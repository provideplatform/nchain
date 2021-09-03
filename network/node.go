package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"

	// natsutil "github.com/kthomas/go-natsutil"

	// "github.com/provideplatform/nchain/gpgputil"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network/p2p"
	provide "github.com/provideplatform/provide-go/api"
	c2 "github.com/provideplatform/provide-go/api/c2"
)

const defualtNodeLogRPP = int64(500)
const nodeReachabilityTimeout = time.Millisecond * 2500

const resolveGenesisTimeout = time.Minute * 10
const resolveHostTimeout = time.Minute * 5
const resolvePeerTimeout = time.Second * 30
const securityGroupTerminationTickerInterval = time.Millisecond * 30000
const securityGroupTerminationTimeout = time.Minute * 5

const nodeConfigClient = "client"
const nodeConfigCredentials = "credentials"
const nodeConfigEntrypoint = "entrypoint"
const nodeConfigEnv = "env"
const nodeConfigImage = "image"
const nodeConfigP2P = "p2p"
const nodeConfigPeerURL = "peer_url"
const nodeConfigRegion = "region"
const nodeConfigResources = "resources"
const nodeConfigRole = "role"
const nodeConfigSecurity = "security"
const nodeConfigSecurityGroupIDs = "target_security_group_ids"
const nodeConfigTargetID = "target_id"
const nodeConfigTaskRole = "task_role"
const nodeConfigTargetTaskIDs = "target_task_id"
const nodeConfigVpcID = "vpc_id"

const nodeStatusFailed = "failed"
const nodeStatusGenesis = "genesis"
const nodeStatusPeering = "peering"
const nodeStatusRunning = "running"
const nodeStatusUnreachable = "unreachable"

const nodeRoleBlockExplorer = "explorer"
const nodeRoleFull = "full"
const nodeRolePeer = "peer"
const nodeRoleValidator = "validator"
const nodeRoleIPFS = "ipfs"

const p2pProtocolPOA = "poa"

// Node instances represent nodes of the network to which they belong, acting in a specific role;
// each Node may have a set or sets of deployed resources, such as application containers, VMs
// or even phyiscal infrastructure
type Node struct {
	provide.Model
	C2NodeID       uuid.UUID  `sql:"not null;type:uuid" json:"c2_node_id"`
	NetworkID      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	UserID         *uuid.UUID `sql:"type:uuid" json:"user_id"`
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id"`
	OrganizationID *uuid.UUID `sql:"type:uuid" json:"organization_id"`
	Bootnode       bool       `sql:"not null;default:'false'" json:"-"`
	Role           *string    `sql:"not null;default:'peer'" json:"role"`

	// ephemeral fields -- enriched from C2
	Host            *string          `sql:"-" json:"host"`
	IPv4            *string          `sql:"-" json:"ipv4"`
	IPv6            *string          `sql:"-" json:"ipv6"`
	PrivateIPv4     *string          `sql:"-" json:"private_ipv4"`
	PrivateIPv6     *string          `sql:"-" json:"private_ipv6"`
	Description     *string          `sql:"-" json:"description"`
	Status          *string          `sql:"-" json:"status"`
	Config          *json.RawMessage `sql:"-" json:"config,omitempty"`
	EncryptedConfig *string          `sql:"-" json:"-"` // FIXME!!!

	privateConfig map[string]interface{} `sql:"-" json:"-"` // cache for decrypted config map in-memory
	Network       *Network               `sql:"-" json:"-"` // in-memory reference to lazy-loaded network
}

// NodeListQuery returns a DB query configured to select columns suitable for a paginated API response
func NodeListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("nodes.id, nodes.created_at, nodes.c2_node_id, nodes.network_id, nodes.user_id, nodes.application_id, nodes.organization_id, nodes.description, nodes.role")
}

