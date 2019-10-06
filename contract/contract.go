package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
	provide "github.com/provideservices/provide-go"
)

const resolveTokenTickerInterval = time.Millisecond * 5000
const resolveTokenTickerTimeout = time.Minute * 1

const natsTxCreateSubject = "goldmine.tx.create"

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
	return dbconf.DatabaseConnection().Select("contracts.id, contracts.created_at, contracts.accessed_at, contracts.application_id, contracts.network_id, contracts.transaction_id, contracts.contract_id, contracts.name, contracts.address")
}

// CompiledArtifact - parse the original JSON params used for contract creation and attempt to unmarshal to a provide.CompiledArtifact
func (c *Contract) CompiledArtifact() *provide.CompiledArtifact {
	artifact := &provide.CompiledArtifact{}
	params := c.ParseParams()
	if params != nil {
		if compiledArtifact, compiledArtifactOk := params["compiled_artifact"].(map[string]interface{}); compiledArtifactOk {
			compiledArtifactJSON, _ := json.Marshal(compiledArtifact)
			compiledArtifactRawJSON := json.RawMessage(compiledArtifactJSON)
			err := json.Unmarshal(compiledArtifactRawJSON, &artifact)
			if err != nil {
				common.Log.Warningf("Failed to unmarshal contract params to compiled artifact; %s", err.Error())
				return nil
			}
		}
	}
	return artifact
}

// GetNetwork - retrieve the associated contract network
func (c *Contract) GetNetwork() (*network.Network, error) {
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(c).Related(&network)
	if network == nil {
		return nil, fmt.Errorf("Failed to retrieve contract network for contract: %s", c.ID)
	}
	return network, nil
}

// ParseParams - parse the original JSON params used for contract creation
func (c *Contract) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if c.Params != nil {
		err := json.Unmarshal(*c.Params, &params)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal contract params; %s", err.Error())
			return nil
		}
	}
	return params
}

// ExecuteEthereumContract to execute an ethereum contract; returns the tx receipt and retvals; if the method is constant, the receipt will be nil.
// If the methid is non-constant, the retvals will be nil.
func (c *Contract) ExecuteEthereumContract(
	network *network.Network,
	txResponseCallback func(
		*Contract, // contract
		*network.Network, //network
		string, // methodDescriptor
		string, // method
		*abi.Method, // abiMethod
		[]interface{}) (*interface{}, *interface{}, error), // params
	method string,
	params []interface{}) (*interface{}, *interface{}, error) { // given tx has been built but broadcast has not yet been attempted
	defer func() {
		go func() {
			accessedAt := time.Now()
			c.AccessedAt = &accessedAt
			db := dbconf.DatabaseConnection()
			db.Save(c)
		}()
	}()

	var err error
	_abi, err := c.ReadEthereumContractAbi()
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

	// call tx callback
	return txResponseCallback(c, network, methodDescriptor, method, abiMethod, params)
}

// ReadEthereumContractAbi is called from token
func (c *Contract) ReadEthereumContractAbi() (*abi.ABI, error) {
	var _abi *abi.ABI
	params := c.ParseParams()
	if contractAbi, ok := params["abi"]; ok {
		abistr, err := json.Marshal(contractAbi)
		if err != nil {
			common.Log.Warningf("Failed to marshal ABI from contract params to json; %s", err.Error())
			return nil, err
		}

		abival, err := abi.JSON(strings.NewReader(string(abistr)))
		if err != nil {
			common.Log.Warningf("Failed to initialize ABI from contract params to json; %s", err.Error())
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
					Message: common.StringOrNil(err.Error()),
				})
			}
		}

		if !db.NewRecord(c) {
			success := rowsAffected > 0
			if success {
				compiledArtifact := c.CompiledArtifact()
				if compiledArtifact != nil {
					params := c.ParseParams()
					delete(params, "compiled_artifact")
					value := uint64(0)
					if val, valOk := params["value"].(float64); valOk {
						value = uint64(val)
					}
					txCreationMsg, _ := json.Marshal(map[string]interface{}{
						"contract_id":  c.ID,
						"data":         compiledArtifact.Bytecode,
						"wallet_id":    common.StringOrNil(params["wallet_id"].(string)),
						"value":        value,
						"params":       params,
						"published_at": time.Now(),
					})
					err := natsutil.NatsPublish(natsTxCreateSubject, txCreationMsg)
					if err != nil {
						common.Log.Warningf("Failed to publish contract deployment tx; %s", err.Error())
					}
				}
			}
			return success
		}
	}

	return false
}

