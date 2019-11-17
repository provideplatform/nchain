package network

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	redisutil "github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

const hostReachabilityTimeout = time.Minute * 5
const hostReachabilityInterval = time.Millisecond * 2500

const natsNetworkContractCreateInvocationSubject = "goldmine.contract.persist"

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

	rand.Seed(time.Now().Unix())
}

type bootnodesInitialized string

func (err bootnodesInitialized) Error() string {
	return "network bootnodes initialized"
}

// Network represents a blockchain network; the network could fall at any level of
// a heirarchy of blockchain networks
type Network struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	Name          *string          `sql:"not null" json:"name"`
	Description   *string          `json:"description"`
	IsProduction  *bool            `sql:"not null" json:"is_production"`
	Cloneable     *bool            `sql:"not null" json:"cloneable"`
	Enabled       *bool            `sql:"not null" json:"enabled"`
	ChainID       *string          `json:"chain_id"`                               // protocol-specific chain id
	SidechainID   *uuid.UUID       `sql:"type:uuid" json:"sidechain_id,omitempty"` // network id used as the transactional sidechain (or null)
	NetworkID     *uuid.UUID       `sql:"type:uuid" json:"network_id"`             // network id used as the parent
	Config        *json.RawMessage `sql:"type:json not null" json:"config,omitempty"`
	// Stats         *provide.NetworkStatus `sql:"-" json:"stats,omitempty"`
}

// NetworkListQuery returns a DB query configured to select columns suitable for a paginated API response
func NetworkListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("networks.id, networks.created_at, networks.application_id, networks.user_id, networks.name, networks.description, networks.cloneable, networks.enabled, networks.chain_id, networks.network_id, networks.sidechain_id, networks.config")
}

// StatsKey returns the network stats key for the given network id, which is guaranteed to be
// unique-per-network; the stats key represents the namespace where real-time stats for the
// network are cached
func StatsKey(networkID uuid.UUID) string {
	return fmt.Sprintf("network.%s.stats", networkID.String())
}

// Stats returns the network stats for the given network id without a network instance
func Stats(networkID uuid.UUID) (*provide.NetworkStatus, error) {
	statsKey := StatsKey(networkID)
	rawstats, err := redisutil.Get(statsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cached network stats from key: %s; %s", statsKey, err.Error())
	}

	stats := &provide.NetworkStatus{}
	json.Unmarshal([]byte(*rawstats), stats)
	return stats, nil
}

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
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(n) {
			success := rowsAffected > 0
			if success {
				n.resolveContracts(db)
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

func (n *Network) requireBootnodes(db *gorm.DB, pending *Node) ([]*Node, error) {
	mutex, mutexOk := networkGenesisMutex[pending.NetworkID.String()]
	if !mutexOk {
		mutex = &sync.Mutex{}
		networkGenesisMutex[pending.NetworkID.String()] = mutex
	}

	mutex.Lock()
	defer mutex.Unlock()

	count := n.BootnodesCount()
	bootnodes := make([]*Node, 0)

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
		common.Log.Debugf("Coerced network node into initial bootnode for network with id: %s", n.ID)
		return bootnodes, *err
	}

	bootnodes, err := n.Bootnodes()
	common.Log.Debugf("Resolved %d initial bootnode(s) for network with id: %s", len(bootnodes), n.ID)

	return bootnodes, err
}

func (n *Network) resolveContracts(db *gorm.DB) {
	cfg := n.ParseConfig()
	if n.IsEthereumNetwork() {
		chainspec, chainspecOk := cfg["chainspec"].(map[string]interface{})
		chainspecAbi, chainspecAbiOk := cfg["chainspec_abi"].(map[string]interface{})
		if chainspecOk && chainspecAbiOk {
			common.Log.Debugf("Resolved configuration for chainspec and ABI for network: %s; attempting to import contracts", n.ID)

			if accounts, accountsOk := chainspec["accounts"].(map[string]interface{}); accountsOk {
				addrs := make([]string, 0)
				for addr := range accounts {
					addrs = append(addrs, addr)
				}
				sort.Strings(addrs)

				for _, addr := range addrs {
					common.Log.Debugf("Processing chainspec account %s for network: %s", addr, n.ID)
					account := accounts[addr]

					_, constructorOk := account.(map[string]interface{})["constructor"].(string)
					abi, abiOk := chainspecAbi[addr].([]interface{})
					if constructorOk && abiOk {
						common.Log.Debugf("Chainspec account %s has a valid constructor and ABI for network: %s; attempting to import contract", addr, n.ID)

						contractName := fmt.Sprintf("Network Contract %s", addr)
						if name, ok := account.(map[string]interface{})["name"].(interface{}); ok {
							contractName = name.(string)
						}
						params := map[string]interface{}{
							"address":    addr,
							"name":       contractName,
							"network_id": n.ID,
							"abi":        abi,
						}

						payload, _ := json.Marshal(params)
						natsutil.NatsPublish(natsNetworkContractCreateInvocationSubject, payload)
					}
				}
			}
		}
	}
}

// setIsLoadBalanced just sets a hint inside the network config
func (n *Network) setIsLoadBalanced(db *gorm.DB, val bool) {
	cfg := n.ParseConfig()
	if val {
		delete(cfg, "json_rpc_url")
		delete(cfg, "websocket_url")
	}
	n.setConfig(cfg)
	db.Save(&n)
}

// Stats returns the network stats
func (n *Network) Stats() (*provide.NetworkStatus, error) {
	return Stats(n.ID)
}

// StatsKey returns a key, which is guaranteed to be unique-per-network, which
// represents the namespace where real-time stats for the network are cached
func (n *Network) StatsKey() string {
	return StatsKey(n.ID)
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
				Message: common.StringOrNil(err.Error()),
			})
		}
	}

	return len(n.Errors) == 0
}

