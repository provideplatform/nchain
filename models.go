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

	"github.com/provideapp/go-core"

	ethereum "github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/jinzhu/gorm"
	"github.com/kthomas/go.uuid"
)

type Network struct {
	gocore.Model
	Name         *string          `sql:"not null" json:"name"`
	Description  *string          `json:"description"`
	IsProduction *bool            `sql:"not null" json:"is_production"`
	SidechainID  *uuid.UUID       `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	Config       *json.RawMessage `sql:"type:json" json:"config"`
}

// NetworkStatus provides network-agnostic status
type NetworkStatus struct {
	Block   *uint64                `json:"block"`   // current block
	Height  *uint64                `json:"height"`  // total height of the blockchain; null after syncing completed
	State   *string                `json:"state"`   // i.e., syncing, synced, etc
	Syncing bool                   `json:"syncing"` // when true, the network is in the process of syncing the ledger; available functionaltiy will be network-specific
	Meta    map[string]interface{} `json:"meta"`    // network-specific metadata
}

// Contract instances must be associated with an application identifier.
type Contract struct {
	gocore.Model
	ApplicationID *uuid.UUID       `sql:"not null;type:uuid" json:"-"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	TransactionID *uuid.UUID       `sql:"type:uuid" json:"transaction_id"` // id of the transaction which created the contract (or null)
	Name          *string          `sql:"not null" json:"name"`
	Address       *string          `sql:"not null" json:"address"` // network-specific token contract address
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	WalletID *uuid.UUID    `json:"wallet_id"`
	Method   *string       `json:"method"`
	Params   []interface{} `json:"params"`
	Value    uint64        `json:"value"`
}

// Token instances must be associated with an application identifier.
type Token struct {
	gocore.Model
	ApplicationID  *uuid.UUID `sql:"not null;type:uuid" json:"-"`
	NetworkID      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	ContractID     *uuid.UUID `sql:"type:uuid" json:"contract_id"`
	SaleContractID *uuid.UUID `sql:"type:uuid" json:"sale_contract_id"`
	Name           *string    `sql:"not null" json:"name"`
	Symbol         *string    `sql:"not null" json:"symbol"`
	Decimals       uint64     `sql:"not null" json:"decimals"`
	Address        *string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress    *string    `json:"sale_address"`           // non-null if token sale contract is specified
}

// Transaction instances are associated with a signing wallet and exactly one matching instance of either an a) application identifier or b) user identifier.
type Transaction struct {
	gocore.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"-"`
	UserID        *uuid.UUID       `sql:"type:uuid" json:"-"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	WalletID      uuid.UUID        `sql:"not null;type:uuid" json:"wallet_id"`
	To            *string          `json:"to"`
	Value         uint64           `sql:"not null;default:0" json:"value"`
	Data          *string          `json:"data"`
	Hash          *string          `sql:"not null" json:"hash"`
	Params        *json.RawMessage `sql:"-" json:"params"`
}

// Wallet instances must be associated with exactly one instance of either an a) application identifier or b) user identifier.
type Wallet struct {
	gocore.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"-"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"-"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	Address       string     `sql:"not null" json:"address"`
	PrivateKey    *string    `sql:"not null;type:bytea" json:"-"`
}

// ParseConfig - parse the persistent network configuration JSON
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

