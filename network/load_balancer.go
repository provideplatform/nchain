package network

import (
	"encoding/json"
	"fmt"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	awswrapper "github.com/kthomas/go-aws-wrapper"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&LoadBalancer{})
	db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_network_id", "network_id")
	db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_region", "region")
	db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_status", "status")
	db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_type", "type")
	db.Model(&LoadBalancer{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// LoadBalancer instances represent a physical or virtual load balancer of a specific type (i.e., JSON-RPC) which belongs to a network
type LoadBalancer struct {
	provide.Model
	NetworkID   uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name        *string          `sql:"not null" json:"name"`
	Type        *string          `sql:"not null" json:"type"`
	Host        *string          `json:"host"`
	IPv4        *string          `json:"ipv4"`
	IPv6        *string          `json:"ipv6"`
	Description *string          `json:"description"`
	Region      *string          `json:"region"`
	Status      *string          `sql:"not null;default:'provisioning'" json:"status"`
	Nodes       []NetworkNode    `gorm:"many2many:load_balancers_network_nodes" json:"-"`
	Config      *json.RawMessage `sql:"type:json" json:"config"`
}

// Create and persist a new load balancer
func (l *LoadBalancer) Create() bool {
	if !l.Validate() {
		return false
	}

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
				msg, _ := json.Marshal(l)
				natsConnection := common.GetDefaultNatsStreamingConnection()
				natsConnection.Publish(natsLoadBalancerProvisioningSubject, msg)
			}
			return success
		}
	}
	return false
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

