package network

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/jinzhu/gorm"
	awswrapper "github.com/kthomas/go-aws-wrapper"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
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

var engineToNetworkNodeClientEnvMapping = map[string]string{"authorityRound": "parity", "handshake": "handshake"}

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&NetworkNode{})
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_network_id", "network_id")
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_user_id", "user_id")
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_application_id", "application_id")
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_role", "role")
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_status", "status")
	db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_bootnode", "bootnode")
	db.Model(&NetworkNode{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// NetworkNode instances represent nodes of the network to which they belong, acting in a specific role;
// each NetworkNode may have a set or sets of deployed resources, such as application containers, VMs
// or even phyiscal infrastructure
type NetworkNode struct {
	provide.Model
	NetworkID       uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	UserID          *uuid.UUID       `sql:"type:uuid" json:"user_id"`
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	Bootnode        bool             `sql:"not null;default:'false'" json:"is_bootnode"`
	Host            *string          `json:"host"`
	IPv4            *string          `json:"ipv4"`
	IPv6            *string          `json:"ipv6"`
	PrivateIPv4     *string          `json:"private_ipv4"`
	PrivateIPv6     *string          `json:"private_ipv6"`
	Description     *string          `json:"description"`
	Role            *string          `sql:"not null;default:'peer'" json:"role"`
	Status          *string          `sql:"not null;default:'init'" json:"status"`
	LoadBalancers   []LoadBalancer   `gorm:"many2many:load_balancers_network_nodes" json:"-"`
	Config          *json.RawMessage `sql:"type:json" json:"config"`
	EncryptedConfig *string          `sql:"type:bytea" json:"-"`
}

func (n *NetworkNode) decryptedConfig() (map[string]interface{}, error) {
	decryptedParams := map[string]interface{}{}
	if n.EncryptedConfig != nil {
		encryptedConfigJSON, err := common.PGPPubDecrypt(*n.EncryptedConfig, common.GpgPrivateKey, common.GpgPassword)
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

func (n *NetworkNode) encryptConfig() bool {
	if n.EncryptedConfig != nil {
		encryptedConfig, err := common.PGPPubEncrypt(*n.EncryptedConfig, common.GpgPublicKey)
		if err != nil {
			common.Log.Warningf("Failed to encrypt network node config; %s", err.Error())
			n.Errors = append(n.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
			return false
		}
		n.EncryptedConfig = encryptedConfig
	}
	return true
}

func (n *NetworkNode) setEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	n.EncryptedConfig = &_paramsJSON
	n.encryptConfig()
}

// Create and persist a new network node
func (n *NetworkNode) Create() bool {
	if !n.Validate() {
		return false
	}

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
		return common.StringOrNil(peerURL)
	}

	return nil
}

func (n *NetworkNode) reachableViaJSONRPC() (bool, uint) {
	cfg := n.ParseConfig()
	defaultJSONRPCPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultJSONRPCPort = common.EngineToDefaultJSONRPCPortMapping[engineID]
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
		defaultWebsocketPort = common.EngineToDefaultWebsocketPortMapping[engineID]
	}
	port := uint(defaultWebsocketPort)
	if websocketPortOverride, websocketPortOverrideOk := cfg["default_websocket_port"].(float64); websocketPortOverrideOk {
		port = uint(websocketPortOverride)
	}

	return n.reachableOnPort(port), port
}

func (n *NetworkNode) reachableOnPort(port uint) bool {
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

func (n *NetworkNode) relatedNetwork() *Network {
	var network = &Network{}
	dbconf.DatabaseConnection().Model(n).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		common.Log.Warningf("Failed to retrieve network for network node: %s", n.ID)
		return nil
	}
	return network
}

func (n *NetworkNode) updateStatus(db *gorm.DB, status string, description *string) {
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
func (n *NetworkNode) Validate() bool {
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
func (n *NetworkNode) ParseConfig() map[string]interface{} {
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
func (n *NetworkNode) Delete() bool {
	n.undeploy()
	msg, _ := json.Marshal(map[string]interface{}{
		"network_node_id": n.ID.String(),
	})
	natsConnection := common.GetDefaultNatsStreamingConnection()
	natsConnection.Publish(natsDeleteTerminatedNetworkNodeSubject, msg)
	return len(n.Errors) == 0
}

// Reload the underlying network node instance
func (n *NetworkNode) Reload() {
	db := dbconf.DatabaseConnection()
	db.Model(&n).Find(n)
}

// Logs exposes the paginated logstream for the underlying node
func (n *NetworkNode) Logs() (*[]string, error) {
	var network = &Network{}
	dbconf.DatabaseConnection().Model(n).Related(&network)
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

func (n *NetworkNode) deploy(db *gorm.DB) {
	if n.Config == nil {
		common.Log.Debugf("Not attempting to deploy network node without a valid configuration; network node id: %s", n.ID)
		return
	}

	go func() {
		var network = &Network{}
		db.Model(n).Related(&network)
		if network == nil || network.ID == uuid.Nil {
			desc := fmt.Sprintf("Failed to retrieve network for network node: %s", n.ID)
			n.updateStatus(db, "failed", &desc)
			common.Log.Warning(desc)
			return
		}

		common.Log.Debugf("Attempting to deploy network node with id: %s; network: %s", n.ID, network.ID)
		n.updateStatus(db, "pending", nil)

		cfg := n.ParseConfig()

		bootnodes, err := network.requireBootnodes(db, n)
		if err != nil {
			switch err.(type) {
			case bootnodesInitialized:
				common.Log.Debugf("Bootnode initialized for network: %s; node: %s; waiting for genesis to complete and peer resolution to become possible", *network.Name, n.ID.String())
				if protocol, protocolOk := cfg["protocol_id"].(string); protocolOk {
					if strings.ToLower(protocol) == "poa" {
						if env, envOk := cfg["env"].(map[string]interface{}); envOk {
							var addr *string
							var privateKey *ecdsa.PrivateKey
							_, masterOfCeremonyPrivateKeyOk := env["ENGINE_SIGNER_PRIVATE_KEY"].(string)
							if masterOfCeremony, masterOfCeremonyOk := env["ENGINE_SIGNER"].(string); masterOfCeremonyOk && !masterOfCeremonyPrivateKeyOk {
								addr = common.StringOrNil(masterOfCeremony)
								out := []string{}
								db.Table("wallets").Select("private_key").Where("wallets.user_id = ? AND wallets.address = ?", n.UserID.String(), addr).Pluck("private_key", &out)
								if out == nil || len(out) == 0 || len(out[0]) == 0 {
									common.Log.Warningf("Failed to retrieve manage engine signing identity for network: %s; generating unmanaged identity...", *network.Name)
									addr, privateKey, err = provide.EVMGenerateKeyPair()
								} else {
									encryptedKey := common.StringOrNil(out[0])
									privateKey, err = common.DecryptECDSAPrivateKey(*encryptedKey, common.GpgPrivateKey, common.GpgPassword)
									if err == nil {
										common.Log.Debugf("Decrypted private key for master of ceremony on network: %s", *network.Name)
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
													chainspecJSON = []byte(strings.Replace(string(chainspecJSON), strings.ToLower(templateMasterOfCeremony[2:]), strings.ToLower(string(*addr)[2:]), -1))
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
									common.Log.Warningf("Failed to generate master of ceremony address for network: %s; %s", *network.Name, err.Error())
								}
							}
						}
					}
				}
				n._deploy(network, bootnodes, db)
			}
		} else {
			if p2p, p2pOk := cfg["p2p"].(bool); p2pOk {
				if p2p {
					n.requireGenesis(network, bootnodes, db)
				} else {
					n._deploy(network, bootnodes, db)
				}
			} else {
				n.requireGenesis(network, bootnodes, db) // default assumes p2p
			}
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
				common.Log.Warning(desc)
				ticker.Stop()
				return
			}

			if daemon, daemonOk := currentNetworkStats[network.ID.String()]; daemonOk {
				if daemon.stats != nil {
					if daemon.stats.Block > 0 {
						common.Log.Warning("Deploying w/o network stats")
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

	containerID, containerOk := cfg["container"].(string)
	targetID, targetOk := cfg["target_id"].(string)
	engineID, engineOk := cfg["engine_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	role, roleOk := cfg["role"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	region, regionOk := cfg["region"].(string)
	vpc, _ := cfg["vpc_id"].(string)
	env, envOk := cfg["env"].(map[string]interface{})
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
				return
			}
			securityCfg = cloneableSecurityCfg
		}

		if !containerOk {
			cloneableTarget, cloneableTargetOk := cloneableCfg[targetID].(map[string]interface{})
			if !cloneableTargetOk {
				desc := fmt.Sprintf("Failed to parse cloneable target configuration for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return
			}

			cloneableProvider, cloneableProviderOk := cloneableTarget[providerID].(map[string]interface{})
			if !cloneableProviderOk {
				desc := fmt.Sprintf("Failed to parse cloneable provider configuration for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return
			}

			cloneableProviderCfgByRegion, cloneableProviderCfgByRegionOk := cloneableProvider["regions"].(map[string]interface{})
			if !cloneableProviderCfgByRegionOk && !regionOk {
				desc := fmt.Sprintf("Failed to parse cloneable provider configuration by region (or a single specific deployment region) for network node: %s", n.ID)
				n.updateStatus(db, "failed", &desc)
				common.Log.Warningf(desc)
				return
			}
			providerCfgByRegion = cloneableProviderCfgByRegion
		}
	}

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

			cfg["security"] = securityCfg
			cfg["region"] = region
			cfg["target_security_group_ids"] = securityGroupIds
			n.setConfig(cfg)

			if err != nil {
				desc := fmt.Sprintf("Failed to create security group in EC2 region %s; network node id: %s; %s", region, n.ID.String(), err.Error())
				n.updateStatus(db, "failed", &desc)
				common.Log.Warning(desc)
				return
			}

			if egress, egressOk := securityCfg["egress"]; egressOk {
				switch egress.(type) {
				case string:
					if egress.(string) == "*" {
						_, err := awswrapper.AuthorizeSecurityGroupEgressAllPortsAllProtocols(accessKeyID, secretAccessKey, region, *securityGroup.GroupId)
						if err != nil {
							common.Log.Warningf("Failed to authorize security group egress across all ports and protocols in EC2 %s region; security group id: %s; %s", region, *securityGroup.GroupId, err.Error())
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
							common.Log.Warningf("Failed to authorize security group egress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
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
							common.Log.Warningf("Failed to authorize security group ingress across all ports and protocols in EC2 %s region; security group id: %s; %s", region, *securityGroup.GroupId, err.Error())
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
							common.Log.Warningf("Failed to authorize security group ingress in EC2 %s region; security group id: %s; tcp ports: %s; udp ports: %s; %s", region, *securityGroup.GroupId, tcp, udp, err.Error())
						}
					}
				}
			}

			if strings.ToLower(providerID) == "docker" {
				common.Log.Debugf("Attempting to deploy network node container(s) in EC2 region: %s", region)
				var resolvedContainer *string

				if containerOk {
					resolvedContainer = common.StringOrNil(containerID)
				} else if containerRolesByRegion, containerRolesByRegionOk := providerCfgByRegion[region].(map[string]interface{}); containerRolesByRegionOk {
					common.Log.Debugf("Resolved deployable containers by region in EC2 region: %s", region)
					if container, containerOk := containerRolesByRegion[role].(string); containerOk {
						resolvedContainer = common.StringOrNil(container)
					}
				} else {
					common.Log.Warningf("Failed to resolve deployable container(s) by region in EC2 region: %s", region)
				}

				if resolvedContainer != nil {
					common.Log.Debugf("Resolved deployable container for role: %s; in EC2 region: %s; container: %s", role, region, resolvedContainer)
					common.Log.Debugf("Attempting to deploy container %s in EC2 region: %s", resolvedContainer, region)
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

					taskIds, err := awswrapper.StartContainer(accessKeyID, secretAccessKey, region, *resolvedContainer, nil, nil, common.StringOrNil(vpc), securityGroupIds, []string{}, overrides)

					if err != nil || len(taskIds) == 0 {
						desc := fmt.Sprintf("Attempt to deploy container %s in EC2 %s region failed; %s", *resolvedContainer, region, err.Error())
						n.updateStatus(db, "failed", &desc)
						n.unregisterSecurityGroups()
						common.Log.Warning(desc)
						return
					}
					common.Log.Debugf("Attempt to deploy container %s in EC2 %s region successful; task ids: %s", *resolvedContainer, region, taskIds)
					cfg["target_task_ids"] = taskIds
					n.setConfig(cfg)
					db.Save(n)

					n.resolveHost(db, network, cfg, taskIds)
					n.resolvePeerURL(db, network, cfg, taskIds)
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
					common.Log.Warning(desc)
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

					if strings.ToLower(providerID) == "docker" {
						containerDetails, err := awswrapper.GetContainerDetails(accessKeyID, secretAccessKey, region, id, nil)
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
												interfaceDetails, err := awswrapper.GetNetworkInterfaceDetails(accessKeyID, secretAccessKey, region, *kvp.Value)
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
			}

			if n.Host != nil {
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				n.Status = common.StringOrNil("running")
				db.Save(n)

				role, roleOk := cfg["role"].(string)
				if roleOk {
					if role == "explorer" {
						go network.resolveAndBalanceExplorerUrls(db, n)
					} else if role == "faucet" {
						common.Log.Warningf("Faucet role not yet supported")
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

	common.Log.Debugf("Attempting to resolve peer url for network node: %s", n.ID.String())

	ticker := time.NewTicker(resolvePeerTickerInterval)
	startedAt := time.Now()
	var peerURL *string
	for {
		select {
		case <-ticker.C:
			if peerURL == nil {
				if time.Now().Sub(startedAt) >= resolvePeerTickerTimeout {
					common.Log.Warningf("Failed to resolve peer url for network node: %s; timing out after %v", n.ID.String(), resolvePeerTickerTimeout)
					n.Status = common.StringOrNil("peer_resolution_failed")
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

					if strings.ToLower(providerID) == "docker" {
						logs, err := awswrapper.GetContainerLogEvents(accessKeyID, secretAccessKey, region, id, nil)
						if err == nil {
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
											ticker.Stop()
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
													ticker.Stop()
													break
												}
											}
										} else if err != nil {
											enodeIndex := strings.LastIndex(msg, "enode://")
											if enodeIndex != -1 {
												enode := msg[enodeIndex:]
												peerURL = common.StringOrNil(enode)
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
				common.Log.Debugf("Resolved peer url for network node with id: %s; peer url: %s", n.ID, *peerURL)
				cfgJSON, _ := json.Marshal(cfg)
				*n.Config = json.RawMessage(cfgJSON)
				db.Save(n)

				if role == "peer" || role == "full" || role == "validator" || role == "faucet" {
					go network.resolveAndBalanceJSONRPCAndWebsocketURLs(db, n)
				}

				ticker.Stop()
				return
			}
		}
	}
}

func (n *NetworkNode) undeploy() error {
	common.Log.Debugf("Attempting to undeploy network node with id: %s", n.ID)

	db := dbconf.DatabaseConnection()
	n.updateStatus(db, "deprovisioning", nil)

	cfg := n.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	providerID, providerOk := cfg["provider_id"].(string)
	region, regionOk := cfg["region"].(string)
	taskIds, taskIdsOk := cfg["target_task_ids"].([]interface{})
	credentials, credsOk := cfg["credentials"].(map[string]interface{})

	common.Log.Debugf("Configuration for network node undeploy: target id: %s; crendentials: %s; target task ids: %s", targetID, credentials, taskIds)

	if targetOk && providerOk && regionOk && credsOk {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if strings.ToLower(providerID) == "docker" && taskIdsOk {
				for i := range taskIds {
					taskID := taskIds[i].(string)

					_, err := awswrapper.StopContainer(accessKeyID, secretAccessKey, region, taskID, nil)
					if err == nil {
						common.Log.Debugf("Terminated ECS docker container with id: %s", taskID)
						n.Status = common.StringOrNil("terminated")
						db.Save(n)

						loadBalancers := make([]*LoadBalancer, 0)
						db.Model(&n).Association("LoadBalancers").Find(&loadBalancers)
						for _, balancer := range loadBalancers {
							common.Log.Debugf("Attempting to unbalance network node %s on load balancer: %s", n.ID, balancer.ID)
							msg, _ := json.Marshal(map[string]interface{}{
								"load_balancer_id": balancer.ID.String(),
								"network_node_id":  n.ID.String(),
							})
							natsConnection := common.GetDefaultNatsStreamingConnection()
							natsConnection.Publish(natsLoadBalancerUnbalanceNodeSubject, msg)
						}
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
	}

	return nil
}

func (n *NetworkNode) unregisterSecurityGroups() error {
	common.Log.Debugf("Attempting to unregister security groups for network node with id: %s", n.ID)

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
						common.Log.Warningf("Failed to unregister security group for network node with id: %s; security group id: %s", n.ID, securityGroupID)
						return err
					}
				}
			}
		}
	}

	return nil
}
