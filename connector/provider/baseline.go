package provider

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	provide "github.com/provideservices/provide-go"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
)

// BaselineProvider is a connector.ProviderAPI implementing orchestration for Baseline
type BaselineProvider struct {
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

// InitBaselineProvider initializes and returns the Baseline connector API provider
func InitBaselineProvider(connectorID uuid.UUID, networkID, applicationID, organizationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *BaselineProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &BaselineProvider{
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

func (p *BaselineProvider) apiClientFactory(basePath *string) *provide.APIClient {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize Baseline api client factory")
		return nil
	}

	parts := strings.Split(*apiURL, "://")
	return &provide.APIClient{
		Host:   parts[len(parts)-1],
		Scheme: parts[0],
	}
}

func (p *BaselineProvider) apiURLFactory(path string) *string {
	if p.apiURL == nil {
		return nil
	}

	suffix := ""
	if path != "" {
		suffix = fmt.Sprintf("/%s", path)
	}
	return common.StringOrNil(fmt.Sprintf("%s%s", *p.apiURL, suffix))
}

func (p *BaselineProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// Deprovision undeploys all associated nodes and load balancers and removes them from the Baseline connector
func (p *BaselineProvider) Deprovision() error {
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
		natsutil.NatsStreamingPublish(natsLoadBalancerDeprovisioningSubject, msg)
	}

	return nil
}

// Provision configures a new load balancer and the initial Baseline nodes and associates the resources with the Baseline connector
func (p *BaselineProvider) Provision() error {
	loadBalancer := &network.LoadBalancer{
		NetworkID:      *p.networkID,
		ApplicationID:  p.applicationID,
		OrganizationID: p.organizationID,
		Type:           common.StringOrNil(RESTConnectorProvider),
		Description:    common.StringOrNil(fmt.Sprintf("Baseline Connector Load Balancer")),
		Region:         p.region,
		Config:         p.rawConfig(),
	}

	if loadBalancer.Create() {
		common.Log.Debugf("Created load balancer %s on connector: %s", loadBalancer.ID, p.connectorID)
		p.model.Association("LoadBalancers").Append(loadBalancer)

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID,
		})
		natsutil.NatsStreamingPublish(natsConnectorDenormalizeConfigSubject, msg)

		err := p.ProvisionNode()
		if err != nil {
			common.Log.Warning(err.Error())
		}
	} else {
		return fmt.Errorf("Failed to provision load balancer on connector: %s; %s", p.connectorID, *loadBalancer.Errors[0].Message)
	}

	return nil
}

