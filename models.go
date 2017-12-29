package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethparams "github.com/ethereum/go-ethereum/params"

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
	Name         *string          `sql:"not null" json:"name"`
	Description  *string          `json:"description"`
	IsProduction *bool            `sql:"not null" json:"is_production"`
	SidechainId  *uuid.UUID       `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	Config       *json.RawMessage `sql:"type:json" json:"config"`
}

type Contract struct {
	Model
	NetworkId     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	WalletId      uuid.UUID        `sql:"not null;type:uuid" json:"wallet_id"`
	TransactionId *uuid.UUID       `sql:"type:uuid" json:"transaction_id"` // id of the transaction which created the contract (or null)
	Name          *string          `sql:"not null" json:"name"`
	Address       *string          `sql:"not null" json:"address"` // network-specific token contract address
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

type Token struct {
	Model
	NetworkId      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	ContractId     *uuid.UUID `sql:"type:uuid" json:"contract_id"`
	SaleContractId *uuid.UUID `sql:"type:uuid" json:"sale_contract_id"`
	Name           *string    `sql:"not null" json:"name"`
	Symbol         *string    `sql:"not null" json:"symbol"`
	Address        *string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress    *string    `json:"sale_address"`           // non-null if token sale contract is specified
}

type Transaction struct {
	Model
	NetworkId uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	WalletId  uuid.UUID `sql:"not null;type:uuid" json:"wallet_id"`
	To        *string   `json:"to"`
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

// Network

func (n *Network) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			Log.Warningf("Failed to unmarshal network config; %s", err.Error())
			return nil
		}
	}
	return config
}

// Contract

func (c *Contract) Create() bool {
	if !c.Validate() {
		return false
	}

	db := DatabaseConnection()
	var network = &Network{}
	var wallet = &Wallet{}
	if c.NetworkId != uuid.Nil {
		db.Model(c).Related(&network)
		db.Model(c).Related(&wallet)
	}
	params := c.ParseParams()

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		value := uint64(0)
		data := make([]byte, 0)
		if val, ok := params["value"].(uint64); ok {
			value = val
		}
		if _data, ok := params["data"].(string); ok {
			data = []byte(_data)
		}
		tx := &Transaction{
			NetworkId: network.Id,
			WalletId:  wallet.Id,
			To:        nil, // recipient is nil to indicate contract creation
			Value:     value,
			Data:      data,
		}
		if tx.Create() {
			Log.Debugf("Created %s contract in tx %s", *network.Name, tx.Hash)
			c.TransactionId = &tx.Id

			Log.Warningf("Contract address will remain nil until contract tx receipt is retreived after mining...")
		} else {
			Log.Warningf("Failed to create %s contract due to tx signing or broadcast failure", *network.Name)
			for _, err := range tx.Errors {
				Log.Warningf("Failed %s contract creation tx error: %s", *network.Name, *err.Message)
			}
			return false
		}
	} else {
		Log.Warningf("Unable to generate contract deployment tx for unsupported network: %s", *network.Name)
	}

	if len(c.Errors) > 0 {
		return false
	}

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(c) {
			return rowsAffected > 0
		}
	}
	return false
}

func (c *Contract) Validate() bool {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(c).Related(&wallet)
	c.Errors = make([]*Error, 0)
	if c.Params == nil {
		c.Errors = append(c.Errors, &Error{
			Message: stringOrNil("Unable to create contract without network-specific params"),
		})
	}
	if c.NetworkId == uuid.Nil {
		c.Errors = append(c.Errors, &Error{
			Message: stringOrNil("Unable to create contract using unspecified network"),
		})
	} else if c.NetworkId != wallet.NetworkId {
		c.Errors = append(c.Errors, &Error{
			Message: stringOrNil("Contract network did not match wallet network"),
		})
	}
	return len(c.Errors) == 0
}

func (c *Contract) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if c.Params != nil {
		err := json.Unmarshal(*c.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal contract params; %s", err.Error())
			return nil
		}
	}
	return params
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
	db := DatabaseConnection()
	var contract = &Contract{}
	var saleContract = &Contract{}
	if t.NetworkId != uuid.Nil {
		db.Model(t).Related(&contract)
		db.Model(t).Related(&saleContract)
	}
	t.Errors = make([]*Error, 0)
	if t.NetworkId == uuid.Nil {
		t.Errors = append(t.Errors, &Error{
			Message: stringOrNil("Unable to deploy token contract using unspecified network"),
		})
	} else if contract != nil {
		if t.NetworkId != contract.NetworkId {
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil("Token network did not match token contract network"),
			})
		}
		if t.Address == nil {
			t.Address = contract.Address
		} else if t.Address != nil && t.Address != contract.Address {
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil("Token sale address did not match referenced token contract address"),
			})
		}
	} else if saleContract != nil {
		if t.NetworkId != saleContract.NetworkId {
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil("Token network did not match token sale contract network"),
			})
		}
		if t.SaleAddress == nil {
			t.SaleAddress = saleContract.Address
		} else if t.SaleAddress != nil && t.SaleAddress != saleContract.Address {
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil("Token sale address did not match referenced token sale contract address"),
			})
		}
	}
	return len(t.Errors) == 0
}

// Transaction

func (t *Transaction) asEthereumCallMsg(gasPrice *big.Int) *ethereum.CallMsg {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	var to *common.Address
	if t.To != nil {
		addr := common.HexToAddress(*t.To)
		to = &addr
	}
	return &ethereum.CallMsg{
		From:     common.HexToAddress(wallet.Address),
		To:       to,
		GasPrice: gasPrice,
		Value:    big.NewInt(int64(t.Value)),
		Data:     t.Data,
	}
}

func (t *Transaction) signEthereumTx(network *Network, wallet *Wallet, cfg *ethparams.ChainConfig) (*types.Transaction, error) {
	client := JsonRpcClient(network)
	syncProgress, err := client.SyncProgress(context.TODO())
	if err == nil {
		hdr, err := client.HeaderByNumber(context.TODO(), nil)
		if err != nil {
			return nil, err
		}
		nonce := *wallet.TxCount()
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		gasLimit := big.NewInt(DefaultEthereumGasLimit)
		callMsg := t.asEthereumCallMsg(gasPrice)
		if callMsg != nil {
			gasLimit, err = client.EstimateGas(context.TODO(), *callMsg)
			Log.Warningf("Failed to estimate gas for %s contract invocation; %s", *network.Name, err.Error())
		}
		var tx *types.Transaction
		if t.To != nil {
			addr := common.HexToAddress(*t.To)
			tx = types.NewTransaction(nonce, addr, big.NewInt(int64(t.Value)), gasLimit, gasPrice, t.Data)
		} else {
			tx = types.NewContractCreation(nonce, big.NewInt(int64(t.Value)), gasLimit, gasPrice, t.Data)
		}
		signer := types.MakeSigner(cfg, hdr.Number)
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
			Log.Warningf("Failed to dial %s JSON-RPC host; %s", *network.Name, err.Error())
			t.Errors = append(t.Errors, &Error{
				Message: stringOrNil(err.Error()),
			})
		} else {
			cfg := GetChainConfig(network)
			tx, err := t.signEthereumTx(network, wallet, cfg)
			if err == nil {
				Log.Debugf("Transmitting signed %s tx to JSON-RPC host", *network.Name)
				err := client.SendTransaction(context.TODO(), tx)
				if err != nil {
					Log.Warningf("Failed to transmit signed %s tx to JSON-RPC host; %s", *network.Name, err.Error())
					t.Errors = append(t.Errors, &Error{
						Message: stringOrNil(err.Error()),
					})
				} else {
					// queue job to check on tx receipt
				}
			} else {
				Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.Id, err.Error())
				t.Errors = append(t.Errors, &Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
	} else {
		Log.Warningf("Unable to generate tx to sign for unsupported network: %s", *network.Name)
	}

	if len(t.Errors) > 0 {
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

func (w *Wallet) TxCount() (count *uint64) {
	db := DatabaseConnection()
	db.Model(&Transaction{}).Where("wallet_id = ?", w.Id).Count(&count)
	return count
}