// setConfig sets the network config in-memory
func (n *Network) setConfig(cfg map[string]interface{}) {
	n.Config = common.MarshalConfig(cfg)
}

// setChainID is an internal method used to set a unique chainID for the network prior to its creation
func (n *Network) setChainID() {
	n.ChainID = common.StringOrNil(fmt.Sprintf("0x%x", time.Now().Unix()))
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

// resolveAndBalanceExplorerUrls updates the network's configured block
// explorer urls (i.e. web-based IDE), and enriches the network cfg
func (n *Network) resolveAndBalanceExplorerUrls(db *gorm.DB, node *Node) {
	ticker := time.NewTicker(hostReachabilityInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			cfg := n.ParseConfig()
			nodeCfg := node.ParseConfig()

			isLoadBalanced := n.isLoadBalanced(db, common.StringOrNil(nodeCfg["region"].(string)), common.StringOrNil(loadBalancerTypeBlockExplorer))

			if time.Now().Sub(startedAt) >= hostReachabilityTimeout {
				common.Log.Warningf("Failed to resolve and balance explorer urls for network node: %s; timing out after %v", n.ID.String(), hostReachabilityTimeout)
				if !isLoadBalanced {
					cfg["block_explorer_url"] = nil
					cfgJSON, _ := json.Marshal(cfg)
					*n.Config = json.RawMessage(cfgJSON)
					db.Save(n)
				}
				ticker.Stop()
				return
			}

			common.Log.Debugf("Attempting to resolve and balance block explorer url for network node: %s", n.ID.String())

			var node = &Node{}
			db.Where("network_id = ? AND status = 'running' AND role IN ('explorer')", n.ID).First(&node)

			if node != nil && node.ID != uuid.Nil {
				if isLoadBalanced {
					common.Log.Warningf("Block explorer load balancer may contain unhealthy or undeployed nodes")
				} else {
					if node.reachableOnPort(defaultWebappPort) {
						common.Log.Debugf("Block explorer reachable via port %d; node id: %s", defaultWebappPort, n.ID)

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
					}

					common.Log.Debugf("Block explorer unreachable via webapp port; node id: %s", n.ID)
					cfg["block_explorer_url"] = nil
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

// resolveAndBalanceJsonRpcAndWebsocketUrls updates the network's configured
// JSON-RPC urls (i.e. web-based IDE), and enriches the network cfg; if no load
// balancer is provisioned for the account-region-type, a load balancer is provisioned
// prior to balancing the given node; FIXME-- refactor this
func (n *Network) resolveAndBalanceJSONRPCAndWebsocketURLs(db *gorm.DB, node *Node) {
	cfg := n.ParseConfig()
	nodeCfg := node.ParseConfig()

	common.Log.Debugf("Attempting to resolve and balance JSON-RPC and websocket urls for network with id: %s", n.ID)

	if node == nil {
		common.Log.Debugf("No network node provided to attempted resolution and load balancing of JSON-RPC/websocket URL for network with id: %s", n.ID)
		db.Where("network_id = ? AND status = 'running' AND role IN ('peer', 'full', 'validator', 'faucet')", n.ID).First(&node)
	}

	isLoadBalanced := n.isLoadBalanced(db, common.StringOrNil(nodeCfg["region"].(string)), common.StringOrNil(loadBalancerTypeRPC))

	if node != nil && node.ID != uuid.Nil {
		var lb *LoadBalancer
		var err error

		balancerCfg := node.privateConfig()
		region, _ := balancerCfg["region"].(string)

		if !isLoadBalanced {
			lbUUID, _ := uuid.NewV4()
			lbName := fmt.Sprintf("%s", lbUUID.String()[0:31])
			lb = &LoadBalancer{
				NetworkID: n.ID,
				Name:      common.StringOrNil(lbName),
				Region:    common.StringOrNil(region),
				Type:      common.StringOrNil(loadBalancerTypeRPC),
			}
			lb.setConfig(balancerCfg)
			if lb.Create() {
				common.Log.Debugf("Provisioned load balancer in region: %s; attempting to balance node: %s", region, lb.ID)
			} else {
				err = fmt.Errorf("Failed to provision load balancer in region: %s; %s", region, *lb.Errors[0].Message)
				common.Log.Warning(err.Error())
			}
			isLoadBalanced = n.isLoadBalanced(db, common.StringOrNil(region), common.StringOrNil(loadBalancerTypeRPC))
		} else {
			balancers, err := n.LoadBalancers(db, common.StringOrNil(region), common.StringOrNil(loadBalancerTypeRPC))
			if err != nil {
				common.Log.Warningf("Failed to retrieve rpc load balancers in region: %s; %s", region, err.Error())
			} else {
				if len(balancers) > 0 {
					common.Log.Debugf("Resolved %v rpc load balancers in region: %s", len(balancers), region)
					lb = balancers[0]
					common.Log.Debugf("Resolved load balancer with id: %s", lb.ID)
				}
			}
		}

		if isLoadBalanced {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": lb.ID.String(),
				"network_node_id":  node.ID.String(),
			})
			natsutil.NatsPublish(natsLoadBalancerBalanceNodeSubject, msg)
		} else {
			if reachable, port := node.reachableViaJSONRPC(); reachable {
				common.Log.Debugf("Node reachable via JSON-RPC port %d; node id: %s", port, n.ID)
				cfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *node.Host, port)
			} else {
				common.Log.Debugf("Node unreachable via JSON-RPC port; node id: %s", n.ID)
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
	}
}

// resolveAndBalanceIPFSURLs updates the network's configured IPFS rpc and gateway url,
// and enriches the network cfg; if no load balancer is provisioned for the account-region-type,
// a load balancer is provisioned prior to balancing the given node; FIXME-- refactor this
func (n *Network) resolveAndBalanceIPFSUrls(db *gorm.DB, node *Node) {
	nodeCfg := node.ParseConfig()

	common.Log.Debugf("Attempting to resolve and balance IPFS RPC and gateway urls for network with id: %s", n.ID)

	if node == nil {
		common.Log.Debugf("No network node provided to attempted resolution and load balancing of IPFS RPC/gateway URL for network with id: %s", n.ID)
		db.Where("network_id = ? AND status = 'running' AND role IN ('ipfs')", n.ID).First(&node)
	}

	isLoadBalanced := n.isLoadBalanced(db, common.StringOrNil(nodeCfg["region"].(string)), common.StringOrNil(loadBalancerTypeIPFS))

	if node != nil && node.ID != uuid.Nil {
		var lb *LoadBalancer
		var err error

		balancerCfg := node.privateConfig()
		region, _ := balancerCfg["region"].(string)

		if !isLoadBalanced {
			lbUUID, _ := uuid.NewV4()
			lbName := fmt.Sprintf("%s", lbUUID.String()[0:31])
			lb = &LoadBalancer{
				NetworkID: n.ID,
				Name:      common.StringOrNil(lbName),
				Region:    common.StringOrNil(region),
				Type:      common.StringOrNil(loadBalancerTypeRPC),
			}
			lb.setConfig(balancerCfg)
			if lb.Create() {
				common.Log.Debugf("Provisioned load balancer in region: %s; attempting to balance node: %s", region, lb.ID)
			} else {
				err = fmt.Errorf("Failed to provision load balancer in region: %s; %s", region, *lb.Errors[0].Message)
				common.Log.Warning(err.Error())
			}
			isLoadBalanced = n.isLoadBalanced(db, common.StringOrNil(region), common.StringOrNil(loadBalancerTypeIPFS))
		} else {
			balancers, err := n.LoadBalancers(db, common.StringOrNil(region), common.StringOrNil(loadBalancerTypeIPFS))
			if err != nil {
				common.Log.Warningf("Failed to retrieve IPFS load balancers in region: %s; %s", region, err.Error())
			} else {
				if len(balancers) > 0 {
					common.Log.Debugf("Resolved %v IPFS load balancers in region: %s", len(balancers), region)
					lb = balancers[0]
					common.Log.Debugf("Resolved load balancer with id: %s", lb.ID)
				}
			}
		}

		if isLoadBalanced {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": lb.ID.String(),
				"network_node_id":  node.ID.String(),
			})
			natsutil.NatsPublish(natsLoadBalancerBalanceNodeSubject, msg)
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

func (n *Network) isLoadBalanced(db *gorm.DB, region, balancerType *string) bool {
	balancers, err := n.LoadBalancers(db, region, balancerType)
	if err != nil {
		common.Log.Warningf("Failed to retrieve network load balancers; %s", err.Error())
		return false
	}
	return len(balancers) > 0
}

// Validate a network for persistence
func (n *Network) Validate() bool {
	n.Errors = make([]*provide.Error, 0)

	if n.Config == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("config object should be defined for network"),
			Status:  common.PtrToInt(10),
		})
	}

	if n.Name == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("name can't be nil"),
			Status:  common.PtrToInt(10),
		})
	}

	if n.Enabled == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("enabled flag can't be nil"),
			Status:  common.PtrToInt(10),
		})
	}

	if n.IsProduction == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("is_production flag can't be nil"),
			Status:  common.PtrToInt(10),
		})
	}

	if n.Cloneable == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("cloneable flag can't be nil"),
			Status:  common.PtrToInt(10),
		})
	}

	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)

		if err == nil && len(config) == 0 {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("config should not be empty"),
				Status:  common.PtrToInt(12),
			})
		}

		if err == nil && n.Cloneable != nil && *n.Cloneable {
			if cfg, cfgOk := config["cloneable_cfg"].(map[string]interface{}); cfgOk {
				if _, ok := cfg["security"]; !ok {
					n.Errors = append(n.Errors, &provide.Error{
						Message: common.StringOrNil("security object should be present for clonable network configuration"),
						Status:  common.PtrToInt(11),
					})
				}
			} else {
				n.Errors = append(n.Errors, &provide.Error{
					Message: common.StringOrNil("cloneable_cfg object should not be null on cloneable network configuration"),
					Status:  common.PtrToInt(11),
				})
			}
		}

		chainspec, chainspecOk := config["chainspec"]
		chainspecURL, chainspecURLOk := config["chainspec_url"].(string)
		if !chainspecOk && !chainspecURLOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chainspec_url or chainspec should be present on network configuration"),
				Status:  common.PtrToInt(11),
			})
		} else {
			if chainspecOk {
				if chainspec == nil || chainspec == "" {
					n.Errors = append(n.Errors, &provide.Error{
						Message: common.StringOrNil("chainspec object should not be empty on network configuration"),
						Status:  common.PtrToInt(11),
					})
				}
			}
			if chainspecURLOk {
				_, err := url.Parse(chainspecURL)
				if err != nil {
					n.Errors = append(n.Errors, &provide.Error{
						Message: common.StringOrNil("chainspec_url should be a valid URL if provided"),
						Status:  common.PtrToInt(11),
					})
				}
			}
		}

		blockExplorerURL, blockExplorerURLOk := config["block_explorer_url"].(string)
		if blockExplorerURLOk {
			_, err := url.Parse(blockExplorerURL)
			if err != nil {
				n.Errors = append(n.Errors, &provide.Error{
					Message: common.StringOrNil("block_explorer_url should be a valid URL if provided"),
					Status:  common.PtrToInt(11),
				})
			}
		}

		chain, chainOk := config["chain"]
		if !chainOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chain should not be nil"),
				Status:  common.PtrToInt(11),
			})
		} else if chain == nil || chain == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chain should not be empty"),
				Status:  common.PtrToInt(11),
			})
		}

		engineID, engineIDOk := config["engine_id"]
		if !engineIDOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("Config engine_id should not be nil"),
				Status:  common.PtrToInt(11)})
		} else if engineID == nil || engineID == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("Config engine_id should not be empty"),
				Status:  common.PtrToInt(11),
			})
		}

		nativeCurrency, nativeCurrencyOk := config["native_currency"]
		if !nativeCurrencyOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("native_currency should not be nil"),
				Status:  common.PtrToInt(11),
			})
		} else if nativeCurrency == nil || nativeCurrency == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("native_currency should not be nil"),
				Status:  common.PtrToInt(11),
			})
		}

		protocolID, protocolIDOk := config["protocol_id"]
		if !protocolIDOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("protocol_id should not be nil"),
				Status:  common.PtrToInt(11),
			})
		} else if protocolID == nil || protocolID == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("protocol_id should not be empty"),
				Status:  common.PtrToInt(11),
			})
		}
	}

	return len(n.Errors) == 0
}

