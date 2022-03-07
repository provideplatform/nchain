package network

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network/p2p"
	provide "github.com/provideplatform/provide-go/api"
	c2 "github.com/provideplatform/provide-go/api/c2"
	provideapi "github.com/provideplatform/provide-go/api/nchain"
	providecrypto "github.com/provideplatform/provide-go/crypto"
)

const defaultWebappPort = 3000
const hostReachabilityTimeout = time.Minute * 5
const hostReachabilityInterval = time.Millisecond * 2500
const networkStateGenesis = "genesis"
const natsNetworkContractCreateInvocationSubject = "nchain.contract.persist"

const loadBalancerTypeRPC = "rpc"
const loadBalancerTypeIPFS = "ipfs"

const networkConfigBootnodes = "bootnodes"
const networkConfigChain = "chain"
const networkConfigChainspec = "chainspec"
const networkConfigChainspecURL = "chainspec_url"
const networkConfigChainspecABI = "chainspec_abi"
const networkConfigChainspecABIURL = "chainspec_abi_url"
const networkConfigEnv = "env"
const networkConfigJSONRPCURL = "json_rpc_url"
const networkConfigJSONRPCPort = "json_rpc_port"
const networkConfigNativeCurrency = "native_currency"
const networkConfigNetworkID = "network_id"
const networkConfigPlatform = "platform"
const networkConfigRPCAPIUser = "rpc_api_user"
const networkConfigRPCAPIKey = "rpc_api_key"
const networkConfigWebsocketURL = "websocket_url"
const networkConfigWebsocketPort = "websocket_port"
const networkConfigIsBaseledgerNetwork = "is_baseledger_network"
const networkConfigIsBcoinNetwork = "is_bcoin_network"
const networkConfigIsEthereumNetwork = "is_ethereum_network"
const networkConfigIsHandshakeNetwork = "is_handshake_network"
const networkConfigIsHyperledgerBesuNetwork = "is_hyperledger_besu_network"
const networkConfigIsHyperledgerFabricNetwork = "is_hyperledger_fabric_network"
const networkConfigIsQuorumNetwork = "is_quorum_network"

const networkConfigEnvBootnodes = "BOOTNODES"
const networkConfigEnvClient = "CLIENT"
const networkConfigEnvPeerSet = "PEER_SET"

type bootnodesInitialized struct{}

func (err *bootnodesInitialized) Error() string {
	return "network bootnodes initialized"
}

// Network represents a blockchain network; the network could fall at any level of
// a heirarchy of blockchain networks
type Network struct {
	provide.Model
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id,omitempty"`
	UserID          *uuid.UUID       `sql:"type:uuid" json:"user_id,omitempty"`
	Name            *string          `sql:"not null" json:"name"`
	Description     *string          `json:"description"`
	IsProduction    *bool            `sql:"not null" json:"-"` // deprecated
	Layer2          *bool            `sql:"not null" json:"layer2,omitempty"`
	Cloneable       *bool            `sql:"not null" json:"-"` // deprecated
	Enabled         *bool            `sql:"not null" json:"enabled"`
	ChainID         *string          `json:"chain_id"`                               // protocol-specific chain id
	SidechainID     *uuid.UUID       `sql:"type:uuid" json:"sidechain_id,omitempty"` // network id used as the transactional sidechain (or null)
	NetworkID       *uuid.UUID       `sql:"type:uuid" json:"network_id,omitempty"`   // network id used as the parent
	Config          *json.RawMessage `sql:"type:json not null" json:"config,omitempty"`
	EncryptedConfig *string          `sql:"-" json:"-"`

	// Stats         *provideapi.NetworkStatus `sql:"-" json:"stats,omitempty"`
}

// ListQuery returns a DB query configured to select columns suitable for a paginated API response
func ListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("networks.id, networks.created_at, networks.application_id, networks.user_id, networks.name, networks.description, networks.chain_id, networks.network_id, networks.sidechain_id, networks.config")
}

// MutexKey returns a key for the given network id, which is guaranteed to be
// unique-per-network, which represents the distributed lock for the network
func MutexKey(networkID uuid.UUID) string {
	return fmt.Sprintf("network.%s.mutex", networkID.String())
}

// StatsKey returns the network stats key for the given network id, which is guaranteed to be
// unique-per-network; the stats key represents the namespace where real-time stats for the
// network are cached
func StatsKey(networkID uuid.UUID) string {
	return fmt.Sprintf("network.%s.stats", networkID.String())
}