func (n *Node) DecryptedConfig() (map[string]interface{}, error) {
	if n.privateConfig != nil {
		return n.privateConfig, nil
	}

	n.privateConfig = map[string]interface{}{}
	if n.EncryptedConfig != nil {
		encryptedConfigJSON, err := pgputil.PGPPubDecrypt([]byte(*n.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to decrypt encrypted network node config; %s", err.Error())
			n.privateConfig = nil
			return nil, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &n.privateConfig)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted network node config; %s", err.Error())
			n.privateConfig = nil
			return nil, err
		}
	}
	return n.privateConfig, nil
}

func (n *Node) encryptConfig() bool {
	if n.EncryptedConfig != nil {
		encryptedConfig, err := pgputil.PGPPubEncrypt([]byte(*n.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to encrypt network node config; %s", err.Error())
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
			return false
		}
		n.EncryptedConfig = common.StringOrNil(string(encryptedConfig))
	}
	return true
}

func (n *Node) SetEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	n.EncryptedConfig = &_paramsJSON
	n.encryptConfig()
	n.privateConfig = params
}

func (n *Node) SanitizeConfig() {
	cfg := n.ParseConfig()

	encryptedCfg, err := n.DecryptedConfig()
	if err != nil {
		encryptedCfg = map[string]interface{}{}
	}

	if credentials, credentialsOk := cfg[nodeConfigCredentials]; credentialsOk {
		encryptedCfg[nodeConfigCredentials] = credentials
		delete(cfg, nodeConfigCredentials)
	}

	if env, envOk := cfg[nodeConfigEnv].(map[string]interface{}); envOk {
		encryptedEnv, encryptedEnvOk := encryptedCfg[nodeConfigEnv].(map[string]interface{})
		if !encryptedEnvOk {
			encryptedEnv = map[string]interface{}{}
			encryptedCfg[nodeConfigEnv] = encryptedEnv
		}

		if engineSignerKeyJSON, engineSignerKeyJSONOk := env["ENGINE_SIGNER_KEY_JSON"]; engineSignerKeyJSONOk {
			encryptedEnv["ENGINE_SIGNER_KEY_JSON"] = engineSignerKeyJSON
			delete(env, "ENGINE_SIGNER_KEY_JSON")
		}

		if engineSignerPrivateKey, engineSignerPrivateKeyOk := env["ENGINE_SIGNER_PRIVATE_KEY"]; engineSignerPrivateKeyOk {
			encryptedEnv["ENGINE_SIGNER_PRIVATE_KEY"] = engineSignerPrivateKey
			delete(env, "ENGINE_SIGNER_PRIVATE_KEY")
		}
	}

	n.SetConfig(cfg)
	n.SetEncryptedConfig(encryptedCfg)
}

// Create and persist a new network node
func (n *Node) Create(token string) bool {
	if !n.Validate() {
		return false
	}

	db := dbconf.DatabaseConnection()
	n.SanitizeConfig()

	err := n.deploy(db, token)
	if err != nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		return false
	}

	if n.C2NodeID == uuid.Nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil("invalid C2 node id"),
		})
		return false
	}

	if db.NewRecord(n) {
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
			return rowsAffected > 0
		}
	}
	return false
}

// SetConfig sets the network config in-memory
func (n *Node) SetConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	_cfgJSON := json.RawMessage(cfgJSON)
	n.Config = &_cfgJSON
}

func (n *Node) peerURL() *string {
	cfg := n.ParseConfig()
	if peerURL, peerURLOk := cfg[nodeConfigPeerURL].(string); peerURLOk {
		return common.StringOrNil(peerURL)
	}

	return nil
}

// mergedConfig returns a merged version of the config and encrypted config
func (n *Node) mergedConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal network node config; %s", err.Error())
			return nil
		}
	}
	encryptedConfig, _ := n.DecryptedConfig()
	for k := range encryptedConfig {
		config[k] = encryptedConfig[k]
	}
	return config
}

func (n *Node) rpcPort() uint {
	cfg := n.ParseConfig()
	port := uint(common.DefaultHTTPPort)
	if jsonRPCPort, jsonRPCPortOk := cfg["json_rpc_port"].(float64); jsonRPCPortOk {
		port = uint(jsonRPCPort)
	}
	return port
}

