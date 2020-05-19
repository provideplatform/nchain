package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // PostgreSQL dialect

	dbconf "github.com/kthomas/go-db-config"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

// SQLProvider is a connector.ProviderAPI implementing orchestration for SQL
type SQLProvider struct {
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

// InitSQLProvider initializes and returns the SQL connector API provider
func InitSQLProvider(connectorID uuid.UUID, networkID, applicationID, organizationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *SQLProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &SQLProvider{
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

func (p *SQLProvider) apiClientFactory(basePath *string) *gorm.DB {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize SQL api client factory")
		return nil
	}

	dbcfg := &dbconf.DBConfig{
		DatabaseHost: strings.Replace(*apiURL, "https://", "", -1), // FIXME
		DatabasePort: uint(p.apiPort),
	}

	if env, envOk := p.config["env"].(map[string]interface{}); envOk {
		if usr, usrok := env["POSTGRES_USER"].(string); usrok {
			dbcfg.DatabaseUser = usr
		}
		if password, passwordOk := env["POSTGRES_PASSWORD"].(string); passwordOk {
			dbcfg.DatabasePassword = password
		}
	}

	client, err := dbconf.DatabaseConnectionFactory(dbcfg)
	if err != nil {
		common.Log.Warningf("failed to establish sql connection for connector: %s; %s", p.connectorID, err.Error())
		return nil
	}

	return client
}

func (p *SQLProvider) apiURLFactory(path string) *string {
	suffix := ""
	if path != "" {
		suffix = fmt.Sprintf("/%s", path)
	}

	// FIXME
	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	if len(nodes) > 0 {
		if nodes[0].Host != nil {
			if strings.Contains(*nodes[0].Host, fmt.Sprintf(":%d", p.apiPort)) {
				return common.StringOrNil(fmt.Sprintf("%s%s", *nodes[0].Host, suffix))
			} else {
				return common.StringOrNil(fmt.Sprintf("%s:%d%s", *nodes[0].Host, p.apiPort, suffix))
			}
		}
	}

	// FIXME-- prefer the below to the above!!! rethink which connector types get load balanced and which represent clustered services that can receive a direct connection to a single node...
	// if p.apiURL == nil {
	// 	return nil
	// }

	// return common.StringOrNil(fmt.Sprintf("%s%s", *p.apiURL, suffix))

	return nil
}

func (p *SQLProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// Deprovision undeploys all associated nodes and load balancers and removes them from the SQL connector
func (p *SQLProvider) Deprovision() error {
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

// Provision configures a new load balancer and the initial SQL nodes and associates the resources with the SQL connector
func (p *SQLProvider) Provision() error {
	loadBalancer := &network.LoadBalancer{
		NetworkID:      *p.networkID,
		ApplicationID:  p.applicationID,
		OrganizationID: p.organizationID,
		Type:           common.StringOrNil(SQLConnectorProvider),
		Description:    common.StringOrNil(fmt.Sprintf("SQL Connector Load Balancer")),
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

// DeprovisionNode undeploys an existing node removes it from the SQL connector
func (p *SQLProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the SQL connector
func (p *SQLProvider) ProvisionNode() error {
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

// Reachable returns true if the SQL API provider is available
func (p *SQLProvider) Reachable() bool {
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

// Create impl for SQLProvider
func (p *SQLProvider) Create(params map[string]interface{}) (*ConnectedEntity, error) {
	var entity *ConnectedEntity
	var err error

	dbconn := p.apiClientFactory(nil)
	if dbconn == nil {
		return nil, fmt.Errorf("failed to establish sql connection for connector: %s", p.connectorID)
	}
	defer dbconn.Close()

	if db, dbok := params["db"].(string); dbok {
		usr, usrok := params["user"].(string)
		passwd, passwdok := params["password"].(string)
		if usrok && passwdok {
			// FIXME-- don't default to superuser... :\
			result := dbconn.Exec("CREATE USER ? WITH SUPERUSER; ALTER USER ? WITH PASSWORD '?'", usr, usr, passwd)
			err = result.Error
			if err != nil {
				err = fmt.Errorf("failed to execute CREATE USER command via sql connector: %s; %s", p.connectorID, err.Error())
			}
		}
		if err == nil {
			result := dbconn.Exec("CREATE DATABASE ? OWNER ?", db, usr)
			err = result.Error
			if err != nil {
				err = fmt.Errorf("failed to execute CREATE DATABASE command via sql connector: %s; %s", p.connectorID, err.Error())
			}
		}
	}

	if err != nil {
		common.Log.Warning(err.Error())
	}

	return entity, err
}

// Find impl for SQLProvider
func (p *SQLProvider) Find(id string) (*ConnectedEntity, error) {
	return nil, errors.New("read not implemented for SQL connectors")
}

// Update impl for SQLProvider
func (p *SQLProvider) Update(id string, params map[string]interface{}) error {
	return errors.New("update not implemented for SQL connectors")
}

// Delete impl for SQLProvider
func (p *SQLProvider) Delete(id string) error {
	return errors.New("delete not implemented for SQL connectors")
}

// List impl for SQLProvider
func (p *SQLProvider) List(params map[string]interface{}) ([]*ConnectedEntity, error) {
	return nil, errors.New("list not implemented for SQL connectors")
}

// Query impl for SQLProvider
func (p *SQLProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for SQL connectors")
}
