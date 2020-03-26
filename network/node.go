package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network/orchestration"
	"github.com/provideapp/goldmine/network/p2p"
	provide "github.com/provideservices/provide-go"
)

const defaultDockerhubBaseURL = "https://hub.docker.com/v2/repositories"
const defualtNodeLogRPP = int64(250)
const nodeReachabilityTimeout = time.Millisecond * 2500
const dockerRepoReachabilityTimeout = time.Millisecond * 2500

const resolveGenesisTickerInterval = time.Millisecond * 10000
const resolveGenesisTickerTimeout = time.Minute * 20
const resolveHostTickerInterval = time.Millisecond * 5000
const resolveHostTickerTimeout = time.Minute * 10
const resolvePeerTickerInterval = time.Millisecond * 5000
const resolvePeerTickerTimeout = time.Minute * 20
const securityGroupTerminationTickerInterval = time.Millisecond * 30000
const securityGroupTerminationTickerTimeout = time.Minute * 10

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
	NetworkID       uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	UserID          *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	OrganizationID  *uuid.UUID       `sql:"type:uuid" json:"organization_id"`
	Bootnode        bool             `sql:"not null;default:'false'" json:"-"`
	Host            *string          `json:"host"`
	IPv4            *string          `json:"ipv4"`
	IPv6            *string          `json:"ipv6"`
	PrivateIPv4     *string          `json:"private_ipv4"`
	PrivateIPv6     *string          `json:"private_ipv6"`
	Description     *string          `json:"description"`
	Role            *string          `sql:"not null;default:'peer'" json:"role"`
	Status          *string          `sql:"not null;default:'init'" json:"status"`
	LoadBalancers   []LoadBalancer   `gorm:"many2many:load_balancers_nodes" json:"-"`
	Config          *json.RawMessage `sql:"type:json" json:"config,omitempty"`
	EncryptedConfig *string          `sql:"type:bytea" json:"-"`

	privateConfig map[string]interface{} `sql:"-" json:"-"` // cache for decrypted config map in-memory
	Network       *Network               `sql:"-" json:"-"` // in-memory reference to lazy-loaded network
}

// NodeListQuery returns a DB query configured to select columns suitable for a paginated API response
func NodeListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("nodes.id, nodes.created_at, nodes.network_id, nodes.user_id, nodes.application_id, nodes.organization_id, nodes.host, nodes.ipv4, nodes.ipv6, nodes.private_ipv4, nodes.private_ipv6, nodes.description, nodes.role, nodes.status, nodes.config")
}

// NodeLog represents an abstract API response containing syslog or similar messages
type NodeLog struct {
	Timestamp       *int64 `json:"timestamp"`
	IngestTimestamp *int64 `json:"ingest_timestamp"`
	Message         string `json:"message"`
}

// NodeLogsResponse represents an abstract API response containing NodeLogs
// and pointer tokens to the next set of events in the stream; this is necessary
// for properly paginating logs
type NodeLogsResponse struct {
	Logs      []*NodeLog `json:"logs"`
	PrevToken *string    `json:"prev_token"`
	NextToken *string    `json:"next_token"`
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
func (n *Node) Create() bool {
	if !n.Validate() {
		return false
	}

	n.SanitizeConfig()

	db := dbconf.DatabaseConnection()

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
			success := rowsAffected > 0
			if success {
				msg, _ := json.Marshal(map[string]interface{}{
					"node_id": n.ID.String(),
				})
				natsutil.NatsStreamingPublish(natsDeployNodeSubject, msg)
			}
			return success
		}
	}
	return false
}