// StatusKey returns the network stats key for the given network id, which is guaranteed to be
// unique-per-network, which represents the namespace where real-time stats/status updates for
// the network are published (i.e., via NATS)
func StatusKey(networkID uuid.UUID) string {
	return fmt.Sprintf("network.%s.status", networkID.String())
}

// Stats returns the network stats for the given network id without a network instance
func Stats(networkID uuid.UUID) (*provideapi.NetworkStatus, error) {
	statsKey := StatsKey(networkID)
	rawstats, err := redisutil.Get(statsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cached network stats from key: %s; %s", statsKey, err.Error())
	}

	stats := &provideapi.NetworkStatus{}
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

func (n *Network) DecryptedConfig() (map[string]interface{}, error) {
	decryptedParams := map[string]interface{}{}
	if n.EncryptedConfig != nil {
		encryptedConfigJSON, err := pgputil.PGPPubDecrypt([]byte(*n.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to decrypt encrypted network config; %s", err.Error())
			return decryptedParams, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &decryptedParams)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted network config; %s", err.Error())
			return decryptedParams, err
		}
	}
	return decryptedParams, nil
}

func (n *Network) encryptConfig() bool {
	if n.EncryptedConfig != nil {
		encryptedConfig, err := pgputil.PGPPubEncrypt([]byte(*n.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to encrypt network config; %s", err.Error())
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
			return false
		}
		n.EncryptedConfig = common.StringOrNil(string(encryptedConfig))
	}
	return true
}

func (n *Network) IsPublic() bool {
	if n.IsEthereumNetwork() {
		switch n.ID.String() {
		case "deca2436-21ba-4ff5-b225-ad1b0b2f5c59":
			return true
		case "07102258-5e49-480e-86af-6d0c3260827d":
			return true
		case "66d44f30-9092-4182-a3c4-bc02736d6ae5":
			return true
		case "8d31bf48-df6b-4a71-9d7c-3cb291111e27":
			return true
		case "1b16996e-3595-4985-816c-043345d22f8c":
			return true
		case "d186de3a-48e9-4d99-8e60-adb98ae87a0c":
			return true
		default:
			return false
		}
	}

	return false
}

func (n *Network) PaymentsNetworkName() *string {
	if n.IsEthereumNetwork() {
		switch n.ID.String() {
		case "deca2436-21ba-4ff5-b225-ad1b0b2f5c59":
			return common.StringOrNil("mainnet")
		case "07102258-5e49-480e-86af-6d0c3260827d":
			return common.StringOrNil("rinkeby")
		case "66d44f30-9092-4182-a3c4-bc02736d6ae5":
			return common.StringOrNil("ropsten")
		case "8d31bf48-df6b-4a71-9d7c-3cb291111e27":
			return common.StringOrNil("kovan")
		case "1b16996e-3595-4985-816c-043345d22f8c":
			return common.StringOrNil("goerli")
		case "d186de3a-48e9-4d99-8e60-adb98ae87a0c":
			return common.StringOrNil("bsn-ope")
		default:
			return nil
		}
	}

	return nil
}

func (n *Network) SetEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	n.EncryptedConfig = &_paramsJSON
	n.encryptConfig()
}

func (n *Network) SanitizeConfig() {
	cfg := n.ParseConfig()

	encryptedCfg, err := n.DecryptedConfig()
	if err != nil {
		encryptedCfg = map[string]interface{}{}
	}

	n.SetConfig(cfg)
	n.SetEncryptedConfig(encryptedCfg)
}

func (n *Network) requireBootnodes(db *gorm.DB, pending *Node) ([]*Node, error) {
	bootnodes := make([]*Node, 0)
	var err error

	redisutil.WithRedlock(n.MutexKey(), func() error {
		count := n.BootnodesCount()
		if count == 0 {
			common.Log.Debugf("Attempting to resolve bootnodes for network: %s", n.ID)
			nodeCfg := pending.ParseConfig()
			if env, envOk := nodeCfg[networkConfigEnv].(map[string]interface{}); envOk {
				_, bootnodesOk := env[networkConfigEnvBootnodes].(string)
				_, peersetOk := env[networkConfigEnvPeerSet].(string)
				if bootnodesOk || peersetOk {
					bootnodes = append(bootnodes, pending)
					err = new(bootnodesInitialized)
					return err
				}
			}

			pending.Bootnode = true
			pending.updateStatus(db, nodeStatusGenesis, nil)
			bootnodes = append(bootnodes, pending)
			err = new(bootnodesInitialized)
			common.Log.Debugf("Coerced network node into initial bootnode for network with id: %s", n.ID)
			return err
		}

		cfg := n.ParseConfig()
		if cfgBootnodes, cfgBootnodesOk := cfg[networkConfigBootnodes].([]interface{}); cfgBootnodesOk {
			if len(cfgBootnodes) > 0 {
				for _, peerURL := range cfgBootnodes {
					tmpnode := &Node{}
					tmpnode.SetConfig(map[string]interface{}{
						"peer_url": peerURL,
					})
					bootnodes = append(bootnodes, tmpnode)
				}
			}
		}

		// bootnodes, err = n.Bootnodes()
		// if err != nil {
		// 	common.Log.Warningf("Failed to resolve bootnodes for network: %s", n.ID)
		// 	return err
		// }

		// if len(bootnodes) == 0 {
		// 	cfg := n.ParseConfig()
		// 	if cfgBootnodes, cfgBootnodesOk := cfg[networkConfigBootnodes].([]interface{}); cfgBootnodesOk {
		// 		if len(cfgBootnodes) > 0 {
		// 			for _, peerURL := range cfgBootnodes {
		// 				tmpnode := &Node{}
		// 				tmpnode.SetConfig(map[string]interface{}{
		// 					"peer_url": peerURL,
		// 				})
		// 				bootnodes = append(bootnodes, tmpnode)
		// 			}
		// 		}
		// 	}
		// }

		return nil
	})

	common.Log.Debugf("Resolved %d initial bootnode(s) for network with id: %s", len(bootnodes), n.ID)
	return bootnodes, err
}

func (n *Network) resolveContracts(db *gorm.DB) {
	cfg := n.ParseConfig()
	if n.IsEthereumNetwork() {
		chainspec, chainspecOk := cfg[networkConfigChainspec].(map[string]interface{})
		chainspecAbi, chainspecAbiOk := cfg[networkConfigChainspecABI].(map[string]interface{})
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
						natsutil.NatsJetstreamPublish(natsNetworkContractCreateInvocationSubject, payload)
					}
				}
			}
		}
	}
}

