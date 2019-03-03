package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

const hostReachabilityTimeout = time.Minute * 5
const hostReachabilityInterval = time.Millisecond * 2500

var networkGenesisMutex = map[string]*sync.Mutex{}

const defaultWebappPort = 3000

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Network{})
	db.Model(&Network{}).AddIndex("idx_networks_application_id", "application_id")
	db.Model(&Network{}).AddIndex("idx_networks_network_id", "network_id")
	db.Model(&Network{}).AddIndex("idx_networks_user_id", "user_id")
	db.Model(&Network{}).AddIndex("idx_networks_cloneable", "cloneable")
	db.Model(&Network{}).AddIndex("idx_networks_enabled", "enabled")
	db.Model(&Network{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Network{}).AddForeignKey("sidechain_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Network{}).AddUniqueIndex("idx_chain_id", "chain_id")
}

type bootnodesInitialized string

func (err bootnodesInitialized) Error() string {
	return "network bootnodes initialized"
}

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
	Config        *json.RawMessage       `sql:"type:json not null" json:"config"`
	Stats         *provide.NetworkStatus `sql:"-" json:"stats"`
}

// config["network_id"] is unique
// config["protocol_id"] is "poa"

// Create and persist a new network
func (n *Network) Create() bool {
	if !n.Validate() {
		return false
	}

	db := dbconf.DatabaseConnection()

	if db.NewRecord(n) {
		n.setChainID()
		result := db.Create(&n)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				n.Errors = append(n.Errors, &provide.Error{
					Message: StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(n) {
			success := rowsAffected > 0
			if success {
				go n.resolveContracts(db)
			}
			return success
		}
	}
	return false
}

func (n *Network) String() string {
	str := ""
	errors := n.Model.Errors
	// move to Model.Errors interface
	for _, e := range errors {
		str = str + " " + *e.Message
	}

	chainID, _ := hexutil.DecodeBig(*n.ChainID)
	errorsStr := str
	name := *n.Name

	return "network: name=" + name + " chainID=" + chainID.String() + " errors=" + errorsStr
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
		nodeCfg := pending.ParseConfig()
		if env, envOk := nodeCfg["env"].(map[string]interface{}); envOk {
			if _, bootnodesOk := env["BOOTNODES"].(string); bootnodesOk {
				bootnodes = append(bootnodes, pending)
				err := new(bootnodesInitialized)
				return bootnodes, *err
			}
		}

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
							Name:          StringOrNil(contractName),
							Address:       StringOrNil(addr),
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
	db := dbconf.DatabaseConnection()
	db.Model(&n).Find(n)
}

// Update an existing network
func (n *Network) Update() bool {
	db := dbconf.DatabaseConnection()

	if !n.Validate() {
		return false
	}

	result := db.Save(&n)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			n.Errors = append(n.Errors, &provide.Error{
				Message: StringOrNil(err.Error()),
			})
		}
	}

	return len(n.Errors) == 0
}

// setConfig sets the network config in-memory
func (n *Network) setConfig(cfg map[string]interface{}) {
	n.Config = marshalConfig(cfg)
}

