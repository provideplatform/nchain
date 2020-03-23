package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/jinzhu/gorm"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

// IPFSProvider is a connector.ProviderAPI implementing orchestration for IPFS
type IPFSProvider struct {
	connectorID    uuid.UUID
	model          *gorm.DB
	config         map[string]interface{}
	networkID      *uuid.UUID
	applicationID  *uuid.UUID
	organizationID *uuid.UUID
	region         *string
	apiURL         *string
	apiPort        int
	gatewayPort    int
}

// InitIPFSProvider initializes and returns the IPFS connector API provider
func InitIPFSProvider(connectorID uuid.UUID, networkID, applicationID, organizationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *IPFSProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	gatewayPort, gatewayPortOk := config["gateway_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || !gatewayPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &IPFSProvider{
		connectorID:    connectorID,
		model:          model,
		config:         config,
		networkID:      networkID,
		applicationID:  applicationID,
		organizationID: organizationID,
		region:         common.StringOrNil(region),
		apiURL:         common.StringOrNil(apiURL),
		apiPort:        int(apiPort),
		gatewayPort:    int(gatewayPort),
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

	var tlsConfig *tls.Config
	if common.DefaultInfrastructureUsesSelfSignedCertificate {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	ipfsClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			Proxy:             http.ProxyFromEnvironment,
			TLSClientConfig:   tlsConfig,
		},
	}

	return ipfs.NewShellWithClient(*apiURL, ipfsClient)
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
		natsutil.NatsStreamingPublish(natsLoadBalancerDeprovisioningSubject, msg)
	}

	return nil
}

// Provision configures a new load balancer and the initial IPFS nodes and associates the resources with the IPFS connector
func (p *IPFSProvider) Provision() error {
	loadBalancer := &network.LoadBalancer{
		NetworkID:      *p.networkID,
		ApplicationID:  p.applicationID,
		OrganizationID: p.organizationID,
		Type:           common.StringOrNil(IPFSConnectorProvider),
		Description:    common.StringOrNil(fmt.Sprintf("IPFS Connector Load Balancer")),
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

// DeprovisionNode undeploys an existing node removes it from the IPFS connector
func (p *IPFSProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the IPFS connector
func (p *IPFSProvider) ProvisionNode() error {
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
func (p *IPFSProvider) Create(params map[string]interface{}) (*ConnectedEntity, error) {
	return nil, errors.New("create not implemented for IPFS connectors")
}

// Read impl for IPFSProvider
func (p *IPFSProvider) Read(id string) (*ConnectedEntity, error) {
	return nil, errors.New("read not implemented for IPFS connectors")
}

// Update impl for IPFSProvider
func (p *IPFSProvider) Update(id string, params map[string]interface{}) error {
	return errors.New("update not implemented for IPFS connectors")
}

// Delete impl for IPFSProvider
func (p *IPFSProvider) Delete(id string) error {
	return errors.New("delete not implemented for IPFS connectors")
}

// List impl for IPFSProvider
func (p *IPFSProvider) List(params map[string]interface{}) ([]*ConnectedEntity, error) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed IPFS connector ls request; %s", r)
		}
	}()

	sh := p.apiClientFactory(nil)
	if sh == nil {
		return nil, fmt.Errorf("unable to list IPFS resources for connector: %s; failed to initialize IPFS shell", p.connectorID)
	}

	resp := make([]*ConnectedEntity, 0)
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
			if entity, entityOk := obj.(map[string]interface{}); entityOk {
				hash := common.StringOrNil(entity["Hash"].(string))

				apiURL := p.apiURLFactory("api/v0/get")
				href := fmt.Sprintf("%s?arg=/ipfs/%s&encoding=json&stream-channels=true", *apiURL, *hash)

				var name *string
				var size *uint64

				if links, linksOk := entity["Links"].([]interface{}); linksOk && len(links) == 1 {
					if link, linkOk := links[0].(map[string]interface{}); linkOk {
						if entityName, entityNameOk := link["Name"].(string); entityNameOk {
							name = common.StringOrNil(entityName)
						}

						if entitySize, entitySizeOk := link["Size"].(float64); entitySizeOk {
							sizeVal := uint64(entitySize)
							size = &sizeVal
						}
					}
				}

				resp = append(resp, &ConnectedEntity{
					ID:       hash,
					Hash:     hash,
					Href:     &href,
					Filename: name,
					Size:     size,

					// CreatedAt  *time.Time             `json:"created_at,omitempty"`
					// DataURL    *string                `json:"data_url,omitempty"`
					// Errors     []*provide.Error       `json:"errors,omitempty"`
					// Type       *string                `json:"type,omitempty"`
					// Hash       *string                `json:"hash,omitempty"`
					// Href       *string                `json:"href,omitempty"`
					// Metadata   map[string]interface{} `json:"metadata,omitempty"`
					// ModifiedAt *time.Time             `json:"modified_at,omitempty"`
					// Name       *string                `json:"name,omitempty"`
					// Raw        *string                `json:"raw,omitempty"`
					// // relations
					// Parent   *ConnectedEntity    `json:"parent,omitempty"`
					// Children *[]*ConnectedEntity `json:"children,omitempty"`
				})
			}
		}
	}

	common.Log.Debugf("retrieved %d object(s) from IPFS ls api", len(resp))
	return resp, nil
}

// Query impl for IPFSProvider
func (p *IPFSProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for IPFS connectors")
}
