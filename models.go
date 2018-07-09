package main

import (
	"context"
	"crypto/ecdsa"
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/provideapp/go-core"
	provide "github.com/provideservices/provide-go"

	"github.com/jinzhu/gorm"
	"github.com/kthomas/go.uuid"
)

const reachabilityTimeout = time.Millisecond * 2500
const receiptTickerInterval = time.Millisecond * 2500
const receiptTickerTimeout = time.Minute * 1
const resolveHostTickerInterval = time.Millisecond * 5000
const resolveHostTickerTimeout = time.Minute * 5
const resolvePeerUrlTickerInterval = time.Millisecond * 5000
const resolvePeerTickerTimeout = time.Minute * 5
const securityGroupTerminationTickerInterval = time.Millisecond * 10000
const securityGroupTerminationTickerTimeout = time.Minute * 10

const defaultJsonRpcPort = 8050
const defaultWebsocketPort = 8051

// Network
type Network struct {
	gocore.Model
	ApplicationID *uuid.UUID             `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID             `sql:"type:uuid" json:"user_id"`
	Name          *string                `sql:"not null" json:"name"`
	Description   *string                `json:"description"`
	IsProduction  *bool                  `sql:"not null" json:"is_production"`
	Cloneable     *bool                  `sql:"not null" json:"cloneable"`
	Enabled       *bool                  `sql:"not null" json:"enabled"`
	ChainID       *string                `json:"chain_id"`                     // protocol-specific chain id
	SidechainID   *uuid.UUID             `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
	NetworkID     *uuid.UUID             `sql:"type:uuid" json:"network_id"`   // network id used as the parent
	Config        *json.RawMessage       `sql:"type:json" json:"config"`
	Stats         *provide.NetworkStatus `sql:"-" json:"stats"`
}

