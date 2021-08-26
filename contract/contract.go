package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
	provide "github.com/provideservices/provide-go/api"
	api "github.com/provideservices/provide-go/api/nchain"
	prvdcrypto "github.com/provideservices/provide-go/crypto"
)

const resolveTokenTickerInterval = time.Millisecond * 5000
const resolveTokenTickerTimeout = time.Minute * 1

const natsTxCreateSubject = "nchain.tx.create"

// Contract instances must be associated with an application identifier.
type Contract struct {
	provide.Model
	ApplicationID  *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	OrganizationID *uuid.UUID       `sql:"type:uuid" json:"organization_id"`
	NetworkID      uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ContractID     *uuid.UUID       `sql:"type:uuid" json:"contract_id"`    // id of the contract which created the contract (or null)
	TransactionID  *uuid.UUID       `sql:"type:uuid" json:"transaction_id"` // id of the transaction which deployed the contract (or null)
	Name           *string          `sql:"not null" json:"name"`
	Address        *string          `sql:"not null" json:"address"`
	Type           *string          `json:"type"`
	Params         *json.RawMessage `sql:"type:json" json:"params,omitempty"`
	AccessedAt     *time.Time       `json:"accessed_at"`
	PubsubPrefix   *string          `sql:"-" json:"pubsub_prefix,omitempty"`
}

// ContractListQuery returns a DB query configured to select columns suitable for a paginated API response
func ContractListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("contracts.id, contracts.created_at, contracts.accessed_at, contracts.application_id, contracts.organization_id, contracts.network_id, contracts.transaction_id, contracts.contract_id, contracts.name, contracts.address, contracts.type")
}

// enrich enriches the contract
func (c *Contract) enrich() {
	c.PubsubPrefix = c.pubsubSubjectPrefix()
}

// CompiledArtifact - parse the original JSON params used for contract creation and attempt to unmarshal to a api.CompiledArtifact
func (c *Contract) CompiledArtifact() *api.CompiledArtifact {
	artifact := &api.CompiledArtifact{}
	params := c.ParseParams()

	if params != nil {
		if compiledArtifact, compiledArtifactOk := params["compiled_artifact"].(map[string]interface{}); compiledArtifactOk {
			compiledArtifactJSON, _ := json.Marshal(compiledArtifact)
			compiledArtifactRawJSON := json.RawMessage(compiledArtifactJSON)
			err := json.Unmarshal(compiledArtifactRawJSON, &artifact)
			if err != nil {
				common.Log.Warningf("failed to unmarshal contract params to compiled artifact; %s", err.Error())
				return nil
			}
		}
	}
	return artifact
}

// Find - retrieve a specific contract for the given network and address
func Find(db *gorm.DB, networkID uuid.UUID, addr string) *Contract {
	cntract := &Contract{}
	db.Where("network_id = ? AND address = ?", networkID, addr).Find(&cntract)
	if cntract == nil || cntract.ID == uuid.Nil {
		return nil
	}
	return cntract
}

