package network

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network/orchestration"
	provide "github.com/provideservices/provide-go"
)

const loadBalancerReachabilityTimeout = time.Millisecond * 2500
const loadBalancerTypeBlockExplorer = "explorer"
const loadBalancerTypeRPC = "rpc"
const loadBalancerTypeIPFS = "ipfs"

// LoadBalancer instances represent a physical or virtual load balancer of a specific type (i.e., JSON-RPC) which belongs to a network
type LoadBalancer struct {
	provide.Model
	NetworkID       uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id,omitempty"`
	Name            *string          `sql:"not null" json:"name"`
	Type            *string          `sql:"not null" json:"type"`
	Host            *string          `json:"host"`
	IPv4            *string          `json:"ipv4"`
	IPv6            *string          `json:"ipv6"`
	Description     *string          `json:"description"`
	Region          *string          `json:"region"`
	Status          *string          `sql:"not null;default:'provisioning'" json:"status"`
	Nodes           []Node           `gorm:"many2many:load_balancers_nodes" json:"-"`
	Config          *json.RawMessage `sql:"type:json" json:"config,omitempty"`
	EncryptedConfig *string          `sql:"type:bytea" json:"-"`
}

// LoadBalancerListQuery returns a DB query configured to select columns suitable for a paginated API response
func LoadBalancerListQuery() *gorm.DB {
	return dbconf.DatabaseConnection().Select("load_balancers.id, load_balancers.created_at, load_balancers.network_id, load_balancers.application_id, load_balancers.name, load_balancers.type, load_balancers.host, load_balancers.ipv4, load_balancers.ipv6, load_balancers.description, load_balancers.region, load_balancers.status, load_balancers.config")
}

func (l *LoadBalancer) decryptedConfig() (map[string]interface{}, error) {
	decryptedParams := map[string]interface{}{}
	if l.EncryptedConfig != nil {
		encryptedConfigJSON, err := pgputil.PGPPubDecrypt([]byte(*l.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to decrypt encrypted load balancer config; %s", err.Error())
			return decryptedParams, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &decryptedParams)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted load balancer config; %s", err.Error())
			return decryptedParams, err
		}
	}
	return decryptedParams, nil
}