// Update an existing network node
func (n *Node) Update() bool {
	if !n.Validate() {
		return false
	}

	n.SanitizeConfig()

	db := dbconf.DatabaseConnection()

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
	result := db.Save(&n)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
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
func (n *Node) Delete() bool {
	n.undeploy()
	msg, _ := json.Marshal(map[string]interface{}{
		"node_id": n.ID.String(),
	})
	natsutil.NatsStreamingPublish(natsDeleteTerminatedNodeSubject, msg)
	return len(n.Errors) == 0
}

// Reload the underlying network node instance
func (n *Node) Reload() {
	db := dbconf.DatabaseConnection()
	db.Model(&n).Find(n)
}

// Logs exposes the paginated logstream for the underlying node
func (n *Node) Logs(startFromHead bool, limit *int64, nextToken *string) (*NodeLogsResponse, error) {
	var response *NodeLogsResponse
	var network = &Network{}
	dbconf.DatabaseConnection().Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		return nil, fmt.Errorf("Failed to retrieve network for network node: %s", n.ID)
	}

	cfg := n.ParseConfig()

	targetID, targetOk := cfg[nodeConfigTargetID].(string)
	_, regionOk := cfg[nodeConfigRegion].(string)

	if !targetOk {
		return nil, fmt.Errorf("Cannot retrieve logs for network node without a target and provider configuration; target id: %s", targetID)
	}

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to retrieve logs for network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return nil, err
	}

	if regionOk {
		response = &NodeLogsResponse{
			Logs: make([]*NodeLog, 0),
		}

		if ids, idsOk := cfg[nodeConfigTargetTaskIDs].([]interface{}); idsOk {
			for i, id := range ids {
				logEvents, err := orchestrationAPI.GetContainerLogEvents(id.(string), nil, startFromHead, nil, nil, limit, nextToken)
				if err == nil && logEvents != nil {
					events := logEvents.Events
					if !startFromHead {
						for i := len(events)/2 - 1; i >= 0; i-- {
							opp := len(events) - 1 - i
							events[i], events[opp] = events[opp], events[i]
						}
					}
					for i := range logEvents.Events {
						event := logEvents.Events[i]
						response.Logs = append(response.Logs, &NodeLog{
							Message:         string(*event.Message),
							Timestamp:       event.Timestamp,
							IngestTimestamp: event.IngestionTime,
						})
					}
					if i == len(ids)-1 {
						response.NextToken = logEvents.NextForwardToken
						response.PrevToken = logEvents.NextBackwardToken
					}
				}
			}
			return response, nil
		}

		return nil, fmt.Errorf("Unable to retrieve logs for network node: %s", n.ID)
	}

	return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; no region provided", n.ID)
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

func (n *Node) deploy(db *gorm.DB) error {
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
	p2pAPI, err := n.p2pAPIClient()
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

			return n._deploy(network, bootnodes, db)
		default:
			msg := fmt.Sprintf("Failed to deploy node %s to network: %s; %s", n.ID, *network.Name, err.Error())
			common.Log.Warning(msg)
			return errors.New(msg)
		}
	} else {
		if p2p, p2pOk := cfg[nodeConfigP2P].(bool); p2pOk {
			if p2p {
				common.Log.Debugf("Attempting to require peer-to-peer network genesis for node: %s", n.ID)
				return n.requireGenesis(network, bootnodes, db)
			}
			common.Log.Debugf("Attempting to deploy non-p2p node: %s", n.ID)
			return n._deploy(network, bootnodes, db)
		}
		return n.requireGenesis(network, bootnodes, db) // default assumes p2p
	}
}

func (n *Node) requireGenesis(network *Network, bootnodes []*Node, db *gorm.DB) error {
	common.Log.Debugf("Attempting to resolve network genesis...")

	if n.Role != nil && *n.Role == nodeRoleIPFS {
		common.Log.Debugf("Short-circuiting genesis block resolution for IPFS node: %s", n.ID)
		return n._deploy(network, bootnodes, db)
	}

	ticker := time.NewTicker(resolveGenesisTickerInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			if time.Now().Sub(startedAt) >= resolveGenesisTickerTimeout {
				desc := fmt.Sprintf("Failed to resolve genesis block for network bootnode: %s; timing out after %v", n.ID.String(), resolveGenesisTickerTimeout)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				ticker.Stop()
				return errors.New(desc)
			}

			stats, _ := network.Stats()
			if stats != nil && stats.Block > 0 {
				ticker.Stop()
				return n._deploy(network, bootnodes, db)
			}
		}
	}
}