func (n *Node) rpcURL() *string {
	if n.Host == nil {
		return nil
	}
	port := n.rpcPort()
	if port == 0 {
		return nil
	}
	cfg := n.ParseConfig()
	scheme := "http"
	if rpcScheme, rpcSchemeOk := cfg["rpc_scheme"].(string); rpcSchemeOk {
		scheme = rpcScheme
	}
	return common.StringOrNil(fmt.Sprintf("%s://%s:%d", scheme, *n.Host, port))
}

func (n *Node) reachableViaJSONRPC() (bool, uint) {
	port := n.rpcPort()
	return n.ReachableOnPort(port), port
}

func (n *Node) reachableViaWebsocket() (bool, uint) {
	cfg := n.ParseConfig()
	port := uint(common.DefaultWebsocketPort)
	if websocketPortOverride, websocketPortOverrideOk := cfg["websocket_port"].(float64); websocketPortOverrideOk {
		port = uint(websocketPortOverride)
	}

	return n.ReachableOnPort(port), port
}

func (n *Node) ReachableOnPort(port uint) bool {
	if n.Host == nil {
		return false
	}
	addr := fmt.Sprintf("%s:%v", *n.Host, port)
	conn, err := net.DialTimeout("tcp", addr, nodeReachabilityTimeout)
	if err == nil {
		common.Log.Debugf("%s:%v is reachable", *n.Host, port)
		defer conn.Close()
		return true
	}
	common.Log.Debugf("%s:%v is unreachable", *n.Host, port)
	return false
}

func (n *Node) relatedNetwork(db *gorm.DB) *Network {
	var network = &Network{}
	db.Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		common.Log.Warningf("Failed to retrieve network for network node: %s", n.ID)
		return nil
	}
	return network
}

func (n *Node) updateStatus(db *gorm.DB, status string, description *string) {
	n.Status = common.StringOrNil(status)
	n.Description = description
	// result := db.Save(&n)
	// errors := result.GetErrors()
	// if len(errors) > 0 {
	// 	for _, err := range errors {
	// 		n.Errors = append(n.Errors, &provide.Error{
	// 			Message: common.StringOrNil(err.Error()),
	// 		})
	// 	}
	// }
}

// Validate a network node for persistence
func (n *Node) Validate() bool {
	cfg := n.ParseConfig()
	if role, roleOk := cfg[nodeConfigRole].(string); roleOk {
		if n.Role == nil || *n.Role != role {
			common.Log.Debugf("Coercing network node role to match node configuration; role: %s", role)
			n.Role = common.StringOrNil(role)
		}
	}
	return len(n.Errors) == 0
}

// ParseConfig - parse the network node configuration JSON
func (n *Node) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal network node config; %s", err.Error())
			return nil
		}
	}
	return config
}

// Delete a network node
func (n *Node) Delete(token string) bool {
	_, err := c2.DeleteNode(token, n.ID.String())
	if err != nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil(fmt.Sprintf("Failed to delete network node; %s", err.Error())),
		})
	}
	// msg, _ := json.Marshal(map[string]interface{}{
	// 	"node_id": n.ID.String(),
	// })
	// natsutil.NatsJetstreamPublish(natsDeleteTerminatedNodeSubject, msg)
	return len(n.Errors) == 0
}

// Reload the underlying network node instance
func (n *Node) Reload() {
	db := dbconf.DatabaseConnection()
	db.Model(&n).Find(n)
}

func (n *Node) resolveNetwork(db *gorm.DB) error {
	var network = &Network{}
	db.Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		msg := fmt.Sprintf("Failed to resolve network for network node: %s", n.ID)
		common.Log.Warning(msg)
		return errors.New(msg)
	}
	n.Network = network
	return nil
}

