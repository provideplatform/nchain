/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package provider

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	provide "github.com/provideplatform/provide-go/api"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
	c2 "github.com/provideplatform/provide-go/api/c2"
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

func (p *ZokratesProvider) apiClientFactory(basePath *string) *provide.Client {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize zokrates api client factory")
		return nil
	}

	parts := strings.Split(*apiURL, "://")
	return &provide.Client{
		Host:   parts[len(parts)-1],
		Scheme: parts[0],
	}
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
	loadBalancers := make([]*c2.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, balancer := range loadBalancers {
		p.model.Association("LoadBalancers").Delete(balancer)
	}

	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	for _, node := range nodes {
		common.Log.Debugf("Attempting to deprovision node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Delete(node)
		node.Delete("") // FIXME -- needs c2 API token
	}

	for _, balancer := range loadBalancers {
		msg, _ := json.Marshal(map[string]interface{}{
			"load_balancer_id": balancer.ID,
		})
		natsutil.NatsJetstreamPublish(natsLoadBalancerDeprovisioningSubject, msg)
	}

	return nil
}

// Provision configures a new load balancer and the initial Zokrates nodes and associates the resources with the Zokrates connector
func (p *ZokratesProvider) Provision() error {
	loadBalancer, err := c2.CreateLoadBalancer("", map[string]interface{}{
		"network_id":      *p.networkID,
		"application_id":  p.applicationID,
		"organization_id": p.organizationID,
		"type":            common.StringOrNil(ElasticsearchConnectorProvider),
		"description":     common.StringOrNil(fmt.Sprintf("Zokrates Connector Load Balancer")),
		"region":          p.region,
		"config":          p.config,
	})

	if err == nil {
		common.Log.Debugf("Created load balancer %s on connector: %s", loadBalancer.ID, p.connectorID)
		p.model.Association("LoadBalancers").Append(loadBalancer)

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID,
		})
		natsutil.NatsJetstreamPublish(natsConnectorDenormalizeConfigSubject, msg)

		err := p.ProvisionNode()
		if err != nil {
			common.Log.Warning(err.Error())
		}
	} else {
		return fmt.Errorf("Failed to provision load balancer on connector: %s; %s", p.connectorID, err.Error())
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

	if node.Create("") { // FIXME -- needs c2 API token
		common.Log.Debugf("Created node %s on connector: %s", node.ID, p.connectorID)
		p.model.Association("Nodes").Append(node)

		loadBalancers := make([]*c2.LoadBalancer, 0)
		p.model.Association("LoadBalancers").Find(&loadBalancers)
		for _, balancer := range loadBalancers {
			msg, _ := json.Marshal(map[string]interface{}{
				"load_balancer_id": balancer.ID.String(),
				"node_id":          node.ID.String(),
			})
			natsutil.NatsJetstreamPublish(natsLoadBalancerBalanceNodeSubject, msg)
		}

		msg, _ := json.Marshal(map[string]interface{}{
			"connector_id": p.connectorID.String(),
		})
		natsutil.NatsJetstreamPublish(natsConnectorResolveReachabilitySubject, msg)
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
	uri := "compile"
	circuitID, circuitIDOk := params["circuit_id"].(string)
	if circuitIDOk {
		// generate-proof
		uri = "generate-proof"
	}

	if uri == "compile" {
		// TODO: inject a signed NATS bearer JWT into params as `jwt`
	}

	apiClient := p.apiClientFactory(nil)
	status, resp, err := apiClient.PostWithTLSClientConfig(uri, params, p.tlsClientConfigFactory())

	if err != nil {
		msg := "failed to compile circuit"
		if circuitIDOk {
			msg = fmt.Sprintf("failed to generate proof for circuit: %s", circuitID)
		}

		common.Log.Warningf("%s; %s", msg, err.Error())
		return nil, err
	}

	if status == 201 && !circuitIDOk {
		common.Log.Debugf("compiled circuit via zokrates connector; %s", resp)
	} else if status == 201 {
		common.Log.Debugf(fmt.Sprintf("generate proof for circuit: %s; response: %s", circuitID, resp))
	}

	entity := &ConnectedEntity{}
	respJSON, _ := json.Marshal(resp)
	err = json.Unmarshal(respJSON, &entity)

	if err != nil {
		msg := "failed to compile circuit"
		if circuitIDOk {
			msg = fmt.Sprintf("failed to generate proof for circuit: %s", circuitID)
		}

		common.Log.Warningf("%s; %s", msg, err.Error())
		return nil, err
	}

	if circuitIDOk {
		entity.ID = &circuitID
	}

	if name, nameOk := params["name"].(string); nameOk {
		entity.Name = &name
	}

	if source, sourceOk := params["source"].(string); sourceOk {
		entity.Source = &source
	}

	if entity.Raw == nil {
		entity.Raw = common.StringOrNil(string(respJSON))
	}

	return entity, nil
}

// Find impl for ZokratesProvider -- fetches the verifying key for a given circuit id
func (p *ZokratesProvider) Find(id string) (*ConnectedEntity, error) {
	apiClient := p.apiClientFactory(nil)
	status, resp, err := apiClient.GetWithTLSClientConfig(fmt.Sprintf("vk/%s", id), map[string]interface{}{}, p.tlsClientConfigFactory())

	if err != nil {
		common.Log.Warningf("failed to initiate baseline protocol; %s", err.Error())
		return nil, err
	}

	if status == 201 {
		common.Log.Debugf("created agreement via baseline connector; %s", resp)
	}

	entity := &ConnectedEntity{}
	respJSON, _ := json.Marshal(resp)
	json.Unmarshal(respJSON, &entity)

	entity.ID = common.StringOrNil(id)

	if entity.Raw == nil {
		entity.Raw = common.StringOrNil(string(respJSON))
	}

	return entity, nil
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

func (p *ZokratesProvider) tlsClientConfigFactory() *tls.Config {
	var tlsConfig *tls.Config
	if common.DefaultInfrastructureUsesSelfSignedCertificate {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return tlsConfig
}
