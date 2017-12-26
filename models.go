package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

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
	Name         *string                `sql:"not null" json:"name"`
	Description  *string                `json:"description"`
	IsProduction *bool                  `sql:"not null" json:"is_production"`
	SidechainId  *uuid.UUID             `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	Config       map[string]interface{} `sql:"type:json" json:"config"`
}

type Token struct {
	Model
	NetworkId   uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Name        *string   `sql:"not null" json:"name"`
	Symbol      *string   `sql:"not null" json:"symbol"`
	Address     *string   `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress *string   `json:"sale_address"`           // non-null if token sale contract is specified
}

type Transaction struct {
	Model
	NetworkId uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	WalletId  uuid.UUID `sql:"not null;type:uuid" json:"wallet_id"`
	To        *string   `sql:"not null" json:"to"`
	Value     uint64    `sql:"not null;default:0" json:"value"`
	Data      []byte    `json:"data"`
	Hash      *string   `sql:"not null" json:"hash"`
}

type Wallet struct {
	Model
	NetworkId  uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Address    string    `sql:"not null" json:"address"`
	PrivateKey *string   `sql:"not null;type:bytea" json:"-"`
}

// Token

func (t *Token) Create() bool {
	db := DatabaseConnection()

	if !t.Validate() {
		return false
	}

	if db.NewRecord(t) {
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			return rowsAffected > 0
		}
	}
	return false
}

func (t *Token) Validate() bool {
	t.Errors = make([]*Error, 0)
	return len(t.Errors) == 0
}

// Transaction

func (t *Transaction) signEthereumTx(network *Network, wallet *Wallet, cfg *params.ChainConfig) (*types.Transaction, error) {
	client := JsonRpcClient(network)
	syncProgress, err := client.SyncProgress(context.TODO())
	if err == nil {
		block, err := client.BlockByNumber(context.TODO(), nil)
		if err != nil {
			return nil, err
		}
		addr := common.HexToAddress(*t.To)
		nonce, _ := client.PendingNonceAt(context.TODO(), addr)
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		// FIXME-- gasLimit, _ := client.EstimateGas(context.TODO(), tx)
		gasLimit := big.NewInt(DefaultEthereumGasLimit)
		tx := types.NewTransaction(nonce, addr, big.NewInt(int64(t.Value)), gasLimit, gasPrice, t.Data)
		signer := types.MakeSigner(cfg, block.Number())
		hash := signer.Hash(tx).Bytes()
		sig, err := wallet.SignTx(hash)
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			t.Hash = stringOrNil(fmt.Sprintf("%x", signedTx.Hash()))
			Log.Debugf("Signed %s tx for raw broadcast via JSON-RPC: %s", *network.Name, signedTx.String())
			return signedTx, nil
		}
		return nil, err
	} else if syncProgress == nil {
		Log.Debugf("%s JSON-RPC is in sync with the network", *network.Name)
	}
	return nil, err
}

func (t *Transaction) Create() bool {
	if !t.Validate() {
		return false
	}

	db := DatabaseConnection()
	var network = &Network{}
	var wallet = &Wallet{}
	if t.NetworkId != uuid.Nil {
		db.Model(t).Related(&network)
		db.Model(t).Related(&wallet)
	}

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		client, err := DialJsonRpc(network)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host @ %s", *network.Name, DefaultEthereumJsonRpcUrl)
		} else {
			cfg := params.MainnetChainConfig
			tx, err := t.signEthereumTx(network, wallet, cfg)
			if err == nil {
				Log.Debugf("Transmitting signed %s tx to JSON-RPC host @ %s", *network.Name, DefaultEthereumJsonRpcUrl)
				err := client.SendTransaction(context.TODO(), tx)
				if err != nil {
					Log.Warningf("Failed to transmit signed %s tx to JSON-RPC host: %s; %s", *network.Name, DefaultEthereumJsonRpcUrl, err.Error())
				}
			} else {
				Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.Id, err.Error())
			}
		}
	} else {
		Log.Warningf("Unable to generate tx to sign for unsupported network: %s", *network.Name)
	}

	if db.NewRecord(t) {
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			return rowsAffected > 0
		}
	}
	return false
}

func (t *Transaction) Validate() bool {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	t.Errors = make([]*Error, 0)
	if t.NetworkId == uuid.Nil {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Unable to sign tx using unspecified network"),
		})
	} else if t.NetworkId != wallet.NetworkId {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Transaction network did not match wallet network"),
		})
	}
	return len(t.Errors) == 0
}

// Wallet

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

func (w *Wallet) SignTx(msg []byte) ([]byte, error) {
	db := DatabaseConnection()

	var network = &Network{}
	db.Model(w).Related(&network)

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		privateKey, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s", *network.Name, w.Id)
			return nil, err
		}

		Log.Debugf("Signing tx using %s wallet: %s", *network.Name, w.Id)
		sig, err := ethcrypto.Sign(msg, privateKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s; %s", *network.Name, w.Id, err.Error())
			return nil, err
		}
		return sig, nil
	}

	err := fmt.Errorf("Unable to sign tx using unsupported network: %s", *network.Name)
	Log.Warningf(err.Error())
	return nil, err
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