func (n *Node) deploy(db *gorm.DB, token string) error {
	if n.Config == nil {
		msg := fmt.Sprintf("Not attempting to deploy network node without a valid configuration; network node id: %s", n.ID)
		common.Log.Warning(msg)
		return errors.New(msg)
	}

	err := n.resolveNetwork(db)
	if n.Network == nil {
		msg := err.Error()
		n.updateStatus(db, "failed", &msg)
		return err
	}
	network := n.Network // FIXME

	common.Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, n.Network.ID)
	n.updateStatus(db, "pending", nil)

	cfg := n.ParseConfig()

	isP2P, p2pOk := cfg[nodeConfigP2P].(bool)
	role, roleOk := cfg[nodeConfigRole].(string)
	isPeerToPeer := p2pOk && isP2P
	if !isPeerToPeer && !p2pOk && roleOk {
		// coerce p2p flag if applicable for role
		isP2P = role == nodeRoleFull || role == nodeRolePeer || role == nodeRoleValidator || role == nodeRoleBlockExplorer
		cfg[nodeConfigP2P] = isP2P
	}

	if isPeerToPeer {
		p2pAPI, err := n.P2PAPIClient()
		if err != nil {
			common.Log.Warningf("Failed to deploy network node with id: %s; %s", n.ID, err.Error())
			return err
		}

		bootnodes, err := network.requireBootnodes(db, n)
		if err != nil {
			switch err.(type) {
			case *bootnodesInitialized:
				common.Log.Debugf("Bootnode initialized for network: %s; node: %s; waiting for genesis to complete and peer resolution to become possible", *network.Name, n.ID.String())
				err := p2pAPI.RequireBootnodes(db, n.UserID, &n.NetworkID, n)
				if err != nil {
					common.Log.Warningf("Failed to deploy network node with id: %s; %s", n.ID, err.Error())
					return err
				}

				return n._deploy(network, bootnodes, db, token)
			default:
				msg := fmt.Sprintf("Attempt to deploy node %s did not succeed; network: %s; %s", n.ID, *network.Name, err.Error())
				common.Log.Debugf(msg)
				return errors.New(msg)
			}
		} else {
			return n.requireGenesis(network, bootnodes, db, token) // default assumes p2p
		}
	} else {
		common.Log.Debugf("Attempting to deploy non-p2p node: %s", n.ID)
		return n._deploy(network, []*Node{}, db, token)
	}
}

func (n *Node) requireGenesis(network *Network, bootnodes []*Node, db *gorm.DB, token string) error {
	common.Log.Debugf("Attempting to require peer-to-peer network genesis for node: %s", n.ID)

	if len(bootnodes) > 0 {
		common.Log.Debugf("Short-circuiting genesis block resolution for node: %s; %d bootnode(s) resolved for peering", n.ID, len(bootnodes))
		return n._deploy(network, bootnodes, db, token)
	}

	// cfg := network.ParseConfig()
	// if bootnodes, bootnodesOk := cfg["bootnodes"].([]string); bootnodesOk && len(bootnodes) > 0 {
	// 	for i := range bootnodes {
	// 		return n._deploy(network, bootnodes, db)
	// 		n.addPeer(bootnodes[i])
	// 	}
	// 	common.Log.Debugf("Short-circuiting genesis block resolution for node: %s; %d bootnode(s) resolved for peering", n.ID, len(bootnodes))
	// 	return n._deploy(network, bootnodes, db)
	// }

	if n.Role != nil && *n.Role == nodeRoleIPFS {
		common.Log.Debugf("Short-circuiting genesis block resolution for IPFS node: %s", n.ID)
		return n._deploy(network, bootnodes, db, token)
	}

	if time.Now().Sub(n.CreatedAt) >= resolveGenesisTimeout {
		desc := fmt.Sprintf("Failed to resolve genesis block for network bootnode: %s; timing out after %v", n.ID.String(), resolveGenesisTimeout)
		n.updateStatus(db, "failed", &desc)
		common.Log.Warning(desc)
		return errors.New(desc)
	}

	stats, _ := network.Stats()
	if stats == nil || stats.Block == 0 {
		desc := "awaiting genesis peers..."
		n.updateStatus(db, nodeStatusPeering, &desc)
		return fmt.Errorf("Node deployment awaiting network genesis; node: %s; network: %s", n.ID, network.ID)
	}

	return n._deploy(network, bootnodes, db, token)
}