// Status retrieves metadata and metrics specific to the given network
func (n *Network) Status() (*NetworkStatus, error) {
	var status *NetworkStatus
	if strings.HasPrefix(strings.ToLower(*n.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		client, err := DialJsonRpc(n)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host; %s", *n.Name, err.Error())
			return nil, err
		}

		syncProgress, err := client.SyncProgress(context.TODO())
		if err != nil {
			Log.Warningf("Failed to read %s sync progress using JSON-RPC host; %s", *n.Name, err.Error())
			return nil, err
		}
		var state string
		var block *uint64  // current block; will be less than height while syncing in progress
		var height *uint64 // total number of blocks
		var syncing = false
		if syncProgress == nil {
			hdr, err := client.HeaderByNumber(context.TODO(), nil)
			if err != nil {
				Log.Warningf("Failed to read latest block header for %s using JSON-RPC host; %s", *n.Name, err.Error())
				return nil, err
			}
			hdrUint64 := hdr.Number.Uint64()
			block = &hdrUint64
		} else {
			block = &syncProgress.CurrentBlock
			height = &syncProgress.HighestBlock
			syncing = true
		}
		status = &NetworkStatus{
			Block:   block,
			Height:  height,
			State:   stringOrNil(state),
			Syncing: syncing,
			Meta:    map[string]interface{}{},
		}
	} else {
		Log.Warningf("Unable to determine status of unsupported network: %s", *n.Name)
	}
	return status, nil
}

// ParseParams - parse the original JSON params used for contract creation
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

// Execute - execute functionality encapsulated in the contract by invoking a specific method using given parameters
func (c *Contract) Execute(walletID *uuid.UUID, value uint64, method string, params ...interface{}) (*Transaction, error) {
	var err error
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(c).Related(&network)

	tx := &Transaction{
		ApplicationID: c.ApplicationID,
		UserID:        nil,
		NetworkID:     c.NetworkID,
		WalletID:      *walletID,
		To:            c.Address,
		Value:         value,
	}

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		err = c.executeEthereumContract(tx, method, params)
		if err != nil {
			err = fmt.Errorf("Unable to execute %s contract; %s", *network.Name, err.Error())
		}
	} else {
		err = fmt.Errorf("Unable to execute contract for unsupported network: %s", *network.Name)
	}
	if err != nil {
		Log.Warningf(err.Error())
		return nil, err
	}
	return tx, nil
}