// setChainID is an internal method used to set a unique chainID for the network prior to its creation
func (n *Network) setChainID() {
	n.ChainID = StringOrNil(fmt.Sprintf("0x%x", time.Now().Unix()))
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

func (n *Network) getSecurityConfiguration(db *gorm.DB) map[string]interface{} {
	cloneableCfg, cloneableCfgOk := n.ParseConfig()["cloneable_cfg"].(map[string]interface{})
	if !cloneableCfgOk {
		return nil
	}
	securityCfg, securityCfgOk := cloneableCfg["_security"].(map[string]interface{})
	if !securityCfgOk {
		return nil
	}
	return securityCfg
}

// resolveAndBalanceJsonRpcAndWebsocketUrls updates the network's configured block
// JSON-RPC urls (i.e. web-based IDE), and enriches the network cfg; if no load
// balancer is provisioned for the account-region-type, a load balancer is provisioned
// prior to balancing the given node; FIXME-- refactor this
func (n *Network) resolveAndBalanceJSONRPCAndWebsocketURLs(db *gorm.DB, node *NetworkNode) {
	cfg := n.ParseConfig()
	nodeCfg := node.ParseConfig()

	Log.Debugf("Attempting to resolve and balance JSON-RPC and websocket urls for network with id: %s", n.ID)

	if node == nil {
		Log.Debugf("No network node provided to attempted resolution and load balancing of JSON-RPC/websocket URL for network with id: %s", n.ID)
		db.Where("network_id = ? AND status = 'running' AND role IN ('peer', 'full', 'validator', 'faucet')", n.ID).First(&node)
	}

	isLoadBalanced := n.isLoadBalanced(db, nodeCfg["region"].(string), "rpc")

	if node != nil && node.ID != uuid.Nil {
		var lb *LoadBalancer
		var err error

		balancerCfg := node.ParseConfig()
		region, _ := balancerCfg["region"].(string)

		if !isLoadBalanced {
			lbUUID, _ := uuid.NewV4()
			lbName := fmt.Sprintf("%s", lbUUID.String()[0:31])
			lb = &LoadBalancer{
				NetworkID: n.ID,
				Name:      StringOrNil(lbName),
				Region:    StringOrNil(region),
				Type:      StringOrNil("rpc"),
			}
			lb.setConfig(balancerCfg)
			if lb.Create() {
				Log.Debugf("Provisioned load balancer in region: %s; attempting to balance node: %s", region, lb.ID)
			} else {
				err = fmt.Errorf("Failed to provision load balancer in region: %s; %s", region, *lb.Errors[0].Message)
				Log.Warning(err.Error())
			}
			isLoadBalanced = n.isLoadBalanced(db, region, "rpc")
		} else {
			balancers, err := n.LoadBalancers(db, StringOrNil(region), StringOrNil("rpc"))
			if err != nil {
				Log.Warningf("Failed to retrieve rpc load balancers in region: %s; %s", region, err.Error())
			} else {
				if len(balancers) > 0 {
					Log.Debugf("Resolved %v rpc load balancers in region: %s", len(balancers), region)
					lb = balancers[0]
					Log.Debugf("Resolved load balancer with id: %s", lb.ID)
				}
			}
		}

		if isLoadBalanced {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": lb.ID.String(),
				"network_node_id":  node.ID.String(),
			})
			natsConnection := GetDefaultNatsStreamingConnection()
			natsConnection.Publish(natsLoadBalancerBalanceNodeSubject, msg)
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

// LoadBalancers returns the Network load balancers
func (n *Network) LoadBalancers(db *gorm.DB, region, balancerType *string) ([]*LoadBalancer, error) {
	balancers := make([]*LoadBalancer, 0)
	query := db.Where("network_id = ?", n.ID)
	if region != nil {
		query = query.Where("region = ?", region)
	}
	if balancerType != nil {
		query = query.Where("type = ?", balancerType)
	}
	query.Find(&balancers)
	return balancers, nil
}

func (n *Network) isLoadBalanced(db *gorm.DB, region, balancerType string) bool {
	balancers, err := n.LoadBalancers(db, StringOrNil(region), StringOrNil(balancerType))
	if err != nil {
		Log.Warningf("Failed to retrieve network load balancers; %s", err.Error())
		return false
	}
	return len(balancers) > 0
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

			isLoadBalanced := n.isLoadBalanced(db, nodeCfg["region"].(string), "explorer")

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

			isLoadBalanced := n.isLoadBalanced(db, nodeCfg["region"].(string), "explorer")

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

	if n.Config == nil {
		n.Errors = append(n.Errors, &provide.Error{StringOrNil("Config value should be present"), ptrToInt(10)})
	}

	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)

		if err == nil && len(config) == 0 {
			n.Errors = append(n.Errors, &provide.Error{StringOrNil("Config should not be empty"), ptrToInt(12)})
		}

		if err == nil && *n.Cloneable {
			if cfg, found := config["cloneable_cfg"]; found {
				if cfg_asserted, ok := cfg.(map[string]interface{}); ok {
					if _, ok := cfg_asserted["_security"]; !ok {
						n.Errors = append(n.Errors, &provide.Error{StringOrNil("Config _security value should be present for clonable network"), ptrToInt(11)})
					}
				}
			}
		}
	}
	// add error if Config is empty
	// add error if Clonable and Config[:_security] is empty
	// add error if Config is nil
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
	balancers, _ := n.LoadBalancers(dbconf.DatabaseConnection(), nil, StringOrNil("rpc"))
	if balancers != nil && len(balancers) > 0 {
		balancer := balancers[0] // FIXME-- loadbalance internally here; round-robin is naive-- better would be to factor in geography of end user and/or give weight to balanced regions with more nodes
		balancerCfg := balancer.ParseConfig()
		if url, urlOk := balancerCfg["json_rpc_url"].(string); urlOk {
			return url
		}
		// loadBalancedURL := fmt.Sprintf("wss://%s:%v", balancer.Host, balancer.Port)
	}
	if rpcURL, ok := cfg["json_rpc_url"].(string); ok {
		return rpcURL
	}
	return ""
}