func (l *LoadBalancer) encryptConfig() bool {
	if l.EncryptedConfig != nil {
		encryptedConfig, err := pgputil.PGPPubEncrypt([]byte(*l.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to encrypt load balancer config; %s", err.Error())
			l.Errors = append(l.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
			return false
		}
		l.EncryptedConfig = common.StringOrNil(string(encryptedConfig))
	}
	return true
}

func (l *LoadBalancer) setEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	l.EncryptedConfig = &_paramsJSON
	l.encryptConfig()
}

func (l *LoadBalancer) sanitizeConfig() {
	cfg := l.ParseConfig()

	encryptedCfg, err := l.decryptedConfig()
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

	l.setConfig(cfg)
	l.setEncryptedConfig(encryptedCfg)
}

// Create and persist a new load balancer
func (l *LoadBalancer) Create() bool {
	if l.Name == nil {
		lbUUID, _ := uuid.NewV4()
		l.Name = common.StringOrNil(fmt.Sprintf("%s", lbUUID.String()[0:31]))
	}

	if !l.Validate() {
		return false
	}

	l.sanitizeConfig()

	db := dbconf.DatabaseConnection()

	if db.NewRecord(l) {
		result := db.Create(&l)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				l.Errors = append(l.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(l) {
			success := rowsAffected > 0
			if success {
				msg, _ := json.Marshal(map[string]interface{}{
					"load_balancer_id": l.ID,
				})
				natsutil.NatsPublish(natsLoadBalancerProvisioningSubject, msg)
			}
			return success
		}
	}
	return false
}

// Update an existing load balancer
func (l *LoadBalancer) Update() bool {
	if !l.Validate() {
		return false
	}

	l.sanitizeConfig()

	db := dbconf.DatabaseConnection()

	result := db.Save(&l)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Errors = append(l.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}

	return len(l.Errors) == 0
}

// Delete a load balancer
func (l *LoadBalancer) Delete() bool {
	db := dbconf.DatabaseConnection()
	result := db.Delete(l)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Errors = append(l.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
	return len(l.Errors) == 0
}

// Validate a load balancer for persistence
func (l *LoadBalancer) Validate() bool {
	l.Errors = make([]*provide.Error, 0)
	return len(l.Errors) == 0
}

// ParseConfig - parse the persistent load balancer configuration JSON
func (l *LoadBalancer) ParseConfig() map[string]interface{} {
	config := map[string]interface{}{}
	if l.Config != nil {
		err := json.Unmarshal(*l.Config, &config)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal load balancer config; %s", err.Error())
			return nil
		}
	}
	return config
}

// ReachableOnPort returns true if the load balancer is available on the named port
func (l *LoadBalancer) ReachableOnPort(port uint) bool {
	var addr *string
	if l.IPv4 != nil {
		addr = l.IPv4
	} else if l.Host != nil {
		addr = l.Host
	}
	if addr == nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%v", *addr, port), loadBalancerReachabilityTimeout) // FIXME-- use configured protocol instead of assuming tcp
	if err == nil {
		common.Log.Debugf("%s:%v is reachable", *addr, port)
		defer conn.Close()
		return true
	}
	common.Log.Debugf("%s:%v is unreachable", *addr, port)
	return false
}

// orchestrationAPIClient returns an instance of the load balancer's underlying OrchestrationAPI
func (l *LoadBalancer) orchestrationAPIClient() (OrchestrationAPI, error) {
	cfg := l.ParseConfig()
	encryptedCfg, _ := l.decryptedConfig()
	targetID, targetIDOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	credentials, credsOk := encryptedCfg["credentials"].(map[string]interface{})
	if !targetIDOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider for load balancer: %s", l.ID)
	}
	if !regionOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider region for load balancer: %s", l.ID)
	}
	if !credsOk {
		return nil, fmt.Errorf("Failed to resolve orchestration provider credentials for load balancer: %s", l.ID)
	}

	var apiClient OrchestrationAPI

	switch targetID {
	case awsOrchestrationProvider:
		apiClient = orchestration.InitAWSOrchestrationProvider(credentials, region)
	case azureOrchestrationProvider:
		// apiClient = orchestration.InitAzureOrchestrationProvider(credentials)
		return nil, fmt.Errorf("Azure orchestration provider not yet implemented")
	case googleOrchestrationProvider:
		// apiClient = orchestration.InitGoogleOrchestrationProvider(credentials)
		return nil, fmt.Errorf("Google orchestration provider not yet implemented")
	default:
		return nil, fmt.Errorf("Failed to resolve orchestration provider for load balancer %s", l.ID)
	}

	return apiClient, nil
}

func (l *LoadBalancer) buildTargetGroupName(port int64) string {
	return ethcommon.Bytes2Hex(provide.Keccak256(fmt.Sprintf("%s-port-%v", l.ID.String(), port)))[0:31]
}