// ParseConfig - parse the persistent network configuration JSON
func (n *Network) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal network config; %s", err.Error())
			return nil
		}
	}
	return config
}

// RPCURL retrieves a load-balanced RPC URL for the network
func (n *Network) RPCURL() string {
	cfg := n.ParseConfig()
	balancers, _ := n.LoadBalancers(dbconf.DatabaseConnection(), nil, common.StringOrNil(loadBalancerTypeRPC))
	if balancers != nil && len(balancers) > 0 {
		balancer := balancers[rand.Intn(len(balancers))] // FIXME-- better would be to factor in geography of end user and/or give weight to balanced regions with more nodes
		balancerCfg := balancer.ParseConfig()
		if url, urlOk := balancerCfg["json_rpc_url"].(string); urlOk {
			return url
		}
	}
	if rpcURL, ok := cfg["json_rpc_url"].(string); ok {
		return rpcURL
	}
	return ""
}

// WebsocketURL retrieves a load-balanced websocket URL for the network
func (n *Network) WebsocketURL() string {
	cfg := n.ParseConfig()
	balancers, _ := n.LoadBalancers(dbconf.DatabaseConnection(), nil, common.StringOrNil(loadBalancerTypeRPC))
	if balancers != nil && len(balancers) > 0 {
		balancer := balancers[rand.Intn(len(balancers))] // FIXME-- better would be to factor in geography of end user and/or give weight to balanced regions with more nodes
		balancerCfg := balancer.ParseConfig()
		if url, urlOk := balancerCfg["websocket_url"].(string); urlOk {
			return url
		}
	}
	if websocketURL, ok := cfg["websocket_url"].(string); ok {
		return websocketURL
	}
	return ""
}

