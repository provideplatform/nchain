package network

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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

const nodeReachabilityTimeout = time.Millisecond * 2500

const resolveGenesisTickerInterval = time.Millisecond * 10000
const resolveGenesisTickerTimeout = time.Minute * 20
const resolveHostTickerInterval = time.Millisecond * 5000
const resolveHostTickerTimeout = time.Minute * 10
const resolvePeerTickerInterval = time.Millisecond * 5000
const resolvePeerTickerTimeout = time.Minute * 20
const securityGroupTerminationTickerInterval = time.Millisecond * 30000
const securityGroupTerminationTickerTimeout = time.Minute * 10

const defaultClient = "parity"

const nodeRoleBlockExplorer = "explorer"
const nodeRoleFull = "full"
const nodeRolePeer = "peer"
const nodeRoleValidator = "validator"
const nodeRoleIPFS = "ipfs"

var engineToNodeClientEnvMapping = map[string]string{"aura": "parity", "handshake": "handshake"}

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

func (n *Node) decryptedConfig() (map[string]interface{}, error) {
	decryptedParams := map[string]interface{}{}
	if n.EncryptedConfig != nil {
		encryptedConfigJSON, err := pgputil.PGPPubDecrypt([]byte(*n.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to decrypt encrypted network node config; %s", err.Error())
			return decryptedParams, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &decryptedParams)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted network node config; %s", err.Error())
			return decryptedParams, err
		}
	}
	return decryptedParams, nil
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

func (n *Node) setEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	n.EncryptedConfig = &_paramsJSON
	n.encryptConfig()
}

func (n *Node) sanitizeConfig() {
	cfg := n.ParseConfig()

	encryptedCfg, err := n.decryptedConfig()
	if err != nil {
		encryptedCfg = map[string]interface{}{}
	}

	if credentials, credentialsOk := cfg["credentials"]; credentialsOk {
		encryptedCfg["credentials"] = credentials
		delete(cfg, "credentials")
	}

	if env, envOk := cfg["env"].(map[string]interface{}); envOk {
		encryptedEnv, encryptedEnvOk := encryptedCfg["env"].(map[string]interface{})
		if !encryptedEnvOk {
			encryptedEnv = map[string]interface{}{}
			encryptedCfg["env"] = encryptedEnv
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

	n.setConfig(cfg)
	n.setEncryptedConfig(encryptedCfg)
}

// Create and persist a new network node
func (n *Node) Create() bool {
	if !n.Validate() {
		return false
	}

	n.sanitizeConfig()

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
					"network_node_id": n.ID.String(),
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

	n.sanitizeConfig()

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

// setConfig sets the network config in-memory
func (n *Node) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	_cfgJSON := json.RawMessage(cfgJSON)
	n.Config = &_cfgJSON
}

func (n *Node) peerURL() *string {
	cfg := n.ParseConfig()
	if peerURL, peerURLOk := cfg["peer_url"].(string); peerURLOk {
		return common.StringOrNil(peerURL)
	}

	return nil
}

// privateConfig returns a merged version of the config and encrypted config
func (n *Node) privateConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if n.Config != nil {
		err := json.Unmarshal(*n.Config, &config)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal network node config; %s", err.Error())
			return nil
		}
	}
	encryptedConfig, _ := n.decryptedConfig()
	for k := range encryptedConfig {
		config[k] = encryptedConfig[k]
	}
	return config
}

func (n *Node) rpcPort() uint {
	cfg := n.ParseConfig()
	defaultJSONRPCPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultJSONRPCPort = common.EngineToDefaultJSONRPCPortMapping[engineID]
	}
	port := uint(defaultJSONRPCPort)
	if jsonRPCPortOverride, jsonRPCPortOverrideOk := cfg["default_json_rpc_port"].(float64); jsonRPCPortOverrideOk {
		port = uint(jsonRPCPortOverride)
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
	return common.StringOrNil(fmt.Sprintf("http://%s:%d", *n.Host, port)) // FIXME-- allow specification of url scheme in cfg := n.ParseConfig()
}

func (n *Node) reachableViaJSONRPC() (bool, uint) {
	port := n.rpcPort()
	return n.reachableOnPort(port), port
}

