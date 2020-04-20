package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/jinzhu/gorm"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

// ZokratesProvider is a connector.ProviderAPI implementing orchestration for Zokrates
type ZokratesProvider struct {
	connectorID    uuid.UUID
	model          *gorm.DB
	config         map[string]interface{}
	networkID      *uuid.UUID
	applicationID  *uuid.UUID
	organizationID *uuid.UUID
	region         *string
	apiURL         *string
	apiPort        int
}

// InitZokratesProvider initializes and returns the Zokrates connector API provider
func InitZokratesProvider(connectorID uuid.UUID, networkID, applicationID, organizationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *ZokratesProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &ZokratesProvider{
		connectorID:    connectorID,
		model:          model,
		config:         config,
		networkID:      networkID,
		applicationID:  applicationID,
		organizationID: organizationID,
		region:         common.StringOrNil(region),
		apiURL:         common.StringOrNil(apiURL),
		apiPort:        int(apiPort),
	}
}

func (p *ZokratesProvider) apiClientFactory(basePath *string) *ipfs.Shell {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize Zokrates api client factory")
		return nil
	}

	return ipfs.NewShell(*apiURL)
}

func (p *ZokratesProvider) apiURLFactory(path string) *string {
	if p.apiURL == nil {
		return nil
	}

	suffix := ""
	if path != "" {
		suffix = fmt.Sprintf("/%s", path)
	}
	return common.StringOrNil(fmt.Sprintf("%s%s", *p.apiURL, suffix))
}

func (p *ZokratesProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// Deprovision undeploys all associated nodes and load balancers and removes them from the Zokrates connector
func (p *ZokratesProvider) Deprovision() error {
	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	for _, node := range nodes {
		common.Log.Debugf("Attempting to deprovision node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Delete(node)
		node.Delete()
	}

	return nil
}

// Provision configures a new load balancer and the initial Zokrates nodes and associates the resources with the Zokrates connector
func (p *ZokratesProvider) Provision() error {
	msg, _ := json.Marshal(map[string]interface{}{
		"connector_id": p.connectorID,
	})
	natsutil.NatsStreamingPublish(natsConnectorDenormalizeConfigSubject, msg)

	err := p.ProvisionNode()
	if err != nil {
		common.Log.Warning(err.Error())
	}

	return nil
}

// DeprovisionNode undeploys an existing node removes it from the Zokrates connector
func (p *ZokratesProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the Zokrates connector
func (p *ZokratesProvider) ProvisionNode() error {
	node := &network.Node{
		NetworkID:      *p.networkID,
		ApplicationID:  p.applicationID,
		OrganizationID: p.organizationID,
		Config:         p.rawConfig(),
	}

	if node.Create() {
		common.Log.Debugf("Created node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Append(node)

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID.String(),
		})
		natsutil.NatsStreamingPublish(natsConnectorResolveReachabilitySubject, msg)
	} else {
		return fmt.Errorf("Failed to provision node on connector: %s", p.connectorID)
	}

	return nil
}

// Reachable returns true if the Tableau API provider is available
func (p *ZokratesProvider) Reachable() bool {
	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	for _, node := range nodes {
		if node.ReachableOnPort(uint(p.apiPort)) {
			return true
		}
	}
	common.Log.Debugf("Connector is unreachable: %s", p.connectorID)
	return false
}

// Create impl for ZokratesProvider
func (p *ZokratesProvider) Create(params map[string]interface{}) (*ConnectedEntity, error) {
	return nil, errors.New("create not implemented for Zokrates connectors")
}

// Read impl for ZokratesProvider
func (p *ZokratesProvider) Read(id string) (*ConnectedEntity, error) {
	return nil, errors.New("read not implemented for Zokrates connectors")
}

// Update impl for ZokratesProvider
func (p *ZokratesProvider) Update(id string, params map[string]interface{}) error {
	return errors.New("update not implemented for Zokrates connectors")
}

// Delete impl for ZokratesProvider
func (p *ZokratesProvider) Delete(id string) error {
	return errors.New("delete not implemented for Zokrates connectors")
}

// List impl for ZokratesProvider
func (p *ZokratesProvider) List(params map[string]interface{}) ([]*ConnectedEntity, error) {
	return nil, errors.New("list not implemented for Zokrates connectors")
}

// Query impl for ZokratesProvider
func (p *ZokratesProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for Zokrates connectors")
}