// Deprovision underlying infrastructure for the load balancer instance
func (l *LoadBalancer) Deprovision(db *gorm.DB) error {
	common.Log.Debugf("Attempting to deprovision infrastructure for load balancer with id: %s", l.ID)
	cfg := l.ParseConfig()
	l.updateStatus(db, "deprovisioning", l.Description)

	targetID, targetIDOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)

	if !targetIDOk || !regionOk || !arnOk {
		err := fmt.Errorf("Cannot deprovision load balancer for network node without target, region, credentials and target balancer id configuration; target: %s, region: %s, balancer: %s", targetID, region, targetBalancerArn)
		common.Log.Warningf(err.Error())
		return err
	}

	orchestrationAPI, err := l.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Cannot deprovision load balancer for network node; failed to resolve orchestration API client; %s", err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if region != "" {
		if strings.ToLower(targetID) == "aws" {
			_, err := orchestrationAPI.DeleteLoadBalancerV2(common.StringOrNil(targetBalancerArn))
			if err != nil {
				common.Log.Warningf("Failed to delete load balancer: %s; %s", *l.Name, err.Error())
				return err
			}

			if targetGroups, targetGroupsOk := cfg["target_groups"].(map[string]interface{}); targetGroupsOk {
				for _, targetGroupArn := range targetGroups {
					_, err := orchestrationAPI.DeleteTargetGroup(common.StringOrNil(targetGroupArn.(string)))
					if err != nil {
						common.Log.Warningf("Failed to delete load balanced target group: %s; %s", targetGroupArn, err.Error())
						return err
					}
				}
			}

			if securityGroupIds, securityGroupIdsOk := cfg["target_security_group_ids"].([]interface{}); securityGroupIdsOk {
				for _, securityGroupID := range securityGroupIds {
					_, err := orchestrationAPI.DeleteSecurityGroup(securityGroupID.(string))
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case "InvalidGroup.NotFound":
								common.Log.Debugf("Attempted to delete security group %s which does not exist for balancer: %s", securityGroupID.(string), l.ID)
							default:
								common.Log.Warningf("Failed to delete security group with id: %s; %s", securityGroupID.(string), err.Error())
								return err
							}
						}
					}
				}
			}

			if l.Delete() {
				common.Log.Debugf("Dropped load balancer: %s", l.ID)
			} else {
				common.Log.Warningf("Failed to drop load balancer: %s", l.ID)
			}
		}
	}

	return nil
}