func (n *Node) _deploy(network *Network, bootnodes []*Node, db *gorm.DB) error {
	cfg := n.ParseConfig()
	encryptedCfg, _ := n.DecryptedConfig()
	networkCfg := network.ParseConfig()

	image, imageOk := cfg[nodeConfigImage].(string)
	resources, resourcesOk := cfg[nodeConfigResources].(map[string]interface{})
	entrypoint, entrypointOk := cfg[nodeConfigEntrypoint].([]string)
	taskRole, taskRoleOk := cfg[nodeConfigTaskRole].(string)
	// script, scriptOk := cfg["script"].(map[string]interface{})
	_, targetOk := cfg[nodeConfigTargetID].(string)
	role, roleOk := cfg[nodeConfigRole].(string)
	region, regionOk := cfg[nodeConfigRegion].(string)
	vpc, _ := cfg[nodeConfigVpcID].(string)
	env, envOk := cfg[nodeConfigEnv].(map[string]interface{})
	encryptedEnv, encryptedEnvOk := encryptedCfg[nodeConfigEnv].(map[string]interface{})
	securityCfg, securityCfgOk := cfg[nodeConfigSecurity].(map[string]interface{})
	isP2P, p2pOk := cfg[nodeConfigP2P].(bool)

	if networkEnv, networkEnvOk := networkCfg[nodeConfigEnv].(map[string]interface{}); envOk && networkEnvOk {
		common.Log.Debugf("Applying environment overrides to network node per network env configuration")
		for k := range networkEnv {
			env[k] = networkEnv[k]
		}
	}

	var p2pAPI p2p.API
	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to deploy network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if !imageOk {
		desc := fmt.Sprintf("Failed to deploy node in region %s; network node id: %s; image must be specified", region, n.ID.String())
		n.updateStatus(db, "failed", &desc)
		common.Log.Warning(desc)
		return errors.New(desc)
	}

	if !securityCfgOk {
		desc := fmt.Sprintf("Failed to deploy node in region %s; network node id: %s; security config must be provided", region, n.ID.String())
		n.updateStatus(db, "failed", &desc)
		common.Log.Warning(desc)
		return errors.New(desc)
	}

	if targetOk && regionOk {
		isPeerToPeer := p2pOk && isP2P
		if !isPeerToPeer && !p2pOk && roleOk {
			// coerce p2p flag if applicable for role
			isP2P = role == nodeRoleFull || role == nodeRolePeer || role == nodeRoleValidator || role == nodeRoleBlockExplorer
		}

		securityGroupDesc := fmt.Sprintf("security group for network node: %s", n.ID.String())
		securityGroupIds, err := orchestrationAPI.CreateSecurityGroup(securityGroupDesc, securityGroupDesc, nil, securityCfg)
		if err != nil {
			n.updateStatus(db, "failed", common.StringOrNil(err.Error()))
			return err
		}

		cfg[nodeConfigSecurity] = securityCfg
		cfg[nodeConfigRegion] = region
		cfg[nodeConfigSecurityGroupIDs] = securityGroupIds
		n.SetConfig(cfg)
		db.Save(&n)

		common.Log.Debugf("Attempting to deploy network node in region: %s", region)
		var imageRef *string

		if imageOk {
			imageRef = common.StringOrNil(image)

			// prefer provide dockerhub repo, if it exists for the requested image...
			imageRef, err = dockerhubRepoExists(*imageRef)
			if err != nil {
				n.updateStatus(db, "failed", common.StringOrNil(err.Error()))
				return err
			}

			common.Log.Debugf("Resolved container image to deploy in region %s; image: %s", region, *imageRef)
		} else {
			err := fmt.Errorf("Failed to resolve deployable image or container(s) to deploy in region: %s; network node: %s", region, n.ID)
			n.updateStatus(db, "failed", common.StringOrNil(err.Error()))
			return err
		}

		ref := imageRef
		common.Log.Debugf("Attempting to deploy container %s in region: %s", *ref, region)

		envOverrides := map[string]interface{}{}
		if envOk {
			for k := range env {
				envOverrides[k] = env[k]
			}
		}
		if encryptedEnvOk {
			for k := range encryptedEnv {
				envOverrides[k] = encryptedEnv[k]
			}
		}

		if isPeerToPeer {
			common.Log.Debugf("Applying peer-to-peer environment sanity rules to deploy network node: %s; role: %s", n.ID, role)
			p2pAPI, err = n.p2pAPIClient()
			if err != nil {
				err := fmt.Errorf("Failed to deploy network node %s; %s", n.ID, err.Error())
				return err
			}

			if bnodes, bootnodesOk := envOverrides[networkConfigEnvBootnodes].(string); bootnodesOk {
				envOverrides[networkConfigEnvBootnodes] = bnodes
			} else {
				bootnodesTxt, err := network.BootnodesTxt()
				if err == nil && bootnodesTxt != nil && *bootnodesTxt != "" {
					envOverrides[networkConfigEnvBootnodes] = bootnodesTxt
				}
			}

			networkChain, networkChainOk := networkCfg[networkConfigChain].(string)
			if _, chainOk := envOverrides[networkConfigChain].(string); !chainOk {
				if networkChainOk {
					envOverrides[networkConfigChain] = networkChain
				}
			} else if networkChainOk {
				chain := envOverrides[networkConfigChain].(string)
				if chain != networkChain {
					common.Log.Warningf("Overridden chain %s did not match network chain %s; network id: %s", chain, networkChain, network.ID)
				}
			}
		}

		overrides := map[string]interface{}{
			"environment": envOverrides,
		}
		cfg[nodeConfigEnv] = envOverrides

		n.SetConfig(cfg)
		n.SanitizeConfig()
		db.Save(&n)

		containerSecurity := map[string]interface{}{} // for now, this should only be populated when imageRef != nil (awswrapper does not yet support providing security cfg when a task def is provided...)
		if imageRef != nil {
			containerSecurity = securityCfg
		}

		var cpu *int64
		var memory *int64
		if resourcesOk {
			if _cpu, cpuOk := resources["cpu"].(float64); cpuOk {
				cpuInt := int64(_cpu)
				cpu = &cpuInt
			}
			if _memory, memoryOk := resources["memory"].(float64); memoryOk {
				memoryInt := int64(_memory)
				memory = &memoryInt
			}
		}

		_entrypoint := make([]*string, 0)
		if entrypointOk {
			for i := range entrypoint {
				_entrypoint = append(_entrypoint, &entrypoint[i])
			}
		} else if p2pAPI != nil {
			defaultEntrypoint := p2pAPI.DefaultEntrypoint()
			for i := range defaultEntrypoint {
				_entrypoint = append(_entrypoint, &defaultEntrypoint[i])
			}
		}

		if p2pAPI != nil {
			cmdEnrichment := p2pAPI.EnrichStartCommand()
			if len(_entrypoint) > 0 && len(cmdEnrichment) > 0 {
				_entrypoint = append(_entrypoint, common.StringOrNil("&&"))
			}

			for i := range cmdEnrichment {
				_entrypoint = append(_entrypoint, &cmdEnrichment[i])
			}
		}

		var containerRole *string
		if taskRoleOk {
			containerRole = &taskRole
		}

		taskIds, err := orchestrationAPI.StartContainer(
			imageRef,
			nil,
			containerRole,
			nil,
			nil,
			common.StringOrNil(vpc),
			cpu,
			memory,
			_entrypoint,
			securityGroupIds,
			[]string{},
			overrides,
			containerSecurity,
		)

		if err != nil || len(taskIds) == 0 {
			desc := fmt.Sprintf("Attempt to deploy container %s in %s region failed; %s", *ref, region, err.Error())
			n.updateStatus(db, "failed", &desc)
			n.unregisterSecurityGroups()
			common.Log.Warning(desc)
			return errors.New(desc)
		}

		if imageRef != nil {
			common.Log.Warningf("FIXME-- leaking the task definition that was used to start this container... %s", taskIds[0])
		}

		common.Log.Debugf("Attempt to deploy container %s in %s region successful; task ids: %s", *ref, region, taskIds)
		cfg[nodeConfigTargetTaskIDs] = taskIds
		n.SetConfig(cfg)
		n.SanitizeConfig()
		db.Save(&n)

		msg, _ := json.Marshal(map[string]interface{}{
			"node_id": n.ID.String(),
		})
		natsutil.NatsStreamingPublish(natsResolveNodeHostSubject, msg)
	}

	return nil
}

