package tx

import (
	"context"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	"github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/contract"
	"github.com/provideapp/nchain/network"
	"github.com/provideapp/nchain/token"
	"github.com/provideapp/nchain/wallet"
	provide "github.com/provideservices/provide-go/api"
	provideapi "github.com/provideservices/provide-go/api/nchain"
	vault "github.com/provideservices/provide-go/api/vault"
	util "github.com/provideservices/provide-go/common/util"
	providecrypto "github.com/provideservices/provide-go/crypto"
)

const defaultDerivedCoinType = uint32(60)
const defaultDerivedChainPath = uint32(0) // i.e., the external or internal chain (also known as change addresses if internal chain)
const firstHardenedChildIndex = uint32(0x80000000)

const DefaultJSONRPCRetries = 3

// Signer interface for signing transactions
type Signer interface {
	Address() string
	Sign(tx *Transaction) (signedTx interface{}, hash []byte, err error)
	String() string
}

// Transaction instances are associated with a signing wallet and exactly one matching instance of either an a) application
// identifier or b) user identifier.
type Transaction struct {
	provide.Model
	NetworkID uuid.UUID `sql:"not null;type:uuid" json:"network_id,omitempty"`

	// Application, organization or user id, if populated, is the entity for which the transaction was custodially signed and broadcast
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	OrganizationID *uuid.UUID `sql:"type:uuid" json:"organization_id,omitempty"`
	UserID         *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`

	// Account or HD wallet which custodially signed the transaction; when an HD wallet is used, if no HD derivation path is provided,
	// the most recently derived non-zero account is used to sign
	AccountID *uuid.UUID `sql:"type:uuid" json:"account_id,omitempty"`
	WalletID  *uuid.UUID `sql:"type:uuid" json:"wallet_id,omitempty"`
	Path      *string    `gorm:"column:hd_derivation_path" json:"hd_derivation_path,omitempty"`

	// Network-agnostic tx fields
	Signer      *string          `sql:"-" json:"signer,omitempty"`
	To          *string          `json:"to"`
	Value       *TxValue         `sql:"not null;type:text" json:"value"`
	Data        *string          `json:"data"`
	Hash        *string          `json:"hash"`
	Status      *string          `sql:"not null;default:'pending'" json:"status"`
	Params      *json.RawMessage `sql:"-" json:"params,omitempty"`
	Ref         *string          `json:"ref"`
	Description *string          `json:"description"`

	// Ephemeral fields for managing the tx/rx and tracing lifecycles
	Response *contract.ExecutionResponse `sql:"-" json:"-"`
	SignedTx interface{}                 `sql:"-" json:"-"`
	Traces   interface{}                 `sql:"-" json:"traces,omitempty"`

	// Transaction metadata/instrumentation
	Block          *uint64    `json:"block"`
	BlockTimestamp *time.Time `json:"block_timestamp,omitempty"`                       // timestamp when the tx was finalized on-chain, according to its tx receipt
	BroadcastAt    *time.Time `json:"broadcast_at,omitempty"`                          // timestamp when the tx was broadcast to the network
	FinalizedAt    *time.Time `json:"finalized_at,omitempty"`                          // timestamp when the tx was finalized on-platform
	PublishedAt    *time.Time `json:"published_at,omitempty"`                          // timestamp when the tx was published to NATS cluster
	QueueLatency   *uint64    `json:"queue_latency,omitempty"`                         // broadcast_at - published_at (in millis) -- the amount of time between when a message is enqueued to the NATS broker and when it is broadcast to the network
	NetworkLatency *uint64    `json:"network_latency,omitempty"`                       // finalized_at - broadcast_at (in millis) -- the amount of time between when a message is broadcast to the network and when it is finalized on-chain
	E2ELatency     *uint64    `gorm:"column:e2e_latency" json:"e2e_latency,omitempty"` // finalized_at - published_at (in millis) -- the amount of time between when a message is published to the NATS broker and when it is finalized on-chain
}

// TransactionSigner is either an account or HD wallet; implements the Signer interface
type TransactionSigner struct {
	DB *gorm.DB

	Network *network.Network

	Account *wallet.Account
	Wallet  *wallet.Wallet
}

var w sync.WaitGroup
var m sync.Mutex

// Address returns the public network address of the underlying Signer
func (txs *TransactionSigner) Address() string {
	var address string
	if txs.Account != nil {
		address = txs.Account.Address
	} else if txs.Wallet != nil {

		// if we have a path provided for the wallet, then we'll validate it and use that to derive the address
		// if we don't have a path, we'll use the default path when deriving an address
		txAddress, _, err := txs.GetSignerDetails()
		if err != nil {
			// TODO sort this not being squashed
			common.Log.Warningf("error obtaining address from transactionsigner. Error: %s", err.Error())
		}
		address = *txAddress
	}

	return address
}

// GetSignerDetails gets the address and derivation path used for wallet signing
func (txs *TransactionSigner) GetSignerDetails() (address, derivationPath *string, err error) {

	// first check if we have a provided path
	// we'll ignore the other branch (no hd wallet path for the moment TODO)
	if txs.Wallet.Path != nil {
		// ensure the path is valid
		_, err := hdwallet.ParseDerivationPath(*txs.Wallet.Path)
		if err != nil {
			err := fmt.Errorf("failed to parse derivation path provided (%s). Error: %s", *txs.Wallet.Path, err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}
		derivationPath = txs.Wallet.Path

		// if we have a valid path, we need to obtain the address associated with that path
		// to do this, we derive a key from the base wallet using this path
		key, err := vault.DeriveKey(util.DefaultVaultAccessJWT, txs.Wallet.VaultID.String(), txs.Wallet.KeyID.String(), map[string]interface{}{
			"hd_derivation_path": *derivationPath,
		})
		if err != nil {
			err := fmt.Errorf("Unable to generate key material for HD wallet for path %s; Error: %s", *derivationPath, err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}
		common.Log.Debugf("address generated using derivation path %s: %s", *derivationPath, *key.Address)
		address = key.Address
	}

	if txs.Wallet.Path == nil {
		key, err := vault.DeriveKey(util.DefaultVaultAccessJWT, txs.Wallet.VaultID.String(), txs.Wallet.KeyID.String(), map[string]interface{}{})
		if err != nil {
			err := fmt.Errorf("unable to generate key material for HD wallet with no derivation path; %s", err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}
		common.Log.Debugf("sequential address generated by hd wallet: %s", *key.Address)
		common.Log.Debugf("derivation path created: %s", *key.HDDerivationPath)

		address = key.Address
		derivationPath = key.HDDerivationPath
	}
	return address, derivationPath, nil
}

func incrementNonce(txAddress, txRef string, txNonce uint64) (*uint64, error) {

	common.Log.Debugf("XXX: Incrementing redis nonce for tx ref %s", txRef)
	var cachedNonce *string
	readErr := redisutil.WithRedlock(txAddress, func() error {
		var err error
		cachedNonce, err = redisutil.Get(txAddress)
		if err != nil {
			return err
		}
		return nil
	})
	if readErr != nil {
		common.Log.Debugf("XXX: Error, possibly nil value for redis nonce (incrementing) for tx ref %s. Error: %s", txRef, readErr.Error())
	}

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis to increment for address: %s, tx ref: %s", txAddress, txRef)
		updatedNonce := txNonce + 1
		lockErr := redisutil.WithRedlock(txAddress, func() error {
			err := redisutil.Set(txAddress, updatedNonce, nil)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}

		return &updatedNonce, nil
		// the evmtxfactory will get the current nonce from the chain
	}
	if cachedNonce != nil {
		common.Log.Debugf("XXX: Nonce of %v found on redis for address: %s, tx ref: %s", *cachedNonce, txAddress, txRef)
		//convert cached Nonce and increment, returning updated nonce for reference
		int64nonce, err := strconv.ParseUint(string(*cachedNonce), 10, 64)
		if err != nil {
			common.Log.Debugf("XXX: Error converting cached nonce to int64 for tx ref: %s. Error: %s", txRef, err.Error())
			return nil, err
		}
		updatedNonce := int64nonce + 1
		common.Log.Debugf("XXX: Incrementing redis nonce for address %s after tx ref %s to %v", txAddress, txRef, updatedNonce)
		lockErr := redisutil.WithRedlock(txAddress, func() error {
			err := redisutil.Set(txAddress, updatedNonce, nil)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}
		common.Log.Debugf("XXX: Nonce incremented for address %s after tx ref %s to %v", txAddress, txRef, updatedNonce)
		return &updatedNonce, nil
		// the evmtxfactory will get the current nonce from the chain
	}

	return nil, nil
}

func generateTx(wg *sync.WaitGroup, txs *TransactionSigner, tx *Transaction, txAddress *string, gas float64, gasPrice *uint64) (types.Signer, *types.Transaction, []byte, error) {
	m.Lock()

	defer func() {
		m.Unlock()
		wg.Done()
	}()

	nonce, err := getNonce(*txAddress, tx, txs)
	if err != nil {
		common.Log.Debugf("Error getting nonce for Address %s, tx ref %s. Error: %s", *txAddress, *tx.Ref, err.Error())
	}

	var signer types.Signer
	var _tx *types.Transaction
	var hash []byte

	err = common.Retry(DefaultJSONRPCRetries, 1*time.Second, func() (err error) {
		signer, _tx, hash, err = providecrypto.EVMTxFactory(
			txs.Network.ID.String(),
			txs.Network.RPCURL(),
			*txAddress,
			tx.To,
			tx.Data,
			tx.Value.BigInt(),
			nonce,
			uint64(gas),
			gasPrice,
		)
		return
	})

	if err != nil {
		// change the message depending on whether we're using a wallet or an account (TODO: correc this)
		if txs.Wallet != nil {
			err = fmt.Errorf("failed to sign %d-byte transaction payload using hardened account for HD wallet: %s; %s", len(hash), txs.Wallet.ID, err.Error())
		} else {
			err = fmt.Errorf("failed to sign %d-byte transaction payload using account ID: %s; %s", len(hash), txs.Account.ID, err.Error())
		}

		common.Log.Debugf("%s", err.Error())
		common.Log.Warning(err.Error())
		return nil, nil, nil, err
	}

	if err == nil {
		_, err = incrementNonce(*txAddress, *tx.Ref, _tx.Nonce())
		if err != nil {
			common.Log.Debugf("XXX: Error incrementing nonce for Address %s, tx ref %s. Error: %s", *txAddress, *tx.Ref, err.Error())
		}
	}

	return signer, _tx, hash, err
}

func getNonce(txAddress string, tx *Transaction, txs *TransactionSigner) (*uint64, error) {

	var nonce *uint64

	var cachedNonce *string
	readErr := redisutil.WithRedlock(txAddress, func() error {
		var err error
		cachedNonce, err = redisutil.Get(txAddress)
		if err != nil {
			return err
		}
		return nil
	})
	if readErr != nil {
		common.Log.Debugf("XXX: Error, possibly nil value for tx ref %s. Error: %s", *tx.Ref, readErr.Error())
	}

	if cachedNonce == nil {
		common.Log.Debugf("XXX: No nonce found on redis for address: %s, tx ref: %s", txAddress, *tx.Ref)
		// get the nonce from the EVM
		common.Log.Debugf("XXX: dialling evm for tx ref %s", *tx.Ref)
		client, err := providecrypto.EVMDialJsonRpc(txs.Network.ID.String(), txs.Network.RPCURL())
		if err != nil {
			return nil, err
		}
		common.Log.Debugf("XXX: Getting nonce from chain for tx ref %s", *tx.Ref)
		// get the last mined nonce, we don't want to rely on the tx pool
		pendingNonce, err := client.NonceAt(context.TODO(), providecrypto.HexToAddress(txAddress), nil)
		if err != nil {
			common.Log.Debugf("XXX: Error getting pending nonce for tx ref %s. Error: %s", *tx.Ref, err.Error())
			return nil, err
		}
		common.Log.Debugf("XXX: Pending nonce found for tx Ref %s. Nonce: %v", *tx.Ref, pendingNonce)
		// put this in redis
		lockErr := redisutil.WithRedlock(txAddress, func() error {
			err := redisutil.Set(txAddress, pendingNonce, nil)
			if err != nil {
				return err
			}
			return nil
		})
		if lockErr != nil {
			return nil, lockErr
		}
		return &pendingNonce, nil
	} else {
		int64nonce, err := strconv.ParseUint(string(*cachedNonce), 10, 64)
		if err != nil {
			common.Log.Debugf("XXX: Error converting cached nonce to int64 for tx ref: %s. Error: %s", *tx.Ref, err.Error())
			return nil, err
		} else {
			common.Log.Debugf("XXX: Assigning nonce of %v to tx ref: %s", int64nonce, *tx.Ref)
			nonce = &int64nonce
		}
	}
	return nonce, nil
}

// Sign implements the Signer interface
func (txs *TransactionSigner) Sign(tx *Transaction) (signedTx interface{}, hash []byte, err error) {
	if tx == nil {
		err := errors.New("cannot sign nil transaction payload")
		common.Log.Warning(err.Error())
		return nil, nil, err
	}

	if txs.Network == nil {
		err := fmt.Errorf("failed to sign %d-byte transaction payload with incorrectly configured signer; no network specified", len(*tx.Data))
		common.Log.Warning(err.Error())
		return nil, nil, err
	}

	if txs.Network.IsEthereumNetwork() {
		params := tx.ParseParams()
		gas, gasOk := params["gas"].(float64)
		if !gasOk {
			gas = float64(0)
		}

		var gasPrice *uint64
		gp, gpOk := params["gas_price"].(float64)
		if gpOk {
			_gasPrice := uint64(gp)
			gasPrice = &_gasPrice
		}

		var nonce *uint64
		if nonceFloat, nonceOk := params["nonce"].(float64); nonceOk {
			nonceUint := uint64(nonceFloat)
			nonce = &nonceUint
		}

		var signer types.Signer
		var _tx *types.Transaction

		// we are using an account to sign the transaction
		if txs.Account != nil && txs.Account.VaultID != nil && txs.Account.KeyID != nil {
			txAddress := common.StringOrNil(txs.Account.Address)

			common.Log.Debugf("XXX: provided nonce of %v for tx ref %s", nonce, *tx.Ref)
			if nonce == nil {
				w.Add(1)
				signer, _tx, hash, err = generateTx(&w, txs, tx, txAddress, gas, gasPrice)
				w.Wait()
			} else {
				// create a signed transaction using the provided nonce
				err = common.Retry(DefaultJSONRPCRetries, 1*time.Second, func() (err error) {
					signer, _tx, hash, err = providecrypto.EVMTxFactory(
						txs.Network.ID.String(),
						txs.Network.RPCURL(),
						*txAddress,
						tx.To,
						tx.Data,
						tx.Value.BigInt(),
						nonce,
						uint64(gas),
						gasPrice,
					)
					return
				})
			}

			// signer, _tx, hash, err = providecrypto.EVMTxFactory(
			// 	txs.Network.ID.String(),
			// 	txs.Network.RPCURL(),
			// 	txs.Account.Address,
			// 	tx.To,
			// 	tx.Data,
			// 	tx.Value.BigInt(),
			// 	nonce,
			// 	uint64(gas),
			// 	gasPrice,
			// )
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using signing account %s; %s", txs.Account.Address, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			sig, err := vault.SignMessage(
				util.DefaultVaultAccessJWT,
				txs.Account.VaultID.String(),
				txs.Account.KeyID.String(),
				fmt.Sprintf("%x", hash),
				map[string]interface{}{},
			)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using signing account %s; %s", txs.Account.Address, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			_sig, err := hex.DecodeString(*sig.Signature)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using signing account %s; %s", txs.Account.Address, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			signedTx, err = _tx.WithSignature(signer, _sig)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using signing account %s; %s", txs.Account.Address, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			if err == nil {
				signedTxJSON, _ := signedTx.(*types.Transaction).MarshalJSON()
				common.Log.Debugf("signed eth tx: %s", signedTxJSON)

				accessedAt := time.Now()
				go func() {
					txs.Account.AccessedAt = &accessedAt
					txs.DB.Save(&txs.Account)
				}()
			}
		}

		if txs.Wallet != nil && txs.Wallet.VaultID != nil && txs.Wallet.KeyID != nil {

			txAddress, txDerivationPath, err := txs.GetSignerDetails()
			if err != nil {
				return nil, nil, err
			}

			common.Log.Debugf("XXX: provided nonce of %v for tx ref %s", nonce, *tx.Ref)
			if nonce == nil {
				w.Add(1)
				signer, _tx, hash, err = generateSignedTx(&w, txs, tx, txAddress, gas, gasPrice)
				w.Wait()
			} else {
				// create a signed transaction using the provided nonce
				err = common.Retry(DefaultJSONRPCRetries, 1*time.Second, func() (err error) {
					signer, _tx, hash, err = providecrypto.EVMTxFactory(
						txs.Network.ID.String(),
						txs.Network.RPCURL(),
						*txAddress,
						tx.To,
						tx.Data,
						tx.Value.BigInt(),
						nonce,
						uint64(gas),
						gasPrice,
					)
					return
				})
			}

			if err != nil {
				err = fmt.Errorf("failed to sign %d-byte transaction payload using hardened account for HD wallet: %s; %s", len(hash), txs.Wallet.ID, err.Error())
				common.Log.Debugf("%s", err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			// if we were provided, or have generated, a hd derivation path, pass it to the signer
			opts := map[string]interface{}{}
			if txDerivationPath != nil {
				opts = map[string]interface{}{
					"hdwallet": map[string]interface{}{
						"hd_derivation_path": txDerivationPath,
					},
				}
			}

			//common.Log.Debugf("vault to sign tx... hash: %s", fmt.Sprintf("%x", hash))
			//check if the hash is actually hex. not sure if it is, it looks like bytes returned from the signer function
			// TODO sometimes the db stores the raw hash (no 0x prefix), not the tx hash (0x prefix) - investigate why this occurs

			sig, err := vault.SignMessage(
				util.DefaultVaultAccessJWT,
				txs.Wallet.VaultID.String(),
				txs.Wallet.KeyID.String(),
				fmt.Sprintf("%x", hash),
				opts,
			)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using hardened account for HD wallet: %s; %s", txs.Wallet.ID, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			_sig, err := hex.DecodeString(*sig.Signature)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction using hardened account for HD wallet: %s; %s", txs.Wallet.ID, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			signedTx, err = _tx.WithSignature(signer, _sig)
			if err != nil {
				err = fmt.Errorf("failed to sign transaction payload using hardened account for HD wallet: %s; %s", txs.Wallet.ID, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}
		}
	} else {
		return nil, nil, fmt.Errorf("unable to generate signed tx for unsupported network: %s", *txs.Network.Name)
	}

	return signedTx, hash, err
}

// String prints a description of the transaction signer
func (txs *TransactionSigner) String() string {
	if txs.Account != nil {
		return fmt.Sprintf("account: %s", txs.Account.ID)
	}

	if txs.Wallet != nil {
		return fmt.Sprintf("HD wallet: %s", txs.Wallet.ID)
	}

	return "(misconfigured tx signer)"
}

// TxValue provides JSON marshaling and gorm driver support for wrapping/unwrapping big.Int
type TxValue struct {
	value *big.Int
}

// NewTxValue is a convenience method to return a TxValue
func NewTxValue(val int64) *TxValue {
	return &TxValue{value: big.NewInt(val)}
}

// Value returns the underlying big.Int as a string for use by the gorm driver (psql)
func (v *TxValue) Value() (driver.Value, error) {
	return v.value.String(), nil
}

// Scan reads the persisted value using the gorm driver and marshals it into a TxValue
func (v *TxValue) Scan(val interface{}) error {
	v.value = new(big.Int)
	if str, ok := val.(string); ok {
		v.value.SetString(str, 10)
	}
	return nil
}

// BigInt returns the value represented as big.Int
func (v *TxValue) BigInt() *big.Int {
	return v.value
}

// MarshalJSON marshals the tx value to bytes
func (v *TxValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON sets the tx value big.Int from its string representation
func (v *TxValue) UnmarshalJSON(data []byte) error {
	v.value = new(big.Int)
	v.value.SetString(string(data), 10)
	return nil
}

func (t *Transaction) asEthereumCallMsg(address string, gasPrice, gasLimit uint64) ethereum.CallMsg {
	var to *ethcommon.Address
	var data []byte
	if t.To != nil {
		addr := ethcommon.HexToAddress(*t.To)
		to = &addr
	}
	if t.Data != nil {
		data = ethcommon.FromHex(*t.Data)
	}
	return ethereum.CallMsg{
		From:     ethcommon.HexToAddress(address),
		To:       to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    t.Value.BigInt(),
		Data:     data,
	}
}

func (t *Transaction) signerFactory(db *gorm.DB) (*TransactionSigner, error) {
	var ntwrk *network.Network
	if t.NetworkID != uuid.Nil {
		ntwrk = &network.Network{}
		db.Model(t).Related(&ntwrk)
	}

	var acct *wallet.Account
	if t.AccountID != nil && *t.AccountID != uuid.Nil {
		acct = &wallet.Account{}
		db.Model(t).Related(&acct)
	}

	var wllt *wallet.Wallet
	if t.WalletID != nil && *t.WalletID != uuid.Nil {
		wllt = &wallet.Wallet{}
		db.Model(t).Related(&wllt)
		if wllt != nil {
			wllt.Path = t.Path
		}
	}

	if ntwrk == nil || ntwrk.ID == uuid.Nil {
		return nil, errors.New("invalid network for tx broadcast")
	}

	if (acct == nil || acct.ID == uuid.Nil) && (wllt == nil || wllt.ID == uuid.Nil) {
		return nil, errors.New("no account or HD wallet signing identity to sign tx for broadcast")
	}

	return &TransactionSigner{
		DB:      db,
		Network: ntwrk,
		Account: acct,
		Wallet:  wllt,
	}, nil
}

// Create and persist a new transaction. Side effects include persistence of contract
// and/or token instances when the tx represents a contract and/or token creation.
func (t *Transaction) Create(db *gorm.DB) bool {
	if !t.Validate() {
		return false
	}

	signer, err := t.signerFactory(db)
	if err != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		return false
	}

	// xxx check what triggers a signingErr here...
	signingErr := t.sign(db, signer)

	if db.NewRecord(t) {
		// last check to make sure we don't violate fk constraints with a nil uuid;
		// if that happens, a transaction will end up on-chain before it we have a
		// local record of it...
		if t.AccountID != nil && *t.AccountID == uuid.Nil {
			t.AccountID = nil
		}
		if t.WalletID != nil && *t.WalletID == uuid.Nil {
			t.WalletID = nil
		}
		common.Log.Debugf("XXX: Create: about to create tx with ref: %v", *t.Ref)
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
			return false
		}

		if !db.NewRecord(t) {
			if rowsAffected > 0 {

				// if we have a signing error, which might be insufficient funds, try bookie
				if signingErr != nil {
					common.Log.Debugf("attepting broadcast of tx ref: %s to bookie due to signing error %s.", *t.Ref, signingErr.Error())
					// network specified in t, so not required to be specifically passed to broadcast method
					bookieBroadcastErr := t.broadcast(db, nil, nil)
					// if bookie fails, we're out
					if bookieBroadcastErr != nil {
						common.Log.Warningf("attepted broadcast of tx ref %s to bookie failed %s", *t.Ref, bookieBroadcastErr.Error())

						t.Errors = append(t.Errors, &provide.Error{
							Message: common.StringOrNil(bookieBroadcastErr.Error()),
						})

						desc := bookieBroadcastErr.Error()
						t.updateStatus(db, "failed", &desc)
						return false
					}
					// if bookie succeeds, pop it onto nats
					if bookieBroadcastErr == nil {
						payload, _ := json.Marshal(map[string]interface{}{
							"transaction_id": t.ID.String(),
						})
						natsutil.NatsStreamingPublish(natsTxReceiptSubject, payload)

						return true
					}
				}

				if signingErr == nil {
					// if no signing error, try regular broadcast
					networkBroadcastErr := t.broadcast(db, signer.Network, signer)
					// if regular fails, we're out
					if networkBroadcastErr != nil {
						common.Log.Warningf("network broadcast failed for tx ref: %s. Error: %s", *t.Ref, networkBroadcastErr.Error())

						t.Errors = append(t.Errors, &provide.Error{
							Message: common.StringOrNil(networkBroadcastErr.Error()),
						})

						desc := networkBroadcastErr.Error()
						t.updateStatus(db, "failed", &desc)
						return false
					}
					// if regular succeeds, pop it onto nats
					if networkBroadcastErr == nil {
						common.Log.Debugf("XXX: tx.Create. Broadcast succeeded for tx ref: %s", *t.Ref)
						payload, _ := json.Marshal(map[string]interface{}{
							"transaction_id": t.ID.String(),
						})
						natsutil.NatsStreamingPublish(natsTxReceiptSubject, payload)

						return true
					}
				}
			}
		}
	}
	return false
}

// GetContract - attempt to resolve the contract associated with the tx execution
func (t *Transaction) GetContract(db *gorm.DB) *contract.Contract {
	var c *contract.Contract
	if t.To != nil {
		c = &contract.Contract{}
		db.Where("network_id = ? AND address = ?", t.NetworkID, t.To).Find(&c)
	} else {
		c = &contract.Contract{}
		db.Where("transaction_id = ?", t.NetworkID, t.ID).Find(&c)
	}
	return c
}

// Validate a transaction for persistence
func (t *Transaction) Validate() bool {
	db := dbconf.DatabaseConnection()
	var wal *wallet.Account
	if t.AccountID != nil {
		wal = &wallet.Account{}
		db.Model(t).Related(&wal)
	}
	t.Errors = make([]*provide.Error, 0)
	if t.ApplicationID != nil && t.UserID != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR user identifier should be provided"),
		})
	} else if t.OrganizationID != nil && t.UserID != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("only an organization OR user identifier should be provided"),
		})
	} else if t.ApplicationID != nil && t.OrganizationID != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR organization identifier should be provided"),
		})
	} else if t.ApplicationID != nil && wal != nil && wal.ApplicationID != nil && *t.ApplicationID != *wal.ApplicationID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("unable to sign tx due to mismatched signing application"),
		})
	} else if t.OrganizationID != nil && wal != nil && wal.OrganizationID != nil && *t.OrganizationID != *wal.OrganizationID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("unable to sign tx due to mismatched signing organization"),
		})
	} else if t.UserID != nil && wal != nil && wal.UserID != nil && *t.UserID != *wal.UserID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("unable to sign tx due to mismatched signing user"),
		})
	}
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("unable to broadcast tx on unspecified network"),
		})
	} else if wal != nil && (t.ApplicationID != nil || t.OrganizationID != nil) && wal.NetworkID != nil && t.NetworkID != *wal.NetworkID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("Transaction network did not match wallet network in application context"),
		})
	}

	if t.Signer != nil {
		if t.AccountID != nil {
			t.Errors = append(t.Errors, &provide.Error{
				Message: common.StringOrNil("provided signer and account_id to tx creation, which is ambiguous"),
			})
		} else {
			account := &wallet.Account{}
			db.Where("address = ?", t.Signer).Find(&account)
			if account == nil || account.ID == uuid.Nil {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil("failed to resolve signer account address to account"),
				})
			}
			t.AccountID = &account.ID
		}
	}

	return len(t.Errors) == 0
}

// Reload the underlying tx instance
func (t *Transaction) Reload() {
	db := dbconf.DatabaseConnection()
	db.Model(&t).Find(t)
}

// GetAccount - retrieve the associated transaction account
func (t *Transaction) GetAccount() (*wallet.Account, error) {
	if t.AccountID == nil {
		return nil, fmt.Errorf("unable to retrieve transaction signing account for tx: %s; no signing account id", t.ID)
	}

	db := dbconf.DatabaseConnection()
	var account = &wallet.Account{}
	db.Model(t).Related(&account)
	if account == nil || account.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to retrieve transaction signing account for tx: %s", t.ID)
	}
	return account, nil
}

// GetNetwork - retrieve the associated transaction network
func (t *Transaction) GetNetwork() (*network.Network, error) {
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(t).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to retrieve transaction network for tx: %s", t.ID)
	}
	return network, nil
}

// GetWallet - retrieve the associated transaction wallet
func (t *Transaction) GetWallet() (*wallet.Wallet, error) {
	if t.WalletID == nil {
		return nil, fmt.Errorf("unable to retrieve transaction signing HD wallet for tx: %s; no HD wallet id", t.ID)
	}

	db := dbconf.DatabaseConnection()
	var wallet = &wallet.Wallet{}
	db.Model(t).Related(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to retrieve transaction signing HD wallet for tx: %s", t.ID)
	}
	wallet.Path = t.Path
	return wallet, nil
}

// ParseParams - parse the original JSON params used when the tx was broadcast
func (t *Transaction) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if t.Params != nil {
		err := json.Unmarshal(*t.Params, &params)
		if err != nil {
			common.Log.Warningf("failed to unmarshal transaction params; %s", err.Error())
			return nil
		}
	}
	return params
}

// shouldSubsidize returns true if the transaction should be subsidized using the integration gas service
func (t *Transaction) shouldSubsidize() bool {
	if subsidize, subsidizeOk := t.ParseParams()["subsidize"].(bool); subsidizeOk {
		return subsidize
	}
	return false
}

func (t *Transaction) updateStatus(db *gorm.DB, status string, description *string) {
	t.Status = common.StringOrNil(status)
	t.Description = description
	result := db.Save(&t)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errors = append(t.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
}

// func (t *Transaction) attemptTxBroadcastRecovery(err error) error {
// 	msg := err.Error()
// 	common.Log.Debugf("Attempting to recover from failed transaction broadcast (tx id: %s); %s", t.ID.String(), msg)

// 	gasFailureStr := "not enough gas to cover minimal cost of the transaction (minimal: "
// 	isGasEstimationRecovery := strings.Contains(msg, gasFailureStr) && strings.Contains(msg, "got: 0") // HACK
// 	if isGasEstimationRecovery {
// 		common.Log.Debugf("Attempting to recover from gas estimation failure with supplied gas of 0 for tx id: %s", t.ID)
// 		offset := strings.Index(msg, gasFailureStr) + len(gasFailureStr)
// 		length := strings.Index(msg[offset:], ",")
// 		minimalGas, err := strconv.ParseFloat(msg[offset:offset+length], 64)
// 		if err == nil {
// 			common.Log.Debugf("Resolved minimal gas of %v required to execute tx: %s", minimalGas, t.ID)
// 			params := t.ParseParams()
// 			params["gas"] = minimalGas
// 			t.setParams(params)
// 			return nil
// 		}
// 		common.Log.Debugf("failed to resolve minimal gas requirement for tx: %s; tx execution unrecoverable", t.ID)
// 	}

// 	return err
// }

func (t *Transaction) broadcast(db *gorm.DB, ntwrk *network.Network, signer Signer) error {
	var err error

	if t.SignedTx == nil || ntwrk == nil {
		params := t.ParseParams()

		var _ntwrk *network.Network
		if t.NetworkID != uuid.Nil {
			_ntwrk = &network.Network{}
			db.Model(t).Related(&_ntwrk)
		}

		if _, networkOk := params["network"].(string); !networkOk {
			params["network"] = _ntwrk.PaymentsNetworkName()
		}

		result, err := common.BroadcastTransaction(t.To, t.Data, params)
		if err != nil {
			return fmt.Errorf("failed to broadcast bookie tx; %s", err.Error())
		}

		t.Hash = result
		db.Save(&t)
		common.Log.Debugf("broadcast tx: %s", *t.Hash)
	} else {
		if ntwrk.IsEthereumNetwork() {
			if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
				common.Log.Debugf("XXX: about to broadcast tx ref: %s", *t.Ref)
				// retry broadcast 3 times if it fails
				err = common.Retry(DefaultJSONRPCRetries, 1*time.Second, func() (err error) {
					err = providecrypto.EVMBroadcastSignedTx(ntwrk.ID.String(), ntwrk.RPCURL(), signedTx)
					return
				})
				if err == nil {
					common.Log.Debugf("broadcast tx ref %s with hash %s", *t.Hash)
					// we have successfully broadcast the transaction
					// so update the db with the received transaction hash
					t.Hash = common.StringOrNil(signedTx.Hash().String())
					db.Save(&t)
					common.Log.Debugf("broadcast tx ref %s with hash %s - saved to db", *t.Hash)
				}
			} else {
				err = fmt.Errorf("unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
			}
		} else {
			err = fmt.Errorf("unable to generate signed tx for unsupported network: %s", *ntwrk.Name)
		}
	}

	if err != nil {
		common.Log.Warningf("failed to broadcast tx ref %s on network %s using %s; %s", *t.Ref, *ntwrk.Name, signer.String(), err.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		desc := err.Error()
		t.updateStatus(db, "failed", &desc)
	} else {
		broadcastAt := time.Now()
		t.BroadcastAt = &broadcastAt
	}

	return err
}

func (t *Transaction) sign(db *gorm.DB, signer Signer) error {
	var err error
	var hash []byte
	t.SignedTx, hash, err = signer.Sign(t)

	if err != nil {
		length := 0
		if t.Data != nil {
			length = len(*t.Data)
		}
		common.Log.Warningf("failed to sign %d-byte tx using on behalf of signer: %s; %s", length, signer, err.Error())
		// t.Errors = append(t.Errors, &provide.Error{
		// 	Message: common.StringOrNil(err.Error()),
		// })
		// desc := err.Error()
		// t.updateStatus(db, "failed", &desc)
	} else {
		hashAsString := hex.EncodeToString(hash)
		t.Hash = common.StringOrNil(hashAsString)

		// ok, this looks wrong, for whatever reason as it's returning just Fe
		//t.Hash = common.StringOrNil(string(ethcommon.FromHex(string(hash))))
	}

	return err
}

func (t *Transaction) fetchReceipt(db *gorm.DB, network *network.Network, signerAddress string) error {
	p2pAPI, err := network.P2PAPIClient()
	if err != nil {
		return err
	}

	if t.Hash == nil {
		return fmt.Errorf("unable to fetch tx receipt for nil tx hash; tx id: %s", t.ID)
	}

	receipt, err := p2pAPI.FetchTxReceipt(signerAddress, *t.Hash)
	if err != nil {
		return err
	}

	common.Log.Debugf("Fetched tx receipt for tx hash: %s", *t.Hash)
	t.Response = &contract.ExecutionResponse{
		Receipt:     receipt,
		Transaction: t,
	}

	err = t.handleTxReceipt(db, network, signerAddress, receipt)
	if err != nil {
		common.Log.Warningf("failed to handle fetched tx receipt for tx hash: %s; %s", *t.Hash, err.Error())
		return err
	}

	traces, traceErr := p2pAPI.FetchTxTraces(*t.Hash)
	if traceErr != nil {
		common.Log.Warningf("failed to fetch tx trace for tx hash: %s; %s", *t.Hash, traceErr.Error())
		return traceErr
	}

	t.Traces = traces
	t.Response.Traces = t.Traces

	return t.handleTxTraces(db, network, signerAddress, traces, receipt)
}

func (t *Transaction) handleTxReceipt(
	db *gorm.DB,
	network *network.Network,
	signerAddress string,
	receipt *provideapi.TxReceipt,
) error {
	if t.To == nil {
		common.Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, *t.Hash, receipt.ContractAddress.Hex())
		params := t.ParseParams()
		contractName := fmt.Sprintf("Contract %s", *common.StringOrNil(receipt.ContractAddress.Hex()))
		if name, ok := params["name"].(string); ok {
			contractName = name
		}
		kontract := &contract.Contract{}
		var tok *token.Token

		tokenCreateFn := func(c *contract.Contract, tokenType, name string, decimals *big.Int, symbol string) (createdToken bool, tokenID uuid.UUID, errs []*provide.Error) {
			common.Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)

			tok = &token.Token{
				ApplicationID:  c.ApplicationID,
				OrganizationID: c.OrganizationID,
				NetworkID:      c.NetworkID,
				ContractID:     &c.ID,
				Type:           common.StringOrNil(tokenType),
				Name:           common.StringOrNil(name),
				Symbol:         common.StringOrNil(symbol),
				Decimals:       decimals.Uint64(),
				Address:        common.StringOrNil(receipt.ContractAddress.Hex()),
			}

			createdToken = tok.Create()
			tokenID = tok.ID
			errs = tok.Errors
			return
		}

		db.Where("transaction_id = ?", t.ID).Find(&kontract)
		common.Log.Debugf("XXX: Searching for contract for txID: %s", t.ID)
		if kontract == nil || kontract.ID == uuid.Nil {
			common.Log.Debugf("XXX could not find contract for tx id: %s, appID: %s, walletID: %s", t.ID, *t.ApplicationID, *t.WalletID)
			ref, err := uuid.FromString(*t.Ref)
			if err != nil {
				common.Log.Debugf("XXX: error converting transaction ref to contract. Error: %s", err.Error())
			}
			kontract = &contract.Contract{
				ApplicationID:  t.ApplicationID,
				OrganizationID: t.OrganizationID,
				NetworkID:      t.NetworkID,
				TransactionID:  &t.ID,
				Name:           common.StringOrNil(contractName),
				Address:        common.StringOrNil(receipt.ContractAddress.Hex()),
				Params:         t.Params,
				Reference:      &ref,
			}
			if kontract.Create() {
				common.Log.Debugf("Created contract %s for %s contract creation tx: %s", kontract.ID, *network.Name, *t.Hash)
				kontract.ResolveTokenContract(db, network, signerAddress, receipt, tokenCreateFn)
			} else {
				common.Log.Warningf("failed to create contract for %s contract creation tx %s", *network.Name, *t.Hash)
			}
		} else {
			common.Log.Debugf("Using previously created contract %s for %s contract creation tx: %s, txID: %s", kontract.ID, *network.Name, *t.Hash, t.ID)
			kontract.Address = common.StringOrNil(receipt.ContractAddress.Hex())
			db.Save(&kontract)
			common.Log.Debugf("XXX: updated contract with address for txID: %s", t.ID)
			kontract.ResolveTokenContract(db, network, signerAddress, receipt, tokenCreateFn)
		}
	}

	return nil
}

func (t *Transaction) handleTxTraces(
	db *gorm.DB,
	network *network.Network,
	signerAddress string,
	traces *provideapi.TxTrace,
	receipt *provideapi.TxReceipt,
) error {
	kontract := t.GetContract(db)
	if kontract == nil || kontract.ID == uuid.Nil {
		common.Log.Debugf("contract not resolved as sender of contract-internal opcode")
		return nil
	}

	artifact := kontract.CompiledArtifact()
	if artifact == nil {
		errmsg := fmt.Sprintf("failed to resolve compiled contract artifact required for contract-internal opcode tracing functionality; contract id: %s", kontract.ID)
		common.Log.Warning(errmsg)
		return errors.New(errmsg)
	}

	for _, result := range traces.Result {
		if result.Type != nil && *result.Type == "create" {
			contractAddr := result.Result.Address
			contractCode := result.Result.Code

			if contractAddr == nil || contractCode == nil {
				common.Log.Warningf("No contract address or bytecode resolved for contract-internal CREATE opcode; tx hash: %s", *t.Hash)
				continue
			}

			resultJSON, _ := json.Marshal(result)
			common.Log.Debugf("Observed contract-internal CREATE opcode resulting in deployed contract at address: %s; tx hash: %s; code: %s; tracing result: %s", *contractAddr, *t.Hash, *contractCode, string(resultJSON))

			dep := kontract.ResolveCompiledDependencyArtifact(*contractCode)
			if dep != nil {
				if dep.Fingerprint != nil {
					common.Log.Debugf("Checking if compiled artifact dependency: %s (fingerprint: %s) is target of contract-internal CREATE opcode at address: %s; tx hash: %s", dep.Name, *dep.Fingerprint, *contractAddr, *t.Hash)
					if strings.HasSuffix(*contractCode, *dep.Fingerprint) {
						common.Log.Debugf("Observed fingerprinted dependency %s as target of contract-internal CREATE opcode at contract address %s; fingerprint: %s; tx hash: %s", dep.Name, *contractAddr, *dep.Fingerprint, *t.Hash)
						params, _ := json.Marshal(map[string]interface{}{
							"compiled_artifact": dep,
						})

						rawParams := json.RawMessage(params)

						internalContract := &contract.Contract{
							ApplicationID:  t.ApplicationID,
							OrganizationID: t.OrganizationID,
							NetworkID:      t.NetworkID,
							ContractID:     &kontract.ID,
							TransactionID:  &t.ID,
							Name:           common.StringOrNil(dep.Name),
							Address:        contractAddr,
							Params:         &rawParams,
						}

						if internalContract.Create() {
							common.Log.Debugf("Created contract %s for %s contract-internal tx: %s", internalContract.ID, *network.Name, *t.Hash)
							internalContract.ResolveTokenContract(db, network, signerAddress, receipt,
								func(c *contract.Contract, tokenType, name string, decimals *big.Int, symbol string) (createdToken bool, tokenID uuid.UUID, errs []*provide.Error) {
									common.Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)

									tok := &token.Token{
										ApplicationID:  c.ApplicationID,
										OrganizationID: c.OrganizationID,
										NetworkID:      c.NetworkID,
										ContractID:     &c.ID,
										Type:           common.StringOrNil(tokenType),
										Name:           common.StringOrNil(name),
										Symbol:         common.StringOrNil(symbol),
										Decimals:       decimals.Uint64(),
										Address:        common.StringOrNil(receipt.ContractAddress.Hex()),
									}

									createdToken = tok.Create()
									tokenID = tok.ID
									errs = tok.Errors

									return createdToken, tokenID, errs
								})
						} else {
							common.Log.Warningf("failed to create contract for %s contract-internal creation tx %s", *network.Name, *t.Hash)
						}
					}
				}
			}
		}
	}

	return nil
}

// RefreshDetails populates transaction details which were not necessarily available upon broadcast, including network-specific metadata and VM execution tracing if applicable
func (t *Transaction) RefreshDetails() error {
	if t.Hash == nil {
		return nil
	}

	network, _ := t.GetNetwork()
	p2pAPI, clientErr := network.P2PAPIClient()
	if clientErr != nil {
		return clientErr
	}

	var err error
	t.Traces, err = p2pAPI.FetchTxTraces(*t.Hash)
	if err != nil {
		common.Log.Warningf("failed to fetch tx trace for tx hash: %s; %s", *t.Hash, err.Error())
		return err
	}

	return nil
}

// setParams sets the tx params in-memory
func (t *Transaction) setParams(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := json.RawMessage(paramsJSON)
	t.Params = &_paramsJSON
}