// Provision underlying infrastructure for the load balancer instance
func (l *LoadBalancer) Provision(db *gorm.DB) error {
	common.Log.Debugf("Attempting to provision infrastructure for load balancer with id: %s", l.ID)
	n := &Network{}
	db.Model(&l).Related(&n)
	cfg := n.ParseConfig()

	balancerCfg := l.ParseConfig()

	var balancerType string
	if l.Type != nil {
		balancerType = *l.Type
	}

	defaultJSONRPCPort := uint(0)
	defaultWebsocketPort := uint(0)
	if engineID, engineOk := cfg["engine_id"].(string); engineOk {
		defaultJSONRPCPort = common.EngineToDefaultJSONRPCPortMapping[engineID]
		defaultWebsocketPort = common.EngineToDefaultWebsocketPortMapping[engineID]
	}

	jsonRPCPort := float64(defaultJSONRPCPort)
	websocketPort := float64(defaultWebsocketPort)

	var securityCfg map[string]interface{}
	if security, securityOk := balancerCfg["security"].(map[string]interface{}); securityOk {
		securityCfg = security
	} else if security, securityOk := cfg["security"].(map[string]interface{}); securityOk {
		securityCfg = security
	} else if cloneableCfg, cloneableCfgOk := cfg["cloneable_cfg"].(map[string]interface{}); cloneableCfgOk {
		securityCfg, _ = cloneableCfg["security"].(map[string]interface{})
	}
	if securityCfg == nil || len(securityCfg) == 0 {
		common.Log.Warningf("Failed to parse cloneable security configuration for load balancer: %s; attempting to create sane initial configuration", n.ID)

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
	}
	balancerCfg["security"] = securityCfg

	targetID, targetOk := balancerCfg["target_id"].(string)
	providerID, providerOk := balancerCfg["provider_id"].(string)
	region, regionOk := balancerCfg["region"].(string)
	vpcID, _ := balancerCfg["vpc_id"].(string)

	l.Description = common.StringOrNil(fmt.Sprintf("%s - %s %s", *l.Name, targetID, region))
	l.Region = common.StringOrNil(region)
	db.Save(&l)

	if !targetOk || !providerOk || !regionOk || balancerType == "" {
		err := fmt.Errorf("Cannot provision load balancer for network node without credentials, a target, provider, region and type configuration; target id: %s; provider id: %s; region: %s, type: %s", targetID, providerID, region, balancerType)
		common.Log.Warningf(err.Error())
		return err
	}

	orchestrationAPI, err := l.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Cannot provision load balancer for network node; failed to resolve orchestration API client; %s", err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if region != "" {
		if strings.ToLower(targetID) == "aws" {
			// start security group handling
			securityGroupIds := make([]string, 0)
			var securityGroupID *string
			securityGroupDesc := fmt.Sprintf("security group for load balancer: %s", l.ID.String())
			securityGroup, _ := orchestrationAPI.CreateSecurityGroup(securityGroupDesc, securityGroupDesc, common.StringOrNil(vpcID))
			if securityGroup == nil {
				securityGroups, err := orchestrationAPI.GetSecurityGroups()
				if err == nil {
					for _, secGroup := range securityGroups.SecurityGroups {
						if secGroup.GroupName != nil && *secGroup.GroupName == securityGroupDesc {
							securityGroupID = secGroup.GroupId
							break
						}
					}
				}
			} else {
				securityGroupID = securityGroup.GroupId
			}

			if securityGroupID != nil {
				securityGroupIds = append(securityGroupIds, *securityGroupID)

				balancerCfg["target_security_group_ids"] = securityGroupIds

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
								}
							}

							// UDP not currently supported by classic ELB API; UDP support is not needed here at this time...
							// if _udp, udpOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpOk {
							// 	for i := range _udp {
							// 		_port := int64(_udp[i].(float64))
							// 		udp = append(udp, _port)
							// 	}
							// }

							_, err := orchestrationAPI.AuthorizeSecurityGroupIngress(*securityGroupID, cidr, tcp, udp)
							if err != nil {
								err := fmt.Errorf("Failed to authorize load balancer security group ingress in EC2 %s region; security group id: %s; %d tcp ports; %d udp ports: %s", region, *securityGroupID, len(tcp), len(udp), err.Error())
								common.Log.Warningf(err.Error())
								// return err // FIXME-- gotta be a sane way to check for actual failures; this will workaround duplicates for now as those should be acceptable
							}
						}
					}
				}
			}
			// end security group handling

			if providerID, providerIDOk := balancerCfg["provider_id"].(string); providerIDOk {
				if strings.ToLower(providerID) == "docker" {
					loadBalancersResp, err := orchestrationAPI.CreateLoadBalancerV2(common.StringOrNil(vpcID), l.Name, common.StringOrNil("application"), securityGroupIds)
					if err != nil {
						err := fmt.Errorf("Failed to provision AWS load balancer (v2); %s", err.Error())
						common.Log.Warningf(err.Error())
						return err
					}

					for _, loadBalancer := range loadBalancersResp.LoadBalancers {
						balancerCfg["load_balancer_name"] = l.Name
						balancerCfg["load_balancer_url"] = loadBalancer.DNSName
						balancerCfg["target_balancer_id"] = loadBalancer.LoadBalancerArn
						balancerCfg["vpc_id"] = loadBalancer.VpcId

						if l.Type != nil && *l.Type == loadBalancerTypeRPC {
							common.Log.Debugf("Setting JSON-RPC and websocket URLs on load balancer: %s", l.ID)
							balancerCfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *loadBalancer.DNSName, jsonRPCPort)
							balancerCfg["json_rpc_port"] = jsonRPCPort
							balancerCfg["websocket_url"] = fmt.Sprintf("ws://%s:%v", *loadBalancer.DNSName, websocketPort)
							balancerCfg["websocket_port"] = websocketPort
						}

						l.Host = loadBalancer.DNSName
						l.setConfig(balancerCfg)
						l.sanitizeConfig()
						l.updateStatus(db, "active", l.Description)
						db.Save(&l)
						if len(l.Errors) == 0 {
							return nil
						}
						return fmt.Errorf("%s", *l.Errors[0].Message)
					}
				}
			} else {
				err := fmt.Errorf("Failed to load balance node without provider_id")
				common.Log.Warningf(err.Error())
				return err
			}
		} else {
			err := fmt.Errorf("Failed to load balance node without region")
			common.Log.Warningf(err.Error())
			return err
		}
	}

	return nil
}

