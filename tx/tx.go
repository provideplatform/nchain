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
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/token"
	"github.com/provideapp/goldmine/wallet"
	provide "github.com/provideservices/provide-go"
)

const defaultDerivedCoinType = uint32(60)
const defaultDerivedChainPath = uint32(0) // i.e., the external or internal chain (also known as change addresses if internal chain)
const firstHardenedChildIndex = uint32(0x80000000)

// Signer interface for signing transactions
type Signer interface {
	Sign(tx *Transaction) (signedTx interface{}, hash *string, err error)
	String() string
}

// Transaction instances are associated with a signing wallet and exactly one matching instance of either an a) application
// identifier or b) user identifier.
type Transaction struct {
	provide.Model
	NetworkID uuid.UUID `sql:"not null;type:uuid" json:"network_id,omitempty"`

	// Application or user id, if populated, is the entity for which the transaction was custodially signed and broadcast
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`

	// Account or HD wallet which custodially signed the transaction; when an HD wallet is used, if no HD derivation path is provided,
	// the most recently derived non-zero account is used to sign
	AccountID *uuid.UUID `sql:"type:uuid" json:"account_id,omitempty"`
	WalletID  *uuid.UUID `sql:"type:uuid" json:"wallet_id,omitempty"`
	Path      *string    `gorm:"column:hd_derivation_path" json:"hd_derivation_path,omitempty"`

	// Network-agnostic tx fields
	Signer      *string          `sql:"-" json:"signer,omitempty"`
	To          *string          `json:"to"`
	Value       *txValue         `sql:"not null;type:text" json:"value"`
	Data        *string          `json:"data"`
	Hash        *string          `json:"hash"`
	Status      *string          `sql:"not null;default:'pending'" json:"status"`
	Params      *json.RawMessage `sql:"-" json:"params,omitempty"`
	Ref         *string          `json:"ref"`
	Description *string          `json:"description"`

	// Ephemeral fields for managing the tx/rx and tracing lifecycles
	Response *contract.ExecutionResponse `sql:"-" json:"-"`
	SignedTx interface{}                 `sql:"-" json:"-"`
	Traces   interface{}                 `sql:"-" json:"traces"`

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

// Sign implements the Signer interface
func (txs *TransactionSigner) Sign(tx *Transaction) (signedTx interface{}, hash *string, err error) {
	if tx == nil || tx.Data == nil {
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

		var nonce *uint64
		if nonceFloat, nonceOk := params["nonce"].(float64); nonceOk {
			nonceUint := uint64(nonceFloat)
			nonce = &nonceUint
		}

		if txs.Account != nil {
			if txs.Account.PrivateKey != nil {
				privateKey, _ := common.DecryptECDSAPrivateKey(*txs.Account.PrivateKey)
				_privateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
				signedTx, hash, err = provide.EVMSignTx(
					txs.Network.ID.String(),
					txs.Network.RPCURL(),
					txs.Account.Address,
					_privateKey,
					tx.To,
					tx.Data,
					tx.Value.BigInt(),
					nonce,
					uint64(gas),
				)

				if err == nil {
					accessedAt := time.Now()
					go func() {
						txs.Account.AccessedAt = &accessedAt
						txs.DB.Save(&txs.Account)
					}()
				}
			} else {
				err = fmt.Errorf("unable to sign tx; no private key for account: %s", txs.Account.ID)
			}
		}

		if txs.Wallet != nil {
			if txs.Wallet.Path == nil {
				err := fmt.Errorf("failed to sign %d-byte transaction payload using HD wallet without a specified derivation path", len(*tx.Data))
				common.Log.Warning(err.Error())
				return nil, nil, err
			}

			chain := uint32(0) // FIXME-- don't hardcode this
			hardenedChild, derivationErr := txs.Wallet.DeriveHardened(nil, defaultDerivedCoinType, uint32(0))
			if derivationErr != nil {
				msg := fmt.Sprintf("failed to derive address for HD wallet: %s; %s", txs.Wallet.ID, derivationErr.Error())
				common.Log.Warningf(msg)
				return nil, nil, err
			}

			idx := uint32(0) // FIXME-- don't hardcode this
			derivedAccount, derivationErr := hardenedChild.DeriveAddress(nil, idx, &chain)
			if derivationErr != nil {
				msg := fmt.Sprintf("failed to derive address for HD wallet: %s; %s", txs.Wallet.ID, derivationErr.Error())
				common.Log.Warningf(msg)
				return nil, nil, err
			}

			privateKey, _ := common.DecryptECDSAPrivateKey(*derivedAccount.PrivateKey)
			_privateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
			signedTx, hash, err = provide.EVMSignTx(
				txs.Network.ID.String(),
				txs.Network.RPCURL(),
				txs.Account.Address,
				_privateKey,
				tx.To,
				tx.Data,
				tx.Value.BigInt(),
				nonce,
				uint64(gas),
			)
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

type txValue struct {
	value *big.Int
}

func (v *txValue) Value() (driver.Value, error) {
	return v.value.String(), nil
}

func (v *txValue) Scan(val interface{}) error {
	v.value = new(big.Int)
	if str, ok := val.(string); ok {
		v.value.SetString(str, 10)
	}
	return nil
}

func (v *txValue) BigInt() *big.Int {
	return v.value
}

func (v *txValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *txValue) UnmarshalJSON(data []byte) error {
	v.value = new(big.Int)
	v.value.SetString(string(data), 10)
	return nil
}

func (t *Transaction) asEthereumCallMsg(gasPrice, gasLimit uint64) ethereum.CallMsg {
	db := dbconf.DatabaseConnection()
	var wallet = &wallet.Account{}
	db.Model(t).Related(&wallet)
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
		From:     ethcommon.HexToAddress(wallet.Address),
		To:       to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    t.Value.BigInt(),
		Data:     data,
	}
}

// Create and persist a new transaction. Side effects include persistence of contract and/or token instances
// when the tx represents a contract and/or token creation.
func (t *Transaction) Create(db *gorm.DB) bool {
	if !t.Validate() {
		return false
	}

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
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("invalid network for tx broadcast"),
		})
	}
	if (acct == nil || acct.ID == uuid.Nil) && (wllt == nil || wllt.ID == uuid.Nil) {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("no account or HD wallet signing identity to sign tx for broadcast"),
		})
	}

	if len(t.Errors) > 0 {
		return false
	}

	signer := &TransactionSigner{
		DB:      db,
		Network: ntwrk,
		Account: acct,
		Wallet:  wllt,
	}

	var signingErr error
	err := t.sign(db, ntwrk, signer)
	if err != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		signingErr = err
	}

	if db.NewRecord(t) {
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
				if signingErr != nil {
					t.Errors = append(t.Errors, &provide.Error{
						Message: common.StringOrNil(signingErr.Error()),
					})

					desc := signingErr.Error()
					t.updateStatus(db, "failed", &desc)
				} else {
					err = t.broadcast(db, ntwrk, signer)
					if err == nil {
						txReceiptMsg, _ := json.Marshal(t)
						natsutil.NatsPublish(natsTxReceiptSubject, txReceiptMsg)
					} else {
						desc := err.Error()
						t.updateStatus(db, "failed", &desc)
					}
				}
			}
			return rowsAffected > 0 && len(t.Errors) == 0
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
	} else if t.ApplicationID != nil && wal != nil && wal.ApplicationID != nil && *t.ApplicationID != *wal.ApplicationID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("unable to sign tx due to mismatched signing application"),
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
	} else if wal != nil && t.ApplicationID != nil && wal.NetworkID != nil && t.NetworkID != *wal.NetworkID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("Transaction network did not match wallet network in application context"),
		})
	}
	return len(t.Errors) == 0
}

// Reload the underlying tx instance
func (t *Transaction) Reload() {
	db := dbconf.DatabaseConnection()
	db.Model(&t).Find(t)
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
func (t *Transaction) GetWallet() (*wallet.Account, error) {
	db := dbconf.DatabaseConnection()
	var wallet = &wallet.Account{}
	db.Model(t).Related(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to retrieve transaction wallet for tx: %s", t.ID)
	}
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

func (t *Transaction) attemptTxBroadcastRecovery(err error) error {
	msg := err.Error()
	common.Log.Debugf("Attempting to recover from failed transaction broadcast (tx id: %s); %s", t.ID.String(), msg)

	gasFailureStr := "not enough gas to cover minimal cost of the transaction (minimal: "
	isGasEstimationRecovery := strings.Contains(msg, gasFailureStr) && strings.Contains(msg, "got: 0") // HACK
	if isGasEstimationRecovery {
		common.Log.Debugf("Attempting to recover from gas estimation failure with supplied gas of 0 for tx id: %s", t.ID)
		offset := strings.Index(msg, gasFailureStr) + len(gasFailureStr)
		length := strings.Index(msg[offset:], ",")
		minimalGas, err := strconv.ParseFloat(msg[offset:offset+length], 64)
		if err == nil {
			common.Log.Debugf("Resolved minimal gas of %v required to execute tx: %s", minimalGas, t.ID)
			params := t.ParseParams()
			params["gas"] = minimalGas
			t.setParams(params)
			return nil
		}
		common.Log.Debugf("failed to resolve minimal gas requirement for tx: %s; tx execution unrecoverable", t.ID)
	}

	return err
}

func (t *Transaction) broadcast(db *gorm.DB, network *network.Network, signer Signer) error {
	var err error

	if t.SignedTx == nil {
		return fmt.Errorf("failed to broadcast %s tx using signer: %s; tx not yet signed", *network.Name, signer.String())
	}

	if network.IsEthereumNetwork() {
		if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
			err = provide.EVMBroadcastSignedTx(network.ID.String(), network.RPCURL(), signedTx)
		} else {
			err = fmt.Errorf("unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
		}

		if err != nil {
			if t.attemptTxBroadcastRecovery(err) == nil {
				err = t.sign(db, network, signer)
				if err == nil {
					if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
						err = provide.EVMBroadcastSignedTx(network.ID.String(), network.RPCURL(), signedTx)
					} else {
						err = fmt.Errorf("unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
					}
				}
			}
		}
	} else {
		err = fmt.Errorf("unable to generate signed tx for unsupported network: %s", *network.Name)
	}

	if err != nil {
		common.Log.Warningf("failed to broadcast %s tx using %s; %s", *network.Name, signer.String(), err.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		desc := err.Error()
		t.updateStatus(db, "failed", &desc)
	} else {
		broadcastAt := time.Now()
		t.BroadcastAt = &broadcastAt
		db.Save(&t)
	}

	return err
}

func (t *Transaction) sign(db *gorm.DB, network *network.Network, signer Signer) error {
	var err error
	t.SignedTx, t.Hash, err = signer.Sign(t)

	if err != nil {
		common.Log.Warningf("failed to sign %s tx using %s; %s", *network.Name, signer.String(), err.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		desc := err.Error()
		t.updateStatus(db, "failed", &desc)
	}

	return err
}

func (t *Transaction) fetchReceipt(db *gorm.DB, network *network.Network, wallet *wallet.Account) error {
	if network.IsEthereumNetwork() {
		receipt, err := provide.EVMGetTxReceipt(network.ID.String(), network.RPCURL(), *t.Hash, wallet.Address)
		if err != nil {
			return err
		}

		common.Log.Debugf("Fetched ethereum tx receipt for tx hash: %s", *t.Hash)
		traces, traceErr := provide.EVMTraceTx(network.ID.String(), network.RPCURL(), t.Hash)
		if traceErr != nil {
			common.Log.Warningf("failed to fetch ethereum tx trace for tx hash: %s; %s", *t.Hash, traceErr.Error())
			return traceErr
		}
		t.Response = &contract.ExecutionResponse{
			Receipt:     receipt,
			Traces:      traces,
			Transaction: t,
		}
		t.Traces = traces

		err = t.handleEthereumTxReceipt(db, network, wallet, receipt)
		if err != nil {
			common.Log.Warningf("failed to handle fetched ethereum tx receipt for tx hash: %s; %s", *t.Hash, err.Error())
			return err
		}
		t.handleEthereumTxTraces(db, network, wallet, traces.(*provide.EthereumTxTraceResponse), receipt)
	}

	return nil
}

func (t *Transaction) handleEthereumTxReceipt(db *gorm.DB, network *network.Network, wallet *wallet.Account, receipt *types.Receipt) error {
	client, err := provide.EVMDialJsonRpc(network.ID.String(), network.RPCURL())
	if err != nil {
		common.Log.Warningf("unable to handle ethereum tx receipt; %s", err.Error())
		return err
	}
	if t.To == nil {
		common.Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, *t.Hash, receipt.ContractAddress.Hex())
		params := t.ParseParams()
		contractName := fmt.Sprintf("Contract %s", *common.StringOrNil(receipt.ContractAddress.Hex()))
		if name, ok := params["name"].(string); ok {
			contractName = name
		}
		kontract := &contract.Contract{}
		var tok *token.Token

		tokenCreateFn := func(c *contract.Contract, name string, decimals *big.Int, symbol string) (createdToken bool, tokenID uuid.UUID, errs []*provide.Error) {
			common.Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)

			tok = &token.Token{
				ApplicationID: c.ApplicationID,
				NetworkID:     c.NetworkID,
				ContractID:    &c.ID,
				Name:          common.StringOrNil(name),
				Symbol:        common.StringOrNil(symbol),
				Decimals:      decimals.Uint64(),
				Address:       common.StringOrNil(receipt.ContractAddress.Hex()),
			}

			createdToken = tok.Create()
			tokenID = tok.ID
			errs = tok.Errors
			return
		}
		db.Where("transaction_id = ?", t.ID).Find(&kontract)
		if kontract == nil || kontract.ID == uuid.Nil {
			kontract = &contract.Contract{
				ApplicationID: t.ApplicationID,
				NetworkID:     t.NetworkID,
				TransactionID: &t.ID,
				Name:          common.StringOrNil(contractName),
				Address:       common.StringOrNil(receipt.ContractAddress.Hex()),
				Params:        t.Params,
			}
			if kontract.Create() {
				common.Log.Debugf("Created contract %s for %s contract creation tx: %s", kontract.ID, *network.Name, *t.Hash)
				kontract.ResolveTokenContract(db, network, &wallet.Address, client, receipt, tokenCreateFn)
			} else {
				common.Log.Warningf("failed to create contract for %s contract creation tx %s", *network.Name, *t.Hash)
			}
		} else {
			common.Log.Debugf("Using previously created contract %s for %s contract creation tx: %s", kontract.ID, *network.Name, *t.Hash)
			kontract.Address = common.StringOrNil(receipt.ContractAddress.Hex())
			db.Save(&kontract)
			kontract.ResolveTokenContract(db, network, &wallet.Address, client, receipt, tokenCreateFn)
		}
	}
	return nil
}

func (t *Transaction) handleEthereumTxTraces(db *gorm.DB, network *network.Network, wallet *wallet.Account, traces *provide.EthereumTxTraceResponse, receipt *types.Receipt) {
	kontract := t.GetContract(db)
	if kontract == nil || kontract.ID == uuid.Nil {
		common.Log.Debugf("failed to resolve contract as sender of contract-internal opcode tracing functionality")
		return
	}
	artifact := kontract.CompiledArtifact()
	if artifact == nil {
		common.Log.Warningf("failed to resolve compiled contract artifact required for contract-internal opcode tracing functionality")
		return
	}

	// client, err := provide.EVMDialJsonRpc(network.ID.String(), network.rpcURL())
	// if err != nil {
	// 	common.Log.Warningf("unable to handle ethereum tx traces; %s", err.Error())
	// 	return
	// }

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

			for _, dep := range artifact.Deps {
				dependency := dep.(map[string]interface{})
				name := dependency["name"].(string)
				fingerprint, fingerprintOk := dependency["fingerprint"].(string)
				if !fingerprintOk {
					continue
				}

				common.Log.Debugf("Checking if compiled artifact dependency: %s (fingerprint: %s) is target of contract-internal CREATE opcode at address: %s; tx hash: %s", name, fingerprint, *contractAddr, *t.Hash)
				if strings.HasSuffix(*contractCode, fingerprint) {
					common.Log.Debugf("Observed fingerprinted dependency %s as target of contract-internal CREATE opcode at contract address %s; fingerprint: %s; tx hash: %s", name, *contractAddr, fingerprint, *t.Hash)

					params, _ := json.Marshal(map[string]interface{}{
						"compiled_artifact": dependency,
					})
					rawParams := json.RawMessage(params)
					internalContract := &contract.Contract{
						ApplicationID: t.ApplicationID,
						NetworkID:     t.NetworkID,
						ContractID:    &kontract.ID,
						TransactionID: &t.ID,
						Name:          common.StringOrNil(name),
						Address:       contractAddr,
						Params:        &rawParams,
					}
					if internalContract.Create() {
						common.Log.Debugf("Created contract %s for %s contract-internal tx: %s", internalContract.ID, *network.Name, *t.Hash)
						client, err := provide.EVMDialJsonRpc(network.ID.String(), network.RPCURL())
						if err != nil {
							common.Log.Warningf("unable to attempt token creation for contract-internal tx; %s", err.Error())
							return
						}
						internalContract.ResolveTokenContract(db, network, &wallet.Address, client, receipt,
							func(c *contract.Contract, name string, decimals *big.Int, symbol string) (createdToken bool, tokenID uuid.UUID, errs []*provide.Error) {
								common.Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)

								tok := &token.Token{
									ApplicationID: c.ApplicationID,
									NetworkID:     c.NetworkID,
									ContractID:    &c.ID,
									Name:          common.StringOrNil(name),
									Symbol:        common.StringOrNil(symbol),
									Decimals:      decimals.Uint64(),
									Address:       common.StringOrNil(receipt.ContractAddress.Hex()),
								}

								createdToken = tok.Create()
								tokenID = tok.ID
								errs = tok.Errors
								return
							})
					} else {
						common.Log.Warningf("failed to create contract for %s contract-internal creation tx %s", *network.Name, *t.Hash)
					}
					break
				}
			}
		}
	}
}

// RefreshDetails populates transaction details which were not necessarily available upon broadcast, including network-specific metadata and VM execution tracing if applicable
func (t *Transaction) RefreshDetails() error {
	if t.Hash == nil {
		return nil
	}

	var err error
	network, _ := t.GetNetwork()
	if network.IsEthereumNetwork() {
		t.Traces, err = provide.EVMTraceTx(network.ID.String(), network.RPCURL(), t.Hash)
	}
	if err != nil {
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