func (n *Node) resolveHost(db *gorm.DB) error {
	network := n.relatedNetwork(db)
	if network == nil {
		return fmt.Errorf("Failed to resolve host for network node %s; no network resolved", n.ID)
	}

	cfg := n.ParseConfig()
	taskIds, taskIdsOk := cfg[nodeConfigTargetTaskIDs].([]interface{})

	if !taskIdsOk {
		return fmt.Errorf("Failed to resolve host for network node %s; no target_task_ids provided", n.ID)
	}

	identifiers := make([]string, 0)
	for _, id := range taskIds {
		identifiers = append(identifiers, id.(string))
	}

	if len(identifiers) == 0 {
		return fmt.Errorf("Unable to resolve network node host without any node identifiers")
	}

	taskID := identifiers[len(identifiers)-1]
	_, regionOk := cfg[nodeConfigRegion].(string)

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to resolve host for network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if regionOk {
		interfaces, err := orchestrationAPI.GetContainerInterfaces(taskID, nil)
		if err != nil {
			err := fmt.Errorf("Failed to resolve host for network node %s; %s", n.ID, err.Error())
			common.Log.Warningf(err.Error())
			return err
		}

		if len(interfaces) > 0 {
			networkInterface := interfaces[0]
			n.Host = networkInterface.Host
			n.IPv4 = networkInterface.IPv4
			n.IPv6 = networkInterface.IPv6
			n.PrivateIPv4 = networkInterface.PrivateIPv4
			n.PrivateIPv6 = networkInterface.PrivateIPv6
		}
	}

	if n.Host == nil {
		if time.Now().Sub(n.CreatedAt) >= resolveHostTickerTimeout {
			desc := fmt.Sprintf("Failed to resolve hostname for network node %s after %v", n.ID.String(), resolveHostTickerTimeout)
			n.updateStatus(db, "failed", &desc)
			common.Log.Warning(desc)
			return fmt.Errorf(desc)
		}

		return fmt.Errorf("Failed to resolve host for network node with id: %s", n.ID)
	}

	err = n.dropNonReservedPeers()
	if err != nil {
		common.Log.Debugf("Failed to set node to only accept connections from reserved peers; %s", err.Error())
	}

	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
	n.Status = common.StringOrNil(nodeStatusRunning)
	db.Save(&n)

	role, roleOk := cfg[nodeConfigRole].(string)
	if roleOk {
		if role == nodeRoleBlockExplorer {
			go network.resolveAndBalanceExplorerUrls(db, n)
		} else if role == nodeRoleIPFS {
			go network.resolveAndBalanceIPFSUrls(db, n)
		}
	}

	return nil
}