func (l *LoadBalancer) deprovision(db *gorm.DB) error {
	common.Log.Debugf("Attempting to deprovision infrastructure for load balancer with id: %s", l.ID)
	cfg := l.ParseConfig()
	l.updateStatus(db, "deprovisioning", l.Description)

	targetID, targetIDOk := cfg["target_id"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	region, regionOk := cfg["region"].(string)
	targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)

	if !targetIDOk || !credsOk || !regionOk || !arnOk {
		err := fmt.Errorf("Cannot deprovision load balancer for network node without target, region, credentials and target balancer id configuration; target: %s, region: %s, balancer: %s", targetID, region, targetBalancerArn)
		common.Log.Warningf(err.Error())
		return err
	}

	if region != "" {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if securityGroupIds, securityGroupIdsOk := cfg["target_security_group_ids"].([]interface{}); securityGroupIdsOk {
				for _, securityGroupID := range securityGroupIds {
					_, err := awswrapper.DeleteSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupID.(string))
					if err != nil {
						common.Log.Warningf("Failed to delete security group with id: %s; %s", securityGroupID.(string), err.Error())
						return err
					}
				}
			}

			targetGroups, targetGroupsOk := cfg["target_groups"].(map[float64]string)
			if targetGroupsOk {
				for _, targetGroupArn := range targetGroups {
					_, err := awswrapper.DeleteTargetGroup(accessKeyID, secretAccessKey, region, common.StringOrNil(targetGroupArn))
					if err != nil {
						common.Log.Warningf("Failed to delete load balanced target group: %s; %s", targetGroupArn, err.Error())
						return err
					}
				}
			}

			_, err := awswrapper.DeleteLoadBalancerV2(accessKeyID, secretAccessKey, region, common.StringOrNil(targetBalancerArn))
			if err != nil {
				common.Log.Warningf("Failed to delete load balancer: %s; %s", *l.Name, err.Error())
				return err
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

func (l *LoadBalancer) provision(db *gorm.DB) error {
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
	if cloneableCfg, cloneableCfgOk := cfg["cloneable_cfg"].(map[string]interface{}); cloneableCfgOk {
		securityCfg, _ = cloneableCfg["_security"].(map[string]interface{})
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
		balancerCfg["_security"] = securityCfg
	}

	targetID, targetOk := balancerCfg["target_id"].(string)
	providerID, providerOk := balancerCfg["provider_id"].(string)
	credentials, credsOk := balancerCfg["credentials"].(map[string]interface{})
	region, regionOk := balancerCfg["region"].(string)
	vpcID, _ := balancerCfg["vpc_id"].(string)

	l.Description = common.StringOrNil(fmt.Sprintf("%s - %s %s", *l.Name, targetID, region))
	l.Region = common.StringOrNil(region)
	db.Save(&l)

	if !targetOk || !providerOk || !credsOk || !regionOk || balancerType == "" {
		err := fmt.Errorf("Cannot provision load balancer for network node without credentials, a target, provider, region and type configuration; target id: %s; provider id: %s; region: %s, type: %s", targetID, providerID, region, balancerType)
		common.Log.Warningf(err.Error())
		return err
	}

	if region != "" {
		if strings.ToLower(targetID) == "aws" {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			// start security group handling
			securityGroupIds := make([]string, 0)
			securityGroupDesc := fmt.Sprintf("security group for load balancer: %s", l.ID.String())
			securityGroup, _ := awswrapper.CreateSecurityGroup(accessKeyID, secretAccessKey, region, securityGroupDesc, securityGroupDesc, common.StringOrNil(vpcID))
			if securityGroup != nil {
				securityGroupIds = append(securityGroupIds, *securityGroup.GroupId)
			}
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

						_, err := awswrapper.AuthorizeSecurityGroupIngress(accessKeyID, secretAccessKey, region, *securityGroup.GroupId, cidr, tcp, udp)
						if err != nil {
							err := fmt.Errorf("Failed to authorize load balancer security group ingress in EC2 %s region; security group id: %s; %d tcp ports; %d udp ports: %s", region, *securityGroup.GroupId, len(tcp), len(udp), err.Error())
							common.Log.Warningf(err.Error())
							return err
						}
					}
				}
			}
			// end security group handling

			if providerID, providerIDOk := balancerCfg["provider_id"].(string); providerIDOk {
				// TODO: for ubuntu-vm provider

				if strings.ToLower(providerID) == "docker" {
					loadBalancersResp, err := awswrapper.CreateLoadBalancerV2(accessKeyID, secretAccessKey, region, common.StringOrNil(vpcID), l.Name, common.StringOrNil("application"), securityGroupIds)
					if err != nil {
						err := fmt.Errorf("Failed to provision AWS load balancer (v2); %s", err.Error())
						common.Log.Warningf(err.Error())
						return err
					}

					for _, loadBalancer := range loadBalancersResp.LoadBalancers {
						balancerCfg["load_balancer_name"] = l.Name
						balancerCfg["load_balancer_url"] = loadBalancer.DNSName
						balancerCfg["json_rpc_url"] = fmt.Sprintf("http://%s:%v", *loadBalancer.DNSName, jsonRPCPort)
						balancerCfg["json_rpc_port"] = jsonRPCPort
						balancerCfg["websocket_url"] = fmt.Sprintf("ws://%s:%v", *loadBalancer.DNSName, websocketPort)
						balancerCfg["websocket_port"] = websocketPort
						balancerCfg["target_balancer_id"] = loadBalancer.LoadBalancerArn

						l.Host = loadBalancer.DNSName
						l.setConfig(balancerCfg)
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

func (l *LoadBalancer) balanceNode(db *gorm.DB, node *NetworkNode) error {
	db.Model(l).Association("Nodes").Append(node)

	cfg := l.ParseConfig()

	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)
	securityCfg, securityCfgOk := cfg["_security"].(map[string]interface{})

	if l.Host != nil {
		common.Log.Debugf("Attempting to load balance network node %s on balancer: %s", node.ID, l.ID)

		if !securityCfgOk {
			desc := fmt.Sprintf("Failed to resolve _security configuration for lazy initialization of load balanced target group in region: %s", region)
			common.Log.Warning(desc)
			return fmt.Errorf(desc)
		}

		if strings.ToLower(targetID) == "aws" && targetOk && regionOk && credsOk && arnOk && securityCfgOk {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)
			vpcID := *common.DefaultAWSConfig.DefaultVpcID // FIXME-- create and use managed vpc

			var targetGroups map[float64]string
			targetGroups, targetGroupsOk := cfg["target_groups"].(map[float64]string)
			if !targetGroupsOk {
				common.Log.Debugf("Attempting to lazily initialize load balanced target groups for network node %s on balancer: %s", node.ID, l.ID)
				targetGroups = map[float64]string{}

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
								targetGroupName := ethcommon.Bytes2Hex(provide.Keccak256(fmt.Sprintf("%s-port-%v", l.ID.String(), tcpPort)))[0:31]
								targetGroup, err := awswrapper.CreateTargetGroup(accessKeyID, secretAccessKey, region, common.StringOrNil(vpcID), common.StringOrNil(targetGroupName), common.StringOrNil("HTTP"), tcpPort)
								if err != nil {
									desc := fmt.Sprintf("Failed to configure load balanced target group in region: %s; %s", region, err.Error())
									common.Log.Warning(desc)
									l.updateStatus(db, "failed", &desc)
									db.Save(l)
									return fmt.Errorf(desc)
								}
								common.Log.Debugf("Upserted target group %s in region: %s", targetGroupName, region)
								if len(targetGroup.TargetGroups) == 1 {
									targetGroups[float64(tcpPort)] = *targetGroup.TargetGroups[0].TargetGroupArn
								}
							}

							cfg["target_groups"] = targetGroups
							l.setConfig(cfg)
							db.Save(l)
						}
					}
				} else {
					desc := fmt.Sprintf("Failed to resolve security ingress rules for creation of load balanced target group in region: %s", region)
					common.Log.Warning(desc)
					return fmt.Errorf(desc)
				}

				if targetGroups, targetGroupsOk := cfg["target_groups"].(map[float64]string); targetGroupsOk {
					common.Log.Debugf("Found target groups for load balanced target registration for network node %s on balancer: %s", node.ID, l.ID)

					for port, targetGroupArn := range targetGroups {
						_port := int64(port)
						target, err := awswrapper.RegisterTarget(accessKeyID, secretAccessKey, region, common.StringOrNil(targetGroupArn), node.PrivateIPv4, &_port)
						if err != nil {
							desc := fmt.Sprintf("Failed to add target to load balanced target group in region: %s; %s", region, err.Error())
							common.Log.Warning(desc)
							l.updateStatus(db, "failed", &desc)
							db.Save(l)
							return fmt.Errorf(desc)
						}
						common.Log.Debugf("Registered load balanced target: %s", target)

						_, err = awswrapper.CreateListenerV2(accessKeyID, secretAccessKey, region, common.StringOrNil(targetBalancerArn), common.StringOrNil(targetGroupArn), common.StringOrNil("HTTP"), &_port)
						if err != nil {
							common.Log.Warningf("Failed to register load balanced listener with target group: %s", targetGroupArn)
						}
						common.Log.Debugf("Upserted listener for load balanced target group %s in region: %s", targetGroupArn, region)
					}
				} else {
					desc := fmt.Sprintf("Failed to resolve load balanced target groups needed for listener creation in region: %s", region)
					common.Log.Warning(desc)
					return fmt.Errorf(desc)
				}
			}
		}
	} else {
		desc := fmt.Sprintf("Failed to resolve host configuration for lazy initialization of load balanced target group in region: %s", region)
		common.Log.Warning(desc)
		return fmt.Errorf(desc)
	}

	return nil
}