func (n *Node) _deploy(network *Network, bootnodes []*Node, db *gorm.DB, token string) error {
	cfg := n.ParseConfig()
	// encryptedCfg, _ := n.DecryptedConfig()
	networkCfg := network.ParseConfig()

	isPeerToPeer := false

	// image, imageOk := cfg[nodeConfigImage].(string)
	// resources, resourcesOk := cfg[nodeConfigResources].(map[string]interface{}) // TODO-- make Resources struct
	// entrypoint, entrypointOk := cfg[nodeConfigEntrypoint].([]string)
	// taskRole, taskRoleOk := cfg[nodeConfigTaskRole].(string)
	// script, scriptOk := cfg["script"].(map[string]interface{})
	// target, targetOk := cfg[nodeConfigTargetID].(string)
	role, _ := cfg[nodeConfigRole].(string)
	// region, regionOk := cfg[nodeConfigRegion].(string)
	// vpc, _ := cfg[nodeConfigVpcID].(string)
	env, envOk := cfg[nodeConfigEnv].(map[string]interface{})
	// encryptedEnv, encryptedEnvOk := encryptedCfg[nodeConfigEnv].(map[string]interface{})
	// securityCfg, securityCfgOk := cfg[nodeConfigSecurity].(map[string]interface{}) // TODO-- make Security struct
	isP2P, p2pOk := cfg[nodeConfigP2P].(bool)
	if !p2pOk {
		isPeerToPeer = false
	} else {
		isPeerToPeer = isP2P
	}

	if networkEnv, networkEnvOk := networkCfg[nodeConfigEnv].(map[string]interface{}); envOk && networkEnvOk {
		common.Log.Debugf("Applying environment overrides to network node per network env configuration")
		for k := range networkEnv {
			env[k] = networkEnv[k]
		}
	}

	if isPeerToPeer {
		common.Log.Debugf("applying peer-to-peer environment sanity rules to deploy network node: %s; role: %s", n.ID, role)
		// p2pAPI, err := n.P2PAPIClient()
		// if err != nil {
		// 	err := fmt.Errorf("failed to deploy network node %s; %s", n.ID, err.Error())
		// 	return err
		// }

		// FIXME?
		// if bnodes, bootnodesOk := envOverrides[networkConfigEnvBootnodes].(string); bootnodesOk {
		// 	envOverrides[networkConfigEnvBootnodes] = bnodes
		// } else {
		// 	bootnodesTxt, err := network.BootnodesTxt()
		// 	if err == nil && bootnodesTxt != nil && *bootnodesTxt != "" {
		// 		envOverrides[networkConfigEnvBootnodes] = bootnodesTxt
		// 	}
		// }

		// FIXME?
		// networkChain, networkChainOk := networkCfg[networkConfigChain].(string)
		// if _, chainOk := envOverrides[networkConfigChain].(string); !chainOk {
		// 	if networkChainOk {
		// 		envOverrides[networkConfigChain] = networkChain
		// 	}
		// } else if networkChainOk {
		// 	chain := envOverrides[networkConfigChain].(string)
		// 	if chain != networkChain {
		// 		common.Log.Warningf("Overridden chain %s did not match network chain %s; network id: %s", chain, networkChain, network.ID)
		// 	}
		// }
	}

	// FIXME?
	// overrides := map[string]interface{}{
	// 	"environment": envOverrides,
	// }
	// cfg[nodeConfigEnv] = envOverrides

	// FIXME?
	// _entrypoint := make([]*string, 0)
	// if entrypointOk {
	// 	for i := range entrypoint {
	// 		_entrypoint = append(_entrypoint, &entrypoint[i])
	// 	}
	// } else if p2pAPI != nil {
	// 	defaultEntrypoint := p2pAPI.DefaultEntrypoint()
	// 	for i := range defaultEntrypoint {
	// 		_entrypoint = append(_entrypoint, &defaultEntrypoint[i])
	// 	}
	// }

	// FIXME?
	// if p2pAPI != nil {
	// 	_bootnodes := make([]string, 0)
	// 	for i := range bootnodes {
	// 		peerURL := bootnodes[i].peerURL()
	// 		if peerURL != nil {
	// 			_bootnodes = append(_bootnodes, *peerURL)
	// 		}
	// 	}
	// 	cmdEnrichment := p2pAPI.EnrichStartCommand(_bootnodes)
	// 	for i := range cmdEnrichment {
	// 		_entrypoint = append(_entrypoint, &cmdEnrichment[i])
	// 	}
	// }

	resp, err := c2.CreateNode(token, n.ParseConfig()) // FIXME-- this should be nested under `config`
	if err != nil {
		n.Errors = append(n.Errors, &provide.Error{
			Message: common.StringOrNil(err.Error()),
		})
		return err
	}
	n.C2NodeID = resp.ID

	n.SetConfig(cfg)
	n.SanitizeConfig()
	db.Save(&n)

	// FIXME
	// msg, _ := json.Marshal(map[string]interface{}{
	// 	"node_id": n.ID.String(),
	// })
	// natsutil.NatsJetstreamPublish(natsResolveNodeHostSubject, msg)

	return nil
}