// Execute a transaction on the contract instance using a specific signer, value, method and params
// func (c *Contract) Execute(execution *ContractExecution) (*ContractExecutionResponse, error) {
// 	var err error
// 	db := dbconf.DatabaseConnection()
// 	var network = &Network{}
// 	db.Model(c).Related(&network)

// 	ref := execution.Ref
// 	wallet := execution.Wallet
// 	value := execution.Value
// 	method := execution.Method
// 	params := execution.Params
// 	gas := execution.Gas
// 	nonce := execution.Nonce
// 	publishedAt := execution.PublishedAt

// 	txParams := map[string]interface{}{}

// 	walletID := &uuid.Nil
// 	if wallet != nil {
// 		if wallet.ID != uuid.Nil {
// 			walletID = &wallet.ID
// 		} else if wallet.Address != "" {
// 			tmpWallet := &Wallet{}
// 			db.Where("address = ?", wallet.Address).Find(&tmpWallet)
// 			if tmpWallet != nil && tmpWallet.ID != uuid.Nil {
// 				walletID = &tmpWallet.ID
// 			}
// 		}
// 		if common.StringOrNil(wallet.Address) != nil && wallet.PrivateKey != nil {
// 			txParams["public_key"] = wallet.Address
// 			txParams["private_key"] = wallet.PrivateKey
// 		}
// 	}

// 	if gas == nil {
// 		gas64 := float64(0)
// 		gas = &gas64
// 	}
// 	txParams["gas"] = gas

// 	if nonce != nil {
// 		txParams["nonce"] = *nonce
// 	}

// 	txParamsJSON, _ := json.Marshal(txParams)
// 	_txParamsJSON := json.RawMessage(txParamsJSON)

// 	tx := &Transaction{
// 		ApplicationID: c.ApplicationID,
// 		UserID:        nil,
// 		NetworkID:     c.NetworkID,
// 		WalletID:      walletID,
// 		To:            c.Address,
// 		Value:         &TxValue{value: value},
// 		Params:        &_txParamsJSON,
// 		Ref:           ref,
// 	}

// 	if publishedAt != nil {
// 		tx.PublishedAt = publishedAt
// 	}

// 	var receipt *interface{}
// 	var response *interface{}

// 	if network.IsEthereumNetwork() {
// 		receipt, response, err = c.ExecuteEthereumContract(network, tx, method, params)
// 	} else {
// 		err = fmt.Errorf("unsupported network: %s", *network.Name)
// 	}

// 	if err != nil {
// 		desc := err.Error()
// 		tx.updateStatus(db, "failed", &desc)
// 		return nil, fmt.Errorf("Unable to execute %s contract; %s", *network.Name, err.Error())
// 	}

// 	accessedAt := time.Now()
// 	go func() {
// 		c.AccessedAt = &accessedAt
// 		db.Save(c)
// 	}()

// 	if tx.Response == nil {
// 		tx.Response = &ContractExecutionResponse{
// 			Response:    response,
// 			Receipt:     receipt,
// 			Traces:      tx.Traces,
// 			Transaction: tx,
// 			Ref:         ref,
// 		}
// 	} else if tx.Response.Transaction == nil {
// 		tx.Response.Transaction = tx
// 	}

// 	return tx.Response, nil
// }

// ExecuteFromTx executes a transaction on the contract instance using a tx callback, specific signer, value, method and params
func (c *Contract) ExecuteFromTx(
	execution *ContractExecution,
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *ContractExecution, *json.RawMessage) (*ContractExecutionResponse, error)) (*ContractExecutionResponse, error) {

	var err error
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(c).Related(&network)

	// ref := execution.Ref
	wallet := execution.Wallet
	// value := execution.Value
	// method := execution.Method
	// params := execution.Params
	gas := execution.Gas
	nonce := execution.Nonce
	// publishedAt := execution.PublishedAt

	txParams := map[string]interface{}{}

	walletID := walletFn(wallet, txParams)

	if gas == nil {
		gas64 := float64(0)
		gas = &gas64
	}
	txParams["gas"] = gas

	if nonce != nil {
		txParams["nonce"] = *nonce
	}

	txParamsJSON, _ := json.Marshal(txParams)
	_txParamsJSON := json.RawMessage(txParamsJSON)

	var txResponse *ContractExecutionResponse
	txResponse, err = txCreateFn(c, network, walletID, execution, &_txParamsJSON)
	return txResponse, err
}

