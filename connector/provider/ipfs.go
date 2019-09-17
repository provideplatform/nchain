package provider

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

const IPFSConnectorProvider = "ipfs"
const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"

// IPFSProvider is a connector.ConnectorAPI implementing orchestration for IPFS
type IPFSProvider struct {
	connectorID   uuid.UUID
	model         *gorm.DB
	config        map[string]interface{}
	networkID     *uuid.UUID
	applicationID *uuid.UUID
	region        *string
	apiPort       int
	gatewayPort   int
}

// InitIPFSProvider initializes and returns the IPFS connector API provider
func InitIPFSProvider(connectorID uuid.UUID, networkID, applicationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *IPFSProvider {
	region, regionOk := config["region"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	gatewayPort, gatewayPortOk := config["gateway_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || !gatewayPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &IPFSProvider{
		connectorID:   connectorID,
		model:         model,
		config:        config,
		networkID:     networkID,
		applicationID: applicationID,
		region:        common.StringOrNil(region),
		apiPort:       int(apiPort),
		gatewayPort:   int(gatewayPort),
	}
}

func (p *IPFSProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

func (p *IPFSProvider) Deprovision() error {
	loadBalancers := make([]*network.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, balancer := range loadBalancers {
		p.model.Association("LoadBalancers").Delete(balancer)
	}

	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	if len(nodes) == 0 {
		for _, balancer := range loadBalancers {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": balancer.ID,
			})
			common.NATSPublish(natsLoadBalancerDeprovisioningSubject, msg)
		}
	} else {
		for _, node := range nodes {
			common.Log.Debugf("Attempting to deprovision node %s on connector: %s", node.ID, p.connectorID)
			p.model.Association("Nodes").Delete(node)
			node.Delete()
		}
	}

	return nil
}

func (p *IPFSProvider) Provision() error {
	loadBalancer := &network.LoadBalancer{
		NetworkID:   *p.networkID,
		Type:        common.StringOrNil(IPFSConnectorProvider),
		Description: common.StringOrNil(fmt.Sprintf("IPFS Connector Load Balancer")),
		Region:      p.region,
		Config:      p.rawConfig(),
	}

	if loadBalancer.Create() {
		common.Log.Debugf("Created load balancer %s on connector: %s", loadBalancer.ID, p.connectorID)
		p.model.Association("LoadBalancers").Append(loadBalancer)

		// err := p.ProvisionNode()
		// if err != nil {
		// 	common.Log.Warning(err.Error())
		// }
	} else {
		return fmt.Errorf("Failed to provision load balancer on connector: %s; %s", p.connectorID, *loadBalancer.Errors[0].Message)
	}

	return nil
}

func (p *IPFSProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

func (p *IPFSProvider) ProvisionNode() error {
	node := &network.Node{
		NetworkID:     *p.networkID,
		ApplicationID: p.applicationID,
		Config:        p.rawConfig(),
	}

	if node.Create() {
		common.Log.Debugf("Created node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Append(node)
	} else {
		return fmt.Errorf("Failed to provision node on connector: %s", p.connectorID)
	}

	return nil
}