// DeprovisionNode undeploys an existing node removes it from the Baseline connector
func (p *BaselineProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the Baseline connector
func (p *BaselineProvider) ProvisionNode() error {
	node := &network.Node{
		NetworkID:      *p.networkID,
		ApplicationID:  p.applicationID,
		OrganizationID: p.organizationID,
		Config:         p.rawConfig(),
	}

	if node.Create() {
		common.Log.Debugf("Created node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Append(node)

		loadBalancers := make([]*network.LoadBalancer, 0)
		p.model.Association("LoadBalancers").Find(&loadBalancers)
		for _, balancer := range loadBalancers {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": balancer.ID.String(),
				"node_id":          node.ID.String(),
			})
			natsutil.NatsStreamingPublish(natsLoadBalancerBalanceNodeSubject, msg)
		}

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
func (p *BaselineProvider) Reachable() bool {
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

// Create impl for BaselineProvider
func (p *BaselineProvider) Create(params map[string]interface{}) (*ConnectedEntity, error) {
	// golang DTO validation

	apiClient := p.apiClientFactory(nil)
	status, resp, err := apiClient.PostWithTLSClientConfig("graphql", params, p.tlsClientConfigFactory())

	if err != nil {
		common.Log.Warningf("failed to initiate baseline protocol; %s", err.Error())
		return nil, err
	}

	if status == 201 {
		common.Log.Debugf("created agreement via baseline connector; %s", resp)
	}

	entity := &ConnectedEntity{}
	respJSON, _ := json.Marshal(resp)
	err = json.Unmarshal(respJSON, &entity)

	if err != nil {
		common.Log.Warningf("failed to initiate baseline protocol; %s", err.Error())
		return nil, err
	}

	if entity.Raw == nil {
		entity.Raw = common.StringOrNil(string(respJSON))
	}

	return entity, nil
}

// Find impl for BaselineProvider
func (p *BaselineProvider) Find(id string) (*ConnectedEntity, error) {
	apiClient := p.apiClientFactory(nil)
	status, resp, err := apiClient.PostWithTLSClientConfig("graphql", map[string]interface{}{
		"query": fmt.Sprintf("{ agreement(id: \"%s\") { _id, linkedId, buyerSignature, supplierSignature} } ", id),
	}, p.tlsClientConfigFactory())

	if err != nil {
		common.Log.Warningf("failed to fetch baseline protocol document with id %s; %s", id, err.Error())
		return nil, err
	}

	if status == 200 {
		common.Log.Debugf("fetched baseline protocol document with id %s via baseline connector; %s", id, resp)
	}

	entity := &ConnectedEntity{}
	respJSON, _ := json.Marshal(resp)
	err = json.Unmarshal(respJSON, &entity)

	if err != nil {
		common.Log.Warningf("failed to fetch baseline protocol document with id %s; %s", id, err.Error())
		return nil, err
	}

	if entity.Raw == nil {
		entity.Raw = common.StringOrNil(string(respJSON))
	}

	return entity, nil
}

// Update impl for BaselineProvider
func (p *BaselineProvider) Update(id string, params map[string]interface{}) error {
	// golang DTO validation

	apiClient := p.apiClientFactory(nil)
	status, resp, err := apiClient.PostWithTLSClientConfig("graphql", params, p.tlsClientConfigFactory())

	if err != nil {
		common.Log.Warningf("failed to update baseline protocol; %s", err.Error())
		return err
	}

	if status == 200 || status == 202 || status == 204 {
		common.Log.Debugf("updated baseline protocol; %s", resp)
	}

	entity := &ConnectedEntity{}
	respJSON, _ := json.Marshal(resp)
	err = json.Unmarshal(respJSON, &entity)

	if err != nil {
		common.Log.Warningf("failed to update baseline protocol; failed handling baseline api response; %s", err.Error())
		return err
	}

	return nil
}

// Delete impl for BaselineProvider
func (p *BaselineProvider) Delete(id string) error {
	return errors.New("delete not implemented for Baseline connectors")
}

// List impl for BaselineProvider
func (p *BaselineProvider) List(params map[string]interface{}) ([]*ConnectedEntity, error) {
	apiClient := p.apiClientFactory(nil)
	if apiClient == nil {
		return nil, fmt.Errorf("failed to resolve baseline api client for connector: %s", p.connectorID)
	}

	var status int
	var resp interface{}
	var err error

	if id, idOk := params["id"].(string); idOk {
		status, resp, err = apiClient.PostWithTLSClientConfig("graphql", map[string]interface{}{
			"query": fmt.Sprintf("{ agreement(id: \"%s\") { _id, linkedId, buyerSignature, supplierSignature} } ", id),
		}, p.tlsClientConfigFactory())
	} else {
		status, resp, err = apiClient.PostWithTLSClientConfig("graphql", map[string]interface{}{
			"query": "{ agreements { _id, linkedId, buyerSignature, supplierSignature} } ",
		}, p.tlsClientConfigFactory())
	}

	if err != nil {
		common.Log.Warningf("failed to list baseline protocol documents; %s", err.Error())
		return nil, err
	}

	entities := make([]*ConnectedEntity, 0)

	if status == 200 {
		respJSON, _ := json.Marshal(resp)
		json.Unmarshal(respJSON, &entities)

		if id, idOk := params["id"].(string); idOk {
			status, resp, err = apiClient.PostWithTLSClientConfig("graphql", map[string]interface{}{
				"query": fmt.Sprintf("{ agreementByLinkedId(linkedId: \"%s\") { _id, linkedId, buyerSignature, supplierSignature} } ", id),
			}, p.tlsClientConfigFactory())
		}

		for _, entity := range entities {
			if entity.Raw == nil {
				entityJSON, _ := json.Marshal(entity)
				entity.Raw = common.StringOrNil(string(entityJSON))
			}
		}

		common.Log.Debugf("fetched %d baseline protocol documents via baseline connector", len(entities))
	} else {
		return nil, fmt.Errorf("failed to fetch connected entities via baseline connector; status: %d", status)
	}

	return entities, nil
}

// Query impl for BaselineProvider
func (p *BaselineProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for Baseline connectors")
}

func (p *BaselineProvider) tlsClientConfigFactory() *tls.Config {
	var tlsConfig *tls.Config
	if common.DefaultInfrastructureUsesSelfSignedCertificate {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return tlsConfig
}