func (n *Node) resolvePeerURL(db *gorm.DB) error {
	network := n.relatedNetwork(db)
	if network == nil {
		return fmt.Errorf("Failed to resolve peer url for network node %s; no network resolved", n.ID)
	}

	cfg := n.ParseConfig()
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

	var orchestrationAPI orchestration.API
	var p2pAPI p2p.API

	var peerURL *string
	var err error

	id := identifiers[len(identifiers)-1]
	_, regionOk := cfg[nodeConfigRegion].(string)

	if regionOk {
		p2pAPI, err = n.p2pAPIClient()
		if err != nil {
			common.Log.Warningf("Failed to resolve peer url for network node %s; %s", n.ID, err.Error())
			return err
		}

		orchestrationAPI, err = n.orchestrationAPIClient()
		if err != nil {
			err := fmt.Errorf("Failed to resolve peer url for network node %s; %s", n.ID, err.Error())
			common.Log.Warningf(err.Error())
			return err
		}

		peerURL, err = p2pAPI.ResolvePeerURL()
		if err != nil {
			common.Log.Debugf("No peer url or equivalent resolved for network node %s; %s", n.ID, err.Error())
		}

		if peerURL == nil {
			logs, err := orchestrationAPI.GetContainerLogEvents(id, nil, true, nil, nil, nil, nil)
			if err == nil && logs != nil {
				for i := range logs.Events {
					event := logs.Events[i]
					if event.Message != nil {
						msg := string(*event.Message)
						peerURL, err = p2pAPI.ParsePeerURL(msg)
						if err == nil && peerURL != nil {
							if n.IPv4 != nil && n.PrivateIPv4 != nil {
								url := strings.Replace(*peerURL, *n.PrivateIPv4, *n.IPv4, 1)
								peerURL = &url
							}
							cfg[nodeConfigPeerURL] = peerURL
							break
						}
					}
				}
			}
		}
	}

	if peerURL == nil {
		if time.Now().Sub(n.CreatedAt) >= resolvePeerTickerTimeout {
			desc := fmt.Sprintf("Failed to resolve peer url for network node %s after %v", n.ID.String(), resolvePeerTickerTimeout)
			n.updateStatus(db, "failed", &desc)
			common.Log.Warning(desc)
			return fmt.Errorf(desc)
		}

		return fmt.Errorf("Failed to resolve peer url for network node with id: %s", n.ID)
	}

	common.Log.Debugf("Resolved peer url for network node with id: %s; peer url: %s", n.ID, *peerURL)
	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
	db.Save(&n)

	if role == nodeRolePeer || role == nodeRoleFull || role == nodeRoleValidator {
		go network.resolveAndBalanceJSONRPCAndWebsocketURLs(db, n)
		// TODO: determine if the node is running IPFS; if so: go network.resolveAndBalanceIPFSUrls(db, n)
	}

	return nil
}

