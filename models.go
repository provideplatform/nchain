package main

import (
	"bytes"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jinzhu/gorm"
	"github.com/kthomas/go-aws-wrapper"
	natsutil "github.com/kthomas/go-natsutil"
	"github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

const hostReachabilityTimeout = time.Minute * 5
const hostReachabilityInterval = time.Millisecond * 2500
const reachabilityTimeout = time.Millisecond * 2500
const receiptTickerInterval = time.Millisecond * 2500
const receiptTickerTimeout = time.Minute * 1
const resolveGenesisTickerInterval = time.Millisecond * 10000
const resolveGenesisTickerTimeout = time.Minute * 20
const resolveHostTickerInterval = time.Millisecond * 5000
const resolveHostTickerTimeout = time.Minute * 5
const resolvePeerTickerInterval = time.Millisecond * 5000
const resolvePeerTickerTimeout = time.Minute * 20
const resolveTokenTickerInterval = time.Millisecond * 5000
const resolveTokenTickerTimeout = time.Minute * 1
const securityGroupTerminationTickerInterval = time.Millisecond * 10000
const securityGroupTerminationTickerTimeout = time.Minute * 10
const streamingTxFilterReturnTimeout = time.Millisecond * 50

const defaultClient = "parity"
const defaultWebappPort = 3000

var engineToDefaultJSONRPCPortMapping = map[string]uint{"authorityRound": 8050, "handshake": 13037}
var engineToDefaultWebsocketPortMapping = map[string]uint{"authorityRound": 8051}
var engineToNetworkNodeClientEnvMapping = map[string]string{"authorityRound": "parity", "handshake": "handshake"}
var networkGenesisMutex = map[string]*sync.Mutex{}
var txFilters = map[string][]*Filter{}

// Network represents a blockchain network; the network could fall at any level of
// a heirarchy of blockchain networks
type Network struct {
	provide.Model
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

// NetworkNode instances represent nodes of the network to which they belong, acting in a specific role;
// each NetworkNode may have a set or sets of deployed resources, such as application containers, VMs
// or even phyiscal infrastructure
type NetworkNode struct {
	provide.Model
	NetworkID   uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	UserID      *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	Bootnode    bool             `sql:"not null;default:'false'" json:"is_bootnode"`
	Host        *string          `json:"host"`
	IPv4        *string          `json:"ipv4"`
	IPv6        *string          `json:"ipv6"`
	Description *string          `json:"description"`
	Role        *string          `sql:"not null;default:'peer'" json:"role"`
	Status      *string          `sql:"not null;default:'pending'" json:"status"`
	Config      *json.RawMessage `sql:"type:json" json:"config"`
}

// Bridge instances are still in the process of being defined.
type Bridge struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
}

// Connector instances represent a logical connection to IPFS or other decentralized filesystem;
// in the future it may represent a logical connection to services of other types
type Connector struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name          *string          `sql:"not null" json:"name"`
	Type          *string          `sql:"not null" json:"type"`
	Config        *json.RawMessage `sql:"type:json" json:"config"`
	AccessedAt    *time.Time       `json:"accessed_at"`
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

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	ABI        interface{}   `json:"abi"`
	NetworkID  *uuid.UUID    `json:"network_id"`
	Contract   *Contract     `json:"-"`
	ContractID *uuid.UUID    `json:"contract_id"`
	WalletID   *uuid.UUID    `json:"wallet_id"`
	Wallet     *Wallet       `json:"wallet"`
	Gas        *float64      `json:"gas"`
	Method     string        `json:"method"`
	Params     []interface{} `json:"params"`
	Value      *big.Int      `json:"value"`
	Ref        *string       `json:"ref"`
}

// ContractExecutionResponse is returned upon successful contract execution
type ContractExecutionResponse struct {
	Response    interface{}  `json:"response"`
	Receipt     interface{}  `json:"receipt"`
	Traces      interface{}  `json:"traces"`
	Transaction *Transaction `json:"transaction"`
	Ref         *string      `json:"ref"`
}

// Filter instances must be associated with an application identifier.
type Filter struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name          *string          `sql:"not null" json:"name"`
	Priority      uint8            `sql:"not null;default:0" json:"priority"`
	Lang          *string          `sql:"not null" json:"lang"`
	Source        *string          `sql:"not null" json:"source"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

// Oracle instances are smart contracts whose terms are fulfilled by writing data from a configured feed onto the blockchain associated with its configured network
type Oracle struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ContractID    uuid.UUID        `sql:"not null;type:uuid" json:"contract_id"`
	Name          *string          `sql:"not null" json:"name"`
	FeedURL       *url.URL         `sql:"not null" json:"feed_url"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
	AttachmentIds []*uuid.UUID     `sql:"type:uuid[]" json:"attachment_ids"`
}

// Token instances must be associated with an application identifier.
type Token struct {
	provide.Model
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	ContractID     *uuid.UUID `sql:"type:uuid" json:"contract_id"`
	SaleContractID *uuid.UUID `sql:"type:uuid" json:"sale_contract_id"`
	Name           *string    `sql:"not null" json:"name"`
	Symbol         *string    `sql:"not null" json:"symbol"`
	Decimals       uint64     `sql:"not null" json:"decimals"`
	Address        *string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress    *string    `json:"sale_address"`           // non-null if token sale contract is specified
	AccessedAt     *time.Time `json:"accessed_at"`
}