// Create and persist a new contract
func (c *Contract) Create() bool {
	db := DatabaseConnection()

	if !c.Validate() {
		return false
	}

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &gocore.Error{
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

// GetTransaction - retrieve the associated contract creation transaction
func (c *Contract) GetTransaction() (*Transaction, error) {
	var tx = &Transaction{}
	db := DatabaseConnection()
	db.Model(c).Related(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		return nil, fmt.Errorf("Failed to retrieve tx for contract: %s", c.ID)
	}
	return tx, nil
}

// Validate a contract for persistence
func (c *Contract) Validate() bool {
	db := DatabaseConnection()
	var transaction = &Transaction{}
	db.Model(c).Related(&transaction)
	c.Errors = make([]*gocore.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &gocore.Error{
			Message: stringOrNil("Unable to associate contract with unspecified network"),
		})
	} else if c.NetworkID != transaction.NetworkID {
		c.Errors = append(c.Errors, &gocore.Error{
			Message: stringOrNil("Contract network did not match transaction network"),
		})
	}
	return len(c.Errors) == 0
}

func (c *Contract) executeEthereumContract(tx *Transaction, method string, params ...interface{}) error { // given tx has been built but broadcast has not yet been attempted
	var err error
	abi, err := c.readEthereumContractAbi()
	if err != nil {
		err := fmt.Errorf("Failed to execute contract method %s on contract: %s; no ABI resolved: %s", method, c.ID, err.Error())
		return err
	}
	if _, ok := abi.Methods[method]; ok {
		Log.Debugf("Attempting to encode %d parameters prior to attempting execution of contract method %s on contract: %s", len(params), method, c.ID)
		Log.Debugf("Attempting to encode parameters: %s", params)
		invocationSig, err := abi.Pack(method, params)
		if err != nil {
			Log.Warningf("Failed to encode %d parameters prior to attempting execution of contract method %s on contract: %s; %s", len(params), method, c.ID, err.Error())
			return err
		}

		data := common.Bytes2Hex(invocationSig)
		tx.Data = &data

		if tx.Create() {
			Log.Debugf("Executed contract method %s on contract: %s", method, c.ID)
		} else {
			err = fmt.Errorf("Failed to execute contract method %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", method, *tx.Data, c.ID)
			Log.Warning(err.Error())
		}
	} else {
		err = fmt.Errorf("Failed to execute contract method %s on contract: %s; method not found in ABI", method, c.ID)
	}
	return err
}

func (c *Contract) readEthereumContractAbi() (*ethabi.ABI, error) {
	var abi *ethabi.ABI
	params := c.ParseParams()
	if contractAbi, ok := params["abi"]; ok {
		abistr, err := json.Marshal(contractAbi)
		if err != nil {
			Log.Warningf("Failed to marshal ABI from contract params to json; %s", err.Error())
			return nil, err
		}

		abival, err := ethabi.JSON(strings.NewReader(string(abistr)))
		if err != nil {
			Log.Warningf("Failed to initialize ABI from contract  params to json; %s", err.Error())
			return nil, err
		}

		abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read ABI from params for contract: %s", c.ID)
	}
	return abi, nil
}

// Create and persist a token
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
				t.Errors = append(t.Errors, &gocore.Error{
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

// Validate a token for persistence
func (t *Token) Validate() bool {
	db := DatabaseConnection()
	var contract = &Contract{}
	if t.NetworkID != uuid.Nil {
		db.Model(t).Related(&contract)
	}
	t.Errors = make([]*gocore.Error, 0)
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("Unable to deploy token contract using unspecified network"),
		})
	} else {
		if contract != nil {
			if t.NetworkID != contract.NetworkID {
				t.Errors = append(t.Errors, &gocore.Error{
					Message: stringOrNil("Token network did not match token contract network"),
				})
			}
			if t.Address == nil {
				t.Address = contract.Address
			} else if t.Address != nil && *t.Address != *contract.Address {
				t.Errors = append(t.Errors, &gocore.Error{
					Message: stringOrNil("Token contract address did not match referenced contract address"),
				})
			}
		}
		// if t.SaleContractID != nil {
		// 	if t.NetworkID != saleContract.NetworkID {
		// 		t.Errors = append(t.Errors, &gocore.Error{
		// 			Message: stringOrNil("Token network did not match token sale contract network"),
		// 		})
		// 	}
		// 	if t.SaleAddress == nil {
		// 		t.SaleAddress = saleContract.Address
		// 	} else if t.SaleAddress != nil && *t.SaleAddress != *saleContract.Address {
		// 		t.Errors = append(t.Errors, &gocore.Error{
		// 			Message: stringOrNil("Token sale address did not match referenced token sale contract address"),
		// 		})
		// 	}
		// }
	}
	return len(t.Errors) == 0
}

// GetContract - retreieve the associated token contract
func (t *Token) GetContract() (*Contract, error) {
	db := DatabaseConnection()
	var contract = &Contract{}
	db.Model(t).Related(&contract)
	if contract == nil {
		return nil, fmt.Errorf("Failed to retrieve token contract for token: %s", t.ID)
	}
	return contract, nil
}

func (t *Token) readEthereumContractAbi() (*ethabi.ABI, error) {
	contract, err := t.GetContract()
	if err != nil {
		return nil, err
	}
	return contract.readEthereumContractAbi()
}

// ParseParams - parse the original JSON params used when the tx was broadcast
func (t *Transaction) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if t.Params != nil {
		err := json.Unmarshal(*t.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal transaction params; %s", err.Error())
			return nil
		}
	}
	return params
}

