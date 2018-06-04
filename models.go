package main

import (
	"crypto/ecdsa"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/provideapp/go-core"
	provide "github.com/provideservices/provide-go"

	"github.com/jinzhu/gorm"
	"github.com/kthomas/go.uuid"
)

// Network
type Network struct {
	gocore.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	Name          *string          `sql:"not null" json:"name"`
	Description   *string          `json:"description"`
	IsProduction  *bool            `sql:"not null" json:"is_production"`
	Cloneable     *bool            `sql:"not null" json:"cloneable"`
	Enabled       *bool            `sql:"not null" json:"enabled"`
	ChainID       *string          `json:"chain_id"`                     // protocol-specific chain id
	SidechainID   *uuid.UUID       `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	NetworkID     *uuid.UUID       `sql:"type:uuid" json:"network_id"`   // network id used as the parent
	Config        *json.RawMessage `sql:"type:json" json:"config"`
}

// NetworkNode
type NetworkNode struct {
	gocore.Model
	NetworkID   uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	UserID      *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	Host        *string          `json:"host"`
	Description *string          `json:"description"`
	Status      *string          `sql:"not null;default:'pending'" json:"status"`
	Config      *json.RawMessage `sql:"type:json" json:"config"`
}

// Bridge instances are still in the process of being defined.
type Bridge struct {
	gocore.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"-"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
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
	Method   string        `json:"method"`
	Params   []interface{} `json:"params"`
	Value    *big.Int      `json:"value"`
}

// ContractExecutionResponse is returned upon successful contract execution
type ContractExecutionResponse struct {
	Receipt     interface{}  `json:"receipt"`
	Traces      interface{}  `json:"traces"`
	Transaction *Transaction `json:"transaction"`
}

// Oracle instances are smart contracts whose terms are fulfilled by writing data from a configured feed onto the blockchain associated with its configured network
type Oracle struct {
	gocore.Model
	ApplicationID *uuid.UUID       `sql:"not null;type:uuid" json:"-"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ContractID    uuid.UUID        `sql:"not null;type:uuid" json:"contract_id"`
	Name          *string          `sql:"not null" json:"name"`
	FeedURL       *url.URL         `sql:"not null" json:"feed_url"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
	AttachmentIds []*uuid.UUID     `sql:"type:uuid[]" json:"attachment_ids"`
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
	ApplicationID *uuid.UUID                 `sql:"type:uuid" json:"-"`
	UserID        *uuid.UUID                 `sql:"type:uuid" json:"-"`
	NetworkID     uuid.UUID                  `sql:"not null;type:uuid" json:"network_id"`
	WalletID      uuid.UUID                  `sql:"not null;type:uuid" json:"wallet_id"`
	To            *string                    `json:"to"`
	Value         *TxValue                   `sql:"not null;type:decimal;default:0" json:"value"`
	Data          *string                    `json:"data"`
	Hash          *string                    `sql:"not null" json:"hash"`
	Status        *string                    `sql:"not null;default:'pending'" json:"status"`
	Params        *json.RawMessage           `sql:"-" json:"params"`
	Response      *ContractExecutionResponse `sql:"-" json:"-"`
	SignedTx      interface{}                `sql:"-" json:"-"`
	Traces        interface{}                `sql:"-" json:"traces"`
}

// Wallet instances must be associated with exactly one instance of either an a) application identifier or b) user identifier.
type Wallet struct {
	gocore.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"-"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"-"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	Address       string     `sql:"not null" json:"address"`
	PrivateKey    *string    `sql:"not null;type:bytea" json:"-"`
	Balance       *big.Int   `sql:"-" json:"balance"`
}

type TxValue struct {
	value *big.Int
}

func (v *TxValue) Value() (driver.Value, error) {
	f := new(big.Float)
	f.SetUint64(v.value.Uint64())
	f64, _ := f.Float64()
	return f64, nil
}

func (v *TxValue) Scan(val interface{}) error {
	f64 := new(sql.NullFloat64)
	err := f64.Scan(val)
	if err != nil {
		return err
	}
	v.value = new(big.Int)
	if f64.Valid {
		v.value.SetUint64(uint64(f64.Float64)) // HACK -- loss of precision possible
	}
	return nil
}

func (v *TxValue) BigInt() *big.Int {
	return v.value
}

func (v *TxValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *TxValue) UnmarshalJSON(data []byte) error {
	v.value = new(big.Int)
	v.value.SetString(string(data), 10)
	return nil
}

// Create and persist a new network
func (n *Network) Create() bool {
	if !n.Validate() {
		return false
	}

	db := DatabaseConnection()

	if db.NewRecord(n) {
		result := db.Create(&n)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				n.Errors = append(n.Errors, &gocore.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(n) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate a network for persistence
func (n *Network) Validate() bool {
	n.Errors = make([]*gocore.Error, 0)
	return len(n.Errors) == 0
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

func (n *Network) rpcURL() string {
	cfg := n.ParseConfig()
	if rpcURL, ok := cfg["json_rpc_url"].(string); ok {
		return rpcURL
	} else if rpcURL, ok := cfg["parity_json_rpc_url"].(string); ok {
		return rpcURL
	}
	return ""
}

// Status retrieves metadata and metrics specific to the given network
func (n *Network) Status() (status *provide.NetworkStatus, err error) {
	if n.isEthereumNetwork() {
		status, err = provide.GetNetworkStatus(n.ID.String(), n.rpcURL())
	} else {
		Log.Warningf("Unable to determine status of unsupported network: %s", *n.Name)
	}
	if err != nil {
		Log.Warningf("Unable to determine status of %s network; %s", *n.Name, err.Error())
	}
	return status, err
}

func (n *Network) isEthereumNetwork() bool {
	if strings.HasPrefix(strings.ToLower(*n.Name), "eth") {
		return true
	}

	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg["is_ethereum_network"]; ok {
			if _ok := isEthereumNetwork.(bool); _ok {
				return isEthereumNetwork.(bool)
			}
		}
		if _, ok := cfg["parity_json_rpc_url"]; ok {
			return true
		}
	}
	return false
}

// Create and persist a new network node
func (n *NetworkNode) Create() bool {
	if !n.Validate() {
		return false
	}

	db := DatabaseConnection()

	if db.NewRecord(n) {
		result := db.Create(&n)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				n.Errors = append(n.Errors, &gocore.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(n) {
			success := rowsAffected > 0
			if success {
				n.deploy()
			}
			return success
		}
	}
	return false
}

// Validate a network node for persistence
func (n *NetworkNode) Validate() bool {
	n.Errors = make([]*gocore.Error, 0)
	return len(n.Errors) == 0
}

// ParseConfig - parse the network node configuration JSON
func (n *NetworkNode) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			Log.Warningf("Failed to unmarshal network node config; %s", err.Error())
			return nil
		}
	}
	return config
}

// Delete a network node
func (n *NetworkNode) Delete() bool {
	n.undeploy()
	db := DatabaseConnection()
	result := db.Delete(n)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			n.Errors = append(n.Errors, &gocore.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
	return len(n.Errors) == 0
}

func (n *NetworkNode) clone(cfg json.RawMessage) *NetworkNode {
	clone := &NetworkNode{
		NetworkID:   n.NetworkID,
		UserID:      n.UserID,
		Description: n.Description,
		Status:      n.Status,
		Config:      &cfg,
	}
	clone.Create()
	return clone
}

func (n *NetworkNode) deploy() error {
	Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, n)

	db := DatabaseConnection()

	var network = &Network{}
	db.Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		return fmt.Errorf("Failed to retrieve network for network node: %s", n.ID)
	}

	cfg := n.ParseConfig()
	networkCfg := network.ParseConfig()

	targetID, targetOk := cfg["target_id"].(string)
	role, roleOk := cfg["role"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	rcd, rcdOk := cfg["rc.d"].(string)
	region, regionOk := cfg["region"].(string)
	cloneableImages, cloneableImagesOk := networkCfg["cloneable_images"].(map[string]interface{})
	cloneableImagesByRegion, cloneableImagesByRegionOk := cloneableImages[targetID].(map[string]interface{})["regions"].(map[string]interface{})

	regions, regionsOk := cfg["regions"].(map[string]interface{})
	if regionsOk && !regionOk {
		delete(cfg, "regions")
		Log.Debugf("Handling multi-region deployment request for network node: %s", n.ID)
		accountedForInitialDeploy := false
		for _region := range regions {
			deployCount, deployCountOk := regions[_region]
			if deployCountOk && deployCount.(float64) > 0 {
				Log.Debugf("Multi-region deployment request specified %v instances in %s region for network node: %s", deployCount, _region, n.ID)
				for i := float64(0); i < deployCount.(float64); i++ {
					if !accountedForInitialDeploy {
						region = _region
						regionOk = true
						accountedForInitialDeploy = true
						continue
					}

					_cfg := map[string]interface{}{}
					for key, val := range cfg {
						_cfg[key] = val
					}
					_cfg["region"] = _region
					_cfgJSON, _ := json.Marshal(_cfg)
					n.clone(json.RawMessage(_cfgJSON))
				}
			}
		}
	}

	Log.Debugf("Configuration for network node deploy: target id: %s; role: %s; crendentials: %s; region: %s, rc.d: %s; cloneable images: %s; network config: %s",
		targetID, role, credentials, region, rcd, cloneableImages, networkCfg)

	if targetOk && roleOk && credsOk && regionOk && cloneableImagesOk && cloneableImagesByRegionOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			var userData = ""
			if rcdOk {
				userData = rcd
			}

			Log.Debugf("Attempting to deploy network node instance(s) in EC2 region: %s", region)
			if imagesByRegion, imagesByRegionOk := cloneableImagesByRegion[region].(map[string]interface{}); imagesByRegionOk {
				Log.Debugf("Resolved deployable images by region in EC2 region: %s", region)
				if imageVersionsByRole, imageVersionsByRoleOk := imagesByRegion[role].(map[string]interface{}); imageVersionsByRoleOk {
					Log.Debugf("Resolved deployable image versions for role: %s; in EC2 region: %s", role, region)
					versions := make([]string, 0)
					for version := range imageVersionsByRole {
						versions = append(versions, version)
					}
					Log.Debugf("Resolved %v deployable image version(s) for role: %s", len(versions), role)
					version := versions[len(versions)-1] // defaults to latest version for now
					Log.Debugf("Attempting to lookup update for version: %s", version)
					if imageID, imageIDOk := imageVersionsByRole[version].(string); imageIDOk {
						Log.Debugf("Attempting to deploy image %s@@%s in EC2 region: %s", imageID, version, region)
						instanceIds, err := LaunchAMI(accessKeyID, secretAccessKey, region, imageID, userData, 1, 1)
						if err != nil {
							return fmt.Errorf("Attempt to deploy image %s@%s in EC2 %s region failed; %s", imageID, version, region, err.Error())
						}
						Log.Debugf("Attempt to deploy image %s@%s in EC2 %s region successful; instance ids: %s", imageID, version, region, instanceIds)
						cfg["region"] = region
						cfg["target_instance_ids"] = instanceIds
						for n.Host == nil {
							instanceID := instanceIds[len(instanceIds)-1]
							instanceDetails, err := GetInstanceDetails(accessKeyID, secretAccessKey, region, instanceID)
							if err == nil {
								if len(instanceDetails.Reservations) > 0 {
									reservation := instanceDetails.Reservations[0]
									if len(reservation.Instances) > 0 {
										instance := reservation.Instances[0]
										n.Host = stringOrNil(*instance.PublicDnsName)
									}
								}
							}
						}
						cfgJSON, _ := json.Marshal(cfg)
						*n.Config = json.RawMessage(cfgJSON)
						n.Status = stringOrNil("running")
						db.Save(n)
						Log.Debugf("Depoyed %v %s@%s instances in EC2 %s region", len(instanceIds), imageID, version, region)
					}
				}
			}
		}
	}

	return nil
}

func (n *NetworkNode) undeploy() error {
	Log.Debugf("Attempting to undeploy network node with id: %s", n.ID, n)

	db := DatabaseConnection()

	cfg := n.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	instanceIds, instanceIdsOk := cfg["target_instance_ids"].([]interface{})
	credentials, credsOk := cfg["credentials"].(map[string]interface{})

	Log.Debugf("Configuration for network node undeploy: target id: %s; crendentials: %s; target instance ids: %s",
		targetID, credentials, instanceIds)

	if targetOk && regionOk && instanceIdsOk && credsOk {
		for i := range instanceIds {
			instanceID := instanceIds[i].(string)

			if strings.ToLower(targetID) == "aws" {
				accessKeyID := credentials["aws_access_key_id"].(string)
				secretAccessKey := credentials["aws_secret_access_key"].(string)

				_, err := TerminateInstance(accessKeyID, secretAccessKey, region, instanceID)
				if err == nil {
					Log.Debugf("Terminated EC2 instance with id: %s", instanceID)
					n.Status = stringOrNil("terminated")
					db.Save(n)
				} else {
					Log.Warningf("Failed to terminate EC2 instance with id: %s", instanceID)
				}
			}
		}
	}

	return nil
}

// ContractListQuery returns a DB query configured to select columns suitable for a paginated API response
func ContractListQuery() *gorm.DB {
	return DatabaseConnection().Select("contracts.id, contracts.application_id, contracts.network_id, contracts.transaction_id, contracts.name, contracts.address")
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
func (c *Contract) Execute(walletID *uuid.UUID, value *big.Int, method string, params []interface{}) (*ContractExecutionResponse, error) {
	var err error
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(c).Related(&network)
	var wallet = &Wallet{}
	db.Where("id = ?", walletID).Find(&wallet)

	tx := &Transaction{
		ApplicationID: c.ApplicationID,
		UserID:        nil,
		NetworkID:     c.NetworkID,
		WalletID:      *walletID,
		To:            c.Address,
		Value:         &TxValue{value: value},
	}

	var receipt *interface{}

	if network.isEthereumNetwork() {
		contractAbi, _ := c.readEthereumContractAbi()
		receipt, err = provide.ExecuteContract(network.ID.String(), network.rpcURL(), wallet.Address, tx.To, tx.Data, value, method, contractAbi, params)
	} else {
		err = fmt.Errorf("unsupported network: %s", *network.Name)
	}

	if err != nil {
		tx.updateStatus(db, "failed")
		return nil, fmt.Errorf("Unable to execute %s contract; %s", *network.Name, err.Error())
	} else {
		tx.updateStatus(db, "success")
	}

	if tx.Response == nil {
		tx.Response = &ContractExecutionResponse{
			Receipt:     receipt,
			Traces:      tx.Traces,
			Transaction: tx,
		}
	} else if tx.Response.Transaction == nil {
		tx.Response.Transaction = tx
	}

	return tx.Response, nil
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
			Log.Warningf("Failed to initialize ABI from contract  params to json; %s", err.Error())
			return nil, err
		}

		_abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read ABI from params for contract: %s", c.ID)
	}
	return _abi, nil
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
	var transaction *Transaction
	if c.TransactionID != nil {
		transaction = &Transaction{}
		db.Model(c).Related(&transaction)
	}
	c.Errors = make([]*gocore.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &gocore.Error{
			Message: stringOrNil("Unable to associate contract with unspecified network"),
		})
	} else if transaction != nil && c.NetworkID != transaction.NetworkID {
		c.Errors = append(c.Errors, &gocore.Error{
			Message: stringOrNil("Contract network did not match transaction network"),
		})
	}
	return len(c.Errors) == 0
}

// ParseParams - parse the original JSON params used for oracle creation
func (o *Oracle) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if o.Params != nil {
		err := json.Unmarshal(*o.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal oracle params; %s", err.Error())
			return nil
		}
	}
	return params
}

// Create and persist a new oracle
func (o *Oracle) Create() bool {
	db := DatabaseConnection()

	if !o.Validate() {
		return false
	}

	if db.NewRecord(o) {
		result := db.Create(&o)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				o.Errors = append(o.Errors, &gocore.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(o) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate an oracle for persistence
func (o *Oracle) Validate() bool {
	o.Errors = make([]*gocore.Error, 0)
	if o.NetworkID == uuid.Nil {
		o.Errors = append(o.Errors, &gocore.Error{
			Message: stringOrNil("Unable to deploy oracle using unspecified network"),
		})
	}
	return len(o.Errors) == 0
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

func (t *Token) readEthereumContractAbi() (*abi.ABI, error) {
	contract, err := t.GetContract()
	if err != nil {
		return nil, err
	}
	return contract.readEthereumContractAbi()
}

// GetNetwork - retrieve the associated transaction network
func (t *Transaction) GetNetwork() (*Network, error) {
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(t).Related(&network)
	if network == nil {
		return nil, fmt.Errorf("Failed to retrieve transaction network for tx: %s", t.ID)
	}
	return network, nil
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

func (t *Transaction) updateStatus(db *gorm.DB, status string) {
	t.Status = stringOrNil(status)
	result := db.Save(&t)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errors = append(t.Errors, &gocore.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
}

func (t *Transaction) broadcast(db *gorm.DB, network *Network, wallet *Wallet) error {
	var err error

	if t.SignedTx == nil {
		return fmt.Errorf("Failed to broadcast %s tx using wallet: %s; tx not yet signed", *network.Name, wallet.ID)
	}

	if network.isEthereumNetwork() {
		if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
			err = provide.BroadcastSignedTx(network.ID.String(), network.rpcURL(), signedTx)
		} else {
			err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
		}
	} else {
		err = fmt.Errorf("Unable to generate signed tx for unsupported network: %s", *network.Name)
	}

	if err != nil {
		Log.Warningf("Failed to broadcast %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil(err.Error()),
		})
		t.updateStatus(db, "failed")
	}

	return err
}

func (t *Transaction) sign(db *gorm.DB, network *Network, wallet *Wallet) error {
	var err error

	if network.isEthereumNetwork() {
		privateKey := wallet.PrivateKey // FIXME: decrypt
		t.SignedTx, err = provide.SignTx(network.ID.String(), network.rpcURL(), wallet.Address, *privateKey, t.To, t.Data, t.Value.BigInt())
	} else {
		err = fmt.Errorf("Unable to generate signed tx for unsupported network: %s", *network.Name)
	}

	if err != nil {
		Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil(err.Error()),
		})
		t.updateStatus(db, "failed")
	}

	return err
}

func (t *Transaction) fetchReceipt(db *gorm.DB, network *Network, wallet *Wallet) {
	if network.isEthereumNetwork() {
		receipt, err := provide.GetTxReceipt(network.ID.String(), network.rpcURL(), *t.Hash, wallet.Address)
		if err != nil {
			Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; %s", t.Hash, err.Error())
			t.Errors = append(t.Errors, &gocore.Error{
				Message: stringOrNil(err.Error()),
			})
		} else {
			Log.Debugf("Fetched ethereum tx receipt with tx hash: %s; receipt: %s", t.Hash, receipt)

			traces, traceErr := provide.TraceTx(network.ID.String(), network.rpcURL(), t.Hash)
			if traceErr != nil {
				Log.Warningf("Failed to fetch ethereum tx trace for tx hash: %s; %s", *t.Hash, traceErr.Error())
			}
			t.Response = &ContractExecutionResponse{
				Receipt:     receipt,
				Traces:      traces,
				Transaction: t,
			}
			t.Traces = traces
		}
	}
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

	err := t.sign(db, network, wallet)
	if err != nil {
		t.Errors = append(t.Errors, &gocore.Error{
			Message: stringOrNil(err.Error()),
		})
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
			return false
		}

		t.broadcast(db, network, wallet)
		if len(t.Errors) > 0 {
			return false
		}

		if !db.NewRecord(t) {
			if rowsAffected > 0 {
				t.fetchReceipt(db, network, wallet)
				t.updateStatus(db, "success")
			}
			return rowsAffected > 0 && len(t.Errors) == 0
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

// RefreshDetails populates transaction details which were not necessarily available upon broadcast, including network-specific metadata and VM execution tracing if applicable
func (t *Transaction) RefreshDetails() error {
	var err error
	network, _ := t.GetNetwork()
	if network.isEthereumNetwork() {
		t.Traces, err = provide.TraceTx(network.ID.String(), network.rpcURL(), t.Hash)
	}
	if err != nil {
		return err
	}
	return nil
}

func (w *Wallet) generate(db *gorm.DB, gpgPublicKey string) {
	network, _ := w.GetNetwork()

	if network == nil || network.ID == uuid.Nil {
		Log.Warningf("Unable to generate private key for wallet without an associated network")
		return
	}

	var encodedPrivateKey *string

	if network.isEthereumNetwork() {
		addr, privateKey, err := provide.GenerateKeyPair()
		if err == nil {
			w.Address = *addr
			encodedPrivateKey = stringOrNil(hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
		}
	} else {
		Log.Warningf("Unable to generate private key for wallet using unsupported network: %s", *network.Name)
	}

	if encodedPrivateKey != nil {
		// Encrypt the private key
		db.Raw("SELECT pgp_pub_encrypt(?, dearmor(?)) as private_key", encodedPrivateKey, gpgPublicKey).Scan(&w)
		Log.Debugf("Generated wallet signing address: %s", w.Address)
	}
}

// GetNetwork - retrieve the associated transaction network
func (w *Wallet) GetNetwork() (*Network, error) {
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(w).Related(&network)
	if network == nil {
		return nil, fmt.Errorf("Failed to retrieve associated network for wallet: %s", w.ID)
	}
	return network, nil
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
	var err error
	if network.isEthereumNetwork() {
		_, err = decryptECDSAPrivateKey(*w.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
	}
	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &gocore.Error{
			Message: &msg,
		})
	}
	return len(w.Errors) == 0
}

// NativeCurrencyBalance retrieves a wallet's native currency/token balance
func (w *Wallet) NativeCurrencyBalance() (*big.Int, error) {
	var balance *big.Int
	var err error
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(w).Related(&network)
	if network.isEthereumNetwork() {
		balance, err = provide.GetNativeBalance(network.ID.String(), network.rpcURL(), w.Address)
		if err != nil {
			return nil, err
		}
	} else {
		Log.Warningf("Unable to read native currency balance for unsupported network: %s", *network.Name)
	}
	return balance, nil
}

// TokenBalance retrieves a wallet's token balance for a given token id
func (w *Wallet) TokenBalance(tokenID string) (*big.Int, error) {
	var balance *big.Int
	db := DatabaseConnection()
	var network = &Network{}
	var token = &Token{}
	db.Model(w).Related(&network)
	db.Where("id = ?", tokenID).Find(&token)
	if token == nil {
		return nil, fmt.Errorf("Unable to read token balance for invalid token: %s", tokenID)
	}
	if network.isEthereumNetwork() {
		contractAbi, err := token.readEthereumContractAbi()
		balance, err = provide.GetTokenBalance(network.ID.String(), network.rpcURL(), *token.Address, w.Address, contractAbi)
		if err != nil {
			return nil, err
		}
	} else {
		Log.Warningf("Unable to read token balance for unsupported network: %s", *network.Name)
	}
	return balance, nil
}

// TxCount retrieves a count of transactions signed by the wallet
func (w *Wallet) TxCount() (count *uint64) {
	db := DatabaseConnection()
	db.Model(&Transaction{}).Where("wallet_id = ?", w.ID).Count(&count)
	return count
}

// decryptECDSAPrivateKey - read the wallet-specific ECDSA private key; required for signing transactions on behalf of the wallet
func decryptECDSAPrivateKey(encryptedKey, gpgPrivateKey, gpgEncryptionKey string) (*ecdsa.PrivateKey, error) {
	results := make([]byte, 1)
	db := DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as private_key", encryptedKey, gpgPrivateKey, gpgEncryptionKey).Rows()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&results)
		privateKeyBytes, err := hex.DecodeString(string(results))
		if err != nil {
			Log.Warningf("Failed to read ecdsa private key from encrypted storage; %s", err.Error())
			return nil, err
		}
		return ethcrypto.ToECDSA(privateKeyBytes)
	}
	return nil, errors.New("Failed to decode ecdsa private key after retrieval from encrypted storage")
}