// Stats returns the network stats
func (n *Network) Stats() (*provideapi.NetworkStatus, error) {
	return Stats(n.ID)
}

// MutexKey returns a key, which is guaranteed to be unique-per-network, which
// represents the distributed lock for the network
func (n *Network) MutexKey() string {
	return MutexKey(n.ID)
}

// StatsKey returns a key, which is guaranteed to be unique-per-network, which
// represents the namespace where real-time stats for the network are cached
func (n *Network) StatsKey() string {
	return StatsKey(n.ID)
}

// StatusKey returns a key, which is guaranteed to be unique-per-network, which
// represents the namespace where real-time stats/status updates for the network
// are published (i.e., via NATS)
func (n *Network) StatusKey() string {
	return StatusKey(n.ID)
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

// SetConfig sets the network config in-memory
func (n *Network) SetConfig(cfg map[string]interface{}) {
	// FIXME-- use mutex?
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
				cfg[networkConfigNetworkID] = networkID.Uint64()
				if chainspec, chainspecOk := cfg[networkConfigChainspec].(map[string]interface{}); chainspecOk {
					// FIXME -- delegate this to p2p client API impl...
					if params, paramsOk := chainspec["params"].(map[string]interface{}); paramsOk {
						params["chainID"] = n.ChainID
						params["networkID"] = n.ChainID
					}
				}
				n.SetConfig(cfg)
			}
		}
	}
}

