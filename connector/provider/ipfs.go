package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"

	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

const IPFSConnectorProvider = "ipfs"

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
	apiPort, apiPortOk := config["api_port"].(int)
	gatewayPort, gatewayPortOk := config["gateway_port"].(int)
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
		apiPort:       apiPort,
		gatewayPort:   gatewayPort,
	}
}

func (p *IPFSProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

func (p *IPFSProvider) Deprovision() error {
	db := dbconf.DatabaseConnection()

	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	for _, node := range nodes {
		if node.Delete() {
			common.Log.Debugf("Removed node %s on connector: %s", node.ID, p.connectorID)
		} else {
			msg := fmt.Sprintf("Failed to deprovision node %s on connector: %s", node.ID, p.connectorID)
			common.Log.Warningf(msg)
			return errors.New(msg)
		}
	}

	loadBalancers := make([]*network.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, balancer := range loadBalancers {
		// TODO: should this be async to wait for nodes to be deprovisioned?
		common.Log.Debugf("Attempting to deprovision load balancer %s on connector: %s", balancer.ID, p.connectorID)
		err := balancer.Deprovision(db)
		if err != nil {
			msg := fmt.Sprintf("Failed to deprovision load balancer %s on connector: %s; %s", balancer.ID, p.connectorID, err.Error())
			common.Log.Warningf(msg)
			return errors.New(msg)
		} else {
			p.model.Association("LoadBalancers").Delete(balancer)
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
		p.model.Association("LoadBalancers").Append(loadBalancer)
	} else {
		return fmt.Errorf("Failed to provision load balancer on connector: %s", p.connectorID)
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
		p.model.Association("Nodes").Append(node)

	} else {
		return fmt.Errorf("Failed to provision node on connector: %s", p.connectorID)
	}

	return nil
}