func (l *LoadBalancer) balanceNode(db *gorm.DB, node *Node) error {
	db.Model(l).Association("Nodes").Append(node)

	network := node.relatedNetwork(db)
	if l.Type != nil && *l.Type == loadBalancerTypeRPC {
		network.setIsLoadBalanced(db, true)
	}

	cfg := l.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)
	vpcID, _ := cfg["vpc_id"].(string)
	securityCfg, securityCfgOk := cfg["security"].(map[string]interface{})

	if l.Host != nil {
		common.Log.Debugf("Attempting to load balance network node %s on balancer: %s", node.ID, l.ID)

		if !securityCfgOk {
			desc := fmt.Sprintf("Failed to resolve security configuration for lazy initialization of load balanced target group in region: %s", region)
			common.Log.Warning(desc)
			return fmt.Errorf(desc)
		}

		orchestrationAPI, err := l.orchestrationAPIClient()
		if err != nil {
			err := fmt.Errorf("Failed to load balance network node %s on balancer: %s; %s", node.ID, l.ID, err.Error())
			common.Log.Warningf(err.Error())
			return err
		}

		if strings.ToLower(targetID) == "aws" && targetOk && regionOk && arnOk && securityCfgOk {
			targetGroups := map[int64]string{}

			_, targetGroupsOk := cfg["target_groups"].(map[string]interface{})
			if !targetGroupsOk {
				common.Log.Debugf("Attempting to lazily initialize load balanced target groups for network node %s on balancer: %s", node.ID, l.ID)

				var healthCheckPort *int64
				var healthCheckStatusCode *int64
				var healthCheckPath *string
				if healthCheck, healthCheckOk := securityCfg["health_check"].(map[string]interface{}); healthCheckOk {
					if hcPort, hcPortOk := healthCheck["port"].(float64); hcPortOk {
						_hcPort := int64(hcPort)
						healthCheckPort = &_hcPort
					}
					if hcStatusCode, hcStatusCodeOk := healthCheck["status_code"].(float64); hcStatusCodeOk {
						_hcStatusCode := int64(hcStatusCode)
						healthCheckStatusCode = &_hcStatusCode
					}
					if hcPath, hcPathOk := healthCheck["path"].(string); hcPathOk {
						healthCheckPath = common.StringOrNil(hcPath)
					}
				}

				if ingress, ingressOk := securityCfg["ingress"]; ingressOk {
					common.Log.Debugf("Found security group ingress rules to apply to load balanced target group for network node %s on balancer: %s", node.ID, l.ID)

					switch ingress.(type) {
					case map[string]interface{}:
						ingressCfg := ingress.(map[string]interface{})
						for cidr := range ingressCfg {
							tcp := make([]int64, 0)
							// udp := make([]int64, 0)
							if _tcp, tcpOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
								for i := range _tcp {
									_port := int64(_tcp[i].(float64))
									tcp = append(tcp, _port)
								}
							}

							for _, tcpPort := range tcp {
								targetGroupName := l.buildTargetGroupName(tcpPort)
								targetGroup, err := orchestrationAPI.CreateTargetGroup(
									common.StringOrNil(vpcID),
									common.StringOrNil(targetGroupName),
									common.StringOrNil("HTTP"),
									tcpPort,
									healthCheckPort,
									healthCheckStatusCode,
									healthCheckPath,
								)
								if err != nil {
									if aerr, ok := err.(awserr.Error); ok {
										switch aerr.Code() {
										case elbv2.ErrCodeDuplicateTargetGroupNameException:
											common.Log.Debugf("Load balancer target group %s already exists for balancer %s", targetGroupName, l.ID)
										default:
											desc := fmt.Sprintf("Failed to configure load balanced target group in region: %s; %s", region, err.Error())
											common.Log.Warning(desc)
											return fmt.Errorf(desc)
										}
									}
								} else {
									common.Log.Debugf("Upserted target group %s in region: %s", targetGroupName, region)
									if len(targetGroup.TargetGroups) == 1 {
										targetGroups[tcpPort] = *targetGroup.TargetGroups[0].TargetGroupArn
									}
								}
							}

							cfg["target_groups"] = targetGroups
							l.setConfig(cfg)
							db.Save(&l)
						}
					}
				} else {
					desc := fmt.Sprintf("Failed to resolve security ingress rules for creation of load balanced target groups in region: %s", region)
					common.Log.Warning(desc)
					return fmt.Errorf(desc)
				}
			} else {
				common.Log.Debugf("Attempting to read previously-initialized load balanced target groups for network node %s on balancer: %s", node.ID, l.ID)
				if cfgTargetGroups, cfgTargetGroupsOk := cfg["target_groups"].(map[string]interface{}); cfgTargetGroupsOk {
					for portStr, targetGroupArn := range cfgTargetGroups {
						_port, _ := strconv.Atoi(portStr)
						port := int64(_port)
						targetGroups[port] = targetGroupArn.(string)
					}
				}
			}

			common.Log.Debugf("Resolved %d target group(s) for load balanced target registration for network node %s on balancer: %s", len(targetGroups), node.ID, l.ID)
			for port, targetGroupArn := range targetGroups {
				_, err := orchestrationAPI.RegisterTarget(common.StringOrNil(targetGroupArn), node.PrivateIPv4, &port)
				if err != nil {
					desc := fmt.Sprintf("Failed to add target to load balanced target group in region: %s; %s", region, err.Error())
					common.Log.Warning(desc)
					return fmt.Errorf(desc)
				}
				common.Log.Debugf("Registered load balanced target for balancer %s on port: %d", l.ID, port)

				_, err = orchestrationAPI.CreateListenerV2(common.StringOrNil(targetBalancerArn), common.StringOrNil(targetGroupArn), common.StringOrNil("HTTP"), &port)
				if err != nil {
					common.Log.Warningf("Failed to register load balanced listener with target group: %s", targetGroupArn)
				}
				common.Log.Debugf("Upserted listener for load balanced target group %s in region: %s", targetGroupArn, region)
			}

			// desc := fmt.Sprintf("Failed to resolve load balanced target groups needed for listener creation in region: %s", region)
			// common.Log.Warning(desc)
			// return fmt.Errorf(desc)
		}
	} else {
		desc := fmt.Sprintf("Failed to resolve host configuration for lazy initialization of load balanced target group in region: %s", region)
		common.Log.Warning(desc)
		return fmt.Errorf(desc)
	}

	return nil
}