// LoadBalancers returns the Network load balancers
func (n *Network) LoadBalancers(db *gorm.DB, region, balancerType *string) ([]*c2.LoadBalancer, error) {
	balancers := make([]*c2.LoadBalancer, 0)
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

// Validate a network for persistence
func (n *Network) Validate() bool {
	n.Errors = make([]*provide.Error, 0)

	if n.Config == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("config object should be defined for network"),
		})
	}

	if n.Name == nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("name can't be nil"),
		})
	}

	if n.Enabled == nil {
		enabled := true
		n.Enabled = &enabled
	}

	if n.IsProduction == nil {
		isProduction := false
		n.IsProduction = &isProduction
	}

	if n.Cloneable == nil {
		isCloneable := false
		n.Cloneable = &isCloneable
	}

	config := map[string]interface{}{}

	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("error parsing config"),
			})
		}

		if err == nil && len(config) == 0 {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("config should not be empty"),
			})
		}

		chainspec, chainspecOk := config[networkConfigChainspec]
		chainspecURL, chainspecURLOk := config[networkConfigChainspecURL].(string)
		if !chainspecOk && !chainspecURLOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chainspec or chainspec_url should be present in network configuration"),
			})
		} else if chainspecOk && chainspecURLOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("specify chainspec ar chainspec_url in network configuration; not both"),
			})
		} else {
			if chainspecOk {
				if chainspec == nil || chainspec == "" {
					n.Errors = append(n.Errors, &provide.Error{
						Message: common.StringOrNil("chainspec object should not be empty"),
					})
				}
			}
			if chainspecURLOk {
				_, err := url.Parse(chainspecURL)
				if err != nil {
					n.Errors = append(n.Errors, &provide.Error{
						Message: common.StringOrNil("chainspec_url should be a valid URL if provided"),
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
				})
			}
		}

		chain, chainOk := config[networkConfigChain]
		if !chainOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chain should not be nil"),
			})
		} else if chain == nil || chain == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("chain should not be empty"),
			})
		}

		nativeCurrency, nativeCurrencyOk := config[networkConfigNativeCurrency]
		if !nativeCurrencyOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("native_currency should not be nil"),
			})
		} else if nativeCurrency == nil || nativeCurrency == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("native_currency should not be nil"),
			})
		}

		platform, platformOk := config[networkConfigPlatform]
		if !platformOk {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("platform should not be nil"),
			})
		} else if platform == nil || platform == "" {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil("platform should not be empty"),
			})
		}

		_, isBcoinNetworkOk := config[networkConfigIsBcoinNetwork].(bool)
		_, isEthereumNetworkOk := config[networkConfigIsEthereumNetwork].(bool)
		_, isHandshakeNetworkOk := config[networkConfigIsHandshakeNetwork].(bool)
		_, isHyperLedgerBesuNetworkOk := config[networkConfigIsHyperledgerBesuNetwork].(bool)
		_, isHyperLedgerFabricNetworkOk := config[networkConfigIsHyperledgerFabricNetwork].(bool)
		_, isQuorumNetworkOk := config[networkConfigIsQuorumNetwork].(bool)

		if !isBcoinNetworkOk && platform != nil && platform == p2p.PlatformBcoin {
			config[networkConfigIsBcoinNetwork] = true
		} else if !isEthereumNetworkOk && platform != nil && platform == p2p.PlatformEVM {
			config[networkConfigIsEthereumNetwork] = true
		} else if !isHandshakeNetworkOk && platform != nil && platform == p2p.PlatformHandshake {
			config[networkConfigIsHandshakeNetwork] = true
		} else if !isHyperLedgerBesuNetworkOk && platform != nil && platform == p2p.PlatformHyperledgerBesu {
			config[networkConfigIsHyperledgerBesuNetwork] = true
		} else if !isHyperLedgerFabricNetworkOk && platform != nil && platform == p2p.PlatformHyperledgerFabric {
			config[networkConfigIsHyperledgerFabricNetwork] = true
		} else if !isQuorumNetworkOk && platform != nil && platform == p2p.PlatformQuorum {
			config[networkConfigIsEthereumNetwork] = true
			config[networkConfigIsQuorumNetwork] = true
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
		balancerCfg := balancer.Config
		balancerDNSName := balancer.Host
		if balancerDNSName != nil {
			if port, portOk := balancerCfg[networkConfigJSONRPCPort].(float64); portOk {
				return fmt.Sprintf("https://%s:%v", *balancerDNSName, port)
			}
		}
		if url, urlOk := balancerCfg[networkConfigJSONRPCURL].(string); urlOk {
			return url
		}
	}
	if rpcURL, ok := cfg[networkConfigJSONRPCURL].(string); ok {
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
		balancerCfg := balancer.Config
		balancerDNSName := balancer.Host
		if balancerDNSName != nil {
			if port, portOk := balancerCfg[networkConfigWebsocketPort].(float64); portOk {
				return fmt.Sprintf("wss://%s:%v", *balancerDNSName, port)
			}
		}
		if url, urlOk := balancerCfg[networkConfigWebsocketURL].(string); urlOk {
			return url
		}
	}
	if websocketURL, ok := cfg[networkConfigWebsocketURL].(string); ok {
		return websocketURL
	}
	return ""
}