func (n *Node) resolvePeerURL(db *gorm.DB) error {
	network := n.relatedNetwork(db)
	if network == nil {
		return fmt.Errorf("Failed to resolve peer url for network node %s; no network resolved", n.ID)
	}

	cfg := n.ParseConfig()
	// targetID, targetOk := cfg["target_id"].(string)
	taskIds, taskIdsOk := cfg[nodeConfigTargetTaskIDs].([]interface{})

	if !taskIdsOk {
		return fmt.Errorf("Failed to deploy network node %s; no target_task_ids provided", n.ID)
	}

	identifiers := make([]string, len(taskIds))
	for _, id := range taskIds {
		identifiers = append(identifiers, id.(string))
	}

	if len(identifiers) == 0 {
		return fmt.Errorf("Unable to resolve network node peer url without any node identifiers")
	}

	role, roleOk := cfg[nodeConfigRole].(string)
	if !roleOk || role != nodeRolePeer && role != nodeRoleFull && role != nodeRoleValidator {
		return nil
	}

	common.Log.Debugf("Attempting to resolve peer url for network node: %s", n.ID.String())

	var p2pAPI p2p.API

	var peerURL *string
	var err error

	id := identifiers[len(identifiers)-1]

	p2pAPI, err = n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to resolve peer url for network node %s; %s", n.ID, err.Error())
		return err
	}

	peerURL, err = p2pAPI.ResolvePeerURL()
	if err != nil {
		common.Log.Debugf("No peer url or equivalent resolved for network node %s; %s", n.ID, err.Error())
	}

	if peerURL == nil {
		resp, err := c2.GetNodeLogs("", id, map[string]interface{}{}) // FIXME-- resolve proper c2 token
		if err == nil && resp != nil {
			for i := range resp.Logs {
				peerURL, err = p2pAPI.ParsePeerURL(resp.Logs[i].Message)
				if err == nil && peerURL != nil {
					if n.IPv4 != nil && n.PrivateIPv4 != nil {
						url := strings.Replace(*peerURL, *n.PrivateIPv4, *n.IPv4, 1)
						url = strings.Replace(url, "127.0.0.1", *n.IPv4, 1)
						peerURL = &url
					}
					cfg[nodeConfigPeerURL] = peerURL
					break
				}
			}
		}
	}

	if peerURL == nil {
		if time.Now().Sub(n.CreatedAt) >= resolvePeerTimeout {
			desc := fmt.Sprintf("Failed to resolve peer url for network node %s after %v", n.ID.String(), resolvePeerTimeout)
			n.updateStatus(db, "failed", &desc)
			common.Log.Warning(desc)
			return fmt.Errorf(desc)
		}

		return fmt.Errorf("Failed to resolve peer url for network node with id: %s", n.ID)
	}

	common.Log.Debugf("Resolved peer url for network node with id: %s; peer url: %s", n.ID, *peerURL)
	if n.Bootnode {
		err := network.AddBootnode(db, *peerURL)
		if err != nil {
			common.Log.Warningf("Failed to add peer url as bootnode; peer url: %s; network: %s; %s", *peerURL, network.ID, err.Error())
		}
	}

	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
	n.Status = common.StringOrNil(nodeStatusRunning)
	n.Description = nil
	db.Save(&n)

	// FIXME!
	// if role == nodeRolePeer || role == nodeRoleFull || role == nodeRoleValidator {
	// 	go network.resolveAndBalanceJSONRPCAndWebsocketURLs(db, n)
	// 	// TODO: determine if the node is running IPFS; if so: go network.resolveAndBalanceIPFSUrls(db, n)
	// }

	return nil
}