func (l *LoadBalancer) unbalanceNode(db *gorm.DB, node *Node) error {
	common.Log.Debugf("Attempting to unbalance network node %s from balancer: %s", node.ID, l.ID)
	cfg := l.ParseConfig()
	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	// targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)
	securityCfg, securityCfgOk := cfg["security"].(map[string]interface{})

	orchestrationAPI, err := l.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to unbalance network node %s on balancer: %s; %s", node.ID, l.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if strings.ToLower(targetID) == "aws" && targetOk && regionOk && securityCfgOk {
		if ingress, ingressOk := securityCfg["ingress"]; ingressOk {
			switch ingress.(type) {
			case map[string]interface{}:
				ingressCfg := ingress.(map[string]interface{})
				for cidr := range ingressCfg {
					tcp := make([]int64, 0)
					// udp := make([]int64, 0)
					if _tcp, tcpOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpOk {
						for i := range _tcp {
							_port := int64(_tcp[i].(float64))
							tcp = append(tcp, _port)
						}
					}

					if targetGroups, targetGroupsOk := cfg["target_groups"].(map[string]interface{}); targetGroupsOk {
						common.Log.Debugf("Attempting to deregister target groups for load balancer: %s", l.ID)
						for portStr, targetGroupArn := range targetGroups {
							_port, _ := strconv.Atoi(portStr)
							port := int64(_port)
							_, err := orchestrationAPI.DeregisterTarget(common.StringOrNil(targetGroupArn.(string)), node.PrivateIPv4, &port)
							if err != nil {
								desc := fmt.Sprintf("Failed to deregister target from load balanced target group in region: %s; %s", region, err.Error())
								common.Log.Warning(desc)
								return err
							}
							common.Log.Debugf("Deregistered target: %s:%v", *node.PrivateIPv4, port)
						}
					}
				}
			}
		}
	} else {
		err := fmt.Errorf("Failed to unbalance network node %s from balancer: %s; invalid configuration", node.ID, l.ID)
		common.Log.Warning(err.Error())
		return err
	}

	db.Model(&l).Association("Nodes").Delete(node)
	balancedNodeCount := db.Model(&l).Association("Nodes").Count() // TODO-- create a balancer_daemon (like stats daemon) to monitor balancer state
	common.Log.Debugf("Load balancer %s contains %d remaining balanced nodes", l.ID, balancedNodeCount)
	if balancedNodeCount == 0 {
		common.Log.Debugf("Attempting to deprovision load balancer %s in region: %s", l.ID, region)
		msg, _ := json.Marshal(map[string]interface{}{
			"load_balancer_id": l.ID,
		})
		natsutil.NatsPublish(natsLoadBalancerDeprovisioningSubject, msg)

		if l.Type != nil && *l.Type == loadBalancerTypeRPC {
			network := node.relatedNetwork(db)
			network.setIsLoadBalanced(db, network.isLoadBalanced(db, nil, nil))
		}
	}

	return nil
}