func (n *Node) reachableViaWebsocket() (bool, uint) {
	cfg := n.ParseConfig()
	defaultWebsocketPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultWebsocketPort = common.EngineToDefaultWebsocketPortMapping[engineID]
	}
	port := uint(defaultWebsocketPort)
	if websocketPortOverride, websocketPortOverrideOk := cfg["default_websocket_port"].(float64); websocketPortOverrideOk {
		port = uint(websocketPortOverride)
	}

	return n.reachableOnPort(port), port
}

func (n *Node) reachableOnPort(port uint) bool {
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
	// if _, protocolOk := cfg["protocol_id"].(string); !protocolOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Failed to parse protocol_id in network node configuration"),
	// 	})
	// }
	// if _, engineOk := cfg["engine_id"].(string); !engineOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Failed to parse engine_id in network node configuration"),
	// 	})
	// }
	// if targetID, targetOk := cfg["target_id"].(string); targetOk {
	// 	if creds, credsOk := cfg["credentials"].(map[string]interface{}); credsOk {
	// 		if strings.ToLower(targetID) == "aws" {
	// 			if _, accessKeyIdOk := creds["aws_access_key_id"].(string); !accessKeyIdOk {
	// 				n.Errors = append(n.Errors, &provide.Error{
	// 					Message: common.StringOrNil("Failed to parse aws_access_key_id in network node credentials configuration for AWS target"),
	// 				})
	// 			}
	// 			if _, secretAccessKeyOk := creds["aws_secret_access_key"].(string); !secretAccessKeyOk {
	// 				n.Errors = append(n.Errors, &provide.Error{
	// 					Message: common.StringOrNil("Failed to parse aws_secret_access_key in network node credentials configuration for AWS target"),
	// 				})
	// 			}
	// 		}
	// 	} else {
	// 		n.Errors = append(n.Errors, &provide.Error{
	// 			Message: common.StringOrNil("Failed to parse credentials in network node configuration"),
	// 		})
	// 	}
	// } else {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Failed to parse target_id in network node configuration"),
	// 	})
	// }
	// if _, providerOk := cfg["provider_id"].(string); !providerOk {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Failed to parse provider_id in network node configuration"),
	// 	})
	// }
	if role, roleOk := cfg["role"].(string); roleOk {
		if n.Role == nil || *n.Role != role {
			common.Log.Debugf("Coercing network node role to match node configuration; role: %s", role)
			n.Role = common.StringOrNil(role)
		}
	}
	// } else {
	// 	n.Errors = append(n.Errors, &provide.Error{
	// 		Message: common.StringOrNil("Failed to parse role in network node configuration"),
	// 	})
	// }

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
		"network_node_id": n.ID.String(),
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

	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	_, regionOk := cfg["region"].(string)

	if !targetOk || !providerOk {
		return nil, fmt.Errorf("Cannot retrieve logs for network node without a target and provider configuration; target id: %s; provider id: %s", targetID, providerID)
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

		if providerID, providerIDOk := cfg["provider_id"].(string); providerIDOk {

			if strings.ToLower(providerID) == "docker" {
				if ids, idsOk := cfg["target_task_ids"].([]interface{}); idsOk {
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
			}

			return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; unsupported AWS provider: %s", *network.Name, providerID)
		}
	} else {
		return nil, fmt.Errorf("Unable to retrieve logs for network node: %s; no region provided: %s", *network.Name, providerID)
	}

	return nil, fmt.Errorf("Unable to retrieve logs for network node on unsupported network: %s", *network.Name)
}

func (n *Node) deploy(db *gorm.DB) error {
	if n.Config == nil {
		msg := fmt.Sprintf("Not attempting to deploy network node without a valid configuration; network node id: %s", n.ID)
		common.Log.Warning(msg)
		return errors.New(msg)
	}

	var network = &Network{}
	db.Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		msg := fmt.Sprintf("Failed to retrieve network for network node: %s", n.ID)
		n.updateStatus(db, "failed", &msg)
		common.Log.Warning(msg)
		return errors.New(msg)
	}

	common.Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, network.ID)
	n.updateStatus(db, "pending", nil)

	cfg := n.ParseConfig()
	encryptedCfg, err := n.decryptedConfig()
	if err != nil {
		return fmt.Errorf("Failed to decrypt config for network node: %s", n.ID)
	}

	env, envOk := cfg["env"].(map[string]interface{})
	encryptedEnv, encryptedEnvOk := encryptedCfg["env"].(map[string]interface{})

	// TODO: move the following to P2PAPI.requireBootnotes()
	bootnodes, err := network.requireBootnodes(db, n)
	if err != nil {
		switch err.(type) {
		case bootnodesInitialized:
			common.Log.Debugf("Bootnode initialized for network: %s; node: %s; waiting for genesis to complete and peer resolution to become possible", *network.Name, n.ID.String())
			if protocol, protocolOk := cfg["protocol_id"].(string); protocolOk {
				if strings.ToLower(protocol) == "poa" {
					if envOk && encryptedEnvOk {
						var addr *string
						var privateKey *ecdsa.PrivateKey
						_, masterOfCeremonyPrivateKeyOk := encryptedEnv["ENGINE_SIGNER_PRIVATE_KEY"].(string)
						if masterOfCeremony, masterOfCeremonyOk := env["ENGINE_SIGNER"].(string); masterOfCeremonyOk && !masterOfCeremonyPrivateKeyOk {
							addr = common.StringOrNil(masterOfCeremony)
							out := []string{}
							db.Table("wallets").Select("private_key").Where("wallets.user_id = ? AND wallets.address = ?", n.UserID.String(), addr).Pluck("private_key", &out)
							if out == nil || len(out) == 0 || len(out[0]) == 0 {
								common.Log.Warningf("Failed to retrieve manage engine signing identity for network: %s; generating unmanaged identity...", *network.Name)
								addr, privateKey, err = provide.EVMGenerateKeyPair()
							} else {
								encryptedKey := common.StringOrNil(out[0])
								privateKey, err = common.DecryptECDSAPrivateKey(*encryptedKey)
								if err == nil {
									common.Log.Debugf("Decrypted private key for master of ceremony on network: %s", *network.Name)
								} else {
									msg := fmt.Sprintf("Failed to decrypt private key for master of ceremony on network: %s", *network.Name)
									common.Log.Warning(msg)
									return errors.New(msg)
								}
							}
						} else if !masterOfCeremonyPrivateKeyOk {
							common.Log.Debugf("Generating managed master of ceremony signing identity for network: %s", *network.Name)
							addr, privateKey, err = provide.EVMGenerateKeyPair()
						}

						if addr != nil && privateKey != nil {
							keystoreJSON, err := provide.EVMMarshalEncryptedKey(ethcommon.HexToAddress(*addr), privateKey, hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
							if err == nil {
								common.Log.Debugf("Master of ceremony has initiated the initial key ceremony: %s; network: %s", *addr, *network.Name)
								env["ENGINE_SIGNER"] = addr
								encryptedEnv["ENGINE_SIGNER_PRIVATE_KEY"] = hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
								encryptedEnv["ENGINE_SIGNER_KEY_JSON"] = string(keystoreJSON)

								n.setConfig(cfg)
								n.setEncryptedConfig(encryptedCfg)
								n.sanitizeConfig()
								db.Save(&n)

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
												chainspecJSON = []byte(strings.Replace(string(chainspecJSON), strings.ToLower(templateMasterOfCeremony[2:]), strings.ToLower(string(*addr)[2:]), -1))
												var newChainspec map[string]interface{}
												err = json.Unmarshal(chainspecJSON, &newChainspec)
												if err == nil {
													networkCfg["chainspec"] = newChainspec
													network.setConfig(networkCfg)
													db.Save(&network)
												}
											}
										}
									}
								}
							} else {
								common.Log.Warningf("Failed to generate master of ceremony address for network: %s; %s", *network.Name, err.Error())
							}
						}
					}
				}
			}
			return n._deploy(network, bootnodes, db)
		default:
			msg := fmt.Sprintf("Failed to deploy node %s to network: %s", n.ID, *network.Name)
			common.Log.Warning(msg)
			return errors.New(msg)
		}
	} else {
		if p2p, p2pOk := cfg["p2p"].(bool); p2pOk {
			if p2p {
				return n.requireGenesis(network, bootnodes, db)
			}
			return n._deploy(network, bootnodes, db)
		}
		return n.requireGenesis(network, bootnodes, db) // default assumes p2p
	}
}

