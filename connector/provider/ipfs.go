package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/jinzhu/gorm"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

// IPFSConnectorProvider "ipfs"
const IPFSConnectorProvider = "ipfs"

const natsConnectorDenormalizeConfigSubject = "goldmine.connector.config.denormalize"
const natsConnectorResolveReachabilitySubject = "goldmine.connector.reachability.resolve"
const natsLoadBalancerBalanceNodeSubject = "goldmine.node.balance"
const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"

// IPFSProvider is a connector.ProviderAPI implementing orchestration for IPFS
type IPFSProvider struct {
	connectorID   uuid.UUID
	model         *gorm.DB
	config        map[string]interface{}
	networkID     *uuid.UUID
	applicationID *uuid.UUID
	region        *string
	apiURL        *string
	apiPort       int
	gatewayPort   int
}

// InitIPFSProvider initializes and returns the IPFS connector API provider
func InitIPFSProvider(connectorID uuid.UUID, networkID, applicationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *IPFSProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
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
		apiURL:        common.StringOrNil(apiURL),
		apiPort:       int(apiPort),
		gatewayPort:   int(gatewayPort),
	}
}

func (p *IPFSProvider) apiClientFactory(basePath *string) *ipfs.Shell {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize IPFS api client factory")
		return nil
	}

	return ipfs.NewShell(*apiURL)
}

func (p *IPFSProvider) apiURLFactory(path string) *string {
	if p.apiURL == nil {
		return nil
	}

	suffix := ""
	if path != "" {
		suffix = fmt.Sprintf("/%s", path)
	}
	return common.StringOrNil(fmt.Sprintf("%s%s", *p.apiURL, suffix))
}

func (p *IPFSProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// Deprovision undeploys all associated nodes and load balancers and removes them from the IPFS connector
func (p *IPFSProvider) Deprovision() error {
	loadBalancers := make([]*network.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, balancer := range loadBalancers {
		p.model.Association("LoadBalancers").Delete(balancer)
	}

	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	for _, node := range nodes {
		common.Log.Debugf("Attempting to deprovision node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Delete(node)
		node.Delete()
	}

	for _, balancer := range loadBalancers {
		msg, _ := json.Marshal(map[string]interface{}{
			"load_balancer_id": balancer.ID,
		})
		natsutil.NatsPublish(natsLoadBalancerDeprovisioningSubject, msg)
	}

	return nil
}

// Provision configures a new load balancer and the initial IPFS nodes and associates the resources with the IPFS connector
func (p *IPFSProvider) Provision() error {
	loadBalancer := &network.LoadBalancer{
		NetworkID:     *p.networkID,
		ApplicationID: p.applicationID,
		Type:          common.StringOrNil(IPFSConnectorProvider),
		Description:   common.StringOrNil(fmt.Sprintf("IPFS Connector Load Balancer")),
		Region:        p.region,
		Config:        p.rawConfig(),
	}

	if loadBalancer.Create() {
		common.Log.Debugf("Created load balancer %s on connector: %s", loadBalancer.ID, p.connectorID)
		p.model.Association("LoadBalancers").Append(loadBalancer)

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID,
		})
		natsutil.NatsPublish(natsConnectorDenormalizeConfigSubject, msg)

		err := p.ProvisionNode()
		if err != nil {
			common.Log.Warning(err.Error())
		}
	} else {
		return fmt.Errorf("Failed to provision load balancer on connector: %s; %s", p.connectorID, *loadBalancer.Errors[0].Message)
	}

	return nil
}

// DeprovisionNode undeploys an existing node removes it from the IPFS connector
func (p *IPFSProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the IPFS connector
func (p *IPFSProvider) ProvisionNode() error {
	node := &network.Node{
		NetworkID:     *p.networkID,
		ApplicationID: p.applicationID,
		Config:        p.rawConfig(),
	}

	if node.Create() {
		common.Log.Debugf("Created node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Append(node)

		loadBalancers := make([]*network.LoadBalancer, 0)
		p.model.Association("LoadBalancers").Find(&loadBalancers)
		for _, balancer := range loadBalancers {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": balancer.ID.String(),
				"network_node_id":  node.ID.String(),
			})
			natsutil.NatsPublish(natsLoadBalancerBalanceNodeSubject, msg)
		}

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID.String(),
		})
		natsutil.NatsPublish(natsConnectorResolveReachabilitySubject, msg)
	} else {
		return fmt.Errorf("Failed to provision node on connector: %s", p.connectorID)
	}

	return nil
}

// Reachable returns true if the IPFS API provider is available
func (p *IPFSProvider) Reachable() bool {
	loadBalancers := make([]*network.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, loadBalancer := range loadBalancers {
		if loadBalancer.ReachableOnPort(uint(p.apiPort)) {
			return true
		}
	}
	common.Log.Debugf("Connector is unreachable: %s", p.connectorID)
	return false
}

// Create impl for IPFSProvider
func (p *IPFSProvider) Create(params map[string]interface{}) (interface{}, error) {
	return nil, errors.New("create not implemented for IPFS connectors")
}

// Read impl for IPFSProvider
func (p *IPFSProvider) Read(id string) (interface{}, error) {
	return nil, errors.New("read not implemented for IPFS connectors")

}

// Update impl for IPFSProvider
func (p *IPFSProvider) Update(id string, params map[string]interface{}) (interface{}, error) {
	return nil, errors.New("update not implemented for IPFS connectors")
}

// Delete impl for IPFSProvider
func (p *IPFSProvider) Delete(id string) error {
	return errors.New("delete not implemented for IPFS connectors")
}

// List impl for IPFSProvider
func (p *IPFSProvider) List(params map[string]interface{}) ([]interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed IPFS connector ls request; %s", r)
		}
	}()

	sh := p.apiClientFactory(nil)
	if sh == nil {
		return nil, fmt.Errorf("unable to list IPFS resources for connector: %s; failed to initialize IPFS shell", p.connectorID)
	}

	resp := make([]interface{}, 0)
	args := make([]string, 0)

	if objects, objectsOk := params["objects"].([]string); objectsOk {
		args = objects
	}

	lsresp, err := sh.Request("ls", args...).Send(context.Background())
	if err != nil {
		common.Log.Warningf("failed to invoke IPFS ls api; %s", err.Error())
		return nil, err
	}
	defer lsresp.Close()

	buf, err := ioutil.ReadAll(lsresp.Output)
	if err != nil {
		common.Log.Warningf("failed to read IPFS ls output: %s", err.Error())
		return nil, err
	}

	common.Log.Debugf("received %d-byte response from IPFS ls api", len(buf))

	var respobj map[string]interface{}
	err = json.Unmarshal(buf, &respobj)
	if err != nil {
		common.Log.Warningf("failed to unmarshal IPFS ls output: %s", err.Error())
		return nil, err
	}

	if objects, objectsOk := respobj["Objects"].([]interface{}); objectsOk {
		for _, obj := range objects {
			resp = append(resp, obj)
		}
	}

	common.Log.Debugf("retrieved %d object(s) from IPFS ls api", len(resp))
	return resp, nil
}

// Query impl for IPFSProvider
func (p *IPFSProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for IPFS connectors")
}
