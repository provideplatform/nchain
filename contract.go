package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

const resolveTokenTickerInterval = time.Millisecond * 5000
const resolveTokenTickerTimeout = time.Minute * 1

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Contract{})
	db.Model(&Contract{}).AddIndex("idx_contracts_application_id", "application_id")
	db.Model(&Contract{}).AddIndex("idx_contracts_accessed_at", "accessed_at")
	db.Model(&Contract{}).AddIndex("idx_contracts_address", "address")
	db.Model(&Contract{}).AddIndex("idx_contracts_contract_id", "contract_id")
	db.Model(&Contract{}).AddIndex("idx_contracts_network_id", "network_id")
	db.Model(&Contract{}).AddIndex("idx_contracts_transaction_id", "transaction_id")
	db.Model(&Contract{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
	db.Model(&Contract{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Contract{}).AddForeignKey("transaction_id", "transactions(id)", "SET NULL", "CASCADE")
}

// Contract instances must be associated with an application identifier.
type Contract struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ContractID    *uuid.UUID       `sql:"type:uuid" json:"contract_id"`    // id of the contract which created the contract (or null)
	TransactionID *uuid.UUID       `sql:"type:uuid" json:"transaction_id"` // id of the transaction which deployed the contract (or null)
	Name          *string          `sql:"not null" json:"name"`
	Address       *string          `sql:"not null" json:"address"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
	AccessedAt    *time.Time       `json:"accessed_at"`
}

// ContractListQuery returns a DB query configured to select columns suitable for a paginated API response
func ContractListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("contracts.id, contracts.application_id, contracts.network_id, contracts.transaction_id, contracts.name, contracts.address")
}

// CompiledArtifact - parse the original JSON params used for contract creation and attempt to unmarshal to a provide.CompiledArtifact
func (c *Contract) CompiledArtifact() *provide.CompiledArtifact {
	artifact := &provide.CompiledArtifact{}
	params := c.ParseParams()
	if params != nil {
		err := json.Unmarshal(*c.Params, &artifact)
		if err != nil {
			Log.Warningf("Failed to unmarshal contract params to compiled artifact; %s", err.Error())
			return nil
		}
	}
	return artifact
}

// Compile the contract if possible
func (c *Contract) Compile() (*provide.CompiledArtifact, error) {
	var artifact *provide.CompiledArtifact
	var err error

	params := c.ParseParams()
	lang, langOk := params["lang"].(string)
	if !langOk {
		return nil, fmt.Errorf("Failed to parse wallet id for solidity source compile; %s", err.Error())
	}
	rawSource, rawSourceOk := params["raw_source"].(string)
	if !rawSourceOk {
		return nil, fmt.Errorf("Failed to compile contract; no source code resolved")
	}
	Log.Debugf("Attempting to compile %d-byte raw source code; lang: %s", len(rawSource), lang)
	db := dbconf.DatabaseConnection()

	var walletID *uuid.UUID
	if _walletID, walletIdOk := params["wallet_id"].(string); walletIdOk {
		__walletID, err := uuid.FromString(_walletID)
		walletID = &__walletID
		if err != nil {
			return nil, fmt.Errorf("Failed to parse wallet id for solidity source compile; %s", err.Error())
		}
	}

	var network = &Network{}
	db.Model(c).Related(&network)

	argv := make([]interface{}, 0)
	if _argv, argvOk := params["argv"].([]interface{}); argvOk {
		argv = _argv
	}

	if network.isEthereumNetwork() {
		optimizerRuns := 200
		if _optimizerRuns, optimizerRunsOk := params["optimizer_runs"].(int); optimizerRunsOk {
			optimizerRuns = _optimizerRuns
		}

		artifact, err = compileSolidity(*c.Name, rawSource, argv, optimizerRuns)
		if err != nil {
			return nil, fmt.Errorf("Failed to compile solidity source; %s", err.Error())
		}
	}

	artifactJSON, _ := json.Marshal(artifact)
	deployableArtifactJSON := json.RawMessage(artifactJSON)

	tx := &Transaction{
		ApplicationID: c.ApplicationID,
		Data:          &artifact.Bytecode,
		NetworkID:     c.NetworkID,
		WalletID:      walletID,
		To:            nil,
		Value:         &TxValue{value: big.NewInt(0)},
		Params:        &deployableArtifactJSON,
	}

	if tx.Create() {
		c.TransactionID = &tx.ID
		db.Save(&c)
		Log.Debugf("Contract compiled from source and deployed via tx: %s", *tx.Hash)
	} else {
		return nil, fmt.Errorf("Failed to deploy compiled contract; tx failed with %d error(s)", len(tx.Errors))
	}
	return artifact, nil
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

// Execute an ethereum contract; returns the tx receipt and retvals; if the method is constant, the receipt will be nil.
// If the methid is non-constant, the retvals will be nil.
func (c *Contract) executeEthereumContract(network *Network, tx *Transaction, method string, params []interface{}) (*interface{}, *interface{}, error) { // given tx has been built but broadcast has not yet been attempted
	defer func() {
		go func() {
			accessedAt := time.Now()
			c.AccessedAt = &accessedAt
			db := dbconf.DatabaseConnection()
			db.Save(c)
		}()
	}()

	var err error
	_abi, err := c.readEthereumContractAbi()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to execute contract method %s on contract: %s; no ABI resolved: %s", method, c.ID, err.Error())
	}
	var methodDescriptor = fmt.Sprintf("method %s", method)
	var abiMethod *abi.Method
	if mthd, ok := _abi.Methods[method]; ok {
		abiMethod = &mthd
	} else if method == "" {
		abiMethod = &_abi.Constructor
		methodDescriptor = "constructor"
	}
	if abiMethod != nil {
		Log.Debugf("Attempting to encode %d parameters %s prior to executing method %s on contract: %s", len(params), params, methodDescriptor, c.ID)
		invocationSig, err := provide.EVMEncodeABI(abiMethod, params...)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to encode %d parameters prior to attempting execution of %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
		}

		data := fmt.Sprintf("0x%s", common.Bytes2Hex(invocationSig))
		tx.Data = &data

		if abiMethod.Const {
			Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
			network, _ := tx.GetNetwork()
			client, err := provide.EVMDialJsonRpc(network.ID.String(), network.rpcURL())
			msg := tx.asEthereumCallMsg(0, 0)
			result, err := client.CallContract(context.TODO(), msg, nil)
			if err != nil {
				err = fmt.Errorf("Failed to read constant method %s on contract: %s; %s", method, c.ID, err.Error())
				return nil, nil, err
			}
			var out interface{}
			if len(abiMethod.Outputs) == 1 {
				err = abiMethod.Outputs.Unpack(&out, result)
				if err == nil {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[0].Type)
					Log.Debugf("Attempting to marshal %s result of constant contract execution of %s on contract: %s", typestr, methodDescriptor, c.ID)
					switch out.(type) {
					case [32]byte:
						arrbytes, _ := out.([32]byte)
						out = string(bytes.Trim(arrbytes[:], "\x00"))
					case [][32]byte:
						arrbytesarr, _ := out.([][32]byte)
						vals := make([]string, len(arrbytesarr))
						for i, item := range arrbytesarr {
							vals[i] = string(bytes.Trim(item[:], "\x00"))
						}
						out = vals
					default:
						Log.Debugf("Noop during marshaling of constant contract execution of %s on contract: %s", methodDescriptor, c.ID)
					}
				}
			} else if len(abiMethod.Outputs) > 1 {
				// handle tuple
				vals := make([]interface{}, len(abiMethod.Outputs))
				for i := range abiMethod.Outputs {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[i].Type)
					Log.Debugf("Reflectively adding type hint for unpacking %s in return values slot %v", typestr, i)
					typ, err := abi.NewType(typestr)
					if err != nil {
						return nil, nil, fmt.Errorf("Failed to reflectively add appropriately-typed %s value for in return values slot %v); %s", typestr, i, err.Error())
					}
					vals[i] = reflect.New(typ.Type).Interface()
				}
				err = abiMethod.Outputs.Unpack(&vals, result)
				out = vals
				Log.Debugf("Unpacked %v returned values from read of constant %s on contract: %s; values: %s", len(vals), methodDescriptor, c.ID, vals)
				if vals != nil && len(vals) == abiMethod.Outputs.LengthNonIndexed() {
					err = nil
				}
			}
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to read constant %s on contract: %s (signature with encoded parameters: %s); %s", methodDescriptor, c.ID, *tx.Data, err.Error())
			}
			return nil, &out, nil
		}

		var txResponse *ContractExecutionResponse
		if tx.Create() {
			Log.Debugf("Executed %s on contract: %s", methodDescriptor, c.ID)
			if tx.Response != nil {
				txResponse = tx.Response
			}
		} else {
			Log.Debugf("Failed tx errors: %s", *tx.Errors[0].Message)
			txParams := tx.ParseParams()
			publicKey, publicKeyOk := txParams["public_key"].(interface{})
			privateKey, privateKeyOk := txParams["private_key"].(interface{})
			gas, gasOk := txParams["gas"].(float64)
			if !gasOk {
				gas = float64(0)
			}
			delete(txParams, "private_key")
			tx.setParams(txParams)

			if publicKeyOk && privateKeyOk {
				Log.Debugf("Attempting to execute %s on contract: %s; arbitrarily-provided signer for tx: %s; gas supplied: %v", methodDescriptor, c.ID, publicKey, gas)
				tx.SignedTx, tx.Hash, err = provide.EVMSignTx(network.ID.String(), network.rpcURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), nil, uint64(gas))
				if err == nil {
					if signedTx, ok := tx.SignedTx.(*types.Transaction); ok {
						err = provide.EVMBroadcastSignedTx(network.ID.String(), network.rpcURL(), signedTx)
					} else {
						err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", tx.SignedTx)
						Log.Warning(err.Error())
					}
				}

				if err != nil {
					err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed using arbitrarily-provided signer: %s; %s", methodDescriptor, c.ID, *tx.Data, publicKey, err.Error())
					Log.Warning(err.Error())
				}
			} else {
				err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", methodDescriptor, c.ID, *tx.Data)
				Log.Warning(err.Error())
			}
		}

		if txResponse != nil {
			Log.Debugf("Received response to tx broadcast attempt calling method %s on contract: %s", methodDescriptor, c.ID)

			var out interface{}
			switch (txResponse.Receipt).(type) {
			case []byte:
				out = (txResponse.Receipt).([]byte)
				Log.Debugf("Received response: %s", out)
			case types.Receipt:
				client, _ := provide.EVMDialJsonRpc(network.ID.String(), network.rpcURL())
				receipt := txResponse.Receipt.(*types.Receipt)
				txdeets, _, err := client.TransactionByHash(context.TODO(), receipt.TxHash)
				if err != nil {
					err = fmt.Errorf("Failed to retrieve %s transaction by tx hash: %s", *network.Name, *tx.Hash)
					Log.Warning(err.Error())
					return nil, nil, err
				}
				out = txdeets
			default:
				// no-op
				Log.Warningf("Unhandled transaction receipt type; %s", tx.Response.Receipt)
			}
			return &out, nil, nil
		}
	} else {
		err = fmt.Errorf("Failed to execute method %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
	}
	return nil, nil, err
}
func (c *Contract) readEthereumContractAbi() (*abi.ABI, error) {
	var _abi *abi.ABI
	params := c.ParseParams()
	if contractAbi, ok := params["abi"]; ok {
		abistr, err := json.Marshal(contractAbi)
		if err != nil {
			Log.Warningf("Failed to marshal ABI from contract params to json; %s", err.Error())
			return nil, err
		}

		abival, err := abi.JSON(strings.NewReader(string(abistr)))
		if err != nil {
			Log.Warningf("Failed to initialize ABI from contract params to json; %s", err.Error())
			return nil, err
		}

		_abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read ABI from params for contract: %s", c.ID)
	}
	return _abi, nil
}

// Create and persist a new contract
func (c *Contract) Create() bool {
	db := dbconf.DatabaseConnection()

	if !c.Validate() {
		return false
	}

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &provide.Error{
					Message: StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(c) {
			success := rowsAffected > 0
			if success {
				params := c.ParseParams()
				_, rawSourceOk := params["raw_source"].(string)
				if rawSourceOk {
					Log.Debugf("Found raw source...")
					contractCompilerInvocationMsg, err := json.Marshal(c)
					if err != nil {
						Log.Warningf("Failed to marshal contract for raw source compilation; %s", err.Error())
					}
					natsConnection := GetDefaultNatsStreamingConnection()
					natsConnection.Publish(natsContractCompilerInvocationSubject, contractCompilerInvocationMsg)
				}
			}
			return success
		}
	}
	return false
}

// Execute a transaction on the contract instance using a specific signer, value, method and params
func (c *Contract) Execute(ref *string, wallet *Wallet, value *big.Int, method string, params []interface{}, gas uint64) (*ContractExecutionResponse, error) {
	var err error
	db := dbconf.DatabaseConnection()
	var network = &Network{}
	db.Model(c).Related(&network)

	txParams := map[string]interface{}{}

	walletID := &uuid.Nil
	if wallet != nil {
		if wallet.ID != uuid.Nil {
			walletID = &wallet.ID
		}
		if StringOrNil(wallet.Address) != nil && wallet.PrivateKey != nil {
			txParams["public_key"] = wallet.Address
			txParams["private_key"] = wallet.PrivateKey
		}
	}

	txParams["gas"] = gas

	txParamsJSON, _ := json.Marshal(txParams)
	_txParamsJSON := json.RawMessage(txParamsJSON)

	tx := &Transaction{
		ApplicationID: c.ApplicationID,
		UserID:        nil,
		NetworkID:     c.NetworkID,
		WalletID:      walletID,
		To:            c.Address,
		Value:         &TxValue{value: value},
		Params:        &_txParamsJSON,
		Ref:           ref,
	}

	var receipt *interface{}
	var response *interface{}

	if network.isEthereumNetwork() {
		receipt, response, err = c.executeEthereumContract(network, tx, method, params)
	} else {
		err = fmt.Errorf("unsupported network: %s", *network.Name)
	}

	accessedAt := time.Now()
	go func() {
		c.AccessedAt = &accessedAt
		db.Save(c)
	}()

	if err != nil {
		desc := err.Error()
		tx.updateStatus(db, "failed", &desc)
		return nil, fmt.Errorf("Unable to execute %s contract; %s", *network.Name, err.Error())
	}

	if tx.Response == nil {
		tx.Response = &ContractExecutionResponse{
			Response:    response,
			Receipt:     receipt,
			Traces:      tx.Traces,
			Transaction: tx,
			Ref:         ref,
		}
	} else if tx.Response.Transaction == nil {
		tx.Response.Transaction = tx
	}

	return tx.Response, nil
}

func (c *Contract) resolveTokenContract(db *gorm.DB, network *Network, wallet *Wallet, client *ethclient.Client, receipt *types.Receipt) {
	ticker := time.NewTicker(resolveTokenTickerInterval)
	go func() {
		startedAt := time.Now()
		for {
			select {
			case <-ticker.C:
				if time.Now().Sub(startedAt) >= resolveTokenTickerTimeout {
					Log.Warningf("Failed to resolve ERC20 token for contract: %s; timing out after %v", c.ID, resolveTokenTickerTimeout)
					ticker.Stop()
					return
				}

				params := c.ParseParams()
				if contractAbi, ok := params["abi"]; ok {
					abistr, err := json.Marshal(contractAbi)
					if err != nil {
						Log.Warningf("Failed to marshal contract abi to json...  %s", err.Error())
					}
					_abi, err := abi.JSON(strings.NewReader(string(abistr)))
					if err == nil {
						msg := ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     common.FromHex(provide.EVMHashFunctionSelector("name()")),
						}

						result, _ := client.CallContract(context.TODO(), msg, nil)
						var name string
						if method, ok := _abi.Methods["name"]; ok {
							err = method.Outputs.Unpack(&name, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract name from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     common.FromHex(provide.EVMHashFunctionSelector("decimals()")),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var decimals *big.Int
						if method, ok := _abi.Methods["decimals"]; ok {
							err = method.Outputs.Unpack(&decimals, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract decimals from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     common.FromHex(provide.EVMHashFunctionSelector("symbol()")),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var symbol string
						if method, ok := _abi.Methods["symbol"]; ok {
							err = method.Outputs.Unpack(&symbol, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract symbol from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						if name != "" && decimals != nil && symbol != "" { // isERC20Token
							Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)
							token := &Token{
								ApplicationID: c.ApplicationID,
								NetworkID:     c.NetworkID,
								ContractID:    &c.ID,
								Name:          StringOrNil(name),
								Symbol:        StringOrNil(symbol),
								Decimals:      decimals.Uint64(),
								Address:       StringOrNil(receipt.ContractAddress.Hex()),
							}
							if token.Create() {
								Log.Debugf("Created token %s for associated %s contract: %s", token.ID, *network.Name, c.ID)
								ticker.Stop()
								return
							} else {
								Log.Warningf("Failed to create token for associated %s contract creation %s; %d errs: %s", *network.Name, c.ID, len(token.Errors), *StringOrNil(*token.Errors[0].Message))
							}
						}
					} else {
						Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
						ticker.Stop()
						return
					}
				}
			}
		}
	}()
}

// setParams sets the contract params in-memory
func (c *Contract) setParams(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := json.RawMessage(paramsJSON)
	c.Params = &_paramsJSON
}

// GetTransaction - retrieve the associated contract creation transaction
func (c *Contract) GetTransaction() (*Transaction, error) {
	var tx = &Transaction{}
	db := dbconf.DatabaseConnection()
	db.Model(c).Related(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		return nil, fmt.Errorf("Failed to retrieve tx for contract: %s", c.ID)
	}
	return tx, nil
}

// Validate a contract for persistence
func (c *Contract) Validate() bool {
	db := dbconf.DatabaseConnection()
	var transaction *Transaction
	if c.TransactionID != nil {
		transaction = &Transaction{}
		db.Model(c).Related(&transaction)
	}
	c.Errors = make([]*provide.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &provide.Error{
			Message: StringOrNil("Unable to associate contract with unspecified network"),
		})
	} else if transaction != nil && c.NetworkID != transaction.NetworkID {
		c.Errors = append(c.Errors, &provide.Error{
			Message: StringOrNil("Contract network did not match transaction network"),
		})
	}
	return len(c.Errors) == 0
}