// Transaction instances are associated with a signing wallet and exactly one matching instance of either an a) application identifier or b) user identifier.
type Transaction struct {
	provide.Model
	ApplicationID *uuid.UUID                 `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID                 `sql:"type:uuid" json:"user_id"`
	NetworkID     uuid.UUID                  `sql:"not null;type:uuid" json:"network_id"`
	WalletID      *uuid.UUID                 `sql:"type:uuid" json:"wallet_id"`
	To            *string                    `json:"to"`
	Value         *TxValue                   `sql:"not null;type:text" json:"value"`
	Data          *string                    `json:"data"`
	Hash          *string                    `sql:"not null" json:"hash"`
	Status        *string                    `sql:"not null;default:'pending'" json:"status"`
	Params        *json.RawMessage           `sql:"-" json:"params"`
	Response      *ContractExecutionResponse `sql:"-" json:"-"`
	SignedTx      interface{}                `sql:"-" json:"-"`
	Traces        interface{}                `sql:"-" json:"traces"`
	Ref           *string                    `json:"ref"`
	Description   *string                    `json:"description"`
}

// Wallet instances must be associated with exactly one instance of either an a) application identifier or b) user identifier.
type Wallet struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"user_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	Address       string     `sql:"not null" json:"address"`
	PrivateKey    *string    `sql:"not null;type:bytea" json:"-"`
	Balance       *big.Int   `sql:"-" json:"balance"`
	AccessedAt    *time.Time `json:"accessed_at"`
}

// Create and persist a new filter
func (f *Filter) Create() bool {
	if !f.Validate() {
		return false
	}

	db := DatabaseConnection()

	if db.NewRecord(f) {
		result := db.Create(&f)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				f.Errors = append(f.Errors, &provide.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(f) {
			success := rowsAffected > 0
			if success {
				go f.cache()
			}
			return success
		}
	}
	return false
}

// cache the filter in memory
func (f *Filter) cache() {
	appFilters := txFilters[f.ApplicationID.String()]
	if appFilters == nil {
		appFilters = make([]*Filter, 0)
		txFilters[f.ApplicationID.String()] = appFilters
	}
	appFilters = append(appFilters, f)
}

// ParseParams - parse the original JSON params used for filter creation
func (f *Filter) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if f.Params != nil {
		err := json.Unmarshal(*f.Params, &params)
		if err != nil {
			Log.Warningf("Failed to unmarshal filter params; %s", err.Error())
			return nil
		}
	}
	return params
}

// Invoke a filter for the given tx payload
func (f *Filter) Invoke(txPayload []byte) *float64 {
	subjectUUID, _ := uuid.NewV4()
	natsStreamingTxFilterReturnSubject := fmt.Sprintf("%s.return.%s", natsStreamingTxFilterExecSubjectPrefix, subjectUUID.String())

	natsMsg := map[string]interface{}{
		"sub":     natsStreamingTxFilterReturnSubject,
		"payload": txPayload,
	}
	natsPayload, _ := json.Marshal(natsMsg)

	natsConnection := getNatsStreamingConnection()
	natsConnection.Publish(natsStreamingTxFilterExecSubjectPrefix, natsPayload)

	natsConn := natsutil.GetNatsConnection()
	defer natsConn.Close()

	sub, err := natsConn.SubscribeSync(natsStreamingTxFilterReturnSubject)
	if err != nil {
		Log.Warningf("Failed to create a synchronous NATS subscription to subject: %s; %s", natsStreamingTxFilterReturnSubject, err.Error())
		return nil
	}

	var confidence *float64
	msg, err := sub.NextMsg(streamingTxFilterReturnTimeout)
	if err != nil {
		Log.Warningf("Failed to parse confidence from streaming tx filter; %s", err.Error())
		return nil
	}
	_confidence, err := strconv.ParseFloat(string(msg.Data), 64)
	if err != nil {
		Log.Warningf("Failed to parse confidence from streaming tx filter; %s", err.Error())
		return nil
	}
	confidence = &_confidence
	return confidence
}

// Validate a filter for persistence
func (f *Filter) Validate() bool {
	f.Errors = make([]*provide.Error, 0)
	return len(f.Errors) == 0
}

func (w *Wallet) setID(walletID uuid.UUID) {
	if w.ID != uuid.Nil {
		Log.Warningf("Attempted to change a wallet id in memory; wallet id not changed: %s", w.ID)
		return
	}
	w.ID = walletID
}

type TxValue struct {
	value *big.Int
}

type bootnodesInitialized string

func (err bootnodesInitialized) Error() string {
	return "network bootnodes initialized"
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
				n.Errors = append(n.Errors, &provide.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(n) {
			success := rowsAffected > 0
			if success {
				go n.provisionLoadBalancers(db)
				go n.resolveContracts(db)
			}
			return success
		}
	}
	return false
}

func (n *Network) provisionLoadBalancers(db *gorm.DB) {
	cfg := n.ParseConfig()

	defaultJSONRPCPort := uint(0)
	defaultWebsocketPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultJSONRPCPort = engineToDefaultJSONRPCPortMapping[engineID]
		defaultWebsocketPort = engineToDefaultWebsocketPortMapping[engineID]
	}

	if n.isEthereumNetwork() {
		Log.Debugf("Attempting to provision JSON-RPC load balancer for EVM-based network: %s", n.ID)

		lbUUID, _ := uuid.NewV4()
		lbName := fmt.Sprintf("loadbalancer-%s", lbUUID)

		jsonRPCPort := float64(defaultJSONRPCPort)
		websocketPort := float64(defaultWebsocketPort)

		securityCfg, securityCfgOk := cfg["_security"].(map[string]interface{})
		if !securityCfgOk {
			Log.Warningf("Failed to parse cloneable security configuration for network: %s; attempting to create sane initial configuration", n.ID)

			tcpIngressCfg := make([]float64, 0)
			if _jsonRPCPort, jsonRPCPortOk := cfg["default_json_rpc_port"].(float64); jsonRPCPortOk {
				jsonRPCPort = _jsonRPCPort
			}
			if _websocketPort, websocketPortOk := cfg["default_websocket_port"].(float64); websocketPortOk {
				websocketPort = _websocketPort
			}

			tcpIngressCfg = append(tcpIngressCfg, float64(jsonRPCPort))
			tcpIngressCfg = append(tcpIngressCfg, float64(websocketPort))
			ingressCfg := map[string]interface{}{
				"tcp": tcpIngressCfg,
				"udp": make([]float64, 0),
			}
			securityCfg = map[string]interface{}{
				"egress": map[string]interface{}{},
				"ingress": map[string]interface{}{
					"0.0.0.0/0": ingressCfg,
				},
			}
			cfg["_security"] = securityCfg
		}

		accessKeyID := *DefaultAWSConfig.AccessKeyId
		secretAccessKey := *DefaultAWSConfig.SecretAccessKey
		region := *DefaultAWSConfig.DefaultRegion
		vpcID := *DefaultAWSConfig.DefaultVpcID

		// start security group handling
		securityGroupDesc := fmt.Sprintf("security group for network load balancer: %s", n.ID.String())
		securityGroup, err := awswrapper.CreateSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupDesc, securityGroupDesc, DefaultAWSConfig.DefaultVpcID)
		securityGroupIds := make([]string, 0)

		if securityGroup != nil {
			securityGroupIds = append(securityGroupIds, *securityGroup.GroupId)
		}

		cfg["region"] = region
		cfg["target_security_group_ids"] = securityGroupIds
		n.setConfig(cfg)

		if err != nil {
			Log.Warningf("Failed to create security group in EC2 region %s; network id: %s; %s", region, n.ID.String(), err.Error())
			awswrapper.DeleteLoadBalancer(accessKeyID, secretAccessKey, region, &lbName)
			return
		}

		listeners := make([]*elb.Listener, 0)

		if ingress, ingressOk := securityCfg["ingress"]; ingressOk {
			switch ingress.(type) {
			case map[string]interface{}:
				ingressCfg := ingress.(map[string]interface{})
				for cidr := range ingressCfg {
					tcp := make([]int64, 0)
					udp := make([]int64, 0)
					if _tcp, tcpOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
						for i := range _tcp {
							_port := int64(_tcp[i].(float64))
							tcp = append(tcp, _port)
							listeners = append(listeners, &elb.Listener{
								InstancePort:     &_port,
								InstanceProtocol: stringOrNil("TCP"),
								LoadBalancerPort: &_port,
								Protocol:         stringOrNil("TCP"),
								SSLCertificateId: nil, // FIXME-- enable provisioning of SSL certificate for with dynamically created ELB
							})
						}
					}

					// UDP not currently supported by classic ELB API; UDP support is not needed here at this time...
					// if _udp, udpOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
					// 	for i := range _udp {
					// 		_port := int64(_udp[i].(float64))
					// 		udp = append(udp, _port)
					// 		listeners = append(listeners, &elb.Listener{
					// 			InstancePort:     &_port,
					// 			InstanceProtocol: stringOrNil("TCP"),
					// 			LoadBalancerPort: &_port,
					// 			Protocol:         stringOrNil("TCP"),
					// 			SSLCertificateId: nil, // FIXME-- enable provisioning of SSL certificate for with dynamically created ELB
					// 		})
					// 	}
					// }

					_, err := awswrapper.AuthorizeSecurityGroupIngress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
					if err != nil {
						Log.Warningf("Failed to authorize load balancer security group ingress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
					}
				}
			}
		}
		// end security group handling

		loadBalancer, err := awswrapper.CreateLoadBalancer(accessKeyID, secretAccessKey, region, &vpcID, &lbName, securityGroupIds, listeners)
		if err != nil {
			Log.Warningf("Failed to provision load balancer for network: %s", n.ID, err.Error())
			return
		}

		cfg["load_balancer_name"] = lbName
		cfg["load_balancer_url"] = loadBalancer.DNSName
		cfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *loadBalancer.DNSName, jsonRPCPort)
		cfg["websocket_url"] = fmt.Sprintf("ws://%s:%v", *loadBalancer.DNSName, websocketPort)

		n.setConfig(cfg)
		db.Save(n)
	}
}

func (n *Network) resolveContracts(db *gorm.DB) {
	cfg := n.ParseConfig()
	if n.isEthereumNetwork() {
		chainspec, chainspecOk := cfg["chainspec"].(map[string]interface{})
		chainspecAbi, chainspecAbiOk := cfg["chainspec_abi"].(map[string]interface{})
		if chainspecOk && chainspecAbiOk {
			Log.Debugf("Resolved configuration for chainspec and ABI for network: %s; attempting to import contracts", n.ID)

			if accounts, accountsOk := chainspec["accounts"].(map[string]interface{}); accountsOk {
				addrs := make([]string, 0)
				for addr := range accounts {
					addrs = append(addrs, addr)
				}
				sort.Strings(addrs)

				for _, addr := range addrs {
					Log.Debugf("Processing chainspec account %s for network: %s", addr, n.ID)
					account := accounts[addr]

					_, constructorOk := account.(map[string]interface{})["constructor"].(string)
					abi, abiOk := chainspecAbi[addr].([]interface{})
					if constructorOk && abiOk {
						Log.Debugf("Chainspec account %s has a valid constructor and ABI for network: %s; attempting to import contract", addr, n.ID)

						contractName := fmt.Sprintf("Network Contract %s", addr)
						if name, ok := account.(map[string]interface{})["name"].(interface{}); ok {
							contractName = name.(string)
						}
						params := map[string]interface{}{
							"name": contractName,
							"abi":  abi,
						}
						contract := &Contract{
							ApplicationID: nil,
							NetworkID:     n.ID,
							TransactionID: nil,
							Name:          stringOrNil(contractName),
							Address:       stringOrNil(addr),
							Params:        nil,
						}
						contract.setParams(params)
						if contract.Create() {
							Log.Debugf("Created contract %s for %s network chainspec account: %s", contract.ID, *n.Name, addr)
						} else {
							Log.Warningf("Failed to create contract for %s network chainspec account: %s; %s", *n.Name, addr, *contract.Errors[0].Message)
						}
					}
				}
			}
		}
	}
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
			n.Errors = append(n.Errors, &provide.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}

	return len(n.Errors) == 0
}

// setConfig sets the network config in-memory
func (n *Network) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	_cfgJSON := json.RawMessage(cfgJSON)
	n.Config = &_cfgJSON
}

// setChainID is an internal method used to set a unique chainID for the network prior to its creation
func (n *Network) setChainID() {
	n.ChainID = stringOrNil(fmt.Sprintf("0x%x", time.Now().Unix()))
	cfg := n.ParseConfig()
	if cfg != nil {
		if n.ChainID != nil {
			networkID, err := hexutil.DecodeBig(*n.ChainID)
			if err == nil {
				cfg["network_id"] = networkID.Uint64()
				if chainspec, chainspecOk := cfg["chainspec"].(map[string]interface{}); chainspecOk {
					if params, paramsOk := chainspec["params"].(map[string]interface{}); paramsOk {
						params["chainID"] = n.ChainID
						params["networkID"] = n.ChainID
					}
				}
				n.setConfig(cfg)
			}
		}
	}
}

// resolveAndBalanceJsonRpcAndWebsocketUrls updates the network's configured block
// JSON-RPC urls (i.e. web-based IDE), and enriches the network cfg
func (n *Network) resolveAndBalanceJsonRpcAndWebsocketUrls(db *gorm.DB) {
	cfg := n.ParseConfig()

	// accessKeyID := *DefaultAWSConfig.AccessKeyId
	// secretAccessKey := *DefaultAWSConfig.SecretAccessKey
	// region := *DefaultAWSConfig.DefaultRegion
	// vpcID := *DefaultAWSConfig.DefaultVpcID

	isLoadBalanced := false
	if loadBalanced, loadBalancedOk := cfg["is_load_balanced"].(bool); loadBalancedOk {
		isLoadBalanced = loadBalanced
	}

	Log.Debugf("Attempting to resolve and balance JSON-RPC and websocket urls for network with id: %s", n.ID)

	var node = &NetworkNode{}
	db.Where("network_id = ? AND status = 'running' AND role IN ('peer', 'full', 'validator', 'faucet')", n.ID).First(&node)

	if node != nil && node.ID != uuid.Nil {
		if isLoadBalanced {
			Log.Warningf("JSON-RPC/websocket load balancer may contain unhealthy or undeployed nodes")
			// FIXME: stick the new node behind the loadbalancer...
			// FIXME: if ubuntu vm, use elb classic
			// awswrapper.RegisterInstanceWithLoadBalancer(accessKeyID, secretAccessKey, region, lbName)
		} else {
			if reachable, port := node.reachableViaJSONRPC(); reachable {
				Log.Debugf("Node reachable via JSON-RPC port %d; node id: %s", port, n.ID)
				cfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *node.Host, port)
			} else {
				Log.Debugf("Node unreachable via JSON-RPC port; node id: %s", n.ID)
				cfg["json_rpc_url"] = nil
			}

			if reachable, port := node.reachableViaWebsocket(); reachable {
				cfg["websocket_url"] = fmt.Sprintf("ws://%s:%v", *node.Host, port)
			} else {
				cfg["websocket_url"] = nil
			}

			cfgJSON, _ := json.Marshal(cfg)
			*n.Config = json.RawMessage(cfgJSON)

			db.Save(n)
		}
	} else if !isLoadBalanced {
		cfg["json_rpc_url"] = nil
		cfg["websocket_url"] = nil

		cfgJSON, _ := json.Marshal(cfg)
		*n.Config = json.RawMessage(cfgJSON)

		db.Save(n)

		if n.AvailablePeerCount() == 0 {
			EvictNetworkStatsDaemon(n)
		}
	}
}

// resolveAndBalanceExplorerUrls updates the network's configured block
// explorer urls (i.e. web-based IDE), and enriches the network cfg
func (n *Network) resolveAndBalanceExplorerUrls(db *gorm.DB, node *NetworkNode) {
	ticker := time.NewTicker(hostReachabilityInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			cfg := n.ParseConfig()
			nodeCfg := node.ParseConfig()

			isLoadBalanced := false
			if loadBalanced, loadBalancedOk := cfg["is_load_balanced"].(bool); loadBalancedOk {
				isLoadBalanced = loadBalanced
			}

			if time.Now().Sub(startedAt) >= hostReachabilityTimeout {
				Log.Warningf("Failed to resolve and balance explorer urls for network node: %s; timing out after %v", n.ID.String(), hostReachabilityTimeout)
				if !isLoadBalanced {
					cfg["block_explorer_url"] = nil
					cfgJSON, _ := json.Marshal(cfg)
					*n.Config = json.RawMessage(cfgJSON)
					db.Save(n)
				}
				ticker.Stop()
				return
			}

			Log.Debugf("Attempting to resolve and balance block explorer url for network node: %s", n.ID.String())

			var node = &NetworkNode{}
			db.Where("network_id = ? AND status = 'running' AND role IN ('explorer')", n.ID).First(&node)

			if node != nil && node.ID != uuid.Nil {
				if isLoadBalanced {
					Log.Warningf("Block explorer load balancer may contain unhealthy or undeployed nodes")
				} else {
					if node.reachableOnPort(defaultWebappPort) {
						Log.Debugf("Block explorer reachable via port %d; node id: %s", defaultWebappPort, n.ID)

						cfg["block_explorer_url"] = fmt.Sprintf("http://%s:%v", *node.Host, defaultWebappPort)
						cfgJSON, _ := json.Marshal(cfg)
						*n.Config = json.RawMessage(cfgJSON)

						nodeCfg["url"] = cfg["block_explorer_url"]
						nodeCfgJSON, _ := json.Marshal(nodeCfg)
						*node.Config = json.RawMessage(nodeCfgJSON)

						db.Save(n)
						db.Save(node)
						ticker.Stop()
						return
					} else {
						Log.Debugf("Block explorer unreachable via webapp port; node id: %s", n.ID)
						cfg["block_explorer_url"] = nil
					}
				}
			} else if !isLoadBalanced {
				cfg["block_explorer_url"] = nil
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				db.Save(n)
			}
		}
	}
}

// resolveAndBalanceStudioUrls updates the network's configured studio url
// (i.e. web-based IDE), and enriches the network cfg
func (n *Network) resolveAndBalanceStudioUrls(db *gorm.DB, node *NetworkNode) {
	ticker := time.NewTicker(hostReachabilityInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			cfg := n.ParseConfig()
			nodeCfg := node.ParseConfig()

			isLoadBalanced := false
			if loadBalanced, loadBalancedOk := cfg["is_load_balanced"].(bool); loadBalancedOk {
				isLoadBalanced = loadBalanced
			}

			if time.Now().Sub(startedAt) >= hostReachabilityTimeout {
				Log.Warningf("Failed to resolve and balance studio (IDE) url for network node: %s; timing out after %v", n.ID.String(), hostReachabilityTimeout)
				if !isLoadBalanced {
					cfg["studio_url"] = nil
					cfgJSON, _ := json.Marshal(cfg)
					*n.Config = json.RawMessage(cfgJSON)
					db.Save(n)
				}
				ticker.Stop()
				return
			}

			if n.isEthereumNetwork() {
				Log.Debugf("Attempting to resolve and balance studio IDE url for network node: %s", n.ID.String())

				var node = &NetworkNode{}
				db.Where("network_id = ? AND status = 'running' AND role IN ('studio')", n.ID).First(&node)

				if node != nil && node.ID != uuid.Nil {
					if isLoadBalanced {
						Log.Warningf("Studio IDE load balancer may contain unhealthy or undeployed nodes")
					} else {
						if node.reachableOnPort(defaultWebappPort) {
							cfg["studio_url"] = fmt.Sprintf("http://%s:%v", *node.Host, defaultWebappPort)
							cfgJSON, _ := json.Marshal(cfg)
							*n.Config = json.RawMessage(cfgJSON)

							nodeCfg["url"] = cfg["studio_url"]
							nodeCfgJSON, _ := json.Marshal(nodeCfg)
							*node.Config = json.RawMessage(nodeCfgJSON)

							db.Save(n)
							db.Save(node)
							ticker.Stop()
							return
						} else {
							cfg["studio_url"] = nil
						}
					}
				} else if !isLoadBalanced {
					cfg["studio_url"] = nil
					cfgJSON, _ := json.Marshal(cfg)
					*n.Config = json.RawMessage(cfgJSON)
					db.Save(n)
				}
			}
		}
	}
}

// Validate a network for persistence
func (n *Network) Validate() bool {
	n.Errors = make([]*provide.Error, 0)
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
	if n.isBcoinNetwork() {
		status, err = provide.BcoinGetNetworkStatus(n.ID.String(), n.rpcURL())
	} else if n.isEthereumNetwork() {
		status, err = provide.EVMGetNetworkStatus(n.ID.String(), n.rpcURL())
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

// AvailablePeerCount retrieves a count of platform-managed network nodes which also have the 'peer' or 'full' role
// and currently are listed with a status of 'running'; this method does not currently check real-time availability
// of these peers-- it is assumed the are still available. FIXME?
func (n *Network) AvailablePeerCount() (count uint64) {
	DatabaseConnection().Model(&NetworkNode{}).Where("network_nodes.network_id = ? AND network_nodes.status = 'running' AND network_nodes.role IN ('peer', 'full', 'validator', 'faucet')", n.ID).Count(&count)
	return count
}

// BootnodesTxt retrieves the current bootnodes string for the network; this value can be used
// to set peer/bootnodes list from which new network nodes are initialized
func (n *Network) BootnodesTxt() (*string, error) {
	bootnodes, err := n.Bootnodes()
	if err != nil {
		return nil, err
	}

	txt := ""
	for i := range bootnodes {
		bootnode := bootnodes[i]
		peerURL := bootnode.peerURL()
		if peerURL != nil {
			txt += *peerURL
		}
	}

	Log.Debugf("Resolved bootnodes environment variable for network with id: %s; bootnodes: %s", n.ID, txt)
	return stringOrNil(txt), err
}

// Bootnodes retrieves a list of network bootnodes
func (n *Network) Bootnodes() (nodes []*NetworkNode, err error) {
	query := DatabaseConnection().Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// BootnodesCount returns a count of the number of bootnodes on the network
func (n *Network) BootnodesCount() (count uint64) {
	db := DatabaseConnection()
	db.Model(&NetworkNode{}).Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID).Count(&count)
	return count
}

// Nodes retrieves a list of network nodes
func (n *Network) Nodes() (nodes []*NetworkNode, err error) {
	query := DatabaseConnection().Where("network_nodes.network_id = ?", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

func (n *Network) isBcoinNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isBcoinNetwork, ok := cfg["is_bcoin_network"].(bool); ok {
			return isBcoinNetwork
		}
	}
	return false
}

func (n *Network) isEthereumNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg["is_ethereum_network"].(bool); ok {
			return isEthereumNetwork
		}
	}
	return false
}

func (n *Network) isHandshakeNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isHandshakeNetwork, ok := cfg["is_handshake_network"].(bool); ok {
			return isHandshakeNetwork
		}
	}
	return false
}

func (n *Network) isLcoinNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isLcoinNetwork, ok := cfg["is_lcoin_network"].(bool); ok {
			return isLcoinNetwork
		}
	}
	return false
}

func (n *Network) isQuorumNetwork() bool {
	if n.Name != nil && strings.HasPrefix(strings.ToLower(*n.Name), "eth") {
		return true
	}

	cfg := n.ParseConfig()
	if cfg != nil {
		if isQuorumNetwork, ok := cfg["is_quorum_network"].(bool); ok {
			return isQuorumNetwork
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
				n.Errors = append(n.Errors, &provide.Error{
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
	_cfgJSON := json.RawMessage(cfgJSON)
	n.Config = &_cfgJSON
}

func (n *NetworkNode) peerURL() *string {
	cfg := n.ParseConfig()
	if peerURL, peerURLOk := cfg["peer_url"].(string); peerURLOk {
		return stringOrNil(peerURL)
	}

	return nil
}

func (n *NetworkNode) reachableViaJSONRPC() (bool, uint) {
	cfg := n.ParseConfig()
	defaultJSONRPCPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultJSONRPCPort = engineToDefaultJSONRPCPortMapping[engineID]
	}
	port := uint(defaultJSONRPCPort)
	if jsonRPCPortOverride, jsonRPCPortOverrideOk := cfg["default_json_rpc_port"].(float64); jsonRPCPortOverrideOk {
		port = uint(jsonRPCPortOverride)
	}

	return n.reachableOnPort(port), port
}

func (n *NetworkNode) reachableViaWebsocket() (bool, uint) {
	cfg := n.ParseConfig()
	defaultWebsocketPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultWebsocketPort = engineToDefaultWebsocketPortMapping[engineID]
	}
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

func (n *NetworkNode) relatedNetwork() *Network {
	var network = &Network{}
	DatabaseConnection().Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		Log.Warningf("Failed to retrieve network for network node: %s", n.ID)
		return nil
	}
	return network
}

func (n *NetworkNode) updateStatus(db *gorm.DB, status string, description *string) {
	n.Status = stringOrNil(status)
	n.Description = description
	result := db.Save(&n)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			n.Errors = append(n.Errors, &provide.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
}

// Validate a network node for persistence
func (n *NetworkNode) Validate() bool {
	cfg := n.ParseConfig()
	// if _, protocolOk := cfg["protocol_id"].(string); !protocolOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: stringOrNil("Failed to parse protocol_id in network node configuration"),
	// 	})
	// }
	// if _, engineOk := cfg["engine_id"].(string); !engineOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: stringOrNil("Failed to parse engine_id in network node configuration"),
	// 	})
	// }
	// if targetID, targetOk := cfg["target_id"].(string); targetOk {
	// 	if creds, credsOk := cfg["credentials"].(map[string]interface{}); credsOk {
	// 		if strings.ToLower(targetID) == "aws" {
	// 			if _, accessKeyIdOk := creds["aws_access_key_id"].(string); !accessKeyIdOk {
	// 				n.Errors = append(n.Errors, &provide.Error{
	// 					Message: stringOrNil("Failed to parse aws_access_key_id in network node credentials configuration for AWS target"),
	// 				})
	// 			}
	// 			if _, secretAccessKeyOk := creds["aws_secret_access_key"].(string); !secretAccessKeyOk {
	// 				n.Errors = append(n.Errors, &provide.Error{
	// 					Message: stringOrNil("Failed to parse aws_secret_access_key in network node credentials configuration for AWS target"),
	// 				})
	// 			}
	// 		}
	// 	} else {
	// 		n.Errors = append(n.Errors, &provide.Error{
	// 			Message: stringOrNil("Failed to parse credentials in network node configuration"),
	// 		})
	// 	}
	// } else {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: stringOrNil("Failed to parse target_id in network node configuration"),
	// 	})
	// }
	// if _, providerOk := cfg["provider_id"].(string); !providerOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: stringOrNil("Failed to parse provider_id in network node configuration"),
	// 	})
	// }
	if role, roleOk := cfg["role"].(string); roleOk {
		if n.Role == nil || *n.Role != role {
			Log.Debugf("Coercing network node role to match node configuration; role: %s", role)
			n.Role = stringOrNil(role)
		}
	}
	// } else {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: stringOrNil("Failed to parse role in network node configuration"),
	// 	})
	// }

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
			n.Errors = append(n.Errors, &provide.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
	return len(n.Errors) == 0
}

// Reload the underlying network node instance
func (n *NetworkNode) Reload() {
	db := DatabaseConnection()
	db.Model(&n).Find(n)
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

	if regionOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if providerID, providerIdOk := cfg["provider_id"].(string); providerIdOk {
				if strings.ToLower(providerID) == "docker" {
					if ids, idsOk := cfg["target_task_ids"].([]interface{}); idsOk {
						logs := make([]string, 0)
						for i := range ids {
							logEvents, err := awswrapper.GetContainerLogEvents(accessKeyID, secretAccessKey, region, ids[i].(string), nil)
							if err == nil && logEvents != nil {
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
	} else {
		return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; no region provided: %s", *network.Name, providerID)
	}

	return nil, fmt.Errorf("Unable to retrieve logs for network node on unsupported network: %s", *network.Name)
}

func (n *Network) requireBootnodes(db *gorm.DB, pending *NetworkNode) ([]*NetworkNode, error) {
	mutex, mutexOk := networkGenesisMutex[pending.NetworkID.String()]
	if !mutexOk {
		mutex = &sync.Mutex{}
		networkGenesisMutex[pending.NetworkID.String()] = mutex
	}

	mutex.Lock()
	defer mutex.Unlock()

	count := n.BootnodesCount()
	bootnodes := make([]*NetworkNode, 0)

	if count == 0 {
		pending.Bootnode = true
		pending.updateStatus(db, "genesis", nil)
		bootnodes = append(bootnodes, pending)
		err := new(bootnodesInitialized)
		Log.Debugf("Coerced network node into initial bootnode for network with id: %s", n.ID)
		return bootnodes, *err
	}

	bootnodes, err := n.Bootnodes()
	Log.Debugf("Resolved %d initial bootnode(s) for network with id: %s", len(bootnodes), n.ID)

	return bootnodes, err
}

func (n *NetworkNode) deploy(db *gorm.DB) {
	if n.Config == nil {
		Log.Debugf("Not attempting to deploy network node without a valid configuration; network node id: %s", n.ID)
		return
	}

	go func() {
		Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, n)

		var network = &Network{}
		db.Model(n).Related(&network)
		if network == nil || network.ID == uuid.Nil {
			desc := fmt.Sprintf("Failed to retrieve network for network node: %s", n.ID)
			n.updateStatus(db, "failed", &desc)
			Log.Warning(desc)
			return
		}

		bootnodes, err := network.requireBootnodes(db, n)
		if err != nil {
			switch err.(type) {
			case bootnodesInitialized:
				Log.Debugf("Bootnode initialized for network: %s; node: %s; waiting for genesis to complete and peer resolution to become possible", *network.Name, n.ID.String())
				cfg := n.ParseConfig()
				if protocol, protocolOk := cfg["protocol_id"].(string); protocolOk {
					if strings.ToLower(protocol) == "poa" {
						if env, envOk := cfg["env"].(map[string]interface{}); envOk {
							var addr *string
							var privateKey *ecdsa.PrivateKey
							_, masterOfCeremonyPrivateKeyOk := env["ENGINE_SIGNER_PRIVATE_KEY"].(string)
							if masterOfCeremony, masterOfCeremonyOk := env["ENGINE_SIGNER"].(string); masterOfCeremonyOk && !masterOfCeremonyPrivateKeyOk {
								addr = stringOrNil(masterOfCeremony)

								wallet := &Wallet{}
								DatabaseConnection().Where("wallets.user_id = ? AND wallets.address = ?", n.UserID.String(), addr).Find(&wallet)
								if wallet == nil || wallet.ID == uuid.Nil {
									Log.Warningf("Failed to retrieve manage engine signing identity for network: %s; generating unmanaged identity...", *network.Name)
									addr, privateKey, err = provide.EVMGenerateKeyPair()
								} else {
									privateKey, err = decryptECDSAPrivateKey(*wallet.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
									if err == nil {
										Log.Debugf("Decrypted private key for master of ceremony on network: %s", *network.Name)
									}
								}
							} else if !masterOfCeremonyPrivateKeyOk {
								Log.Debugf("Generating managed master of ceremony signing identity for network: %s", *network.Name)
								addr, privateKey, err = provide.EVMGenerateKeyPair()
							}

							if addr != nil && privateKey != nil {
								keystoreJSON, err := provide.EVMMarshalEncryptedKey(common.HexToAddress(*addr), privateKey, hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
								if err == nil {
									Log.Debugf("Master of ceremony has initiated the initial key ceremony: %s; network: %s", addr, *network.Name)
									env["ENGINE_SIGNER"] = addr
									env["ENGINE_SIGNER_PRIVATE_KEY"] = hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
									env["ENGINE_SIGNER_KEY_JSON"] = string(keystoreJSON)

									n.setConfig(cfg)

									networkCfg := network.ParseConfig()
									if chainspec, chainspecOk := networkCfg["chainspec"].(map[string]interface{}); chainspecOk {
										if accounts, accountsOk := chainspec["accounts"].(map[string]interface{}); accountsOk {
											nonSystemAccounts := make([]string, 0)
											for account := range accounts {
												if !strings.HasPrefix(account, "0x000000000000000000000000000000000") { // 7 chars truncated
													nonSystemAccounts = append(nonSystemAccounts, account)
												}
											}
											if len(nonSystemAccounts) == 1 {
												templateMasterOfCeremony := nonSystemAccounts[0]
												chainspecJSON, err := json.Marshal(chainspec)
												if err == nil {
													chainspecJSON = []byte(strings.Replace(string(chainspecJSON), templateMasterOfCeremony[2:], string(*addr)[2:], -1))
													var newChainspec map[string]interface{}
													err = json.Unmarshal(chainspecJSON, &newChainspec)
													if err == nil {
														networkCfg["chainspec"] = newChainspec
														network.setConfig(networkCfg)
														db.Save(network)
													}
												}
											}
										}
									}
								} else {
									Log.Warningf("Failed to generate master of ceremony address for network: %s; %s", *network.Name, err.Error())
								}
							}
						}
					}
				}
				n._deploy(network, bootnodes, db)
			}
		} else {
			n.requireGenesis(network, bootnodes, db)
		}
	}()
}

func (n *NetworkNode) requireGenesis(network *Network, bootnodes []*NetworkNode, db *gorm.DB) {
	ticker := time.NewTicker(resolveGenesisTickerInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			if time.Now().Sub(startedAt) >= resolveGenesisTickerTimeout {
				desc := fmt.Sprintf("Failed to resolve genesis block for network bootnode: %s; timing out after %v", n.ID.String(), resolveGenesisTickerTimeout)
				n.updateStatus(db, "failed", &desc)
				Log.Warning(desc)
				ticker.Stop()
				return
			}

			if daemon, daemonOk := currentNetworkStats[network.ID.String()]; daemonOk {
				if daemon.stats != nil {
					if daemon.stats.Block > 0 {
						n._deploy(network, bootnodes, db)
						ticker.Stop()
						return
					}
				}
			}
		}
	}
}

func (n *NetworkNode) _deploy(network *Network, bootnodes []*NetworkNode, db *gorm.DB) {
	cfg := n.ParseConfig()
	networkCfg := network.ParseConfig()

	cfg["default_json_rpc_port"] = networkCfg["default_json_rpc_port"]
	cfg["default_websocket_port"] = networkCfg["default_websocket_port"]

	targetID, targetOk := cfg["target_id"].(string)
	engineID, engineOk := cfg["engine_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	role, roleOk := cfg["role"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	rcd, rcdOk := cfg["rc.d"].(string)
	region, regionOk := cfg["region"].(string)
	env, envOk := cfg["env"].(map[string]interface{})

	if networkEnv, networkEnvOk := networkCfg["env"].(map[string]interface{}); envOk && networkEnvOk {
		Log.Debugf("Applying environment overrides to network node per network env configuration")
		for k := range networkEnv {
			env[k] = networkEnv[k]
		}
	}

	cloneableCfg, cloneableCfgOk := networkCfg["cloneable_cfg"].(map[string]interface{})
	if !cloneableCfgOk {
		desc := fmt.Sprintf("Failed to parse cloneable configuration for network node: %s", n.ID)
		n.updateStatus(db, "failed", &desc)
		Log.Warning(desc)
		return
	}

	securityCfg, securityCfgOk := cloneableCfg["_security"].(map[string]interface{})
	if !securityCfgOk {
		desc := fmt.Sprintf("Failed to parse cloneable security configuration for network node: %s", n.ID)
		n.updateStatus(db, "failed", &desc)
		Log.Warning(desc)
		return
	}

	cloneableTarget, cloneableTargetOk := cloneableCfg[targetID].(map[string]interface{})
	if !cloneableTargetOk {
		desc := fmt.Sprintf("Failed to parse cloneable target configuration for network node: %s", n.ID)
		n.updateStatus(db, "failed", &desc)
		Log.Warning(desc)
		return
	}

	cloneableProvider, cloneableProviderOk := cloneableTarget[providerID].(map[string]interface{})
	if !cloneableProviderOk {
		desc := fmt.Sprintf("Failed to parse cloneable provider configuration for network node: %s", n.ID)
		n.updateStatus(db, "failed", &desc)
		Log.Warning(desc)
		return
	}

	providerCfgByRegion, providerCfgByRegionOk := cloneableProvider["regions"].(map[string]interface{})
	if !providerCfgByRegionOk && !regionOk {
		desc := fmt.Sprintf("Failed to parse cloneable provider configuration by region (or a single specific deployment region) for network node: %s", n.ID)
		n.updateStatus(db, "failed", &desc)
		Log.Warningf(desc)
		return
	}

	Log.Debugf("Configuration for network node deploy: target id: %s; provider: %s; role: %s; crendentials: %s; region: %s, rc.d: %s; cloneable provider cfg: %s; network config: %s",
		targetID, providerID, role, credentials, region, rcd, providerCfgByRegion, networkCfg)

	if targetOk && engineOk && providerOk && roleOk && credsOk && regionOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			// start security group handling
			securityGroupDesc := fmt.Sprintf("security group for network node: %s", n.ID.String())
			securityGroup, err := awswrapper.CreateSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupDesc, securityGroupDesc, nil)
			securityGroupIds := make([]string, 0)

			if securityGroup != nil {
				securityGroupIds = append(securityGroupIds, *securityGroup.GroupId)
			}

			cfg["region"] = region
			cfg["target_security_group_ids"] = securityGroupIds
			n.setConfig(cfg)

			if err != nil {
				desc := fmt.Sprintf("Failed to create security group in EC2 region %s; network node id: %s; %s", region, n.ID.String(), err.Error())
				n.updateStatus(db, "failed", &desc)
				Log.Warning(desc)
				return
			}

			if egress, egressOk := securityCfg["egress"]; egressOk {
				switch egress.(type) {
				case string:
					if egress.(string) == "*" {
						_, err := awswrapper.AuthorizeSecurityGroupEgressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, *securityGroup.GroupId)
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
						_, err := awswrapper.AuthorizeSecurityGroupEgress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
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
						_, err := awswrapper.AuthorizeSecurityGroupIngressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, *securityGroup.GroupId)
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
						_, err := awswrapper.AuthorizeSecurityGroupIngress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
						if err != nil {
							Log.Warningf("Failed to authorize security group ingress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
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
							instanceIds, err := awswrapper.LaunchAMI(accessKeyID, secretAccessKey, region, imageID, userData, 1, 1)
							if err != nil || len(instanceIds) == 0 {
								desc := fmt.Sprintf("Attempt to deploy image %s@%s in EC2 %s region failed; %s", imageID, version, region, err.Error())
								n.updateStatus(db, "failed", &desc)
								n.unregisterSecurityGroups()
								Log.Warning(desc)
								return
							}
							Log.Debugf("Attempt to deploy image %s@%s in EC2 %s region successful; instance ids: %s", imageID, version, region, instanceIds)
							cfg["target_instance_ids"] = instanceIds

							Log.Debugf("Assigning %v security groups for deployed image %s@%s in EC2 %s region; instance ids: %s", len(securityGroupIds), imageID, version, region, instanceIds)
							for i := range instanceIds {
								awswrapper.SetInstanceSecurityGroups(accessKeyID, secretAccessKey, region, instanceIds[i], securityGroupIds)
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

						if bnodes, bootnodesOk := envOverrides["BOOTNODES"].(string); bootnodesOk {
							envOverrides["BOOTNODES"] = bnodes
						} else {
							bootnodesTxt, err := network.BootnodesTxt()
							if err == nil && bootnodesTxt != nil && *bootnodesTxt != "" {
								envOverrides["BOOTNODES"] = bootnodesTxt
							}
						}
						if _, peerSetOk := envOverrides["PEER_SET"]; !peerSetOk && envOverrides["BOOTNODES"] != nil {
							if bnodes, bootnodesOk := envOverrides["BOOTNODES"].(string); bootnodesOk {
								envOverrides["PEER_SET"] = strings.Replace(strings.Replace(bnodes, "enode://", "required:", -1), ",", " ", -1)
							} else if bnodes, bootnodesOk := envOverrides["BOOTNODES"].(*string); bootnodesOk {
								envOverrides["PEER_SET"] = strings.Replace(strings.Replace(*bnodes, "enode://", "required:", -1), ",", " ", -1)
							}
						}

						if client, clientOk := networkCfg["client"].(string); clientOk {
							envOverrides["CLIENT"] = client
						} else {
							if defaultClientEnv, defaultClientEnvOk := engineToNetworkNodeClientEnvMapping[engineID]; defaultClientEnvOk {
								envOverrides["CLIENT"] = defaultClientEnv
							} else {
								envOverrides["CLIENT"] = defaultClient
							}
						}

						if chain, chainOk := networkCfg["chain"].(string); chainOk {
							envOverrides["CHAIN"] = chain
						}
						overrides := map[string]interface{}{
							"environment": envOverrides,
						}
						cfg["env"] = envOverrides
						n.setConfig(cfg)
						db.Save(n)

						taskIds, err := awswrapper.StartContainer(accessKeyID, secretAccessKey, region, container, nil, nil, securityGroupIds, []string{}, overrides)

						if err != nil || len(taskIds) == 0 {
							desc := fmt.Sprintf("Attempt to deploy container %s in EC2 %s region failed; %s", container, region, err.Error())
							n.updateStatus(db, "failed", &desc)
							n.unregisterSecurityGroups()
							Log.Warning(desc)
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
}

func (n *NetworkNode) resolveHost(db *gorm.DB, network *Network, cfg map[string]interface{}, identifiers []string) {
	ticker := time.NewTicker(resolveHostTickerInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			if n.Host == nil {
				if time.Now().Sub(startedAt) >= resolveHostTickerTimeout {
					desc := fmt.Sprintf("Failed to resolve hostname for network node: %s; timing out after %v", n.ID.String(), resolveHostTickerTimeout)
					n.updateStatus(db, "failed", &desc)
					ticker.Stop()
					Log.Warning(desc)
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
						instanceDetails, err := awswrapper.GetInstanceDetails(accessKeyID, secretAccessKey, region, id)
						if err == nil {
							if len(instanceDetails.Reservations) > 0 {
								reservation := instanceDetails.Reservations[0]
								if len(reservation.Instances) > 0 {
									instance := reservation.Instances[0]
									n.Host = instance.PublicDnsName
									n.IPv4 = instance.PublicIpAddress
								}
							}
						}
					} else if strings.ToLower(providerID) == "docker" {
						containerDetails, err := awswrapper.GetContainerDetails(accessKeyID, secretAccessKey, region, id, nil)
						if err == nil {
							if len(containerDetails.Tasks) > 0 {
								task := containerDetails.Tasks[0]
								if len(task.Attachments) > 0 {
									attachment := task.Attachments[0]
									if attachment.Type != nil && *attachment.Type == "ElasticNetworkInterface" {
										for i := range attachment.Details {
											kvp := attachment.Details[i]
											if kvp.Name != nil && *kvp.Name == "networkInterfaceId" && kvp.Value != nil {
												interfaceDetails, err := awswrapper.GetNetworkInterfaceDetails(accessKeyID, secretAccessKey, region, *kvp.Value)
												if err == nil {
													if len(interfaceDetails.NetworkInterfaces) > 0 {
														Log.Debugf("Retrieved interface details for container instance: %s", interfaceDetails)
														interfaceAssociation := interfaceDetails.NetworkInterfaces[0].Association
														if interfaceAssociation != nil {
															n.Host = interfaceAssociation.PublicDnsName
															n.IPv4 = interfaceAssociation.PublicIp
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

				role, roleOk := cfg["role"].(string)
				if roleOk {
					if role == "explorer" {
						go network.resolveAndBalanceExplorerUrls(db, n)
					} else if role == "faucet" {
						Log.Warningf("Faucet role not yet supported")
					} else if role == "studio" {
						go network.resolveAndBalanceStudioUrls(db, n)
					}
				}

				ticker.Stop()
				return
			}
		}
	}
}

func (n *NetworkNode) resolvePeerURL(db *gorm.DB, network *Network, cfg map[string]interface{}, identifiers []string) {
	role, roleOk := cfg["role"].(string)
	if !roleOk || role != "peer" && role != "full" && role != "validator" {
		return
	}

	Log.Debugf("Attempting to resolve peer url for network node: %s", n.ID.String())

	ticker := time.NewTicker(resolvePeerTickerInterval)
	startedAt := time.Now()
	var peerURL *string
	for {
		select {
		case <-ticker.C:
			if peerURL == nil {
				if time.Now().Sub(startedAt) >= resolvePeerTickerTimeout {
					Log.Warningf("Failed to resolve peer url for network node: %s; timing out after %v", n.ID.String(), resolvePeerTickerTimeout)
					n.Status = stringOrNil("peer_resolution_failed")
					db.Save(n)
					ticker.Stop()
					return
				}

				id := identifiers[len(identifiers)-1]
				targetID, targetOk := cfg["target_id"].(string)
				engineID, engineOk := cfg["engine_id"].(string)
				providerID, providerOk := cfg["provider_id"].(string)
				region, regionOk := cfg["region"].(string)
				credentials, credsOk := cfg["credentials"].(map[string]interface{})

				if strings.ToLower(targetID) == "aws" && targetOk && engineOk && providerOk && regionOk && credsOk {
					accessKeyID := credentials["aws_access_key_id"].(string)
					secretAccessKey := credentials["aws_secret_access_key"].(string)

					if strings.ToLower(providerID) == "ubuntu-vm" {
						Log.Warningf("Peer URL resolution is not yet implemented for non-containerized AWS deployments")
						ticker.Stop()
						return
					} else if strings.ToLower(providerID) == "docker" {
						logs, err := awswrapper.GetContainerLogEvents(accessKeyID, secretAccessKey, region, id, nil)
						if err == nil {
							for i := range logs.Events {
								event := logs.Events[i]
								if event.Message != nil {
									msg := string(*event.Message)

									if network.isBcoinNetwork() {
										const bcoinPoolIdentitySearchString = "Pool identity key:"
										poolIdentityFoundIndex := strings.LastIndex(msg, bcoinPoolIdentitySearchString)
										if poolIdentityFoundIndex != -1 {
											defaultJSONRPCPort := engineToDefaultJSONRPCPortMapping[engineID]
											poolIdentity := strings.TrimSpace(msg[poolIdentityFoundIndex+len(bcoinPoolIdentitySearchString) : len(msg)-1])
											node := fmt.Sprintf("%s@%s:%v", poolIdentity, *n.IPv4, defaultJSONRPCPort)
											peerURL = &node
											cfg["peer_url"] = node
											cfg["peer_identity"] = poolIdentity
											ticker.Stop()
											break
										}
									} else if network.isEthereumNetwork() {
										nodeInfo := &provide.EthereumJsonRpcResponse{}
										err := json.Unmarshal([]byte(msg), &nodeInfo)
										if err == nil && nodeInfo != nil {
											result, resultOk := nodeInfo.Result.(map[string]interface{})
											if resultOk {
												if enode, enodeOk := result["enode"].(string); enodeOk {
													peerURL = stringOrNil(enode)
													cfg["peer"] = result
													cfg["peer_url"] = enode
													ticker.Stop()
													break
												}
											}
										} else if err != nil {
											enodeIndex := strings.LastIndex(msg, "enode://")
											if enodeIndex != -1 {
												enode := msg[enodeIndex:]
												peerURL = stringOrNil(enode)
												// FIXME? do we need this? cfg["peer"] = result
												cfg["peer_url"] = enode
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

			if peerURL != nil {
				Log.Debugf("Resolved peer url for network node with id: %s; peer url: %s", n.ID, *peerURL)
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				db.Save(n)

				role, roleOk := cfg["role"].(string)
				if roleOk {
					if role == "peer" || role == "full" || role == "validator" || role == "faucet" {
						network.resolveAndBalanceJsonRpcAndWebsocketUrls(db)
					}
				}

				ticker.Stop()
				return
			}
		}
	}
}

func (n *NetworkNode) undeploy() error {
	Log.Debugf("Attempting to undeploy network node with id: %s", n.ID, n)

	db := DatabaseConnection()
	n.updateStatus(db, "deprovisioning", nil)

	cfg := n.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	region, regionOk := cfg["region"].(string)
	role, roleOk := cfg["role"].(string)
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

					_, err := awswrapper.TerminateInstance(accessKeyID, secretAccessKey, region, instanceID)
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

					_, err := awswrapper.StopContainer(accessKeyID, secretAccessKey, region, taskID, nil)
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

		if roleOk {
			network := n.relatedNetwork()

			if role == "peer" || role == "full" || role == "validator" {
				go network.resolveAndBalanceJsonRpcAndWebsocketUrls(db)
			} else if role == "explorer" {
				go network.resolveAndBalanceExplorerUrls(db, n)
			} else if role == "faucet" {
				Log.Warningf("Faucet role not yet supported")
			} else if role == "studio" {
				go network.resolveAndBalanceStudioUrls(db, n)
			}
		}
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
					_, err := awswrapper.DeleteSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupID)
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
	db := DatabaseConnection()

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
			db := DatabaseConnection()
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
				tx.SignedTx, tx.Hash, err = provide.EVMSignTx(network.ID.String(), network.rpcURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), uint64(gas))
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

		if err != nil {
			desc := err.Error()
			tx.updateStatus(DatabaseConnection(), "failed", &desc)
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

// ParseConfig - parse the original JSON params used for Connector creation
func (c *Connector) ParseConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	if c.Config != nil {
		err := json.Unmarshal(*c.Config, &cfg)
		if err != nil {
			Log.Warningf("Failed to unmarshal connector params; %s", err.Error())
			return nil
		}
	}
	return cfg
}

// Create and persist a new Connector
func (c *Connector) Create() bool {
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
				c.Errors = append(c.Errors, &provide.Error{
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

// Validate an Connector for persistence
func (c *Connector) Validate() bool {
	c.Errors = make([]*provide.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &provide.Error{
			Message: stringOrNil("Unable to deploy connector using unspecified network"),
		})
	}
	if c.Type == nil || strings.ToLower(*c.Type) != "ipfs" {
		c.Errors = append(c.Errors, &provide.Error{
			Message: stringOrNil("Unable to define connector of invalid type"),
		})
	}
	return len(c.Errors) == 0
}

// Delete a connector
func (c *Connector) Delete() bool {
	db := DatabaseConnection()
	result := db.Delete(c)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			c.Errors = append(c.Errors, &provide.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
	return len(c.Errors) == 0
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

// IsUnique checks if the transaction hash exists in the database; returns true if the hash is nil or
func (t *Transaction) IsUnique() (bool, error) {
	if t.Hash == nil {
		return false, fmt.Errorf("Unable to determine if transaction hash is unique for null hash")
	}
	var count *uint64
	DatabaseConnection().Model(&Transaction{}).Where("hash = ?", *t.Hash).Count(&count)
	isUnique := *count == 0
	return isUnique, nil
}

// Execute an ephemeral ContractExecution
func (e *ContractExecution) Execute() (interface{}, error) {
	var _abi *abi.ABI
	if __abi, abiOk := e.ABI.(abi.ABI); abiOk {
		_abi = &__abi
	} else if e.Contract != nil {
		__abi, err := e.Contract.readEthereumContractAbi()
		if err != nil {
			Log.Warningf("Cannot attempt contract execution without ABI")
			return nil, err
		}
		_abi = __abi
	}

	if _abi != nil {
		if mthd, ok := _abi.Methods[e.Method]; ok {
			if mthd.Const {
				return e.Contract.Execute(e.Ref, e.Wallet, e.Value, e.Method, e.Params, 0)
			}
		}
	}

	txMsg, _ := json.Marshal(e)
	natsConnection := getNatsStreamingConnection()
	return e, natsConnection.Publish(natsTxSubject, txMsg)
}

// Execute a transaction on the contract instance using a specific signer, value, method and params
func (c *Contract) Execute(ref *string, wallet *Wallet, value *big.Int, method string, params []interface{}, gas uint64) (*ContractExecutionResponse, error) {
	var err error
	db := DatabaseConnection()
	var network = &Network{}
	db.Model(c).Related(&network)

	txParams := map[string]interface{}{}

	walletID := &uuid.Nil
	if wallet != nil {
		if wallet.ID != uuid.Nil {
			walletID = &wallet.ID
		}
		if stringOrNil(wallet.Address) != nil && wallet.PrivateKey != nil {
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

	tx.updateStatus(db, "success", nil)

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
				c.Errors = append(c.Errors, &provide.Error{
					Message: stringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(c) {
			success := rowsAffected > 0
			if success {
				params := c.ParseParams()
				_, rawSourceOk := params["raw_source"].(string)
				if rawSourceOk {
					contractCompilerInvocationMsg, _ := json.Marshal(c)
					natsConnection := getNatsStreamingConnection()
					natsConnection.Publish(natsContractCompilerInvocationSubject, contractCompilerInvocationMsg)
				}
			}
			return success
		}
	}
	return false
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
								Name:          stringOrNil(name),
								Symbol:        stringOrNil(symbol),
								Decimals:      decimals.Uint64(),
								Address:       stringOrNil(receipt.ContractAddress.Hex()),
							}
							if token.Create() {
								Log.Debugf("Created token %s for associated %s contract: %s", token.ID, *network.Name, c.ID)
								ticker.Stop()
								return
							} else {
								Log.Warningf("Failed to create token for associated %s contract creation %s; %d errs: %s", *network.Name, c.ID, len(token.Errors), *stringOrNil(*token.Errors[0].Message))
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
	c.Errors = make([]*provide.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &provide.Error{
			Message: stringOrNil("Unable to associate contract with unspecified network"),
		})
	} else if transaction != nil && c.NetworkID != transaction.NetworkID {
		c.Errors = append(c.Errors, &provide.Error{
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
				o.Errors = append(o.Errors, &provide.Error{
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
	o.Errors = make([]*provide.Error, 0)
	if o.NetworkID == uuid.Nil {
		o.Errors = append(o.Errors, &provide.Error{
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
				t.Errors = append(t.Errors, &provide.Error{
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
	t.Errors = make([]*provide.Error, 0)
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("Unable to deploy token contract using unspecified network"),
		})
	} else {
		if contract != nil {
			if t.NetworkID != contract.NetworkID {
				t.Errors = append(t.Errors, &provide.Error{
					Message: stringOrNil("Token network did not match token contract network"),
				})
			}
			if t.Address == nil {
				t.Address = contract.Address
			} else if t.Address != nil && *t.Address != *contract.Address {
				t.Errors = append(t.Errors, &provide.Error{
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

// GetWallet - retrieve the associated transaction wallet
func (t *Transaction) GetWallet() (*Wallet, error) {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	if wallet == nil {
		return nil, fmt.Errorf("Failed to retrieve transaction wallet for tx: %s", t.ID)
	}
	return wallet, nil
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

func (t *Transaction) updateStatus(db *gorm.DB, status string, description *string) {
	t.Status = stringOrNil(status)
	t.Description = description
	result := db.Save(&t)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errors = append(t.Errors, &provide.Error{
				Message: stringOrNil(err.Error()),
			})
		}
	}
}

func (t *Transaction) attemptGasEstimationRecovery(err error) error {
	msg := err.Error()
	gasFailureStr := "not enough gas to cover minimal cost of the transaction (minimal: "
	if strings.Contains(msg, gasFailureStr) && strings.Contains(msg, "got: 0") { // HACK
		Log.Debugf("Attempting to recover from gas estimation failure with supplied gas of 0 for tx id: %s", t.ID)
		offset := strings.Index(msg, gasFailureStr) + len(gasFailureStr)
		length := strings.Index(msg[offset:], ",")
		minimalGas, err := strconv.ParseFloat(msg[offset:offset+length], 64)
		if err == nil {
			Log.Debugf("Resolved minimal gas of %v required to execute tx: %s", minimalGas, t.ID)
			params := t.ParseParams()
			params["gas"] = minimalGas
			t.setParams(params)
			return nil
		}
	}
	Log.Debugf("Failed to resolve minimal gas requirement for tx: %s; tx execution unrecoverable", t.ID)
	return err
}

func (t *Transaction) broadcast(db *gorm.DB, network *Network, wallet *Wallet) error {
	var err error

	if t.SignedTx == nil {
		return fmt.Errorf("Failed to broadcast %s tx using wallet: %s; tx not yet signed", *network.Name, wallet.ID)
	}

	if network.isEthereumNetwork() {
		if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
			err = provide.EVMBroadcastSignedTx(network.ID.String(), network.rpcURL(), signedTx)
		} else {
			err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
		}

		if err != nil {
			if t.attemptGasEstimationRecovery(err) == nil {
				err = t.sign(db, network, wallet)
				if err == nil {
					if signedTx, ok := t.SignedTx.(*types.Transaction); ok {
						err = provide.EVMBroadcastSignedTx(network.ID.String(), network.rpcURL(), signedTx)
					} else {
						err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", t.SignedTx)
					}
				}
			}
		}
	} else {
		err = fmt.Errorf("Unable to generate signed tx for unsupported network: %s", *network.Name)
	}

	if err != nil {
		Log.Warningf("Failed to broadcast %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil(err.Error()),
		})
		desc := err.Error()
		t.updateStatus(db, "failed", &desc)
	}

	return err
}

func (t *Transaction) sign(db *gorm.DB, network *Network, wallet *Wallet) error {
	var err error

	if network.isEthereumNetwork() {
		params := t.ParseParams()
		gas, gasOk := params["gas"].(float64)
		if !gasOk {
			gas = float64(0)
		}

		if wallet.PrivateKey != nil {
			privateKey, _ := decryptECDSAPrivateKey(*wallet.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
			_privateKey := hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
			t.SignedTx, t.Hash, err = provide.EVMSignTx(network.ID.String(), network.rpcURL(), wallet.Address, _privateKey, t.To, t.Data, t.Value.BigInt(), uint64(gas))
		} else {
			err = fmt.Errorf("Unable to sign tx; no private key for wallet: %s", wallet.ID)
		}
	} else {
		err = fmt.Errorf("Unable to generate signed tx for unsupported network: %s", *network.Name)
	}

	if err != nil {
		Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil(err.Error()),
		})
		desc := err.Error()
		t.updateStatus(db, "failed", &desc)
	}

	accessedAt := time.Now()
	go func() {
		wallet.AccessedAt = &accessedAt
		db.Save(wallet)
	}()

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
					receipt, err := provide.EVMGetTxReceipt(network.ID.String(), network.rpcURL(), *t.Hash, wallet.Address)
					if err != nil {
						Log.Debugf("Failed to fetch ethereum tx receipt with tx hash: %s; %s", *t.Hash, err.Error())
						if err == ethereum.NotFound {
							if time.Now().Sub(startedAt) >= receiptTickerTimeout {
								Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; timing out after %v", *t.Hash, receiptTickerTimeout)
								t.updateStatus(db, "failed", stringOrNil("failed to fetch tx receipt"))
								ticker.Stop()
								return
							}
						} else {
							if time.Now().Sub(startedAt) >= receiptTickerTimeout {
								Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; timing out after %v", *t.Hash, receiptTickerTimeout)
								t.Errors = append(t.Errors, &provide.Error{
									Message: stringOrNil(err.Error()),
								})
								t.updateStatus(db, "failed", stringOrNil(err.Error()))
								ticker.Stop()
								return
							}
						}
					} else {
						Log.Debugf("Fetched ethereum tx receipt for tx hash: %s", *t.Hash)
						ticker.Stop()

						traces, traceErr := provide.EVMTraceTx(network.ID.String(), network.rpcURL(), t.Hash)
						if traceErr != nil {
							Log.Warningf("Failed to fetch ethereum tx trace for tx hash: %s; %s", *t.Hash, traceErr.Error())
						}
						t.Response = &ContractExecutionResponse{
							Receipt:     receipt,
							Traces:      traces,
							Transaction: t,
						}
						t.Traces = traces

						t.updateStatus(db, "success", nil)
						t.handleEthereumTxReceipt(db, network, wallet, receipt)
						t.handleEthereumTxTraces(db, network, wallet, traces.(*provide.EthereumTxTraceResponse))
						return
					}
				}
			}
		}()
	}
}

func (t *Transaction) handleEthereumTxReceipt(db *gorm.DB, network *Network, wallet *Wallet, receipt *types.Receipt) {
	client, err := provide.EVMDialJsonRpc(network.ID.String(), network.rpcURL())
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
			contract.resolveTokenContract(db, network, wallet, client, receipt)
		} else {
			Log.Warningf("Failed to create contract for %s contract creation tx %s", *network.Name, *t.Hash)
		}
	}
}

func (t *Transaction) handleEthereumTxTraces(db *gorm.DB, network *Network, wallet *Wallet, traces *provide.EthereumTxTraceResponse) {
	contract := t.GetContract(db)
	if contract == nil {
		Log.Warningf("Failed to resolve contract as sender of contract-internal opcode tracing functionality")
		return
	}
	artifact := contract.CompiledArtifact()
	if artifact == nil {
		Log.Warningf("Failed to resolve compiled contract artifact required for contract-internal opcode tracing functionality")
		return
	}

	// client, err := provide.DialJsonRpc(network.ID.String(), network.rpcURL())
	// if err != nil {
	// 	Log.Warningf("Unable to handle ethereum tx traces; %s", err.Error())
	// 	return
	// }

	for _, result := range traces.Result {
		if result.Type != nil && *result.Type == "create" {
			contractAddr := result.Result.Address
			contractCode := result.Result.Code

			if contractAddr == nil || contractCode == nil {
				Log.Warningf("No contract address or bytecode resolved for contract-internal CREATE opcode; tx hash: %s", *t.Hash)
				continue
			}

			for name, dep := range artifact.Deps {
				dependency := dep.(map[string]interface{})
				fingerprint := dependency["fingerprint"].(string)
				if fingerprint == "" {
					continue
				}

				fingerprintSuffix := fmt.Sprintf("%s0029", fingerprint)
				if strings.HasSuffix(*contractCode, fingerprintSuffix) {
					params, _ := json.Marshal(dep)
					rawParams := json.RawMessage(params)
					internalContract := &Contract{
						ApplicationID: t.ApplicationID,
						NetworkID:     t.NetworkID,
						ContractID:    &contract.ID,
						TransactionID: &t.ID,
						Name:          stringOrNil(name),
						Address:       contractAddr,
						Params:        &rawParams,
					}
					if internalContract.Create() {
						Log.Debugf("Created contract %s for %s contract-internal tx: %s", internalContract.ID, *network.Name, *t.Hash)
						// FIXME-- contract.resolveTokenContract(db, network, wallet, client, receipt)
					} else {
						Log.Warningf("Failed to create contract for %s contract-internal creation tx %s", *network.Name, *t.Hash)
					}
					break
				}
			}
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
		t.Errors = append(t.Errors, &provide.Error{
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
				t.Errors = append(t.Errors, &provide.Error{
					Message: stringOrNil(err.Error()),
				})
			}
			return false
		}

		err = t.broadcast(db, network, wallet)
		if len(t.Errors) > 0 {
			return false
		}

		if !db.NewRecord(t) {
			if rowsAffected > 0 {
				if err != nil {
					desc := err.Error()
					t.updateStatus(db, "failed", &desc)
				} else {
					txReceiptMsg, _ := json.Marshal(t)
					natsConnection := getNatsStreamingConnection()
					natsConnection.Publish(natsTxReceiptSubject, txReceiptMsg)
				}
			}
			return rowsAffected > 0 && len(t.Errors) == 0
		}
	}
	return false
}

// GetContract - attempt to resolve the contract associated with the tx execution
func (t *Transaction) GetContract(db *gorm.DB) *Contract {
	var contract *Contract
	if t.To != nil {
		contract = &Contract{}
		db.Model(&Contract{}).Where("network_id = ? AND address = ?", t.NetworkID, t.To).Find(&contract)
	}
	return contract
}

// Validate a transaction for persistence
func (t *Transaction) Validate() bool {
	db := DatabaseConnection()
	var wallet *Wallet
	if t.WalletID != nil {
		wallet = &Wallet{}
		db.Model(t).Related(&wallet)
	}
	t.Errors = make([]*provide.Error, 0)
	if t.ApplicationID != nil && t.UserID != nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("only an application OR user identifier should be provided"),
		})
	} else if t.ApplicationID != nil && wallet != nil && wallet.ApplicationID != nil && *t.ApplicationID != *wallet.ApplicationID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("Unable to sign tx due to mismatched signing application"),
		})
	} else if t.UserID != nil && wallet != nil && wallet.UserID != nil && *t.UserID != *wallet.UserID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("Unable to sign tx due to mismatched signing user"),
		})
	}
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("Unable to broadcast tx on unspecified network"),
		})
	} else if wallet != nil && t.ApplicationID != nil && t.NetworkID != wallet.NetworkID {
		t.Errors = append(t.Errors, &provide.Error{
			Message: stringOrNil("Transaction network did not match wallet network in application context"),
		})
	}
	return len(t.Errors) == 0
}

// RefreshDetails populates transaction details which were not necessarily available upon broadcast, including network-specific metadata and VM execution tracing if applicable
func (t *Transaction) RefreshDetails() error {
	var err error
	network, _ := t.GetNetwork()
	if network.isEthereumNetwork() {
		t.Traces, err = provide.EVMTraceTx(network.ID.String(), network.rpcURL(), t.Hash)
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

func (w *Wallet) generate(db *gorm.DB, gpgPublicKey string) {
	network, _ := w.GetNetwork()

	if network == nil || network.ID == uuid.Nil {
		Log.Warningf("Unable to generate private key for wallet without an associated network")
		return
	}

	var encodedPrivateKey *string

	if network.isEthereumNetwork() {
		addr, privateKey, err := provide.EVMGenerateKeyPair()
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
				w.Errors = append(w.Errors, &provide.Error{
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
	w.Errors = make([]*provide.Error, 0)
	var network = &Network{}
	DatabaseConnection().Model(w).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: stringOrNil(fmt.Sprintf("invalid network association attempted with network id: %s", w.NetworkID.String())),
		})
	}
	if w.ApplicationID == nil && w.UserID == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: stringOrNil("no application or user identifier provided"),
		})
	} else if w.ApplicationID != nil && w.UserID != nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: stringOrNil("only an application OR user identifier should be provided"),
		})
	}
	var err error
	if w.PrivateKey != nil {
		if network.isEthereumNetwork() {
			_, err = decryptECDSAPrivateKey(*w.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
		}
	} else {
		w.Errors = append(w.Errors, &provide.Error{
			Message: stringOrNil("private key generation failed"),
		})
	}

	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &provide.Error{
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
		balance, err = provide.EVMGetNativeBalance(network.ID.String(), network.rpcURL(), w.Address)
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
		balance, err = provide.EVMGetTokenBalance(network.ID.String(), network.rpcURL(), *token.Address, w.Address, contractAbi)
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