func (n *Network) websocketURL() string {
	cfg := n.ParseConfig()
	balancers, _ := n.LoadBalancers(dbconf.DatabaseConnection(), nil, StringOrNil("websocket"))
	if balancers != nil && len(balancers) > 0 {
		balancer := balancers[0] // FIXME-- loadbalance internally here; round-robin is naive-- better would be to factor in geography of end user and/or give weight to balanced regions with more nodes
		balancerCfg := balancer.ParseConfig()
		if url, urlOk := balancerCfg["websocket_url"].(string); urlOk {
			return url
		}
		// loadBalancedURL := fmt.Sprintf("wss://%s:%v", balancer.Host, balancer.Port)
	}
	if websocketURL, ok := cfg["websocket_url"].(string); ok {
		return websocketURL
	}
	return ""
}

// InvokeJSONRPC method with given params
func (n *Network) InvokeJSONRPC(method string, params []interface{}) (map[string]interface{}, error) {
	if n.isBcoinNetwork() {
		cfg := n.ParseConfig()
		rpcAPIUser := cfg["rpc_api_user"].(string)
		rpcAPIKey := cfg["rpc_api_key"].(string)
		var resp map[string]interface{}
		err := provide.BcoinInvokeJsonRpcClient(n.ID.String(), n.rpcURL(), rpcAPIUser, rpcAPIKey, method, params, &resp)
		if err != nil {
			Log.Warningf("Failed to invoke JSON-RPC method %s with params: %s; %s", method, params, err.Error())
			return nil, err
		}
		result, _ := resp["result"].(map[string]interface{})
		return result, nil
	}

	return nil, fmt.Errorf("JSON-RPC invocation not supported by network %s", n.ID)
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
		networkCfg := n.ParseConfig()
		var rpcAPIUser string
		var rpcAPIKey string
		if rpcUser, rpcUserOk := networkCfg["rpc_api_user"].(string); rpcUserOk {
			rpcAPIUser = rpcUser
		}
		if rpcKey, rpcKeyOk := networkCfg["rpc_api_key"].(string); rpcKeyOk {
			rpcAPIKey = rpcKey
		}
		status, err = provide.BcoinGetNetworkStatus(n.ID.String(), n.rpcURL(), rpcAPIUser, rpcAPIKey)
	} else if n.isEthereumNetwork() {
		status, err = provide.EVMGetNetworkStatus(n.ID.String(), n.rpcURL())
	} else {
		Log.Warningf("Unable to determine status of unsupported network: %s", *n.Name)
	}
	return status, err
}

// NodeCount retrieves a count of platform-managed network nodes
func (n *Network) NodeCount() (count *uint64) {
	dbconf.DatabaseConnection().Model(&NetworkNode{}).Where("network_nodes.network_id = ?", n.ID).Count(&count)
	return count
}

// AvailablePeerCount retrieves a count of platform-managed network nodes which also have the 'peer' or 'full' role
// and currently are listed with a status of 'running'; this method does not currently check real-time availability
// of these peers-- it is assumed the are still available. FIXME?
func (n *Network) AvailablePeerCount() (count uint64) {
	dbconf.DatabaseConnection().Model(&NetworkNode{}).Where("network_nodes.network_id = ? AND network_nodes.status = 'running' AND network_nodes.role IN ('peer', 'full', 'validator', 'faucet')", n.ID).Count(&count)
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
	return StringOrNil(txt), err
}

// Bootnodes retrieves a list of network bootnodes
func (n *Network) Bootnodes() (nodes []*NetworkNode, err error) {
	query := dbconf.DatabaseConnection().Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// BootnodesCount returns a count of the number of bootnodes on the network
func (n *Network) BootnodesCount() (count uint64) {
	db := dbconf.DatabaseConnection()
	db.Model(&NetworkNode{}).Where("network_nodes.network_id = ? AND network_nodes.bootnode = true", n.ID).Count(&count)
	return count
}

// Nodes retrieves a list of network nodes
func (n *Network) Nodes() (nodes []*NetworkNode, err error) {
	query := dbconf.DatabaseConnection().Where("network_nodes.network_id = ?", n.ID)
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