func (n *Node) requireGenesis(network *Network, bootnodes []*Node, db *gorm.DB) error {
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
	encryptedCfg, _ := n.decryptedConfig()
	networkCfg := network.ParseConfig()

	cfg["default_json_rpc_port"] = networkCfg["default_json_rpc_port"]
	cfg["default_websocket_port"] = networkCfg["default_websocket_port"]

	containerID, containerOk := cfg["container"].(string)
	image, imageOk := cfg["image"].(string)
	// resources, resourcesOk := cfg["resources"].(map[string]interface{})
	// script, scriptOk := cfg["script"].(map[string]interface{})
	targetID, targetOk := cfg["target_id"].(string)
	engineID, engineOk := cfg["engine_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	role, roleOk := cfg["role"].(string)
	region, regionOk := cfg["region"].(string)
	vpc, _ := cfg["vpc_id"].(string)
	env, envOk := cfg["env"].(map[string]interface{})
	encryptedEnv, encryptedEnvOk := encryptedCfg["env"].(map[string]interface{})
	securityCfg, securityCfgOk := cfg["security"].(map[string]interface{})

	if networkEnv, networkEnvOk := networkCfg["env"].(map[string]interface{}); envOk && networkEnvOk {
		common.Log.Debugf("Applying environment overrides to network node per network env configuration")
		for k := range networkEnv {
			env[k] = networkEnv[k]
		}
	}

	var providerCfgByRegion map[string]interface{}

	cloneableCfg, cloneableCfgOk := networkCfg["cloneable_cfg"].(map[string]interface{})
	if cloneableCfgOk {
		if !securityCfgOk {
			cloneableSecurityCfg, cloneableSecurityCfgOk := cloneableCfg["security"].(map[string]interface{})
			if !cloneableSecurityCfgOk {
				desc := fmt.Sprintf("Failed to parse cloneable security configuration for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return errors.New(desc)
			}
			securityCfg = cloneableSecurityCfg
		}

		if !containerOk && !imageOk {
			cloneableTarget, cloneableTargetOk := cloneableCfg[targetID].(map[string]interface{})
			if !cloneableTargetOk {
				desc := fmt.Sprintf("Failed to parse cloneable target configuration for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return errors.New(desc)
			}

			cloneableProvider, cloneableProviderOk := cloneableTarget[providerID].(map[string]interface{})
			if !cloneableProviderOk {
				desc := fmt.Sprintf("Failed to parse cloneable provider configuration for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return errors.New(desc)
			}

			cloneableProviderCfgByRegion, cloneableProviderCfgByRegionOk := cloneableProvider["regions"].(map[string]interface{})
			if !cloneableProviderCfgByRegionOk && !regionOk {
				desc := fmt.Sprintf("Failed to parse cloneable provider configuration by region (or a single specific deployment region) for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warningf(desc)
				return errors.New(desc)
			}
			providerCfgByRegion = cloneableProviderCfgByRegion
		}
	}

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to deploy network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if imageOk && containerOk {
		desc := fmt.Sprintf("Failed to deploy node in region %s; network node id: %s; both an image and container were specified; only one should be used", region, n.ID.String())
		n.updateStatus(db, "failed", &desc)
		common.Log.Warning(desc)
		return errors.New(desc)
	}

	if targetOk && engineOk && providerOk && roleOk && regionOk {
		securityGroupDesc := fmt.Sprintf("security group for network node: %s", n.ID.String())
		securityGroupIds, err := orchestrationAPI.CreateSecurityGroup(securityGroupDesc, securityGroupDesc, nil, securityCfg)
		if err != nil {
			n.updateStatus(db, "failed", common.StringOrNil(err.Error()))
			return err
		}

		cfg["security"] = securityCfg
		cfg["region"] = region
		cfg["target_security_group_ids"] = securityGroupIds
		n.setConfig(cfg)

		if strings.ToLower(providerID) == "docker" {
			common.Log.Debugf("Attempting to deploy network node container(s) in EC2 region: %s", region)
			var imageRef *string
			var containerRef *string

			if imageOk {
				imageRef = common.StringOrNil(image)
				common.Log.Debugf("Resolved container image to deploy in region %s; image: %s", region, *imageRef)
			} else if containerOk { // HACK -- deprecate container in favor of image
				containerRef = common.StringOrNil(containerID)
				common.Log.Debugf("Resolved container to deploy in region %s; ref: %s", region, *containerRef)
			} else if containerRolesByRegion, containerRolesByRegionOk := providerCfgByRegion[region].(map[string]interface{}); containerRolesByRegionOk {
				common.Log.Debugf("Resolved deployable containers by region in EC2 region: %s", region)
				if container, containerOk := containerRolesByRegion[role].(string); containerOk {
					containerRef = common.StringOrNil(container)
				}
			} else {
				common.Log.Warningf("Failed to resolve deployable container(s) by region in EC2 region: %s", region)
			}

			if imageRef != nil || containerRef != nil {
				common.Log.Debugf("Attempting to deploy image %s in EC2 region: %s", *imageRef, region)
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

				networkClient, networkClientOk := networkCfg["client"].(string)
				if _, clientOk := envOverrides["CLIENT"].(string); !clientOk {
					if networkClientOk {
						envOverrides["CLIENT"] = networkClient
					} else {
						if defaultClientEnv, defaultClientEnvOk := engineToNodeClientEnvMapping[engineID]; defaultClientEnvOk {
							envOverrides["CLIENT"] = defaultClientEnv
						} else {
							envOverrides["CLIENT"] = defaultClient
						}
					}
				} else if networkClientOk {
					client := envOverrides["CLIENT"].(string)
					if client != networkClient {
						common.Log.Warningf("Overridden client %s did not match network client %s; network id: %s", client, networkClient, network.ID)
					}
				}

				networkChain, networkChainOk := networkCfg["chain"].(string)
				if _, chainOk := envOverrides["CHAIN"].(string); !chainOk {
					if networkChainOk {
						envOverrides["CHAIN"] = networkChain
					}
				} else if networkChainOk {
					chain := envOverrides["CHAIN"].(string)
					if chain != networkChain {
						common.Log.Warningf("Overridden chain %s did not match network chain %s; network id: %s", chain, networkChain, network.ID)
					}
				}

				overrides := map[string]interface{}{
					"environment": envOverrides,
				}
				cfg["env"] = envOverrides

				n.setConfig(cfg)
				n.sanitizeConfig()
				db.Save(&n)

				containerSecurity := map[string]interface{}{} // for now, this should only be populated when imageRef != nil (awswrapper does not yet support providing security cfg when a task def is provided...)
				if imageRef != nil {
					containerSecurity = securityCfg
				}

				taskIds, err := orchestrationAPI.StartContainer(imageRef, containerRef, nil, nil, common.StringOrNil(vpc), securityGroupIds, []string{}, overrides, containerSecurity)
				if imageRef != nil {
					common.Log.Warningf("FIXME-- leaking the task definition that was used to start this container... %s", taskIds[0])
				}

				if err != nil || len(taskIds) == 0 {
					desc := fmt.Sprintf("Attempt to deploy container %s in EC2 %s region failed; %s", *imageRef, region, err.Error())
					n.updateStatus(db, "failed", &desc)
					n.unregisterSecurityGroups()
					common.Log.Warning(desc)
					return errors.New(desc)
				}
				common.Log.Debugf("Attempt to deploy container %s in EC2 %s region successful; task ids: %s", *imageRef, region, taskIds)
				cfg["target_task_ids"] = taskIds
				n.setConfig(cfg)
				n.sanitizeConfig()
				db.Save(&n)

				msg, _ := json.Marshal(map[string]interface{}{
					"network_node_id": n.ID.String(),
				})
				natsutil.NatsStreamingPublish(natsResolveNodeHostSubject, msg)
				natsutil.NatsStreamingPublish(natsResolveNodePeerURLSubject, msg)
			}
		}
	}

	return nil
}

func (n *Node) resolveHost(db *gorm.DB) error {
	network := n.relatedNetwork(db)
	if network == nil {
		return fmt.Errorf("Failed to resolve host for network node %s; no network resolved", n.ID)
	}

	cfg := n.ParseConfig()
	taskIds, taskIdsOk := cfg["target_task_ids"].([]interface{})

	if !taskIdsOk {
		return fmt.Errorf("Failed to resolve host for network node %s; no target_task_ids provided", n.ID)
	}

	identifiers := make([]string, len(taskIds))
	for _, id := range taskIds {
		identifiers = append(identifiers, id.(string))
	}

	if len(identifiers) == 0 {
		return fmt.Errorf("Unable to resolve network node host without any node identifiers")
	}

	id := identifiers[len(identifiers)-1]
	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	_, regionOk := cfg["region"].(string)

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to resolve host for network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if strings.ToLower(targetID) == "aws" && targetOk && providerOk && regionOk {
		if strings.ToLower(providerID) == "docker" {
			containerDetails, err := orchestrationAPI.GetContainerDetails(id, nil)
			if err == nil {
				if len(containerDetails.Tasks) > 0 {
					task := containerDetails.Tasks[0] // FIXME-- should this support exposing all tasks?
					taskStatus := ""
					if task.LastStatus != nil {
						taskStatus = strings.ToLower(*task.LastStatus)
					}
					if taskStatus == "running" && len(task.Attachments) > 0 {
						attachment := task.Attachments[0]
						if attachment.Type != nil && *attachment.Type == "ElasticNetworkInterface" {
							for i := range attachment.Details {
								kvp := attachment.Details[i]
								if kvp.Name != nil && *kvp.Name == "networkInterfaceId" && kvp.Value != nil {
									interfaceDetails, err := orchestrationAPI.GetNetworkInterfaceDetails(*kvp.Value)
									if err == nil {
										if len(interfaceDetails.NetworkInterfaces) > 0 {
											common.Log.Debugf("Retrieved interface details for container instance: %s", interfaceDetails)
											n.PrivateIPv4 = interfaceDetails.NetworkInterfaces[0].PrivateIpAddress

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
		common.Log.Warningf("Failed to set node to only accept connections from reserved peers; %s", err.Error())
	}

	cfgJSON, _ := json.Marshal(cfg)
	*n.Config = json.RawMessage(cfgJSON)
	n.Status = common.StringOrNil("running")
	db.Save(&n)

	role, roleOk := cfg["role"].(string)
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
	taskIds, taskIdsOk := cfg["target_task_ids"].([]interface{})

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

	role, roleOk := cfg["role"].(string)
	if !roleOk || role != nodeRolePeer && role != nodeRoleFull && role != nodeRoleValidator {
		return nil
	}

	common.Log.Debugf("Attempting to resolve peer url for network node: %s", n.ID.String())

	var peerURL *string

	id := identifiers[len(identifiers)-1]
	engineID, engineOk := cfg["engine_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	_, regionOk := cfg["region"].(string)

	// TODO: use P2PAPI ResolvePeerURL()

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to resolve peer url for network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if engineOk && providerOk && regionOk {
		if strings.ToLower(providerID) == "docker" {
			logs, err := orchestrationAPI.GetContainerLogEvents(id, nil, true, nil, nil, nil, nil)
			if err == nil && logs != nil {
				for i := range logs.Events {
					event := logs.Events[i]
					if event.Message != nil {
						msg := string(*event.Message)

						if network.IsBcoinNetwork() {
							const bcoinPoolIdentitySearchString = "Pool identity key:"
							poolIdentityFoundIndex := strings.LastIndex(msg, bcoinPoolIdentitySearchString)
							if poolIdentityFoundIndex != -1 {
								defaultPeerListenPort := common.EngineToDefaultPeerListenPortMapping[engineID]
								poolIdentity := strings.TrimSpace(msg[poolIdentityFoundIndex+len(bcoinPoolIdentitySearchString) : len(msg)-1])
								node := fmt.Sprintf("%s@%s:%v", poolIdentity, *n.IPv4, defaultPeerListenPort)
								peerURL = &node
								cfg["peer_url"] = node
								cfg["peer_identity"] = poolIdentity
								break
							}
						} else if network.IsEthereumNetwork() {
							nodeInfo := &provide.EthereumJsonRpcResponse{}
							err := json.Unmarshal([]byte(msg), &nodeInfo)
							if err == nil && nodeInfo != nil {
								result, resultOk := nodeInfo.Result.(map[string]interface{})
								if resultOk {
									if enode, enodeOk := result["enode"].(string); enodeOk {
										peerURL = common.StringOrNil(enode)
										cfg["peer"] = result
										cfg["peer_url"] = enode
										break
									}
								}
							} else if err != nil {
								enodeIndex := strings.LastIndex(msg, "enode://")
								if enodeIndex != -1 {
									enode := msg[enodeIndex:]
									if n.IPv4 != nil && n.PrivateIPv4 != nil {
										enode = strings.Replace(enode, *n.PrivateIPv4, *n.IPv4, 1)
									}
									peerURL = common.StringOrNil(enode)
									cfg["peer_url"] = enode
									break
								}
							}
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
	providerID, providerOk := cfg["provider_id"].(string)
	_, regionOk := cfg["region"].(string)
	taskIds, taskIdsOk := cfg["target_task_ids"].([]interface{})

	orchestrationAPI, err := n.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to undeploy network node %s; %s", n.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if providerOk && regionOk {
		if strings.ToLower(providerID) == "docker" && taskIdsOk {
			for i := range taskIds {
				taskID := taskIds[i].(string)

				_, err := orchestrationAPI.StopContainer(taskID, nil)
				if err == nil {
					common.Log.Debugf("Terminated ECS docker container with id: %s", taskID)
					n.Status = common.StringOrNil("terminated")
					db.Save(n)
					n.unbalance(db)
				} else {
					err = fmt.Errorf("Failed to terminate ECS docker container with id: %s; %s", taskID, err.Error())
					common.Log.Warning(err.Error())
					return err
				}
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
			"network_node_id":  n.ID.String(),
		})
		return natsutil.NatsStreamingPublish(natsLoadBalancerUnbalanceNodeSubject, msg)
	}
	return nil
}

func (n *Node) unregisterSecurityGroups() error {
	common.Log.Debugf("Attempting to unregister security groups for network node with id: %s", n.ID)

	cfg := n.ParseConfig()
	_, regionOk := cfg["region"].(string)
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

// orchestrationAPIClient returns an instance of the node's underlying OrchestrationAPI
func (n *Node) orchestrationAPIClient() (OrchestrationAPI, error) {
	cfg := n.ParseConfig()
	encryptedCfg, _ := n.decryptedConfig()
	targetID, targetIDOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	credentials, credsOk := encryptedCfg["credentials"].(map[string]interface{})
	if !targetIDOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider for network node: %s", n.ID)
	}
	if !regionOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider region for network node: %s", n.ID)
	}
	if !credsOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider credentials for network node: %s", n.ID)
	}

	var apiClient OrchestrationAPI

	switch targetID {
	case awsOrchestrationProvider:
		apiClient = orchestration.InitAWSOrchestrationProvider(credentials, region)
	case azureOrchestrationProvider:
		// apiClient = InitAzureOrchestrationProvider(credentials, region)
		return nil, fmt.Errorf("Azure orchestration provider not yet implemented")
	case googleOrchestrationProvider:
		// apiClient = InitGoogleOrchestrationProvider(credentials, region)
		return nil, fmt.Errorf("Google orchestration provider not yet implemented")
	default:
		return nil, fmt.Errorf("Failed to resolve orchestration provider for network node %s", n.ID)
	}

	return apiClient, nil
}

// p2pAPIClient returns an instance of the node's underlying P2PAPI
func (n *Node) p2pAPIClient() (P2PAPI, error) {
	cfg := n.ParseConfig()
	client, clientOk := cfg["client"].(string)
	if !clientOk {
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node: %s", n.ID)
	}
	rpcURL := n.rpcURL()
	if rpcURL == nil {
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node: %s", n.ID)
	}

	var apiClient P2PAPI

	switch client {
	case bcoinP2PProvider:
		// apiClient = p2p.InitBcoinP2PProvider(*rpcURL)
		return nil, fmt.Errorf("Bcoin p2p provider not yet implemented")
	case parityP2PProvider:
		apiClient = p2p.InitParityP2PProvider(*rpcURL)
	case quorumP2PProvider:
		// apiClient = p2p.InitQuorumP2PProvider(*rpcURL)
		return nil, fmt.Errorf("Quorum p2p not yet implemented")
	default:
		return nil, fmt.Errorf("Failed to resolve p2p provider for network node %s", n.ID)
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