// addPeer adds the given peer url to the network topology and notifies other peers of the new peer's existence
func (n *Network) addPeer(peerURL string) error {
	// FIXME: batch this so networks with lots of nodes still perform well
	nodes, err := n.Nodes()
	if err != nil {
		common.Log.Warningf("Failed to retrieve network nodes for broadcasting peer url addition %s; %s", peerURL, err.Error())
	}

	common.Log.Debugf("Attempting to broadcast peer url %s for inclusion on network %s by %d nodes", peerURL, n.ID, len(nodes))
	for _, node := range nodes {
		params := map[string]interface{}{
			"network_node_id": node.ID,
			"peer_url":        peerURL,
		}
		payload, _ := json.Marshal(params)
		_, err := natsutil.NatsPublishAsync(natsAddNodePeerSubject, payload)
		if err != nil {
			common.Log.Warningf("Failed to add peer %s to network: %s; %s", peerURL, n.ID, err.Error())
			return err
		}
	}
	common.Log.Debugf("Broadcast peer url %s for inclusion on network %s by %d nodes", peerURL, n.ID, len(nodes))
	return nil
}

// removePeer removes the given peer url to the network topology and notifies other peers of the new peer's existence
func (n *Network) removePeer(peerURL string) error {
	// FIXME: batch this so networks with lots of nodes still perform well
	nodes, err := n.Nodes()
	if err != nil {
		common.Log.Warningf("Failed to retrieve network nodes for broadcasting peer url removal %s; %s", peerURL, err.Error())
	}

	common.Log.Debugf("Attempting to broadcast peer url %s for removal on network %s by %d nodes", peerURL, n.ID, len(nodes))
	for _, node := range nodes {
		params := map[string]interface{}{
			"network_node_id": node.ID,
			"peer_url":        peerURL,
		}
		payload, _ := json.Marshal(params)
		_, err := natsutil.NatsPublishAsync(natsRemoveNodePeerSubject, payload)
		if err != nil {
			common.Log.Warningf("Failed to remove peer %s to network: %s; %s", peerURL, n.ID, err.Error())
			return err
		}
	}
	common.Log.Debugf("Broadcast peer url %s for removal from network %s by %d nodes", peerURL, n.ID, len(nodes))
	return nil
}