// FindByTxID - retrieve a specific contract for a given tx id
func FindByTxID(db *gorm.DB, txID uuid.UUID) *Contract {
	cntract := &Contract{}
	db.Where("transaction_id = ?", txID).Find(&cntract)
	if cntract == nil || cntract.ID == uuid.Nil {
		return nil
	}
	return cntract
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

// ExecuteEthereumContract execute an ethereum contract; returns the tx receipt or retvals,
// depending on if the execution is asynchronous or not. If the method is non-constant, the
// response is the tx receipt. When the method is constant, retvals are returned.
func (c *Contract) ExecuteEthereumContract(
	network *network.Network,
	txResponseCallback func(
		*Contract, // contract
		*network.Network, //network
		string, // methodDescriptor
		string, // method
		*abi.Method, // abiMethod
		[]interface{}) (map[string]interface{}, error), // params
	method string,
	params []interface{}) (map[string]interface{}, error) { // given tx has been built but broadcast has not yet been attempted
	defer func() {
		go func() {
			accessedAt := time.Now()
			c.AccessedAt = &accessedAt
			db := dbconf.DatabaseConnection()
			db.Save(c)
		}()
	}()

	if !network.IsEthereumNetwork() {
		return nil, fmt.Errorf("Failed to execute EVM-based smart contract method %s on contract: %s; target network invalid", method, c.ID)
	}

	var err error
	_abi, err := c.ReadEthereumContractAbi()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute contract method %s on contract: %s; no ABI resolved: %s", method, c.ID, err.Error())
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
	contractAbi, contractAbiOk := params["abi"]
	if !contractAbiOk {
		artifact := c.CompiledArtifact()
		if artifact != nil {
			contractAbi = artifact.ABI
		}
	}

	if contractAbi != nil {
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

// ResolveCompiledDependencyArtifact returns the compiled artifact if matched to the given descriptor;
// in the case of EVM-based solidity contracts, the descriptor can be a contract name or its raw bytecode
func (c *Contract) ResolveCompiledDependencyArtifact(descriptor string) *api.CompiledArtifact {
	artifact := c.CompiledArtifact()
	if artifact == nil {
		return nil
	}

	var dependencyArtifact *api.CompiledArtifact

	for _, dep := range artifact.Deps {
		dependency := dep.(map[string]interface{})
		var name string
		if depname, depnameOk := dependency["name"].(string); depnameOk {
			name = depname
		} else if depname, depnameOk := dependency["contractName"].(string); depnameOk {
			name = depname
		}
		nameOk := name != ""

		fingerprint, fingerprintOk := dependency["fingerprint"].(string)
		if !nameOk && !fingerprintOk {
			continue
		}

		common.Log.Debugf("checking if compiled artifact dependency: %s (fingerprint: %s) is target of contract-internal CREATE opcode at address: %s", name, fingerprint, *c.Address)
		if name == descriptor {
			depJSON, _ := json.Marshal(dependency)
			json.Unmarshal(depJSON, &dependencyArtifact)
			break
		} else if strings.HasSuffix(descriptor, fingerprint) {
			depJSON, _ := json.Marshal(dependency)
			json.Unmarshal(depJSON, &dependencyArtifact)
			break
		}
	}

	return dependencyArtifact
}

// persist a contract without deploying it to the network
func (c *Contract) Save() bool {
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
		success := rowsAffected > 0
		return success
	}
	return false
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
			if success && c.ContractID == nil { // when ContractID is non-nil, the deployment happened in a contract-internal tx
				compiledArtifact := c.CompiledArtifact()
				if compiledArtifact != nil {
					params := c.ParseParams()
					delete(params, "compiled_artifact")
					value := uint64(0)
					if val, valOk := params["value"].(float64); valOk {
						value = uint64(val)
					}

					var accountID *string
					if acctID, acctIDOk := params["account_id"].(string); acctIDOk {
						accountID = &acctID
					}

					var walletID *string
					var hdDerivationPath *string
					if wlltID, wlltIDOk := params["wallet_id"].(string); wlltIDOk {
						walletID = &wlltID

						if path, pathOk := params["hd_derivation_path"].(string); pathOk {
							hdDerivationPath = &path
						}
					}

					// FIXME-- should not be here.
					_abi, err := c.ReadEthereumContractAbi()
					if err != nil {
						c.Errors = append(c.Errors, &provide.Error{
							Message: common.StringOrNil(fmt.Sprintf("failed to create contract; %s", err.Error())),
						})
						return false
					}

					data := compiledArtifact.Bytecode
					if argv, argvOk := params["argv"].([]interface{}); argvOk {
						if len(argv) > 0 {
							encodedArgv, err := prvdcrypto.EVMEncodeABI(&_abi.Constructor, argv...)
							if err != nil {
								common.Log.Warningf("failed to encode contract constructor args; %s", err.Error())
								c.Errors = append(c.Errors, &provide.Error{
									Message: common.StringOrNil(fmt.Sprintf("failed to encode contract constructor args; %s", err.Error())),
								})
								return false
							}
							data = fmt.Sprintf("%s%x", data, string(encodedArgv))
						}
					}

					txCreationMsg, _ := json.Marshal(map[string]interface{}{
						"contract_id":        c.ID,
						"data":               data,
						"account_id":         accountID,
						"wallet_id":          walletID,
						"hd_derivation_path": hdDerivationPath,
						"value":              value,
						"params":             params,
						"published_at":       time.Now(),
					})

					err = natsutil.NatsStreamingPublish(natsTxCreateSubject, txCreationMsg)
					if err != nil {
						common.Log.Warningf("Failed to publish contract deployment tx; %s", err.Error())
						c.Errors = append(c.Errors, &provide.Error{
							Message: common.StringOrNil(err.Error()),
						})
						success = false
					}

					// err := consumer.BroadcastFragments(txCreationMsg, false, common.StringOrNil(natsTxCreateSubject))
					// if err != nil {
					// 	common.Log.Warningf("Failed to broadcast fragmented contract deployment message; %s", err.Error())
					// }
				}
			}
			return success
		}
	}

	return false
}

// pubsubSubjectPrefix returns a hash for use as the pub/sub subject prefix for the contract
func (c *Contract) pubsubSubjectPrefix() *string {
	if c.ApplicationID != nil {
		digest := sha256.New()
		digest.Write([]byte(fmt.Sprintf("%s.%s", c.ApplicationID.String(), *c.Address)))
		return common.StringOrNil(hex.EncodeToString(digest.Sum(nil)))
	} else if c.OrganizationID != nil {
		digest := sha256.New()
		digest.Write([]byte(fmt.Sprintf("%s.%s", c.OrganizationID.String(), *c.Address)))
		return common.StringOrNil(hex.EncodeToString(digest.Sum(nil)))
	}

	return nil
}