func (n *Node) undeploy() error {
	common.Log.Debugf("Attempting to undeploy network node with id: %s", n.ID)

	db := dbconf.DatabaseConnection()
	n.updateStatus(db, "deprovisioning", nil)

	cfg := n.ParseConfig()
	_, regionOk := cfg[nodeConfigRegion].(string)
	taskIds, taskIdsOk := cfg[nodeConfigTargetTaskIDs].([]interface{})

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to undeploy network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if regionOk && taskIdsOk {
		for i := range taskIds {
			taskID := taskIds[i].(string)

			_, err := orchestrationAPI.StopContainer(taskID, nil)
			if err == nil {
				common.Log.Debugf("Terminated container with id: %s", taskID)
				n.Status = common.StringOrNil("terminated")
				db.Save(n)
				n.unbalance(db)
			} else {
				err = fmt.Errorf("Failed to terminate docker container with id: %s; %s", taskID, err.Error())
				common.Log.Warning(err.Error())
				return err
			}
		}

		// FIXME-- move the following security group removal to an async NATS operation
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
						common.Log.Warningf("Failed to unregister security groups for network node with id: %s; timing out after %v...", n.ID, securityGroupTerminationTickerTimeout)
						ticker.Stop()
						return
					}
				}
			}
		}()
	}

	return nil
}

func (n *Node) unbalance(db *gorm.DB) error {
	loadBalancers := make([]*LoadBalancer, 0)
	db.Model(&n).Association("LoadBalancers").Find(&loadBalancers)
	for _, balancer := range loadBalancers {
		common.Log.Debugf("Attempting to unbalance network node %s on load balancer: %s", n.ID, balancer.ID)
		msg, _ := json.Marshal(map[string]interface{}{
			"load_balancer_id": balancer.ID.String(),
			"node_id":          n.ID.String(),
		})
		return natsutil.NatsStreamingPublish(natsLoadBalancerUnbalanceNodeSubject, msg)
	}
	return nil
}

func (n *Node) unregisterSecurityGroups() error {
	common.Log.Debugf("Attempting to unregister security groups for network node with id: %s", n.ID)

	cfg := n.ParseConfig()
	_, regionOk := cfg[nodeConfigRegion].(string)
	securityGroupIds, securityGroupIdsOk := cfg["target_security_group_ids"].([]interface{})

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to unregistry security groups for network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if regionOk && securityGroupIdsOk {
		for i := range securityGroupIds {
			securityGroupID := securityGroupIds[i].(string)

			_, err := orchestrationAPI.DeleteSecurityGroup(securityGroupID)
			if err != nil {
				common.Log.Warningf("Failed to unregister security group for network node with id: %s; security group id: %s", n.ID, securityGroupID)
				return err
			}
		}
	}

	return nil
}

// orchestrationAPIClient returns an instance of the node's underlying orchestration.API
func (n *Node) orchestrationAPIClient() (orchestration.API, error) {
	cfg := n.ParseConfig()
	encryptedCfg, _ := n.DecryptedConfig()

	targetID, targetIDOk := cfg[nodeConfigTargetID].(string)
	region, regionOk := cfg[nodeConfigRegion].(string)
	credentials, credsOk := encryptedCfg[nodeConfigCredentials].(map[string]interface{})
	if !targetIDOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider for network node: %s", n.ID)
	}
	if !regionOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider region for network node: %s", n.ID)
	}
	if !credsOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider credentials for network node: %s", n.ID)
	}

	var apiClient orchestration.API

	switch targetID {
	case orchestration.ProviderAWS:
		apiClient = orchestration.InitAWSOrchestrationProvider(credentials, region)
	case orchestration.ProviderAzure:
		apiClient = orchestration.InitAzureOrchestrationProvider(credentials, region)
	case orchestration.ProviderGoogle:
		// apiClient = InitGoogleOrchestrationProvider(credentials, region)
		return nil, fmt.Errorf("Google Cloud orchestration provider not yet implemented")
	default:
		return nil, fmt.Errorf("Failed to resolve orchestration provider for network node %s", n.ID)
	}

	return apiClient, nil
}