// setConfig sets the network config in-memory
func (l *LoadBalancer) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	_cfgJSON := json.RawMessage(cfgJSON)
	l.Config = &_cfgJSON
}

func (l *LoadBalancer) updateStatus(db *gorm.DB, status string, description *string) {
	l.Status = common.StringOrNil(status)
	l.Description = description
	result := db.Save(&l)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			l.Errors = append(l.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
}

func (l *LoadBalancer) unregisterSecurityGroups(db *gorm.DB) error {
	common.Log.Debugf("Attempting to unregister security groups for load balancer with id: %s", l.ID)

	cfg := l.ParseConfig()
	securityGroupIds, securityGroupIdsOk := cfg["target_security_group_ids"].([]interface{})

	orchestrationAPI, err := l.orchestrationAPIClient()
	if err != nil {
		err := fmt.Errorf("Failed to unregister security groups for load balancer with id: %s; %s", l.ID, err.Error())
		common.Log.Warningf(err.Error())
		return err
	}

	if securityGroupIdsOk {
		for i := range securityGroupIds {
			securityGroupID := securityGroupIds[i].(string)

			_, err := orchestrationAPI.DeleteSecurityGroup(securityGroupID)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case "InvalidGroup.NotFound":
						common.Log.Debugf("Attempted to unregister security group which does not exist for load balancer with id: %s; security group id: %s", l.ID, securityGroupID)
					default:
						common.Log.Warningf("Failed to unregister security group for load balancer with id: %s; security group id: %s", l.ID, securityGroupID)
						return err
					}
				}
			}
		}

		delete(cfg, "target_security_group_ids")
		l.setConfig(cfg)
		db.Save(l)
	}

	return nil
}
