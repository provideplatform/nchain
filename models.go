package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
)

type Model struct {
	Id        uuid.UUID `sql:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `sql:"not null;default:now()" json:"created_at"`
	Errors    []*Error  `gorm:"-" json:"-"`
}

type Error struct {
	Message *string `json:"message"`
	Status  *int    `json:"status"`
}

type Network struct {
	Model
	Name         *string    `sql:"not null" json:"name"`
	Description  *string    `json:"description"`
	IsProduction *bool      `sql:"not null" json:"is_production"`
	SidechainId  *uuid.UUID `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
}

type Token struct {
	Model
	NetworkId   uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Name        *string   `sql:"not null" json:"name"`
	Symbol      *string   `sql:"not null" json:"symbol"`
	Address     *string   `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress *string   `json:"sale_address"`           // non-null if token sale contract is specified
}

type Wallet struct {
	Model
	NetworkId  uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Address    string    `sql:"not null" json:"address"`
	PrivateKey *string   `sql:"not null;type:bytea" json:"-"`
}

func (w *Wallet) generate(db *gorm.DB, gpgPublicKey string) {
	var network = &Network{}
	db.Model(w).Related(&network)

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		privateKey, err := ethcrypto.GenerateKey()
		if err == nil {
			w.Address = ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			encodedPrivateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
			db.Raw("SELECT pgp_pub_encrypt(?, dearmor(?)) as private_key", encodedPrivateKey, gpgPublicKey).Scan(&w)
			Log.Debugf("Generated Ethereum address: %s", w.Address)
		}
	} else {
		Log.Warningf("Unable to generate private key for wallet using unsupported network: %s", *network.Name)
	}
}

func (w *Wallet) ECDSAPrivateKey(gpgPrivateKey, gpgEncryptionKey string) (*ecdsa.PrivateKey, error) {
	results := make([]byte, 1)
	db := DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as private_key", w.PrivateKey, gpgPrivateKey, gpgEncryptionKey).Rows()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&results)
		privateKeyBytes, err := hex.DecodeString(string(results))
		if err != nil {
			Log.Warningf("Failed to decode ecdsa private key from encrypted storage; %s", err.Error())
			return nil, err
		}
		return ethcrypto.ToECDSA(privateKeyBytes)
	}
	return nil, errors.New("Failed to decode ecdsa private key from encrypted storage")
}

func (w *Wallet) SignTxn(msg []byte, opts crypto.SignerOpts) ([]byte, error) {
	db := DatabaseConnection()

	var network = &Network{}
	db.Model(w).Related(&network)

	privateKey, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
	if err != nil {
		Log.Warningf("Failed to sign txn using %s wallet: %s", *network.Name, w.Id)
		return nil, err
	}

	Log.Debugf("Signing %v-byte txn using %s wallet: %s", len(msg), *network.Name, w.Id)
	asn1, err := privateKey.Sign(rand.Reader, msg, opts)
	if err != nil {
		Log.Warningf("Failed to sign txn using %s wallet: %s; %s", *network.Name, w.Id, err.Error())
		return nil, err
	}
	return asn1, nil
}

func (w *Wallet) Create() bool {
	db := DatabaseConnection()

	w.generate(db, GpgPublicKey)
	if !w.Validate() {
		return false
	}

	if db.NewRecord(w) {
		result := db.Create(&w)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				w.Errors = append(w.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(w) {
			return rowsAffected > 0
		}
	}
	return false
}

func (w *Wallet) Validate() bool {
	w.Errors = make([]*Error, 0)
	_, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &Error{
			Message: &msg,
		})
	}
	return len(w.Errors) == 0
}