// P2PAPIClient returns an instance of the node's underlying p2p.API
func (n *Node) P2PAPIClient() (p2p.API, error) {
	cfg := n.ParseConfig()
	client, clientOk := cfg[nodeConfigClient].(string)
	if !clientOk {
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node: %s; no configured client", n.ID)
	}
	rpcURL := n.rpcURL()
	if rpcURL == nil {
		common.Log.Debugf("Resolving %s p2p provider for network node which does not yet have a configured rpc url; node id: %s", client, n.ID)
	}

	if n.Network == nil {
		err := n.resolveNetwork(dbconf.DatabaseConnection())
		if err != nil {
			return nil, fmt.Errorf("Failed to resolve p2p provider for network node; %s", err.Error())
		}
	}

	var apiClient p2p.API

	switch client {
	case p2p.ProviderBcoin:
		return nil, fmt.Errorf("Bcoin p2p provider not yet implemented")
	case p2p.ProviderGeth:
		apiClient = p2p.InitGethP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	case p2p.ProviderHyperledgerBesu:
		return nil, fmt.Errorf("besu p2p provider not yet implemented")
	case p2p.ProviderHyperledgerFabric:
		apiClient = p2p.InitHyperledgerFabricP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	case p2p.ProviderNethermind:
		apiClient = p2p.InitNethermindP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	case p2p.ProviderParity:
		apiClient = p2p.InitParityP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	case p2p.ProviderQuorum:
		apiClient = p2p.InitQuorumP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	case p2p.ProviderBaseledger:
		apiClient = p2p.InitBaseledgerP2PProvider(rpcURL, n.NetworkID.String(), n.Network)
	default:
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node %s; unsupported client", n.ID)
	}

	return apiClient, nil
}

func (n *Node) addPeer(peerURL string) error {
	apiClient, err := n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to add peer; %s", err.Error())
		return err
	}
	return apiClient.AddPeer(peerURL)
}

func (n *Node) removePeer(peerURL string) error {
	apiClient, err := n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to remove peer; %s", err.Error())
		return err
	}
	return apiClient.RemovePeer(peerURL)
}

func (n *Node) acceptNonReservedPeers() error {
	apiClient, err := n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to accept non-reserved peers; %s", err.Error())
		return err
	}
	return apiClient.AcceptNonReservedPeers()
}

func (n *Node) dropNonReservedPeers() error {
	apiClient, err := n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to drop non-reserved peers; %s", err.Error())
		return err
	}
	return apiClient.DropNonReservedPeers()
}

func (n *Node) upgrade() error {
	apiClient, err := n.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to execute upgrade; %s", err.Error())
		return err
	}
	return apiClient.Upgrade()
}