// InvokeJSONRPC method with given params
func (n *Network) InvokeJSONRPC(method string, params []interface{}) (map[string]interface{}, error) {
	if n.IsBcoinNetwork() {
		cfg := n.ParseConfig()
		rpcAPIUser := cfg["rpc_api_user"].(string)
		rpcAPIKey := cfg["rpc_api_key"].(string)
		var resp map[string]interface{}
		err := provide.BcoinInvokeJsonRpcClient(n.ID.String(), n.RPCURL(), rpcAPIUser, rpcAPIKey, method, params, &resp)
		if err != nil {
			common.Log.Warningf("Failed to invoke JSON-RPC method %s with params: %s; %s", method, params, err.Error())
			return nil, err
		}
		result, _ := resp["result"].(map[string]interface{})
		return result, nil
	}

	return nil, fmt.Errorf("JSON-RPC invocation not supported by network %s", n.ID)
}

// NodeCount retrieves a count of platform-managed network nodes
func (n *Network) NodeCount() (count *uint64) {
	dbconf.DatabaseConnection().Model(&Node{}).Where("nodes.network_id = ?", n.ID).Count(&count)
	return count
}

// AvailablePeerCount retrieves a count of platform-managed network nodes which also have the 'peer' or 'full' role
// and currently are listed with a status of 'running'; this method does not currently check real-time availability
// of these peers-- it is assumed the are still available. FIXME?
func (n *Network) AvailablePeerCount() (count uint64) {
	dbconf.DatabaseConnection().Model(&Node{}).Where("nodes.network_id = ? AND nodes.status = 'running' AND nodes.role IN ('peer', 'full', 'validator', 'faucet')", n.ID).Count(&count)
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

	common.Log.Debugf("Resolved bootnodes environment variable for network with id: %s; bootnodes: %s", n.ID, txt)
	return common.StringOrNil(txt), err
}