// ResolveTokenContract called in tx/tx
func (c *Contract) ResolveTokenContract(db *gorm.DB, network *network.Network, walletAddress *string, client *ethclient.Client, receipt *types.Receipt, tokenCreateFn func(*Contract, string, *big.Int, string) (bool, uuid.UUID, []*provide.Error)) {
	ticker := time.NewTicker(resolveTokenTickerInterval)
	go func() {
		startedAt := time.Now()
		for {
			select {
			case <-ticker.C:
				if time.Now().Sub(startedAt) >= resolveTokenTickerTimeout {
					common.Log.Warningf("Failed to resolve ERC20 token for contract: %s; timing out after %v", c.ID, resolveTokenTickerTimeout)
					ticker.Stop()
					return
				}

				artifact := c.CompiledArtifact()
				if artifact == nil {
					common.Log.Warningf("Failed to resolve compiled artifact during attempt to resolve ERC20 token for contract: %s", c.ID)
					return
				}
				if artifact.ABI != nil {
					abistr, err := json.Marshal(artifact.ABI)
					if err != nil {
						common.Log.Warningf("Failed to marshal contract abi to json...  %s", err.Error())
					}
					_abi, err := abi.JSON(strings.NewReader(string(abistr)))
					if err == nil {
						msg := ethereum.CallMsg{
							From:     ethcommon.HexToAddress(*walletAddress),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("name()")),
						}

						result, _ := client.CallContract(context.TODO(), msg, nil)
						var name string
						if method, ok := _abi.Methods["name"]; ok {
							err = method.Outputs.Unpack(&name, result)
							if err != nil {
								common.Log.Warningf("Failed to read %s, contract name from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     ethcommon.HexToAddress(*walletAddress),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("decimals()")),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var decimals *big.Int
						if method, ok := _abi.Methods["decimals"]; ok {
							err = method.Outputs.Unpack(&decimals, result)
							if err != nil {
								common.Log.Warningf("Failed to read %s, contract decimals from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     ethcommon.HexToAddress(*walletAddress),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: big.NewInt(0),
							Value:    nil,
							Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("symbol()")),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var symbol string
						if method, ok := _abi.Methods["symbol"]; ok {
							err = method.Outputs.Unpack(&symbol, result)
							if err != nil {
								common.Log.Warningf("Failed to read %s, contract symbol from deployed contract %s; %s", *network.Name, c.ID, err.Error())
							}
						}

						if name != "" && decimals != nil && symbol != "" { // isERC20Token
							common.Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)
							// token := &token.Token{
							// 	ApplicationID: c.ApplicationID,
							// 	NetworkID:     c.NetworkID,
							// 	ContractID:    &c.ID,
							// 	Name:          common.StringOrNil(name),
							// 	Symbol:        common.StringOrNil(symbol),
							// 	Decimals:      decimals.Uint64(),
							// 	Address:       common.StringOrNil(receipt.ContractAddress.Hex()),
							// }
							res, id, errs := tokenCreateFn(c, name, decimals, symbol)

							if res {
								common.Log.Debugf("Created token %s for associated %s contract: %s", id, *network.Name, c.ID)
								ticker.Stop()
								return
							} else if len(errs) > 0 {
								common.Log.Warningf("Failed to create token for associated %s contract creation %s; %d errs: %s", *network.Name, c.ID, len(errs), *errs[0].Message)
							}
						}
					} else {
						common.Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
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

// not used
// GetTransaction - retrieve the associated contract creation transaction
// func (c *Contract) GetTransaction() (*Transaction, error) {
// 	var tx = &Transaction{}
// 	db := dbconf.DatabaseConnection()
// 	db.Model(c).Related(&tx)
// 	if tx == nil || tx.ID == uuid.Nil {
// 		return nil, fmt.Errorf("Failed to retrieve tx for contract: %s", c.ID)
// 	}
// 	return tx, nil
// }

// Validate a contract for persistence
func (c *Contract) Validate() bool {
	// db := dbconf.DatabaseConnection()
	// var transaction *tx.Transaction
	// if c.TransactionID != nil {
	// 	transaction = &Transaction{}
	// 	db.Model(c).Related(&transaction)
	// }
	c.Errors = make([]*provide.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &provide.Error{
			Message: common.StringOrNil("Unable to associate contract with unspecified network"),
		})
	}
	// else if transaction != nil && c.NetworkID != transaction.NetworkID {
	// 	c.Errors = append(c.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Contract network did not match transaction network"),
	// 	})
	// }
	return len(c.Errors) == 0
}