// p2pAPIClient returns an instance of the node's underlying p2p.API
func (n *Node) p2pAPIClient() (p2p.API, error) {
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
		apiClient = p2p.InitGethP2PProvider(rpcURL, n.Network)
	case p2p.ProviderParity:
		apiClient = p2p.InitParityP2PProvider(rpcURL, n.Network)
	case p2p.ProviderQuorum:
		apiClient = p2p.InitQuorumP2PProvider(rpcURL, n.Network)
	default:
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node %s; unsupported client", n.ID)
	}

	return apiClient, nil
}

func (n *Node) addPeer(peerURL string) error {
	apiClient, err := n.p2pAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to add peer; %s", err.Error())
		return err
	}
	return apiClient.AddPeer(peerURL)
}

func (n *Node) removePeer(peerURL string) error {
	apiClient, err := n.p2pAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to remove peer; %s", err.Error())
		return err
	}
	return apiClient.RemovePeer(peerURL)
}

func (n *Node) acceptNonReservedPeers() error {
	apiClient, err := n.p2pAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to accept non-reserved peers; %s", err.Error())
		return err
	}
	return apiClient.AcceptNonReservedPeers()
}

func (n *Node) dropNonReservedPeers() error {
	apiClient, err := n.p2pAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to drop non-reserved peers; %s", err.Error())
		return err
	}
	return apiClient.DropNonReservedPeers()
}

func (n *Node) upgrade() error {
	apiClient, err := n.p2pAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to execute upgrade; %s", err.Error())
		return err
	}
	return apiClient.Upgrade()
}

// TODO: move this elsewhere
func dockerhubRepoExists(name string) (*string, error) {
	if name == "" {
		err := errors.New("invalid dockerhub repo name (empty string)")
		common.Log.Warning(err.Error())
		return nil, err
	}

	if common.DefaultDockerhubOrganization != nil && name != *common.DefaultDockerhubOrganization && !strings.HasPrefix(name, *common.DefaultDockerhubOrganization) { // reentrancy check
		repo := fmt.Sprintf("%s/%s", *common.DefaultDockerhubOrganization, strings.ReplaceAll(name, "/", "-"))
		preferredRepo, err := dockerhubRepoExists(repo)
		if err == nil {
			common.Log.Debugf("short-circuiting dockerhub repo resolution; preferred organization hosts repo: %s", repo)
			return preferredRepo, nil
		}
		common.Log.Debugf("preferred dockerhub organization does not currently host repo: %s", repo)
	}

	idx := strings.Index(name, "/")
	if idx != -1 {
		addr := fmt.Sprintf("%s:443", name[0:idx])
		conn, err := net.DialTimeout("tcp", addr, dockerRepoReachabilityTimeout)
		if err == nil {
			defer conn.Close()
			common.Log.Debugf("short-circuiting dockerhub repo resolution; third-party hosted repo is reachable: %s", name)
			return &name, nil
		}
		common.Log.Debugf("docker repo was not resolved to a third-party host: %s", name)
	}

	dockerhubClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy:             http.ProxyFromEnvironment,
		},
	}

	resp, err := dockerhubClient.Get(fmt.Sprintf("%s/%s", defaultDockerhubBaseURL, name))
	if err != nil {
		common.Log.Warningf("failed to query dockerhub for existance of repo: %s; %s", name, err.Error())
		return nil, err
	} else if resp.StatusCode >= 400 {
		var err error
		if resp.StatusCode == 404 {
			err = fmt.Errorf("docker repository was not resolved: %s", name)
		} else {
			err = fmt.Errorf("failed to query dockerhub for existance of repo: %s; status code: %d", name, resp.StatusCode)
		}
		common.Log.Warning(err.Error())
		return nil, err
	}

	return &name, nil
}
