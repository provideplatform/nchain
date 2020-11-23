package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
	c2 "github.com/provideservices/provide-go/api/c2"
)

// MongoDBProvider is a connector.ProviderAPI implementing orchestration for MongoDB
type MongoDBProvider struct {
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

// InitMongoDBProvider initializes and returns the MongoDB connector API provider
func InitMongoDBProvider(connectorID uuid.UUID, networkID, applicationID, organizationID *uuid.UUID, model *gorm.DB, config map[string]interface{}) *MongoDBProvider {
	region, regionOk := config["region"].(string)
	apiURL, _ := config["api_url"].(string)
	apiPort, apiPortOk := config["api_port"].(float64)
	if connectorID == uuid.Nil || !regionOk || !apiPortOk || networkID == nil || *networkID == uuid.Nil {
		return nil
	}
	return &MongoDBProvider{
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

func (p *MongoDBProvider) apiClientFactory(basePath *string) *mongo.Client {
	uri := ""
	if basePath != nil {
		uri = *basePath
	}
	apiURL := p.apiURLFactory(uri)
	if apiURL == nil {
		common.Log.Warningf("unable to initialize MongoDB api client factory")
		return nil
	}

	mongoURL := strings.Replace(*apiURL, "https://", "mongodb://", -1) // FIXME
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		common.Log.Warningf("failed to establish mongodb connection for connector: %s; %s", p.connectorID, err.Error())
		return nil
	}

	return client
}

func (p *MongoDBProvider) apiURLFactory(path string) *string {
	suffix := ""
	if path != "" {
		suffix = fmt.Sprintf("/%s", path)
	}

	// FIXME
	nodes := make([]*network.Node, 0)
	p.model.Association("Nodes").Find(&nodes)
	if len(nodes) > 0 {
		if nodes[0].IPv4 != nil {
			var user *string
			var passwd *string
			if env, envOk := p.config["env"].(map[string]interface{}); envOk {
				if usr, usrok := env["MONGO_INITDB_ROOT_USERNAME"].(string); usrok {
					user = &usr
				}
				if password, passwordOk := env["MONGO_INITDB_ROOT_PASSWORD"].(string); passwordOk {
					passwd = &password
				}
			}
			if user != nil && passwd != nil {
				return common.StringOrNil(fmt.Sprintf("mongodb://%s:%s@%s:%d%s", *user, *passwd, *nodes[0].IPv4, p.apiPort, suffix))
			}

			return common.StringOrNil(fmt.Sprintf("mongodb://%s:%d%s", *nodes[0].IPv4, p.apiPort, suffix))
		}
	}

	// FIXME-- prefer the below to the above!!! rethink which connector types get load balanced and which represent clustered services that can receive a direct connection to a single node...
	// if p.apiURL == nil {
	// 	return nil
	// }

	// return common.StringOrNil(fmt.Sprintf("%s%s", *p.apiURL, suffix))
	return nil
}

func (p *MongoDBProvider) rawConfig() *json.RawMessage {
	cfgJSON, _ := json.Marshal(p.config)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// Deprovision undeploys all associated nodes and load balancers and removes them from the MongoDB connector
func (p *MongoDBProvider) Deprovision() error {
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
		natsutil.NatsStreamingPublish(natsLoadBalancerDeprovisioningSubject, msg)
	}

	return nil
}

// Provision configures a new load balancer and the initial MongoDB nodes and associates the resources with the MongoDB connector
func (p *MongoDBProvider) Provision() error {
	loadBalancer, err := c2.CreateLoadBalancer("", map[string]interface{}{
		"network_id":      *p.networkID,
		"application_id":  p.applicationID,
		"organization_id": p.organizationID,
		"type":            common.StringOrNil(ElasticsearchConnectorProvider),
		"description":     common.StringOrNil(fmt.Sprintf("MongoDB Connector Load Balancer")),
		"region":          p.region,
		"config":          p.config,
	})

	if err == nil {
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
		return fmt.Errorf("Failed to provision load balancer on connector: %s; %s", p.connectorID, err.Error())
	}

	return nil
}

// DeprovisionNode undeploys an existing node removes it from the MongoDB connector
func (p *MongoDBProvider) DeprovisionNode() error {
	node := &network.Node{}
	p.model.Association("Nodes").Find(node)

	return nil
}

// ProvisionNode deploys and load balances a new node and associates it with the MongoDB connector
func (p *MongoDBProvider) ProvisionNode() error {
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

// Reachable returns true if the MongoDB API provider is available
func (p *MongoDBProvider) Reachable() bool {
	loadBalancers := make([]*c2.LoadBalancer, 0)
	p.model.Association("LoadBalancers").Find(&loadBalancers)
	for _, loadBalancer := range loadBalancers {
		if loadBalancer.ReachableOnPort(uint(p.apiPort)) {
			return true
		}
	}
	common.Log.Debugf("Connector is unreachable: %s", p.connectorID)
	return false
}

// Create impl for MongoDBProvider
func (p *MongoDBProvider) Create(params map[string]interface{}) (*ConnectedEntity, error) {
	var entity *ConnectedEntity
	var err error

	mongo := p.apiClientFactory(nil)
	if mongo == nil {
		return nil, fmt.Errorf("failed to establish mongodb connection for connector: %s; %s", p.connectorID, err.Error())
	}
	defer mongo.Disconnect(context.Background())

	if db, dbok := params["db"].(string); dbok {
		mongoDB := mongo.Database(db)

		usr, usrok := params["user"].(string)
		passwd, passwdok := params["password"].(string)
		roles, rolesok := params["roles"].([]interface{})
		if usrok && passwdok && rolesok {
			result := mongoDB.RunCommand(
				context.Background(),
				&bson.M{
					"createUser": usr,
					"pwd":        passwd,
					"roles":      roles,
				},
			)
			err = result.Err()
			if err != nil {
				err = fmt.Errorf("failed to execute createUser command via mongodb connector: %s; %s", p.connectorID, err.Error())
			}
		} else {
			err = fmt.Errorf("failed to execute arbitrary mongodb command via mongodb connector: %s; only createUser is currently implemented", p.connectorID)
		}
	}

	if err != nil {
		common.Log.Warning(err.Error())
	}

	return entity, err
}

// Find impl for MongoDBProvider
func (p *MongoDBProvider) Find(id string) (*ConnectedEntity, error) {
	return nil, errors.New("read not implemented for MongoDB connectors")
}

// Update impl for MongoDBProvider
func (p *MongoDBProvider) Update(id string, params map[string]interface{}) error {
	return errors.New("update not implemented for MongoDB connectors")
}

// Delete impl for MongoDBProvider
func (p *MongoDBProvider) Delete(id string) error {
	return errors.New("delete not implemented for MongoDB connectors")
}

// List impl for MongoDBProvider
func (p *MongoDBProvider) List(params map[string]interface{}) ([]*ConnectedEntity, error) {
	return nil, errors.New("list not implemented for MongoDB connectors")
}

// Query impl for MongoDBProvider
func (p *MongoDBProvider) Query(q string) (interface{}, error) {
	return nil, errors.New("query not implemented for MongoDB connectors")
}