// Bootnodes retrieves a list of network bootnodes
func (n *Network) Bootnodes() (nodes []*Node, err error) {
	query := dbconf.DatabaseConnection().Where("nodes.network_id = ? AND nodes.bootnode = true", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// BootnodesCount returns a count of the number of bootnodes on the network
func (n *Network) BootnodesCount() (count uint64) {
	db := dbconf.DatabaseConnection()
	db.Model(&Node{}).Where("nodes.network_id = ? AND nodes.bootnode = true", n.ID).Count(&count)
	return count
}

// Nodes retrieves a list of network nodes; FIXME: support pagination
func (n *Network) Nodes() (nodes []*Node, err error) {
	query := dbconf.DatabaseConnection().Where("nodes.network_id = ?", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// IsBcoinNetwork returns true if the network is bcoin-based
func (n *Network) IsBcoinNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isBcoinNetwork, ok := cfg["is_bcoin_network"].(bool); ok {
			return isBcoinNetwork
		}
	}
	return false
}

// IsEthereumNetwork returns true if the network is EVM-based
func (n *Network) IsEthereumNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg["is_ethereum_network"].(bool); ok {
			return isEthereumNetwork
		}
	}
	return false
}

func (n *Network) IsHandshakeNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isHandshakeNetwork, ok := cfg["is_handshake_network"].(bool); ok {
			return isHandshakeNetwork
		}
	}
	return false
}

func (n *Network) IsLcoinNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isLcoinNetwork, ok := cfg["is_lcoin_network"].(bool); ok {
			return isLcoinNetwork
		}
	}
	return false
}

func (n *Network) IsQuorumNetwork() bool {
	// if n.Name != nil && strings.HasPrefix(strings.ToLower(*n.Name), "eth") {
	// 	return true
	// }

	cfg := n.ParseConfig()
	if cfg != nil {
		if isQuorumNetwork, ok := cfg["is_quorum_network"].(bool); ok {
			return isQuorumNetwork
		}
	}
	return false
}