// qualifiedSubject returns a namespaced subject suitable for pub/sub subscriptions
func (c *Contract) qualifiedSubject(suffix string) *string {
	prefix := c.pubsubSubjectPrefix()
	if prefix == nil {
		return nil
	}
	if suffix == "" {
		return prefix
	}
	return common.StringOrNil(fmt.Sprintf("%s.%s", *prefix, suffix))
}

// networkQualifiedSubject returns the contract subject suitable for pub/sub subscriptions
// using the unhashed `networks.<id>.contracts.<address>` approach
func (c *Contract) networkQualifiedSubject(suffix *string) *string {
	if c.Address == nil {
		return nil
	}
	if suffix == nil {
		return common.StringOrNil(fmt.Sprintf("network.%s.contracts.%s", c.NetworkID, *c.Address))
	}
	return common.StringOrNil(fmt.Sprintf("network.%s.contracts.%s.%s", c.NetworkID, *c.Address, *suffix))
}

// ExecuteFromTx executes a transaction on the contract instance using a tx callback, specific signer, value, method and params
func (c *Contract) ExecuteFromTx(
	execution *Execution,
	accountFn func(interface{}, map[string]interface{}) *uuid.UUID,
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *uuid.UUID, *Execution, *json.RawMessage) (*ExecutionResponse, error)) (*ExecutionResponse, error) {

	var err error
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(c).Related(&network)

	// ref := execution.Ref
	account := execution.Account
	wallet := execution.Wallet
	// value := execution.Value
	// method := execution.Method
	// params := execution.Params
	gas := execution.Gas
	gasPrice := execution.GasPrice
	nonce := execution.Nonce

	//xxx add path to params
	path := execution.HDPath

	// publishedAt := execution.PublishedAt

	txParams := map[string]interface{}{}
	if c.Address != nil {
		txParams["to"] = *c.Address
	}

	// TODO get the hd-derivation-path (if present) into the txparams

	accountID := accountFn(account, txParams)
	walletID := walletFn(wallet, txParams)

	if gas == nil {
		gas64 := float64(0)
		gas = &gas64
	}
	txParams["gas"] = gas

	if gasPrice != nil {
		txParams["gas_price"] = gasPrice
	}

	if nonce != nil {
		txParams["nonce"] = *nonce
	}

	// xxx add path to params
	if path != nil {
		txParams["hd_derivation_path"] = *path
	}

	txParamsJSON, _ := json.Marshal(txParams)
	_txParamsJSON := json.RawMessage(txParamsJSON)

	var txResponse *ExecutionResponse
	txResponse, err = txCreateFn(c, network, accountID, walletID, execution, &_txParamsJSON)
	return txResponse, err
}

// ResolveTokenContract called in tx/tx
func (c *Contract) ResolveTokenContract(
	db *gorm.DB,
	network *network.Network,
	signerAddress string,
	receipt interface{},
	tokenCreateFn func(*Contract, string, string, *big.Int, string) (bool, uuid.UUID, []*provide.Error),
) {
	p2pAPI, err := network.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Unable to attempt token contract resolution for contract id %s; no P2P API client; %s", c.ID, err.Error())
		return
	}

	ticker := time.NewTicker(resolveTokenTickerInterval)
	go func() {
		startedAt := time.Now()
		for {
			select {
			case <-ticker.C:
				if time.Since(startedAt) >= resolveTokenTickerTimeout {
					common.Log.Warningf("failed to resolve ERC20 token for contract: %s; timing out after %v", c.ID, resolveTokenTickerTimeout)
					ticker.Stop()
					return
				}

				artifact := c.CompiledArtifact()
				if artifact == nil {
					common.Log.Warningf("unable to attempt token contract resolution for contract id %s; no compiled artifact", c.ID)
					return
				}

				tokenType, name, decimals, symbol, err := p2pAPI.ResolveTokenContract(signerAddress, receipt, artifact)
				if err != nil {
					common.Log.Debugf("contract id %s did not match a supported token contract standard; %s", c.ID, err.Error())
					return
				}

				res, id, errs := tokenCreateFn(c, *tokenType, *name, decimals, *symbol)
				if res {
					common.Log.Debugf("created token %s for associated %s contract: %s", id, *network.Name, c.ID)
					ticker.Stop()
					return
				} else if len(errs) > 0 {
					common.Log.Warningf("failed to create token for associated %s contract creation %s; %d errs: %s", *network.Name, c.ID, len(errs), *errs[0].Message)
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