func (t *Transaction) asEthereumCallMsg(gasPrice, gasLimit uint64) ethereum.CallMsg {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	var to *common.Address
	var data []byte
	if t.To != nil {
		addr := common.HexToAddress(*t.To)
		to = &addr
	}
	if t.Data != nil {
		data = common.FromHex(*t.Data)
	}
	return ethereum.CallMsg{
		From:     common.HexToAddress(wallet.Address),
		To:       to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    big.NewInt(int64(t.Value)),
		Data:     data,
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
		nonce, err := client.PendingNonceAt(context.TODO(), common.HexToAddress(wallet.Address))
		if err != nil {
			return nil, err
		}
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		var data []byte
		if t.Data != nil {
			data = common.FromHex(*t.Data)
		}
		var tx *types.Transaction
		if t.To != nil {
			addr := common.HexToAddress(*t.To)
			gasLimit := big.NewInt(DefaultEthereumGasLimit).Uint64()
			tx = types.NewTransaction(nonce, addr, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		} else {
			Log.Debugf("Attempting to deploy %s contract via tx; estimating total gas requirements", *network.Name)
			callMsg := t.asEthereumCallMsg(gasPrice.Uint64(), 0)
			gasLimit, err := client.EstimateGas(context.TODO(), callMsg)
			if err != nil {
				Log.Warningf("Failed to estimate gas for %s contract deployment tx; %s", *network.Name, err.Error())
				return nil, err
			}
			Log.Debugf("Estimated %d total gas required for %s contract deployment tx with %d-byte data payload", gasLimit, *network.Name, len(data))
			tx = types.NewContractCreation(nonce, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		}
		signer := types.MakeSigner(cfg, hdr.Number)
		hash := signer.Hash(tx).Bytes()
		sig, err := wallet.SignTx(hash)
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			t.Hash = stringOrNil(fmt.Sprintf("%x", signedTx.Hash()))
			Log.Debugf("Signed %s tx for broadcast via JSON-RPC: %s", *network.Name, signedTx.String())
			return signedTx, nil
		}
		return nil, err
	} else if syncProgress == nil {
		Log.Debugf("%s JSON-RPC is in sync with the network", *network.Name)
	}
	return nil, err
}

// Create and persist a new transaction. Side effects include persistence of contract and/or token instances
// when the tx represents a contract and/or token creation.
func (t *Transaction) Create() bool {
	if !t.Validate() {
		return false
	}

	db := DatabaseConnection()
	var network = &Network{}
	var wallet = &Wallet{}
	if t.NetworkID != uuid.Nil {
		db.Model(t).Related(&network)
		db.Model(t).Related(&wallet)
	}

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		client, err := DialJsonRpc(network)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host; %s", *network.Name, err.Error())
			t.Errors = append(t.Errors, &gocore.Error{
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
					t.Errors = append(t.Errors, &gocore.Error{
						Message: stringOrNil(err.Error()),
					})
				}
			} else {
				Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
				t.Errors = append(t.Errors, &gocore.Error{
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
				t.Errors = append(t.Errors, &gocore.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			if t.To == nil && rowsAffected > 0 {
				if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
					var err error
					var receipt *types.Receipt
					client, err := DialJsonRpc(network)
					gasPrice, _ := client.SuggestGasPrice(context.TODO())
					txHash := fmt.Sprintf("0x%s", *t.Hash)
					Log.Debugf("%s contract created by broadcast tx: %s", *network.Name, txHash)
					err = ethereum.NotFound
					for receipt == nil && err == ethereum.NotFound {
						Log.Debugf("Retrieving tx receipt for %s contract creation tx: %s", *network.Name, txHash)
						receipt, err = client.TransactionReceipt(context.TODO(), common.HexToHash(txHash))
						if err != nil && err == ethereum.NotFound {
							Log.Warningf("%s contract created by broadcast tx: %s; address must be retrieved from tx receipt", *network.Name, txHash)
						} else {
							Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, txHash, receipt.ContractAddress.Hex())
							params := t.ParseParams()
							contractName := fmt.Sprintf("Contract %s", *stringOrNil(receipt.ContractAddress.Hex()))
							if name, ok := params["name"].(string); ok {
								contractName = name
							}
							contract := &Contract{
								ApplicationID: t.ApplicationID,
								NetworkID:     t.NetworkID,
								TransactionID: &t.ID,
								Name:          stringOrNil(contractName),
								Address:       stringOrNil(receipt.ContractAddress.Hex()),
								Params:        t.Params,
							}
							if contract.Create() {
								Log.Debugf("Created contract %s for %s contract creation tx: %s", contract.ID, *network.Name, txHash)

								if contractAbi, ok := params["abi"]; ok {
									abistr, err := json.Marshal(contractAbi)
									if err != nil {
										Log.Warningf("failed to marshal abi to json...  %s", err.Error())
									}
									abi, err := ethabi.JSON(strings.NewReader(string(abistr)))
									if err == nil {
										msg := ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("name()"))),
										}

										result, _ := client.CallContract(context.TODO(), msg, nil)
										var name string
										if method, ok := abi.Methods["name"]; ok {
											err = method.Outputs.Unpack(&name, result)
											if err != nil {
												Log.Warningf("Failed to read %s, contract name from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
											}
										}

										msg = ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("decimals()"))),
										}
										result, _ = client.CallContract(context.TODO(), msg, nil)
										var decimals *big.Int
										if method, ok := abi.Methods["decimals"]; ok {
											err = method.Outputs.Unpack(&decimals, result)
											if err != nil {
												Log.Warningf("Failed to read %s, contract decimals from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
											}
										}

										msg = ethereum.CallMsg{
											From:     common.HexToAddress(wallet.Address),
											To:       &receipt.ContractAddress,
											Gas:      0,
											GasPrice: gasPrice,
											Value:    nil,
											Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("symbol()"))),
										}
										result, _ = client.CallContract(context.TODO(), msg, nil)
										var symbol string
										if method, ok := abi.Methods["symbol"]; ok {
											err = method.Outputs.Unpack(&symbol, result)
											if err != nil {
												Log.Warningf("Failed to read %s, contract symbol from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
											}
										}

										if name != "" && decimals != nil && symbol != "" { // isERC20Token
											Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)
											token := &Token{
												ApplicationID: contract.ApplicationID,
												NetworkID:     contract.NetworkID,
												ContractID:    &contract.ID,
												Name:          stringOrNil(name),
												Symbol:        stringOrNil(symbol),
												Decimals:      decimals.Uint64(),
												Address:       stringOrNil(receipt.ContractAddress.Hex()),
											}
											if token.Create() {
												Log.Debugf("Created token %s for associated %s contract creation tx: %s", token.ID, *network.Name, txHash)
											} else {
												Log.Warningf("Failed to create token for associated %s contract creation tx %s; %d errs: %s", *network.Name, txHash, len(token.Errors), *stringOrNil(*token.Errors[0].Message))
											}
										}
									} else {
										Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
									}
								}
							} else {
								Log.Warningf("Failed to create contract for %s contract creation tx %s", *network.Name, txHash)
							}
						}
					}
				}
			}
			return rowsAffected > 0
		}
	}
	return false
}

