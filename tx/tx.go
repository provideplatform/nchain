package tx

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	"github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	stan "github.com/nats-io/stan.go"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/contract"
	"github.com/provideplatform/nchain/network"
	"github.com/provideplatform/nchain/token"
	"github.com/provideplatform/nchain/wallet"
	provide "github.com/provideplatform/provide-go/api"
	provideapi "github.com/provideplatform/provide-go/api/nchain"
	vault "github.com/provideplatform/provide-go/api/vault"
	util "github.com/provideplatform/provide-go/common/util"
	providecrypto "github.com/provideplatform/provide-go/crypto"
)

// const defaultDerivedCoinType = uint32(60)
// const defaultDerivedChainPath = uint32(0) // i.e., the external or internal chain (also known as change addresses if internal chain)
// const firstHardenedChildIndex = uint32(0x80000000)

const DefaultJSONRPCRetries = 3

// chaosMonkeyCounter used to trigger error scenarios for testing
var chaosMonkeyCounter = 0

// Signer interface for signing transactions
type Signer interface {
	Address() (*string, error)
	Sign(tx *Transaction) (signedTx interface{}, hash []byte, err error)
	String() string
}

// idempotency variables
var AttemptReprocess bool

type Parameters struct {
	ContractID  *uuid.UUID `sql:"-" json:"contract_id,omitempty"`
	Data        *string    `sql:"-" json:"data,omitempty"`
	AccountID   *uuid.UUID `sql:"-" json:"account_id,omitempty"`
	WalletID    *uuid.UUID `sql:"-" json:"wallet_id,omitempty"`
	Path        *string    `sql:"-" json:"hd_derivation_path,omitempty"`
	Value       *float64   `sql:"-" json:"value,omitempty"`
	PublishedAt *string    `sql:"-" json:"published_at,omitempty"`
	GasPrice    *float64   `sql:"-" json:"gas_price,omitempty"`
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

	// Ethereum specific nonce fields
	Nonce       *uint64            `gorm:"column:nonce" json:"nonce,omitempty"`
	EthSigner   *TransactionSigner `sql:"-" json:"eth_signer,omitempty"` // TODO remove (marshal generic signer instead)
	Message     *stan.Msg          `sql:"-" json:"nats_msg,omitempty"`   //HACK
	ReBroadcast bool               `sql:"-" json:"rebroadcast,omitempty"`

	// parameter fields as struct (todo replace params above?)
	Parameters *Parameters `sql:"-" json:"parameters,omitempty"`

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

// getAddressIdentifier converts the tx account/wallet+path
// into a unique identifier for that account
// rather than relying on having the signer succeed to get
// the account address before batching up txs
// returns hex encoded value
func (t *Transaction) getAddressIdentifier() *string {

	if t.AccountID != nil {
		// hash the account id and return
		accountid := *t.AccountID
		identifier := crypto.Keccak256Hash(accountid.Bytes())
		retval := common.StringOrNil(identifier.Hex())
		return retval
	}

	if t.WalletID != nil {
		walletid := *t.WalletID
		identifierStr := walletid.String()

		// get the optional path
		if t.Parameters != nil {
			path := t.Parameters.Path
			if path != nil {
				identifierStr = fmt.Sprintf("%s:%s", identifierStr, *path)
			}
		}
		// hash up the walletid + optional path and return
		identifier := crypto.Keccak256Hash([]byte(identifierStr))
		retval := common.StringOrNil(identifier.Hex())
		return retval
	}
	return nil
}

// Address returns the public network address of the underlying Signer
func (txs *TransactionSigner) Address() (*string, error) {
	var address *string
	if txs.Account != nil {
		address = common.StringOrNil(txs.Account.Address)
	} else if txs.Wallet != nil {

		// if we have a path provided for the wallet, then we'll validate it and use that to derive the address
		// if we don't have a path, we'll use the default path when deriving an address
		var err error
		address, _, err = txs.GetSignerDetails()
		if err != nil {
			// TODO sort this not being squashed
			common.Log.Warningf("error obtaining address from transactionsigner. Error: %s", err.Error())
			return nil, err
		}
	}

	return address, nil
}

// GetSignerDetails gets the address and derivation path used for wallet signing
func (txs *TransactionSigner) GetSignerDetails() (address, derivationPath *string, err error) {
	if txs.Account != nil {
		address := common.StringOrNil(txs.Account.Address)
		derivationPath := common.StringOrNil("")
		return address, derivationPath, nil
	}

	// first check if we have a provided path
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

func getCurrentValue(key string) *uint64 {
	var currentValue *uint64
	var cachedValue *string

	start := time.Now()

	cachedValue, _ = redisutil.Get(key)

	if cachedValue == nil {
		common.Log.Debugf("TIMING: No value found for key %s", key)
		return nil
	}

	if cachedValue != nil {
		cv, err := strconv.ParseUint(string(*cachedValue), 10, 64)
		if err != nil {
			common.Log.Debugf("TIMING: Error parsing current value %s from key %s. Error: %s", *cachedValue, key, err.Error())
			// handle this
			return nil
		}
		common.Log.Debugf("TIMING: Got current value %v from key %s.", cv, key)
		currentValue = &cv
	}

	elapsed := time.Since(start)
	common.Log.Debugf("TIMING: Getting value %v for key %s took %s", *currentValue, key, elapsed)
	return currentValue
}

func setCurrentValue(key string, value *uint64) *uint64 {
	common.Log.Debugf("TIMING: Setting value %v to key %s", *value, key)
	setErr := redisutil.WithRedlock(key, func() error {
		// TODO parameterise timeout (10 minutes)
		var ttl time.Duration = 10 * time.Minute
		err := redisutil.Set(key, *value, &ttl)
		if err != nil {
			return err
		}
		return nil
	})
	if setErr != nil {
		// handle this error!
	}
	common.Log.Debugf("TIMING: Set value %v to key %s", *value, key)
	return value
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

	var sig *vault.SignResponse

	if !txs.Network.IsEthereumNetwork() {
		return nil, nil, fmt.Errorf("unable to generate signed tx for tx ref %s for unsupported network: %s", *tx.Ref, *txs.Network.Name)
	}

	if txs.Network.IsEthereumNetwork() {

		params := tx.ParseParams()
		gas, gasOk := params["gas"].(float64)
		if !gasOk {
			gas = float64(0)
		}

		var gasPrice *uint64
		gp := tx.Parameters.GasPrice
		if gp == nil {
			gasprice := uint64(0)
			gasPrice = &gasprice
		} else {
			gasprice := uint64(*gp)
			gasPrice = &gasprice
		}

		var signer types.Signer
		var _tx *types.Transaction
		var signingWithAccount bool
		var signingWithWallet bool

		if txs.Account != nil && txs.Account.VaultID != nil && txs.Account.KeyID != nil {
			common.Log.Debugf("signing with account")
			signingWithAccount = true
		}

		if txs.Wallet != nil && txs.Wallet.VaultID != nil && txs.Wallet.KeyID != nil {
			common.Log.Debugf("signing with wallet")
			signingWithWallet = true
		}

		txAddress, txDerivationPath, err := txs.GetSignerDetails()
		if err != nil {
			common.Log.Debugf("error getting signer details for tx ref %s. Error: %s", *tx.Ref, err.Error())
			return nil, nil, err
		}

		common.Log.Debugf("Provided nonce of %v for tx ref %s", *tx.Nonce, *tx.Ref)
		signer, _tx, hash, err = providecrypto.EVMTxFactory(
			txs.Network.ID.String(),
			txs.Network.RPCURL(),
			*txAddress,
			tx.To,
			tx.Data,
			tx.Value.BigInt(),
			tx.Nonce,
			uint64(gas),
			gasPrice,
		)

		if err != nil {
			err = fmt.Errorf("failed to create raw tx for address %s; %s", *txAddress, err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}

		// we are signing the raw transaction with a vault account id
		if signingWithAccount {
			sig, err = vault.SignMessage(
				util.DefaultVaultAccessJWT,
				txs.Account.VaultID.String(),
				txs.Account.KeyID.String(),
				fmt.Sprintf("%x", hash),
				map[string]interface{}{},
			)

			if err != nil {
				err = fmt.Errorf("failed to sign transaction ref %s using address %s; %s", *tx.Ref, *txAddress, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}
		}

		// we are signing the raw transaction with a vault wallet (+optional derivation path)
		if signingWithWallet {
			// if we were given a hd derivation path, pass it to the signer
			opts := map[string]interface{}{}
			if txDerivationPath != nil {
				opts = map[string]interface{}{
					"hdwallet": map[string]interface{}{
						"hd_derivation_path": txDerivationPath,
					},
				}
			}

			sig, err = vault.SignMessage(
				util.DefaultVaultAccessJWT,
				txs.Wallet.VaultID.String(),
				txs.Wallet.KeyID.String(),
				fmt.Sprintf("%x", hash),
				opts,
			)

			if err != nil {
				err = fmt.Errorf("failed to sign transaction ref %s using address %s; %s", *tx.Ref, *txAddress, err.Error())
				common.Log.Warning(err.Error())
				return nil, nil, err
			}
		}

		sigBytes, err := hex.DecodeString(*sig.Signature)
		if err != nil {
			err = fmt.Errorf("failed to sign transaction ref %s using address %s; %s", *tx.Ref, *txAddress, err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}

		signedTx, err = _tx.WithSignature(signer, sigBytes)
		if err != nil {
			err = fmt.Errorf("failed to sign transaction ref %s using address %s; %s", *tx.Ref, *txAddress, err.Error())
			common.Log.Warning(err.Error())
			return nil, nil, err
		}

		if err == nil {
			accessedAt := time.Now()
			go func() {
				// TODO check this, tx.account can have the wallet id and derivation path
				if signingWithAccount {
					txs.Account.AccessedAt = &accessedAt
					txs.DB.Save(&txs.Account)
				}
			}()
		}
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

func (t *Transaction) SignRawTransaction(db *gorm.DB, nonce *uint64, signer *TransactionSigner) error {

	start := time.Now()

	chaosMonkeyCounter++
	// if chaosMonkeyCounter%5 == 0 {
	// 	chaosErr := fmt.Errorf("chaos monkey error for tx ref %s", *t.Ref)
	// 	return chaosErr
	// }

	var hash []byte
	var err error
	t.SignedTx, hash, err = signer.Sign(t)
	if err != nil {
		return err
	}
	hashAsString := hex.EncodeToString(hash)
	t.Hash = common.StringOrNil(hashAsString)

	elapsed := time.Since(start)
	common.Log.Debugf("TIMING: Signing raw tx for tx ref %s took %s", *t.Ref, elapsed)
	return nil
}

func (t *Transaction) BroadcastSignedTransaction(db *gorm.DB, signer *TransactionSigner) error {
	start := time.Now()
	networkBroadcastErr := t.broadcast(db, signer.Network, signer)
	// if regular succeeds, pop it onto nats
	if networkBroadcastErr == nil {
		common.Log.Debugf("Broadcast to %s network succeeded for tx ref: %s", *signer.Network.Name, *t.Ref)
		payload, _ := json.Marshal(map[string]interface{}{
			"transaction_id": t.ID.String(),
		})
		common.Log.Debugf("Setting tx ref %s status to BROADCAST", *t.Ref)
		broadcastAt := time.Now()
		t.BroadcastAt = &broadcastAt
		t.updateStatus(db, "broadcast", nil)
		natsutil.NatsStreamingPublish(natsTxReceiptSubject, payload)
		elapsed := time.Since(start)
		common.Log.Debugf("TIMING: Broadcasting signed tx for tx ref %s took %s", *t.Ref, elapsed)
		return nil
	} else {
		common.Log.Warningf("Broadcast to %s network failed for tx ref: %s. Error: %s", *signer.Network.Name, *t.Ref, networkBroadcastErr.Error())

		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(networkBroadcastErr.Error()),
		})

		desc := networkBroadcastErr.Error()
		t.updateStatus(db, "failed", &desc)
		return networkBroadcastErr
	}
}

func (t *Transaction) getSequenceCounter(idempotentKey, network, address string) *uint64 {

	var counter uint64

	// first get the sequence counter for this network address
	sequenceKey := fmt.Sprintf("nchain.tx.sequence.%s:%s", network, address)

	if txSequencer.Has(sequenceKey) {
		counter = txSequencer.Get(sequenceKey).(uint64)
		common.Log.Debugf("IDEMPOTENT: Have sequence counter of %v for address of tx ref %s", counter, *t.Ref)
	} else {
		zero := new(uint64)
		txSequencer.Set(sequenceKey, *zero)
		counter = *zero
		common.Log.Debugf("IDEMPOTENT: Have sequence counter of %v for address of tx ref %s", counter, *t.Ref)
	}

	// then check if we have a counter already in the register
	if txRegister.Has(idempotentKey) {
		counter = txRegister.Get(idempotentKey).(uint64)
		common.Log.Debugf("IDEMPOTENT: Have idempotent key (%v)for tx ref %s", counter, *t.Ref)
	} else {
		common.Log.Debugf("IDEMPOTENT: Setting idempotent key to (%v)for tx ref %s", counter, *t.Ref)
		// set the idempotent key to the original sequence counter
		txRegister.Set(idempotentKey, counter)
		// increment the sequence counter
		atomicCounter := counter
		txSequencer.Set(sequenceKey, atomic.AddUint64(&atomicCounter, 1))
	}
	common.Log.Debugf("IDEMPOTENT: Returning sequence counter of %v for tx ref %s", counter, *t.Ref)
	return &counter
}

func (t *Transaction) attemptBroadcast(db *gorm.DB) error {

	// note this will use the t.Nonce if it has one
	// we will need to make sure that it sticks with this
	// because effectively a reprocess means keeping that nonce
	// even if it wasn't user-created
	//common.Log.Debugf("TIMINGNANO: about to get nonce for tx ref: %s at %d", *t.Ref, time.Now().UnixNano())
	nonce, signer, err := t.getNonce(db)
	if err != nil {
		common.Log.Debugf("error getting nonce for tx ref %s. Error: %s", *t.Ref, err.Error())
		return err
	}

	address := t.getAddressIdentifier()
	network := signer.Network.ID.String()

	idempotentKey := fmt.Sprintf("nchain.tx.listing.%s:%s:%s", network, *address, *t.Ref)

	// check if we have to try to broadcast this tx
	if t.ContinueWithBroadcast(idempotentKey) {

		var inChan chan BroadcastConfirmation
		var outChan chan BroadcastConfirmation

		var chanKey *string
		var prevChanKey *string

		// get the sequenceCounter
		// for this address (if new)
		// or this tx (if not new)
		counter := t.getSequenceCounter(idempotentKey, network, *address)

		if *counter == 0 {
			// first tx for this address
			chanKey = common.StringOrNil(fmt.Sprintf("%s.%s:%s:%v", channelKey, network, *address, *counter))
			prevChanKey = nil
			common.Log.Debugf("tx ref: %s counter %v, idempotentKey: %s chankey: %s, prev chankey nil", *t.Ref, *counter, idempotentKey, *chanKey)
		} else {
			// not first tx for this address
			chanKey = common.StringOrNil(fmt.Sprintf("%s.%s:%s:%v", channelKey, network, *address, *counter))
			prevChanKey = common.StringOrNil(fmt.Sprintf("%s.%s:%s:%v", channelKey, network, *address, *counter-1))
			common.Log.Debugf("tx ref: %s counter %v, idempotentKey: %s chankey: %s, prev chankey %s", *t.Ref, *counter, idempotentKey, *chanKey, *prevChanKey)
		}

		if prevChanKey != nil && txChannels.Has(*prevChanKey) {
			// if we have an OUT channel for the previous counter
			// use it as an IN channel for this tx
			inChan = txChannels.Get(*prevChanKey).(channelPair).outgoing
			// TODO if this is closed, we need to recreate it
			common.Log.Debugf("using prev chan for tx ref: %s. inchan: %+v", *t.Ref, inChan)
		} else {
			// otherwise make a new inbound channel for broadcast go ahead
			inChan = make(chan BroadcastConfirmation)
			common.Log.Debugf("making new chan for tx ref: %s. inchan: %+v", *t.Ref, inChan)
		}

		var chanPair channelPair

		oc := txChannels.Get(*chanKey)
		if oc == nil {
			common.Log.Debugf("tx ref %s: nil outbound channel for chan key %s", *t.Ref, *chanKey)
			// we need to make a new tx channels dictionary entry for this nonce
			outChan = make(chan BroadcastConfirmation)
		} else {
			common.Log.Debugf("tx ref %s: outbound channel %+v for chan key %s", *t.Ref, oc.(channelPair).outgoing, *chanKey)
			// we will use the existing tx channels dictionary entry for this nonce
			outChan = oc.(channelPair).outgoing
		}

		chanPair = channelPair{
			inChan,
			outChan,
		}

		common.Log.Debugf("tx ref: %s. chankey: %s inchan %+v, outchan: %+v. chanpair: %+v", *t.Ref, *chanKey, chanPair.incoming, chanPair.outgoing, chanPair)
		// add the IN and OUT channel pair to this tx dictionary
		txChannels.Set(*chanKey, chanPair)

		common.Log.Debugf("TIMINGNANO: tx ref %s chankey %s, signer %v, nonce %v", *t.Ref, *chanKey, signer, nonce)

		broadcast := make(chan bool)
		go t.SignAndReadyForBroadcast(txChannels.Get(*chanKey), signer, db, nonce, broadcast)

		common.Log.Debugf("!!!: waiting for sign and broadcast ok for tx ref: %s", *t.Ref)
		txBroadcast := <-broadcast
		common.Log.Debugf("!!!: got sign and broadcast %v for tx ref: %s", txBroadcast, *t.Ref)
		if !txBroadcast {
			broadcastError := fmt.Errorf("error signing and broadcasting tx ref: %s", *t.Ref)
			return broadcastError
		}

		// todo add channel to capture bool for continue
		// if true, we have signed etc, and should attempt broadcast
		// if false, we have failed to sign, and should exit with an error
		// and wait for the rebroadcast to catch it

		// if this is the first tx for this network address, or we've previously broadcast it, give it the broadcast go straight away
		if prevChanKey == nil {
			if !t.ReBroadcast {
				goForBroadcast := BroadcastConfirmation{address, &network, nonce, false, false}
				common.Log.Debugf("TIMING: FIRST: giving broadcast go-ahead for address %s nonce %v. chan %+v", *address, *nonce, txChannels.Get(*chanKey).(channelPair).incoming)
				txChannels.Get(*chanKey).(channelPair).incoming <- goForBroadcast
			}
			// once it's received, close the channel
			// TODO close channels for this chankey sequence when we get the receipt confirmation
			// channels are cheap?
			// close(txChannels.Get(*chanKey).(channelPair).incoming)
		}

		return nil
	} // continue with broadcast

	return nil
}

func (t *Transaction) SignAndReadyForBroadcast(channels interface{}, signer *TransactionSigner, db *gorm.DB, nonce *uint64, broadcast chan bool) {
	var currentBroadcastNonceKey string

	err := t.SignRawTransaction(db, nonce, signer)
	if err != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		common.Log.Debugf("!!!: Error signing raw tx for tx ref %s. Error: %s", *t.Ref, err.Error())
		errDesc := fmt.Sprintf("signing error: %s", err.Error())

		// only set tx status to failed if it's not a rebroadcast attempt
		if !t.ReBroadcast {
			t.updateStatus(db, "failed", &errDesc)
		}
		broadcast <- false
		return
	} else {
		common.Log.Debugf("Raw tx signed for tx ref %s", *t.Ref)
	}

	// save the nonce and status to the database
	t.Nonce = nonce
	// only set tx status to ready if it's not a rebroadcast attempt
	if !t.ReBroadcast {
		t.updateStatus(db, "ready", nil)
	}

	common.Log.Debugf("!!!: sign and broadcast ok for tx ref: %s", *t.Ref)
	broadcast <- true
	common.Log.Debugf("!!!: gave sign and broadcast ok for tx ref: %s", *t.Ref)

	address := t.getAddressIdentifier()
	network := signer.Network.ID.String()
	currentBroadcastNonceKey = fmt.Sprintf("%s:%s:%s", currentBroadcastNonce, *address, network)
	var goForBroadcast BroadcastConfirmation
	if !t.ReBroadcast {

		common.Log.Debugf("TIMING IN: waiting for go ahead for tx ref %s for key %s nonce %v, on channel %+v", *t.Ref, currentBroadcastNonceKey, *nonce, channels.(channelPair).incoming)
		// BLOCK until channel read
		goForBroadcast = <-channels.(channelPair).incoming
		common.Log.Debugf("TIMING: received go ahead for tx ref %s on incoming channel %+v of %+v", *t.Ref, channels.(channelPair).incoming, goForBroadcast)

		if goForBroadcast.nonce == nil {
			common.Log.Debugf("ERRRRRROOOOOOOOOOOR. nil channel nonce for tx ref %s", *t.Ref)
			zero := uint64(0)
			goForBroadcast.nonce = &zero
		}

		if t.Nonce == nil {
			common.Log.Debugf("ERRRRRROOOOOOOOOOOR. nil nonce for tx ref %s", *t.Ref)
			// set nonce to 0 and attempt to fix in the broadcast
			zero := uint64(0)
			t.Nonce = &zero
		}

		//check if the nonce specified in the channel is different to the one we're currently using
		// if it's different, re-sign the tx
		if *goForBroadcast.nonce != *nonce {
			common.Log.Debugf("updated nonce for tx ref %s to %v", *t.Ref, *goForBroadcast.nonce)
			t.Nonce = goForBroadcast.nonce
			err := t.SignRawTransaction(db, t.Nonce, signer)
			if err != nil {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
				common.Log.Debugf("!!!: Error re-signing raw tx with new nonce for tx ref %s. Error: %s", *t.Ref, err.Error())
				errDesc := fmt.Sprintf("signing error updating nonce: %s", err.Error())
				t.updateStatus(db, "failed", &errDesc)
				broadcast <- false
				return
			}
			common.Log.Debugf("Updated nonce to %v for tx ref %s", *t.Nonce, *t.Ref)
		}
	} else {
		common.Log.Debugf("Rebroadcast of tx ref %s. No go ahead waited on", *t.Ref)
	}

	// broadcast repeatedly until it succeeds
	for {
		err = t.broadcast(db, signer.Network, signer)
		if err == nil {
			//common.Log.Debugf("TIMING: tx ref %s key %s broadcast tx successfully with nonce %v", *t.Ref, currentBroadcastNonceKey, *t.Nonce)
			payload, _ := json.Marshal(map[string]interface{}{
				"transaction_id": t.ID.String(),
			})
			common.Log.Debugf("setting tx ref %s status to BROADCAST", *t.Ref)
			broadcastAt := time.Now()
			t.BroadcastAt = &broadcastAt
			t.updateStatus(db, "broadcast", nil)
			natsutil.NatsStreamingPublish(natsTxReceiptSubject, payload)
			// update the db record with the updated details (currently just nonce, but remove any fail desc and update status)
			break
		}
		if err != nil {
			// if we're reprocessing this
			// status broadcast
			// - ignore already known errors
			common.Log.Debugf("Error broadcasting tx ref %s. Error: %s", *t.Ref, err.Error())
			// check for nonce too low and fix
			// TODO add manualnonce check - do not alter if nonce has been applied manually
			if NonceTooLow(err) && !t.ReBroadcast {
				common.Log.Debugf("Nonce too low error for tx ref: %s", *t.Ref)
				// get the address for the signer
				signerAddress, err := signer.Address()
				if err != nil {
					t.Errors = append(t.Errors, &provide.Error{
						Message: common.StringOrNil(err.Error()),
					})
				} else {
					// get the last mined nonce
					t.Nonce, _ = t.getLastMinedNonce(*signerAddress)
					// and re-sign transaction with this nonce
					err := t.SignRawTransaction(db, t.Nonce, signer)
					if err != nil {
						t.Errors = append(t.Errors, &provide.Error{
							Message: common.StringOrNil(err.Error()),
						})
						common.Log.Debugf("!!!: Error re-signing raw tx (nonce too low) for tx ref %s. Error: %s", *t.Ref, err.Error())
						errDesc := fmt.Sprintf("signing error re-signing nonce too low tx: %s", err.Error())
						t.updateStatus(db, "failed", &errDesc)
						broadcast <- false
						return
					}
					common.Log.Debugf("signed raw tx for tx ref %s", *t.Ref)
				}
			} //NonceTooLow

			if NonceTooLow(err) && t.ReBroadcast {
				common.Log.Debugf("Nonce too low error rebroadcast error for tx ref: %s", *t.Ref)
				break
			} //NonceTooLow

			// TODO parameterise and cap the gas price
			if UnderPriced(err) && !t.ReBroadcast {
				common.Log.Debugf("!!!: Under Priced error for tx ref: %s", *t.Ref)
				updatedGasPrice := float64(0)
				if t.Parameters.GasPrice != nil {
					//common.Log.Debugf("tx ref %s under priced. gas price: %d", *t.Ref, *t.Parameters.GasPrice)
					updatedGasPrice = *t.Parameters.GasPrice + 1000000000 //TODO min/max gas price
					//common.Log.Debugf("tx ref %s under priced. new gas price: %d", *t.Ref, *t.Parameters.GasPrice)
				} else {
					//common.Log.Debugf("tx ref %s under priced. no gas price specified", *t.Ref)
					updatedGasPrice = float64(1000000000) //TODO min/max gas price
				}

				t.Parameters.GasPrice = &updatedGasPrice
				//common.Log.Debugf("re-signing tx ref %s with gas %v", *t.Ref, *t.Parameters.GasPrice)
				// and re-sign transaction with this updated gas price
				err := t.SignRawTransaction(db, t.Nonce, signer)
				if err != nil {
					t.Errors = append(t.Errors, &provide.Error{
						Message: common.StringOrNil(err.Error()),
					})
					common.Log.Debugf("!!!: Error re-signing raw tx (underpriced) for tx ref %s. Error: %s", *t.Ref, err.Error())
					errDesc := fmt.Sprintf("signing error under priced tx: %s", err.Error())
					t.updateStatus(db, "failed", &errDesc)
					broadcast <- false
					return
				}
				common.Log.Debugf("Underpriced tx. re-Signed raw tx for tx ref %s", *t.Ref)
			} //UnderPriced

			if UnderPriced(err) && t.ReBroadcast {
				common.Log.Debugf("!!!: Under Priced rebroadcast error for tx ref: %s", *t.Ref)
				break
			} //UnderPriced

			// this is ok, we're just refreshing the broadcast to ensure it's in the mempool
			// TODO this will get extended for the gas pricing increases when it goes in
			if AlreadyKnown(err) {
				common.Log.Debugf("Already known error for tx ref: %s. Requesting new receipt.", *t.Ref)
				payload, _ := json.Marshal(map[string]interface{}{
					"transaction_id": t.ID.String(),
				})
				natsutil.NatsStreamingPublish(natsTxReceiptSubject, payload)
				break
			} // AlreadyKnown

		} //broadcast error check
	} //for (broadcast loop)

	setCurrentValue(currentBroadcastNonceKey, t.Nonce)

	if !t.ReBroadcast {
		// now broadcast the goahead for the next transaction
		nextNonce := *t.Nonce + 1
		common.Log.Debugf("TIMING: OUT tx ref %s giving broadcast go-ahead for key %s nonce %v on channel %+v", *t.Ref, currentBroadcastNonceKey, nextNonce, channels.(channelPair).outgoing)
		goForBroadcast = BroadcastConfirmation{address, &network, &nextNonce, true, true}

		// below blocks until it is read
		channels.(channelPair).outgoing <- goForBroadcast

		common.Log.Debugf("TIMING: OUT tx ref %s broadcast go-ahead  delivered for address %s nonce %v on channel %+v", *t.Ref, *address, nextNonce, channels.(channelPair).outgoing)
	} else {
		common.Log.Debugf("IDEMPOTENT: rebroadcast tx ref %s. No go-ahead published", *t.Ref)
	}
	// close the outgoing channel once it is read
	// TODO this should also timeout and clear itself up from the dictionary
	// TODO close these when we get the receipt confirmation
	// close(channels.(channelPair).outgoing)

}

// Create and persist a new transaction. Side effects include persistence of contract
// and/or token instances when the tx represents a contract and/or token creation.
func (t *Transaction) Create(db *gorm.DB) bool {
	//common.Log.Debugf("TIMINGNANO: about to validate tx ref: %s at %d", *t.Ref, time.Now().UnixNano())

	if !t.Validate() {
		return false
	}

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

		result := db.Create(&t)

		rowsAffected := result.RowsAffected
		if rowsAffected == 0 {
			common.Log.Debugf("No record added to db for tx ref %s", *t.Ref)
			return false
		}

		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
				// TODO display all errors
				common.Log.Debugf("Error creating tx in db for tx ref %s. Last error: %s", *t.Ref, err.Error())
			}
			return false
		}
		common.Log.Debugf("Attempting to broadcast NEW tx %s", *t.Ref)
	} else {
		common.Log.Debugf("Attempting to broadcast EXISTING tx %s", *t.Ref)
	}

	// attempt to broadcast it, whether it's in the db or not
	err := t.attemptBroadcast(db)
	if err != nil {
		common.Log.Debugf("Error attempting broadcast of tx ref %s. Error: %s", *t.Ref, err.Error())
		return false
	}

	return true
}

func (t *Transaction) ContinueWithBroadcast(idempotentKey string) bool {

	if txRegister.Has(idempotentKey) {
		common.Log.Debugf("IDEMPOTENT: Register has tx ref %s. Status: %s", *t.Ref, *t.Status)
		// we have this tx in memory, so we only need to reprocess it potentially
		// if it's broadcast (unsticking it from mempool)
		switch *t.Status {
		case "ready":
			common.Log.Debugf("IDEMPOTENT: NOT reprocessing tx ref %s that is ready for broadcast", *t.Ref)
			return false
		case "pending":
			common.Log.Debugf("IDEMPOTENT: NOT reprocessing tx ref %s that is pending broadcast", *t.Ref)
			return false
		case "broadcast":
			common.Log.Debugf("IDEMPOTENT: reprocessing tx ref %s that has been broadcast", *t.Ref)
			t.ReBroadcast = true
			return true
		case "failed":
			return true
		case "success":
			common.Log.Debugf("ACK: Acknowledging successful delivery of tx ref %s", *t.Ref)
			// TODO pass this ack back to the consume method
			err := t.Message.Ack()
			if err != nil {
				common.Log.Debugf("ACK: Error acking tx ref %s. Error: %s", *t.Ref, err.Error())
			}
			common.Log.Debugf("ACK: ConsumeTxCreateMsg, msg acked tx ref: %s", *t.Ref)
			return false
		default:
			common.Log.Debugf("IDEMPOTENT: tx ref %s is in progress", *t.Ref)
			return false
		}
	} else {
		common.Log.Debugf("IDEMPOTENT: Register does NOT have tx ref %s. Status: %s", *t.Ref, *t.Status)
		// we don't have this tx process details in memory
		// possibly it's because of an nchain crash
		switch *t.Status {
		case "pending":
			common.Log.Debugf("IDEMPOTENT: processing tx ref %s that is pending", *t.Ref)
			return true
		case "ready":
			// try again, just in case nchain restarted
			common.Log.Debugf("IDEMPOTENT: processing tx ref %s that is ready for broadcast", *t.Ref)
			return true
		case "broadcast":
			// try again, just in case it's stuck in the mempool
			common.Log.Debugf("IDEMPOTENT: processing tx ref %s that has been broadcast", *t.Ref)
			t.ReBroadcast = true
			return true
		case "failed":
			// try again
			common.Log.Debugf("IDEMPOTENT: processing tx ref %s that has failed to broadcast", *t.Ref)
			return true
		case "success":
			// ack the NATS message as we don't need this any more
			common.Log.Debugf("ACK: Acknowledging successful delivery of tx ref %s", *t.Ref)
			err := t.Message.Ack()
			if err != nil {
				common.Log.Debugf("ACK: ConsumeTxCreateMsg, error acking tx ref: %s. Error: %s", *t.Ref, err.Error())
			}
			common.Log.Debugf("ACK: ConsumeTxCreateMsg, msg acked tx ref: %s", *t.Ref)
			return false
		default:
			common.Log.Debugf("IDEMPOTENT: unknown db status found (%s) for tx ref %s", *t.Status, *t.Ref)
			// STOP PROCESSING THIS TX
			return false
		}
	}
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

	// CHECKME signer account and wallet relationship?
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

func (t *Transaction) broadcast(db *gorm.DB, ntwrk *network.Network, signer Signer) error {
	var broadcastError error

	// TODO ensure subsidize param is set, or we don't broadcast to bookie
	if (t.SignedTx == nil || ntwrk == nil) && t.shouldSubsidize() {
		common.Log.Debugf("!!!: tx ref %s going down bookie path! ", *t.Ref)
		// bookie path
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
			// TODO if this fails, don't retry, fail the tx stack
			broadcastError = fmt.Errorf("failed to broadcast bookie tx; %s", err.Error())
			return broadcastError
		}
		// TODO check that this is correct after nonce too low re-broadcast
		t.Hash = result
		db.Save(&t)
		common.Log.Debugf("broadcast tx %s via bookie: got hash %s", *t.Ref, *t.Hash)
	} else {

		// non-bookie path
		if ntwrk.IsEthereumNetwork() {

			if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
				common.Log.Debugf("About to broadcast tx ref: %s", *t.Ref)
				t.Hash = common.StringOrNil(signedTx.Hash().String())

				err := providecrypto.EVMBroadcastSignedTx(ntwrk.ID.String(), ntwrk.RPCURL(), signedTx)
				if err == nil {
					// we have successfully broadcast the transaction
					common.Log.Debugf("Broadcast tx ref %s with hash %s", *t.Ref, *t.Hash)
				} else {
					// we have failed to broadcast the tx (for some reason)
					common.Log.Debugf("Failed to broadcast tx ref %s with hash %s. Error: %s", *t.Ref, *t.Hash, err.Error())
					broadcastError = fmt.Errorf("unable to broadcast signed tx; broadcast failed for signed tx ref %s. Error: %s", *t.Ref, err.Error())
				} //evmbroadcast

			} else {
				broadcastError = fmt.Errorf("unable to broadcast signed tx; typecast failed for tx ref: %s", *t.Ref)
			} // signedTx
		} else {
			broadcastError = fmt.Errorf("unable to generate signed tx for tx ref %s for unsupported network: %s", *t.Ref, *ntwrk.Name)
		} // isEthereumNetwork
	}

	if broadcastError != nil {
		common.Log.Warningf("failed to broadcast tx ref %s on network %s using %s; %s", *t.Ref, *ntwrk.Name, signer.String(), broadcastError.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(broadcastError.Error()),
		})
	} else {
		broadcastAt := time.Now()
		t.BroadcastAt = &broadcastAt
	}

	return broadcastError
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
		common.Log.Debugf("Searching for contract for txID: %s", t.ID)
		if kontract == nil || kontract.ID == uuid.Nil {
			common.Log.Debugf("Could not find contract for tx id: %s, appID: %s, walletID: %s", t.ID, *t.ApplicationID, *t.WalletID)

			kontract = &contract.Contract{
				ApplicationID:  t.ApplicationID,
				OrganizationID: t.OrganizationID,
				NetworkID:      t.NetworkID,
				TransactionID:  &t.ID,
				Name:           common.StringOrNil(contractName),
				Address:        common.StringOrNil(receipt.ContractAddress.Hex()),
				Params:         t.Params,
				Ref:            t.Ref,
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
			common.Log.Debugf("Updated contract with address for txID: %s", t.ID)
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