func (l *LoadBalancer) unbalanceNode(db *gorm.DB, node *NetworkNode) error {
	common.Log.Debugf("Attempting to unbalance network node %s from balancer: %s", node.ID, l.ID)
	cfg := l.ParseConfig()

	targetID, targetOk := cfg["target_id"].(string)
	region, regionOk := cfg["region"].(string)
	credentials, credsOk := cfg["credentials"].(map[string]interface{})
	// targetBalancerArn, arnOk := cfg["target_balancer_id"].(string)
	securityCfg, securityCfgOk := cfg["_security"].(map[string]interface{})

	if strings.ToLower(targetID) == "aws" && targetOk && regionOk && credsOk && securityCfgOk {
		accessKeyID := credentials["aws_access_key_id"].(string)
		secretAccessKey := credentials["aws_secret_access_key"].(string)

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

					if targetGroups, targetGroupsOk := cfg["target_groups"].(map[float64]string); targetGroupsOk {
						for port, targetGroupArn := range targetGroups {
							_port := int64(port)
							_, err := awswrapper.DeregisterTarget(accessKeyID, secretAccessKey, region, common.StringOrNil(targetGroupArn), node.IPv4, &_port)
							if err != nil {
								desc := fmt.Sprintf("Failed to deregister target from load balanced target group in region: %s; %s", region, err.Error())
								common.Log.Warning(desc)
								return err
							}
							common.Log.Debugf("Deregistered target: %s:%v", node.IPv4, port)
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
	if db.Model(&l).Association("Nodes").Count() == 0 {
		if strings.ToLower(targetID) == "aws" && targetOk && regionOk && credsOk && securityCfgOk {
			accessKeyID := credentials["aws_access_key_id"].(string)
			secretAccessKey := credentials["aws_secret_access_key"].(string)

			if targetGroups, targetGroupsOk := cfg["target_groups"].(map[float64]string); targetGroupsOk {
				for _, targetGroupArn := range targetGroups {
					_, err := awswrapper.DeleteTargetGroup(accessKeyID, secretAccessKey, region, common.StringOrNil(targetGroupArn))
					if err != nil {
						desc := fmt.Sprintf("Failed to deregister target from load balanced target group in region: %s; %s", region, err.Error())
						common.Log.Warning(desc)
						return err
					}
					common.Log.Debugf("Dropped target group %s in region: %s", targetGroupArn, region)
				}
			}
		}

		common.Log.Debugf("Attempting to deprovision load balancer %s in region: %s", l.ID, region)
		msg, _ := json.Marshal(l)
		natsConnection := common.GetDefaultNatsStreamingConnection()
		natsConnection.Publish(natsLoadBalancerDeprovisioningSubject, msg)
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