// Validate a transaction for persistence
func (t *Transaction) Validate() bool {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	t.Errors = make([]*gocore.Error, 0)
	if t.ApplicationID == nil && t.UserID == nil {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("no application or user identifier provided"),
		})
	} else if t.ApplicationID != nil && t.UserID != nil {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("only an application OR user identifier should be provided"),
		})
	} else if t.ApplicationID != nil && wallet.ApplicationID != nil && *t.ApplicationID != *wallet.ApplicationID {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("Unable to sign tx due to mismatched signing application"),
		})
	} else if t.UserID != nil && wallet.UserID != nil && *t.UserID != *wallet.UserID {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("Unable to sign tx due to mismatched signing user"),
		})
	}
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("Unable to sign tx using unspecified network"),
		})
	} else if t.NetworkID != wallet.NetworkID {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil("Transaction network did not match wallet network"),
		})
	}
	return len(t.Errors) == 0
}

func (w *Wallet) generate(db *gorm.DB, gpgPublicKey string) {
	var network = &Network{}
	db.Model(w).Related(&network)

	if network == nil || network.ID == uuid.Nil {
		Log.Warningf("Unable to generate private key for wallet without an associated network")
		return
	}

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

// ECDSAPrivateKey - read the wallet-specific ECDSA private key; required for signing transactions on behalf of the wallet
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

// SignTx - sign a raw transaction
func (w *Wallet) SignTx(msg []byte) ([]byte, error) {
	db := DatabaseConnection()

	var network = &Network{}
	db.Model(w).Related(&network)

	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		privateKey, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s", *network.Name, w.ID)
			return nil, err
		}

		Log.Debugf("Signing tx using %s wallet: %s", *network.Name, w.ID)
		sig, err := ethcrypto.Sign(msg, privateKey)
		if err != nil {
			Log.Warningf("Failed to sign tx using %s wallet: %s; %s", *network.Name, w.ID, err.Error())
			return nil, err
		}
		return sig, nil
	}

	err := fmt.Errorf("Unable to sign tx using unsupported network: %s", *network.Name)
	Log.Warningf(err.Error())
	return nil, err
}