// addPeer adds the given peer url to the network topology and notifies other peers of the new peer's existence
func (n *Network) addPeer(peerURL string) error {
	// FIXME: batch this so networks with lots of nodes still perform well
	nodes, err := n.ReachableNodes()
	if err != nil {
		common.Log.Warningf("Failed to retrieve network nodes for broadcasting peer url addition %s; %s", peerURL, err.Error())
	}

	common.Log.Debugf("Attempting to broadcast peer url %s for inclusion on network %s by %d nodes", peerURL, n.ID, len(nodes))
	for _, node := range nodes {
		nodePeerURL := node.peerURL()
		if nodePeerURL != nil && *nodePeerURL != peerURL {
			params := map[string]interface{}{
				"node_id":  node.ID,
				"peer_url": peerURL,
			}
			payload, _ := json.Marshal(params)
			_, err := natsutil.NatsJetstreamPublishAsync(natsAddNodePeerSubject, payload)
			if err != nil {
				common.Log.Warningf("Failed to add peer %s to network: %s; %s", peerURL, n.ID, err.Error())
				return err
			}
		} else if nodePeerURL != nil && *nodePeerURL == peerURL {
			common.Log.Debugf("Not broadcasting peer url to matching node: %s; peer url: %s", node.ID, peerURL)
		}
	}
	common.Log.Debugf("Broadcast peer url %s for inclusion on network %s by %d nodes", peerURL, n.ID, len(nodes))
	return nil
}

// removePeer removes the given peer url to the network topology and notifies other peers of the new peer's existence
func (n *Network) removePeer(peerURL string) error {
	// FIXME: batch this so networks with lots of nodes still perform well
	nodes, err := n.ReachableNodes()
	if err != nil {
		common.Log.Warningf("Failed to retrieve network nodes for broadcasting peer url removal %s; %s", peerURL, err.Error())
	}

	common.Log.Debugf("Attempting to broadcast peer url %s for removal on network %s by %d nodes", peerURL, n.ID, len(nodes))
	for _, node := range nodes {
		params := map[string]interface{}{
			"node_id":  node.ID,
			"peer_url": peerURL,
		}
		payload, _ := json.Marshal(params)
		_, err := natsutil.NatsJetstreamPublishAsync(natsRemoveNodePeerSubject, payload)
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
		rpcAPIUser := cfg[networkConfigRPCAPIUser].(string)
		rpcAPIKey := cfg[networkConfigRPCAPIKey].(string)
		var resp map[string]interface{}
		err := providecrypto.BcoinInvokeJsonRpcClient(n.ID.String(), n.RPCURL(), rpcAPIUser, rpcAPIKey, method, params, &resp)
		if err != nil {
			common.Log.Warningf("Failed to invoke JSON-RPC method %s with params: %s; %s", method, params, err.Error())
			return nil, err
		}
		result, _ := resp["result"].(map[string]interface{})
		return result, nil
	}

	return nil, fmt.Errorf("JSON-RPC invocation not supported by network %s", n.ID)
}

// BootnodesTxt retrieves the current bootnodes string for the network; this value can be used
// to set peer/bootnodes list from which new network nodes are initialized
func (n *Network) BootnodesTxt() (*string, error) {
	peerURLs := make([]string, 0)

	cfg := n.ParseConfig()
	if cfgBootnodes, cfgBootnodesOk := cfg[networkConfigBootnodes].([]interface{}); cfgBootnodesOk {
		for i := range cfgBootnodes {
			peerURL := cfgBootnodes[i].(string)
			peerURLs = append(peerURLs, peerURL)
		}
	}

	bootnodes, err := n.Bootnodes()
	if err != nil {
		return nil, err
	}

	for i := range bootnodes {
		bootnode := bootnodes[i]
		peerURL := bootnode.peerURL()
		if peerURL != nil {
			peerURLs = append(peerURLs, *peerURL)
		}
	}

	if len(peerURLs) == 0 {
		return nil, fmt.Errorf("No bootnodes resolved for network with id: %s", n.ID)
	}

	var txt *string

	p2pAPI, err := n.P2PAPIClient()
	if err == nil {
		txt = common.StringOrNil(p2pAPI.FormatBootnodes(peerURLs))
	} else {
		txt = common.StringOrNil(strings.Join(peerURLs, ","))
	}

	common.Log.Debugf("Resolved bootnodes environment variable for network with id: %s; bootnodes: %s", n.ID, *txt)

	return txt, err
}

// AddBootnode is a mutually-exclusive operation that appends a peer url to the network bootnodes config
func (n *Network) AddBootnode(db *gorm.DB, peerURL string) error {
	return redisutil.WithRedlock(n.MutexKey(), func() error {
		cfg := n.ParseConfig()
		peers := make([]string, 0)
		if bootnodes, bootnodesOk := cfg["bootnodes"].([]string); bootnodesOk {
			copy(peers, bootnodes)
		}

		cfg["bootnodes"] = append(peers, peerURL)
		n.SetConfig(cfg)
		db.Save(&n)

		return nil
	})
}