// NetworkNode
type NetworkNode struct {
	gocore.Model
	NetworkID   uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	UserID      *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	Bootnode    bool             `sql:"not null;default:'false'" json:"is_bootnode"`
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
	Address       *string          `sql:"not null" json:"address"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	ABI       interface{}   `json:"abi"`
	NetworkID *uuid.UUID    `json:"network_id"`
	WalletID  *uuid.UUID    `json:"wallet_id"`
	Method    string        `json:"method"`
	Params    []interface{} `json:"params"`
	Value     *big.Int      `json:"value"`
}

// ContractExecutionResponse is returned upon successful contract execution
type ContractExecutionResponse struct {
	Response    interface{}  `json:"response"`
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
	Value         *TxValue                   `sql:"not null" json:"value"`
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
	return v.value.String(), nil
}

func (v *TxValue) Scan(val interface{}) error {
	v.value = new(big.Int)
	if str, ok := val.(string); ok {
		v.value.SetString(str, 10)
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
		n.setChainID()
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

// Reload the underlying network instance
func (n *Network) Reload() {
	db := DatabaseConnection()
	db.Model(&n).Find(n)
}

// Update an existing network
func (n *Network) Update() bool {
	db := DatabaseConnection()

	if !n.Validate() {
		return false
	}

	result := db.Save(&n)
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

// setConfig sets the network config in-memory
func (n *Network) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
}

// setChainID is an internal method used to set a unique chainID for the network prior to its creation
func (n *Network) setChainID() {
	n.ChainID = stringOrNil(fmt.Sprintf("0x%x", time.Now().Unix()))
	cfg := n.ParseConfig()
	if cfg != nil {
		cfg["network_id"] = n.ChainID
		if chainspec, chainspecOk := cfg["chainspec"].(map[string]interface{}); chainspecOk {
			if params, paramsOk := chainspec["params"].(map[string]interface{}); paramsOk {
				params["networkID"] = n.ChainID
			}
			n.setConfig(cfg)
		}
	}
}

func (n *Network) resolveJsonRpcAndWebsocketUrls(db *gorm.DB) {
	// update the JSON-RPC URL and enrich the network cfg
	cfg := n.ParseConfig()

	isLoadBalanced := false
	if loadBalanced, loadBalancedOk := cfg["is_load_balanced"].(bool); loadBalancedOk {
		isLoadBalanced = loadBalanced
	}

	if n.isEthereumNetwork() {
		var node = &NetworkNode{}
		db.Where("network_id = ? AND status = 'running'", n.ID).First(&node)

		if node != nil && node.ID != uuid.Nil {
			if isLoadBalanced {
				Log.Warningf("JSON-RPC/websocket load balancer may contain unhealthy or undeployed nodes")
			} else {
				if reachable, port := node.reachableViaJsonRpc(); reachable {
					cfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *node.Host, port)
					cfg["parity_json_rpc_url"] = fmt.Sprintf("http://%s:%v", *node.Host, port) // deprecated
				} else {
					cfg["json_rpc_url"] = nil
					cfg["parity_json_rpc_url"] = nil // deprecated
				}

				if reachable, port := node.reachableViaWebsocket(); reachable {
					cfg["websocket_url"] = fmt.Sprintf("wss://%s:%v", *node.Host, port)
				} else {
					cfg["websocket_url"] = nil
				}

				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)

				db.Save(n)
			}
		}
	}
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

func (n *Network) websocketURL() string {
	cfg := n.ParseConfig()
	if websocketURL, ok := cfg["websocket_url"].(string); ok {
		return websocketURL
	}
	return ""
}

// Status retrieves metadata and metrics specific to the given network;
// when force is true, it forces a JSON-RPC request to be made to retrieve
// the latest status; when false, cached network stats are returned if available
func (n *Network) Status(force bool) (status *provide.NetworkStatus, err error) {
	if cachedStatus, ok := currentNetworkStats[n.ID.String()]; ok && !force {
		return cachedStatus.stats, nil
	}
	RequireNetworkStatsDaemon(n)
	if n.isEthereumNetwork() {
		status, err = provide.GetNetworkStatus(n.ID.String(), n.rpcURL())
	} else {
		Log.Warningf("Unable to determine status of unsupported network: %s", *n.Name)
	}
	return status, err
}

// NodeCount retrieves a count of platform-managed network nodes
func (n *Network) NodeCount() (count *uint64) {
	DatabaseConnection().Model(&NetworkNode{}).Where("network_nodes.network_id = ?", n.ID).Count(&count)
	return count
}

// Bootnodes retrieves a list of network bootnodes
func (n *Network) Bootnodes() (nodes []*NetworkNode, err error) {
	query := DatabaseConnection().Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// BootnodesCount returns a count of the number of bootnodes on the network
func (n *Network) BootnodesCount() (count *uint64, err error) {
	db := DatabaseConnection()
	db.Model(&Transaction{}).Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID).Count(&count)
	return count, err
}

// Nodes retrieves a list of network nodes
func (n *Network) Nodes() (nodes []*NetworkNode, err error) {
	query := DatabaseConnection().Where("network_nodes.network_id = ?", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

func (n *Network) isEthereumNetwork() bool {
	if n.Name != nil && strings.HasPrefix(strings.ToLower(*n.Name), "eth") {
		return true
	}

	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg["is_ethereum_network"]; ok {
			if _ok := isEthereumNetwork.(bool); _ok {
				return isEthereumNetwork.(bool)
			}
		}
		if _, ok := cfg["json_rpc_url"]; ok {
			return true
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
				n.deploy(db)
			}
			return success
		}
	}
	return false
}

// setConfig sets the network config in-memory
func (n *NetworkNode) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
}

func (n *NetworkNode) reachableViaJsonRpc() (bool, uint) {
	cfg := n.ParseConfig()
	port := uint(defaultJsonRpcPort)
	if jsonRpcPortOverride, jsonRpcPortOverrideOk := cfg["default_json_rpc_port"].(float64); jsonRpcPortOverrideOk {
		port = uint(jsonRpcPortOverride)
	}

	return n.reachableOnPort(port), port
}

func (n *NetworkNode) reachableViaWebsocket() (bool, uint) {
	cfg := n.ParseConfig()
	port := uint(defaultWebsocketPort)
	if websocketPortOverride, websocketPortOverrideOk := cfg["default_websocket_port"].(float64); websocketPortOverrideOk {
		port = uint(websocketPortOverride)
	}

	return n.reachableOnPort(port), port
}

func (n *NetworkNode) reachableOnPort(port uint) bool {
	addr := fmt.Sprintf("%s:%v", *n.Host, port)
	conn, err := net.DialTimeout("tcp", addr, reachabilityTimeout)
	if err == nil {
		Log.Debugf("%s:%v is reachable", *n.Host, port)
		defer conn.Close()
		return true
	}
	Log.Debugf("%s:%v is unreachable", *n.Host, port)
	return false
}

func (n *NetworkNode) updateStatus(db *gorm.DB, status string) {
	n.Status = stringOrNil(status)
	result := db.Save(&n)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			n.Errors = append(n.Errors, &gocore.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
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

// Logs exposes the paginated logstream for the underlying node
func (n *NetworkNode) Logs() (*[]string, error) {
	var network = &Network{}
	DatabaseConnection().Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		return nil, fmt.Errorf("Failed to retrieve network for network node: %s", n.ID)
	}

	cfg := n.ParseConfig()

	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	region, regionOk := cfg["region"].(string)

	if !targetOk || !providerOk || !credsOk {
		return nil, fmt.Errorf("Cannot retrieve logs for network node without a target and provider configuration; target id: %s; provider id: %s", targetID, providerID)
	}

	if network.isEthereumNetwork() && regionOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if providerID, providerIdOk := cfg["provider_id"].(string); providerIdOk {
				if strings.ToLower(providerID) == "docker" {
					if ids, idsOk := cfg["target_task_ids"].([]string); idsOk {
						logs := make([]string, 0)
						for i := range ids {
							logEvents, err := GetContainerLogEvents(accessKeyID, secretAccessKey, region, ids[i], nil)
							if err == nil {
								for i := range logEvents.Events {
									event := logEvents.Events[i]
									logs = append(logs, string(*event.Message))
								}
							}
						}
						return &logs, nil
					}
				}

				return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; unsupported AWS provider: %s", *network.Name, providerID)
			}
		}
	} else if !regionOk {
		return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; no region provided: %s", *network.Name, providerID)
	}

	return nil, fmt.Errorf("Unable to retrieve logs for network node on unsupported network: %s", *network.Name)
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

func (n *NetworkNode) deploy(db *gorm.DB) {
	go func() {
		Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, n)

		var network = &Network{}
		db.Model(n).Related(&network)
		if network == nil || network.ID == uuid.Nil {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to retrieve network for network node: %s", n.ID)
			return
		}

		cfg := n.ParseConfig()
		networkCfg := network.ParseConfig()

		cfg["default_json_rpc_port"] = networkCfg["default_json_rpc_port"]
		cfg["default_websocket_port"] = networkCfg["default_websocket_port"]

		targetID, targetOk := cfg["target_id"].(string)
		providerID, providerOk := cfg["provider_id"].(string)
		role, roleOk := cfg["role"].(string)
		credentials, credsOk := cfg["credentials"].(map[string]interface{})
		rcd, rcdOk := cfg["rc.d"].(string)
		region, regionOk := cfg["region"].(string)
		env, envOk := cfg["env"].(map[string]string)

		cloneableCfg, cloneableCfgOk := networkCfg["cloneable_cfg"].(map[string]interface{})
		if !cloneableCfgOk {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to parse cloneable configuration for network node: %s", n.ID)
			return
		}

		securityCfg, securityCfgOk := cloneableCfg["_security"].(map[string]interface{})
		if !securityCfgOk {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to parse cloneable security configuration for network node: %s", n.ID)
			return
		}

		cloneableTarget, cloneableTargetOk := cloneableCfg[targetID].(map[string]interface{})
		if !cloneableTargetOk {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to parse cloneable target configuration for network node: %s", n.ID)
			return
		}

		cloneableProvider, cloneableProviderOk := cloneableTarget[providerID].(map[string]interface{})
		if !cloneableProviderOk {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to parse cloneable provider configuration for network node: %s", n.ID)
			return
		}

		providerCfgByRegion, providerCfgByRegionOk := cloneableProvider["regions"].(map[string]interface{})
		if !providerCfgByRegionOk {
			n.updateStatus(db, "failed")
			Log.Warningf("Failed to parse cloneable provider configuration by region for network node: %s", n.ID)
			return
		}

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

		Log.Debugf("Configuration for network node deploy: target id: %s; provider: %s; role: %s; crendentials: %s; region: %s, rc.d: %s; cloneable provider cfg: %s; network config: %s",
			targetID, providerID, role, credentials, region, rcd, providerCfgByRegion, networkCfg)

		if targetOk && providerOk && roleOk && credsOk && regionOk {
			if strings.ToLower(targetID) == "aws" {
				accessKeyID := credentials["aws_access_key_id"].(string)
				secretAccessKey := credentials["aws_secret_access_key"].(string)

				// start security group handling
				securityGroupDesc := fmt.Sprintf("security group for network node: %s", n.ID.String())
				securityGroup, err := CreateSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupDesc, securityGroupDesc, nil)
				securityGroupIds := make([]string, 0)
				securityGroupIds = append(securityGroupIds, *securityGroup.GroupId)

				cfg["region"] = region
				cfg["target_security_group_ids"] = securityGroupIds
				n.setConfig(cfg)

				if err != nil {
					Log.Warningf("Failed to create security group in EC2 %s region %s; network node id: %s", region, n.ID.String())
				} else {
					if egress, egressOk := securityCfg["egress"]; egressOk {
						switch egress.(type) {
						case string:
							if egress.(string) == "*" {
								_, err := AuthorizeSecurityGroupEgressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, *securityGroup.GroupId)
								if err != nil {
									Log.Warningf("Failed to authorize security group egress across all ports and protocols in EC2 %s region; security group id: %s; %s", region, *securityGroup.GroupId, err.Error())
								}
							}
						case map[string]interface{}:
							egressCfg := egress.(map[string]interface{})
							for cidr := range egressCfg {
								tcp := make([]int64, 0)
								udp := make([]int64, 0)
								if _tcp, tcpOk := egressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
									for i := range _tcp {
										tcp = append(tcp, int64(_tcp[i].(float64)))
									}
								}
								if _udp, udpOk := egressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
									for i := range _udp {
										udp = append(udp, int64(_udp[i].(float64)))
									}
								}
								_, err := AuthorizeSecurityGroupEgress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
								if err != nil {
									Log.Warningf("Failed to authorize security group egress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
								}
							}
						}
					}

					if ingress, ingressOk := securityCfg["ingress"]; ingressOk {
						switch ingress.(type) {
						case string:
							if ingress.(string) == "*" {
								_, err := AuthorizeSecurityGroupIngressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, *securityGroup.GroupId)
								if err != nil {
									Log.Warningf("Failed to authorize security group ingress across all ports and protocols in EC2 %s region; security group id: %s; %s", region, *securityGroup.GroupId, err.Error())
								}
							}
						case map[string]interface{}:
							ingressCfg := ingress.(map[string]interface{})
							for cidr := range ingressCfg {
								tcp := make([]int64, 0)
								udp := make([]int64, 0)
								if _tcp, tcpOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
									for i := range _tcp {
										tcp = append(tcp, int64(_tcp[i].(float64)))
									}
								}
								if _udp, udpOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
									for i := range _udp {
										udp = append(udp, int64(_udp[i].(float64)))
									}
								}
								_, err := AuthorizeSecurityGroupIngress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
								if err != nil {
									Log.Warningf("Failed to authorize security group ingress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
								}
							}
						}
					}
				}

				if strings.ToLower(providerID) == "ubuntu-vm" {
					var userData = ""
					if rcdOk {
						userData = base64.StdEncoding.EncodeToString([]byte(rcd))
					}

					Log.Debugf("Attempting to deploy network node instance(s) in EC2 region: %s", region)
					if imagesByRegion, imagesByRegionOk := providerCfgByRegion[region].(map[string]interface{}); imagesByRegionOk {
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
									n.updateStatus(db, "failed")
									n.unregisterSecurityGroups()
									Log.Warningf("Attempt to deploy image %s@%s in EC2 %s region failed; %s", imageID, version, region, err.Error())
									return
								}
								Log.Debugf("Attempt to deploy image %s@%s in EC2 %s region successful; instance ids: %s", imageID, version, region, instanceIds)
								cfg["target_instance_ids"] = instanceIds

								Log.Debugf("Assigning %v security groups for deployed image %s@%s in EC2 %s region; instance ids: %s", len(securityGroupIds), imageID, version, region, instanceIds)
								for i := range instanceIds {
									SetInstanceSecurityGroups(accessKeyID, secretAccessKey, region, instanceIds[i], securityGroupIds)
								}

								n.resolveHost(db, network, cfg, instanceIds)
								n.resolvePeerURL(db, network, cfg, instanceIds)
							}
						}
					}
				} else if strings.ToLower(providerID) == "docker" {
					Log.Debugf("Attempting to deploy network node container(s) in EC2 region: %s", region)
					if containerRolesByRegion, containerRolesByRegionOk := providerCfgByRegion[region].(map[string]interface{}); containerRolesByRegionOk {
						Log.Debugf("Resolved deployable containers by region in EC2 region: %s", region)
						if container, containerOk := containerRolesByRegion[role].(string); containerOk {
							Log.Debugf("Resolved deployable container for role: %s; in EC2 region: %s; container: %s", role, region, container)
							Log.Debugf("Attempting to deploy container %s in EC2 region: %s", container, region)
							envOverrides := map[string]interface{}{}
							if envOk {
								for k := range env {
									envOverrides[k] = env[k]
								}
							}
							if chain, chainOk := networkCfg["chain"].(string); chainOk {
								envOverrides["CHAIN"] = chain
							}
							overrides := map[string]interface{}{
								"environemnt": envOverrides,
							}
							taskIds, err := StartContainer(accessKeyID, secretAccessKey, region, container, nil, nil, securityGroupIds, []string{}, overrides)

							if err != nil {
								n.updateStatus(db, "failed")
								n.unregisterSecurityGroups()
								Log.Warningf("Attempt to deploy container %s in EC2 %s region failed; %s", container, region, err.Error())
								return
							}
							Log.Debugf("Attempt to deploy container %s in EC2 %s region successful; task ids: %s", container, region, taskIds)
							cfg["target_task_ids"] = taskIds

							n.resolveHost(db, network, cfg, taskIds)
							n.resolvePeerURL(db, network, cfg, taskIds)
						}
					}
				}
			}
		}
	}()
}

func (n *NetworkNode) resolveHost(db *gorm.DB, network *Network, cfg map[string]interface{}, identifiers []string) {
	ticker := time.NewTicker(resolveHostTickerInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			if n.Host == nil {
				if time.Now().Sub(startedAt) >= resolveHostTickerTimeout {
					Log.Warningf("Failed to resolve hostname for network node: %s; timing out after %v", n.ID.String(), resolveHostTickerTimeout)
					n.updateStatus(db, "failed")
					ticker.Stop()
					return
				}

				id := identifiers[len(identifiers)-1]
				targetID, targetOk := cfg["target_id"].(string)
				providerID, providerOk := cfg["provider_id"].(string)
				region, regionOk := cfg["region"].(string)
				credentials, credsOk := cfg["credentials"].(map[string]interface{})

				if strings.ToLower(targetID) == "aws" && targetOk && providerOk && regionOk && credsOk {
					accessKeyID := credentials["aws_access_key_id"].(string)
					secretAccessKey := credentials["aws_secret_access_key"].(string)

					if strings.ToLower(providerID) == "ubuntu-vm" {
						instanceDetails, err := GetInstanceDetails(accessKeyID, secretAccessKey, region, id)
						if err == nil {
							if len(instanceDetails.Reservations) > 0 {
								reservation := instanceDetails.Reservations[0]
								if len(reservation.Instances) > 0 {
									instance := reservation.Instances[0]
									n.Host = instance.PublicDnsName
								}
							}
						}
					} else if strings.ToLower(providerID) == "docker" {
						containerDetails, err := GetContainerDetails(accessKeyID, secretAccessKey, region, id, nil)
						if err == nil {
							if len(containerDetails.Tasks) > 0 {
								task := containerDetails.Tasks[0]
								if len(task.Attachments) > 0 {
									attachment := task.Attachments[0]
									if attachment.Type != nil && *attachment.Type == "ElasticNetworkInterface" {
										for i := range attachment.Details {
											kvp := attachment.Details[i]
											if kvp.Name != nil && *kvp.Name == "networkInterfaceId" && kvp.Value != nil {
												interfaceDetails, err := GetNetworkInterfaceDetails(accessKeyID, secretAccessKey, region, *kvp.Value)
												if err == nil {
													if len(interfaceDetails.NetworkInterfaces) > 0 {
														Log.Debugf("Retrieved interface details for container instance: %s", interfaceDetails)
														interfaceAssociation := interfaceDetails.NetworkInterfaces[0].Association
														if interfaceAssociation != nil {
															n.Host = interfaceAssociation.PublicDnsName
														}
													}
												}
												break
											}
										}
									}
								}
							}
						}
					}
				}
			}

			if n.Host != nil {
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				n.Status = stringOrNil("running")
				db.Save(n)
				network.resolveJsonRpcAndWebsocketUrls(db)
				ticker.Stop()
				return
			}
		}
	}
}

func (n *NetworkNode) resolvePeerURL(db *gorm.DB, network *Network, cfg map[string]interface{}, identifiers []string) {
	ticker := time.NewTicker(resolvePeerUrlTickerInterval)
	startedAt := time.Now()
	var peerURL *string
	for {
		select {
		case <-ticker.C:
			if peerURL == nil {
				if time.Now().Sub(startedAt) >= resolvePeerTickerTimeout {
					Log.Warningf("Failed to resolve peer url for network node: %s; timing out after %v", n.ID.String(), resolvePeerTickerTimeout)
					ticker.Stop()
					return
				}

				id := identifiers[len(identifiers)-1]
				targetID, targetOk := cfg["target_id"].(string)
				providerID, providerOk := cfg["provider_id"].(string)
				region, regionOk := cfg["region"].(string)
				credentials, credsOk := cfg["credentials"].(map[string]interface{})

				if strings.ToLower(targetID) == "aws" && targetOk && providerOk && regionOk && credsOk {
					accessKeyID := credentials["aws_access_key_id"].(string)
					secretAccessKey := credentials["aws_secret_access_key"].(string)

					if strings.ToLower(providerID) == "ubuntu-vm" {
						Log.Warningf("Peer URL resolution is not yet implemented for non-containerized AWS deployments")
						ticker.Stop()
						return
					} else if strings.ToLower(providerID) == "docker" {
						if network.isEthereumNetwork() {
							logs, err := GetContainerLogEvents(accessKeyID, secretAccessKey, region, id, nil)
							if err == nil {
								for i := range logs.Events {
									event := logs.Events[i]
									if event.Message != nil {
										msg := string(*event.Message)
										nodeInfo := &provide.EthereumJsonRpcResponse{}
										err := json.Unmarshal([]byte(msg), &nodeInfo)
										if err == nil && nodeInfo != nil {
											result, resultOk := nodeInfo.Result.(map[string]interface{})
											if resultOk {
												if enode, enodeOk := result["enode"].(string); enodeOk {
													peerURL = stringOrNil(enode)
													ticker.Stop()
													break
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}

			if peerURL != nil {
				Log.Debugf("Resolved peer url for network node with id: %s; peer url: %s", n.ID, peerURL)
				cfg["peer_url"] = peerURL
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				db.Save(n)
				ticker.Stop()
				return
			}
		}
	}
}

func (n *NetworkNode) undeploy() error {
	Log.Debugf("Attempting to undeploy network node with id: %s", n.ID, n)

	db := DatabaseConnection()

	cfg := n.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	region, regionOk := cfg["region"].(string)
	instanceIds, instanceIdsOk := cfg["target_instance_ids"].([]interface{})
	taskIds, taskIdsOk := cfg["target_task_ids"].([]interface{})
	credentials, credsOk := cfg["credentials"].(map[string]interface{})

	Log.Debugf("Configuration for network node undeploy: target id: %s; crendentials: %s; target instance ids: %s",
		targetID, credentials, instanceIds)

	if targetOk && providerOk && regionOk && credsOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if strings.ToLower(providerID) == "ubuntu-vm" && instanceIdsOk {
				for i := range instanceIds {
					instanceID := instanceIds[i].(string)

					_, err := TerminateInstance(accessKeyID, secretAccessKey, region, instanceID)
					if err == nil {
						Log.Debugf("Terminated EC2 instance with id: %s", instanceID)
						n.Status = stringOrNil("terminated")
						db.Save(n)
					} else {
						Log.Warningf("Failed to terminate EC2 instance with id: %s; %s", instanceID, err.Error())
					}
				}
			} else if strings.ToLower(providerID) == "docker" && taskIdsOk {
				for i := range taskIds {
					taskID := taskIds[i].(string)

					_, err := StopContainer(accessKeyID, secretAccessKey, region, taskID, nil)
					if err == nil {
						Log.Debugf("Terminated ECS docker container with id: %s", taskID)
						n.Status = stringOrNil("terminated")
						db.Save(n)
					} else {
						Log.Warningf("Failed to terminate ECS docker container with id: %s; %s", taskID, err.Error())
					}
				}
			}

			ticker := time.NewTicker(securityGroupTerminationTickerInterval)
			go func() {
				startedAt := time.Now()
				for {
					select {
					case <-ticker.C:
						err := n.unregisterSecurityGroups()
						if err == nil {
							ticker.Stop()
							return
						}

						if time.Now().Sub(startedAt) >= securityGroupTerminationTickerTimeout {
							Log.Warningf("Failed to unregister security groups for network node with id: %s; timing out after %v...", n.ID, securityGroupTerminationTickerTimeout)
							ticker.Stop()
							return
						}
					}
				}
			}()
		}

		var network = &Network{}
		db.Where("id = ?", n.NetworkID).Find(&network)
		go network.resolveJsonRpcAndWebsocketUrls(db)
	}

	return nil
}

func (n *NetworkNode) unregisterSecurityGroups() error {
	Log.Debugf("Attempting to unregister security groups for network node with id: %s", n.ID, n)

	cfg := n.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	securityGroupIds, securityGroupIdsOk := cfg["target_security_group_ids"].([]interface{})
	credentials, credsOk := cfg["credentials"].(map[string]interface{})

	if targetOk && regionOk && credsOk && securityGroupIdsOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			for i := range securityGroupIds {
				securityGroupID := securityGroupIds[i].(string)

				if strings.ToLower(targetID) == "aws" {
					_, err := DeleteSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupID)
					if err != nil {
						Log.Warningf("Failed to unregister security group for network node with id: %s; security group id: %s", n.ID, securityGroupID)
						return err
					}
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

// Execute an ethereum contract; returns the tx receipt and retvals; if the method is constant, the receipt will be nil.
// If the methid is non-constant, the retvals will be nil.
func (c *Contract) executeEthereumContract(network *Network, tx *Transaction, method string, params []interface{}) (*interface{}, *interface{}, error) { // given tx has been built but broadcast has not yet been attempted
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
		Log.Debugf("Attempting to encode %d parameters [ %s ] prior to executing contract %s on contract: %s", len(params), params, methodDescriptor, c.ID)
		invocationSig, err := provide.EncodeABI(abiMethod, params...)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to encode %d parameters prior to attempting execution of contract %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
		}

		data := common.Bytes2Hex(invocationSig)
		tx.Data = &data

		if abiMethod.Const {
			Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
			network, _ := tx.GetNetwork()
			client, err := provide.DialJsonRpc(network.ID.String(), network.rpcURL())
			gasPrice, _ := client.SuggestGasPrice(context.TODO())
			msg := tx.asEthereumCallMsg(gasPrice.Uint64(), 0)
			result, _ := client.CallContract(context.TODO(), msg, nil)
			var out interface{}
			err = abiMethod.Outputs.Unpack(&out, result)
			if len(abiMethod.Outputs) == 1 {
				err = abiMethod.Outputs.Unpack(&out, result)
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
			}
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to read constant %s on contract: %s (signature with encoded parameters: %s); %s", methodDescriptor, c.ID, *tx.Data, err.Error())
			}
			return nil, &out, nil
		}
		if tx.Create() {
			Log.Debugf("Executed contract %s on contract: %s", methodDescriptor, c.ID)

			if tx.Response != nil {
				Log.Debugf("Received response to tx broadcast attempt calling contract %s on contract: %s", methodDescriptor, c.ID)

				var out interface{}
				switch (tx.Response.Receipt).(type) {
				case []byte:
					out = (tx.Response.Receipt).([]byte)
					Log.Debugf("Received response: %s", out)
				case types.Receipt:
					client, _ := provide.DialJsonRpc(network.ID.String(), network.rpcURL())
					receipt := tx.Response.Receipt.(*types.Receipt)
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
			err = fmt.Errorf("Failed to execute contract %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", methodDescriptor, c.ID, *tx.Data)
			Log.Warning(err.Error())
		}
	} else {
		err = fmt.Errorf("Failed to execute contract %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
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
			Log.Warningf("Failed to initialize ABI from contract  params to json; %s", err.Error())
			return nil, err
		}

		_abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read ABI from params for contract: %s", c.ID)
	}
	return _abi, nil
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
		Value:    t.Value.BigInt(),
		Data:     data,
	}
}

// Execute a transaction on the contract instance using a specific signer, value, method and params
func (c *Contract) Execute(walletID *uuid.UUID, value *big.Int, method string, params []interface{}) (*ContractExecutionResponse, error) {
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
		Value:         &TxValue{value: value},
	}

	var receipt *interface{}
	var response *interface{}

	if network.isEthereumNetwork() {
		receipt, response, err = c.executeEthereumContract(network, tx, method, params)
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
			Response:    response,
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
		if wallet.PrivateKey != nil {
			privateKey, _ := decryptECDSAPrivateKey(*wallet.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
			_privateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
			t.SignedTx, t.Hash, err = provide.SignTx(network.ID.String(), network.rpcURL(), wallet.Address, _privateKey, t.To, t.Data, t.Value.BigInt())
		} else {
			err = fmt.Errorf("Unable to sign tx; no private key for wallet: %s", wallet.ID)
		}
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
		ticker := time.NewTicker(receiptTickerInterval)
		go func() {
			startedAt := time.Now()
			for {
				select {
				case <-ticker.C:
					receipt, err := provide.GetTxReceipt(network.ID.String(), network.rpcURL(), *t.Hash, wallet.Address)
					if err != nil {
						if err == ethereum.NotFound {
							Log.Debugf("Failed to fetch ethereum tx receipt with tx hash: %s; %s", *t.Hash, err.Error())
							if time.Now().Sub(startedAt) >= receiptTickerTimeout {
								Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; timing out after %v", *t.Hash, receiptTickerTimeout)
								t.updateStatus(db, "failed")
								ticker.Stop()
								return
							}
						} else {
							Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; %s", *t.Hash, err.Error())
							t.Errors = append(t.Errors, &gocore.Error{
								Message: stringOrNil(err.Error()),
							})
							t.updateStatus(db, "failed")
							ticker.Stop()
							return
						}
					} else {
						Log.Debugf("Fetched ethereum tx receipt with tx hash: %s; receipt: %s", *t.Hash, receipt)

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

						t.updateStatus(db, "success")
						t.handleEthereumTxReceipt(db, network, wallet, receipt)
						ticker.Stop()
						return
					}
				}
			}
		}()
	}
}

func (t *Transaction) handleEthereumTxReceipt(db *gorm.DB, network *Network, wallet *Wallet, receipt *types.Receipt) {
	client, err := provide.DialJsonRpc(network.ID.String(), network.rpcURL())
	if err != nil {
		Log.Warningf("Unable to handle ethereum tx receipt; %s", err.Error())
		return
	}
	if t.To == nil {
		Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, *t.Hash, receipt.ContractAddress.Hex())
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
			Log.Debugf("Created contract %s for %s contract creation tx: %s", contract.ID, *network.Name, *t.Hash)

			if contractAbi, ok := params["abi"]; ok {
				abistr, err := json.Marshal(contractAbi)
				if err != nil {
					Log.Warningf("failed to marshal abi to json...  %s", err.Error())
				}
				_abi, err := abi.JSON(strings.NewReader(string(abistr)))
				if err == nil {
					msg := ethereum.CallMsg{
						From:     common.HexToAddress(wallet.Address),
						To:       &receipt.ContractAddress,
						Gas:      0,
						GasPrice: big.NewInt(0),
						Value:    nil,
						Data:     common.FromHex(provide.HashFunctionSelector("name()")),
					}

					result, _ := client.CallContract(context.TODO(), msg, nil)
					var name string
					if method, ok := _abi.Methods["name"]; ok {
						err = method.Outputs.Unpack(&name, result)
						if err != nil {
							Log.Warningf("Failed to read %s, contract name from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
						}
					}

					msg = ethereum.CallMsg{
						From:     common.HexToAddress(wallet.Address),
						To:       &receipt.ContractAddress,
						Gas:      0,
						GasPrice: big.NewInt(0),
						Value:    nil,
						Data:     common.FromHex(provide.HashFunctionSelector("decimals()")),
					}
					result, _ = client.CallContract(context.TODO(), msg, nil)
					var decimals *big.Int
					if method, ok := _abi.Methods["decimals"]; ok {
						err = method.Outputs.Unpack(&decimals, result)
						if err != nil {
							Log.Warningf("Failed to read %s, contract decimals from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
						}
					}

					msg = ethereum.CallMsg{
						From:     common.HexToAddress(wallet.Address),
						To:       &receipt.ContractAddress,
						Gas:      0,
						GasPrice: big.NewInt(0),
						Value:    nil,
						Data:     common.FromHex(provide.HashFunctionSelector("symbol()")),
					}
					result, _ = client.CallContract(context.TODO(), msg, nil)
					var symbol string
					if method, ok := _abi.Methods["symbol"]; ok {
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
							Log.Debugf("Created token %s for associated %s contract creation tx: %s", token.ID, *network.Name, *t.Hash)
						} else {
							Log.Warningf("Failed to create token for associated %s contract creation tx %s; %d errs: %s", *network.Name, *t.Hash, len(token.Errors), *stringOrNil(*token.Errors[0].Message))
						}
					}
				} else {
					Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
				}
			}
		} else {
			Log.Warningf("Failed to create contract for %s contract creation tx %s", *network.Name, *t.Hash)
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
	if w.PrivateKey != nil {
		if network.isEthereumNetwork() {
			_, err = decryptECDSAPrivateKey(*w.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
		}
	} else {
		w.Errors = append(w.Errors, &gocore.Error{
			Message: stringOrNil("private key generation failed"),
		})
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
	defer rows.Close()
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