// Create and persist a network-specific wallet used for storing crpyotcurrency or digital tokens native to a specific network
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
				w.Errors = append(w.Errors, &gocore.Error{
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

// Validate a wallet for persistence
func (w *Wallet) Validate() bool {
	w.Errors = make([]*gocore.Error, 0)
	var network = &Network{}
	DatabaseConnection().Model(w).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		w.Errors = append(w.Errors, &gocore.Error{
			Message: stringOrNil(fmt.Sprintf("invalid network association attempted with network id: %s", w.NetworkID.String())),
		})
	}
	if w.ApplicationID == nil && w.UserID == nil {
		w.Errors = append(w.Errors, &gocore.Error{
			Message: stringOrNil("no application or user identifier provided"),
		})
	} else if w.ApplicationID != nil && w.UserID != nil {
		w.Errors = append(w.Errors, &gocore.Error{
			Message: stringOrNil("only an application OR user identifier should be provided"),
		})
	}
	_, err := w.ECDSAPrivateKey(GpgPrivateKey, WalletEncryptionKey)
	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &gocore.Error{
			Message: &msg,
		})
	}
	return len(w.Errors) == 0
}

// TokenBalance
// Retrieve a wallet's token balance for a given token id
func (w *Wallet) TokenBalance(tokenId string) (uint64, error) {
	balance := uint64(0)
	db := DatabaseConnection()
	var network = &Network{}
	var token = &Token{}
	db.Model(w).Related(&network)
	db.Where("id = ?", tokenId).Find(&token)
	if token == nil {
		return 0, fmt.Errorf("Unable to read token balance for invalid token: %s", tokenId)
	}
	if strings.HasPrefix(strings.ToLower(*network.Name), "eth") { // HACK-- this should be simpler; implement protocol switch
		abi, err := token.readEthereumContractAbi()
		if err != nil {
			return 0, err
		}
		client, err := DialJsonRpc(network)
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		to := common.HexToAddress(*token.Address)
		msg := ethereum.CallMsg{
			From:     common.HexToAddress(w.Address),
			To:       &to,
			Gas:      0,
			GasPrice: gasPrice,
			Value:    nil,
			Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("balanceOf(address)"))),
		}
		result, _ := client.CallContract(context.TODO(), msg, nil)
		var out *big.Int
		if method, ok := abi.Methods["balanceOf"]; ok {
			method.Outputs.Unpack(&out, result)
			if out != nil {
				balance = out.Uint64()
				Log.Debugf("Read %s %s token balance (%v) from token contract address: %s", *network.Name, token.Symbol, balance, token.Address)
			}
		} else {
			Log.Warningf("Unable to read balance of unsupported %s token contract address: %s", *network.Name, token.Address)
		}
	} else {
		Log.Warningf("Unable to read token balance for unsupported network: %s", *network.Name)
	}
	return balance, nil
}

// TxCount
// Retrieve a count of transactions signed by the wallet
func (w *Wallet) TxCount() (count *uint64) {
	db := DatabaseConnection()
	db.Model(&Transaction{}).Where("wallet_id = ?", w.ID).Count(&count)
	return count
}