// Bootnodes retrieves a list of network bootnodes
func (n *Network) Bootnodes() (nodes []*Node, err error) {
	query := dbconf.DatabaseConnection().Where("nodes.network_id = ? AND nodes.bootnode = true AND nodes.status IN (?, ?, ?)", n.ID, nodeStatusRunning, nodeStatusGenesis, nodeStatusPeering)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// BootnodesCount returns a count of the number of bootnodes on the network
func (n *Network) BootnodesCount() (count uint64) {
	db := dbconf.DatabaseConnection()
	db.Model(&Node{}).Where("nodes.network_id = ? AND nodes.bootnode = true AND nodes.status IN (?, ?, ?)", n.ID, nodeStatusRunning, nodeStatusGenesis, nodeStatusPeering).Count(&count)
	return count
}

// Nodes retrieves a list of network nodes; FIXME: support pagination
func (n *Network) Nodes() (nodes []*Node, err error) {
	query := dbconf.DatabaseConnection().Where("nodes.network_id = ?", n.ID)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// ReachableNodes retrieves a list of running network nodes; FIXME: support pagination
func (n *Network) ReachableNodes() (nodes []*Node, err error) {
	query := dbconf.DatabaseConnection().Where("nodes.network_id = ? AND nodes.status IN (?)", n.ID, nodeStatusRunning)
	query.Order("created_at ASC").Find(&nodes)
	return nodes, err
}

// IsBaseledgerNetwork returns true if the network is a baseledger testnet or mainnet
func (n *Network) IsBaseledgerNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isBaseledgerNetwork, ok := cfg[networkConfigIsBaseledgerNetwork].(bool); ok {
			return isBaseledgerNetwork
		}
	}
	return false
}

// IsBcoinNetwork returns true if the network is bcoin-based
func (n *Network) IsBcoinNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isBcoinNetwork, ok := cfg[networkConfigIsBcoinNetwork].(bool); ok {
			return isBcoinNetwork
		}
	}
	return false
}

// IsEthereumNetwork returns true if the network is EVM-based
func (n *Network) IsEthereumNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg[networkConfigIsEthereumNetwork].(bool); ok {
			return isEthereumNetwork
		}
	}
	return false
}

// IsHandshakeNetwork returns true if the network is bcoin-based handshake protocol
func (n *Network) IsHandshakeNetwork() bool {
	cfg := n.ParseConfig()
	if cfg != nil {
		if isHandshakeNetwork, ok := cfg[networkConfigIsHandshakeNetwork].(bool); ok {
			return isHandshakeNetwork
		}
	}
	return false
}

// P2PAPIClient returns an instance of the network's underlying p2p.API, if that is possible given the network config
func (n *Network) P2PAPIClient() (p2p.API, error) {
	cfg := n.ParseConfig()
	client, clientOk := cfg[nodeConfigClient].(string)
	if !clientOk {
		return nil, fmt.Errorf("Failed to resolve p2p provider for network: %s; no configured client", n.ID)
	}
	rpcURL := n.RPCURL()
	if rpcURL == "" {
		common.Log.Debugf("Resolving %s p2p provider for network which does not yet have a configured rpc url; network id: %s", client, n.ID)
	}

	var apiClient p2p.API

	switch client {
	case p2p.ProviderBcoin:
		return nil, fmt.Errorf("Bcoin p2p provider not yet implemented")
	case p2p.ProviderGeth:
		apiClient = p2p.InitGethP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	case p2p.ProviderHyperledgerBesu:
		return nil, fmt.Errorf("besu p2p provider not yet implemented")
	case p2p.ProviderHyperledgerFabric:
		apiClient = p2p.InitHyperledgerFabricP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	case p2p.ProviderNethermind:
		apiClient = p2p.InitNethermindP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	case p2p.ProviderParity:
		apiClient = p2p.InitParityP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	case p2p.ProviderQuorum:
		apiClient = p2p.InitQuorumP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	case p2p.ProviderBaseledger:
		apiClient = p2p.InitBaseledgerP2PProvider(common.StringOrNil(rpcURL), n.ID.String(), n)
	default:
		return nil, fmt.Errorf("Failed to resolve p2p provider for network %s; unsupported client", n.ID)
	}

	return apiClient, nil
}
