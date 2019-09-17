package connector

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/connector/provider"
	"github.com/provideapp/goldmine/network"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Connector{})
	db.Model(&Connector{}).AddIndex("idx_connectors_application_id", "application_id")
	db.Model(&Connector{}).AddIndex("idx_connectors_network_id", "network_id")
	db.Model(&Connector{}).AddIndex("idx_connectors_type", "type")
	db.Model(&Connector{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Connector{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

	db.Exec("ALTER TABLE connectors_load_balancers ADD CONSTRAINT connectors_load_balancers_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;")
	db.Exec("ALTER TABLE connectors_load_balancers ADD CONSTRAINT connectors_load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;")

	db.Exec("ALTER TABLE connectors_nodes ADD CONSTRAINT connectors_nodes_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;")
	db.Exec("ALTER TABLE connectors_nodes ADD CONSTRAINT connectors_nodes_node_id_nodes_id_foreign FOREIGN KEY (node_id) REFERENCES nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;")
}

// ConnectorAPI defines an interface for connector provisioning and deprovisioning
type ConnectorAPI interface {
	Deprovision() error
	Provision() error

	DeprovisionNode() error
	ProvisionNode() error
}

// Connector instances represent a logical connection to IPFS or other decentralized filesystem;
// in the future it may represent a logical connection to services of other types
type Connector struct {
	provide.Model
	ApplicationID   *uuid.UUID             `sql:"type:uuid" json:"application_id"`
	NetworkID       uuid.UUID              `sql:"not null;type:uuid" json:"network_id"`
	Name            *string                `sql:"not null" json:"name"`
	Type            *string                `sql:"not null" json:"type"`
	Config          *json.RawMessage       `sql:"type:json" json:"config"`
	EncryptedConfig *string                `sql:"type:bytea" json:"-"`
	AccessedAt      *time.Time             `json:"accessed_at"`
	LoadBalancer    []network.LoadBalancer `gorm:"many2many:connectors_load_balancers" json:"-"`
	Nodes           []network.Node         `gorm:"many2many:connectors_nodes" json:"-"`
}

// ParseConfig - parse the original JSON params used for Connector creation
func (c *Connector) ParseConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	if c.Config != nil {
		err := json.Unmarshal(*c.Config, &cfg)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal connector params; %s", err.Error())
			return nil
		}
	}
	return cfg
}

func (c *Connector) decryptedConfig() (map[string]interface{}, error) {
	decryptedParams := map[string]interface{}{}
	if c.EncryptedConfig != nil {
		encryptedConfigJSON, err := pgputil.PGPPubDecrypt([]byte(*c.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to decrypt encrypted connector config; %s", err.Error())
			return decryptedParams, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &decryptedParams)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted connector config; %s", err.Error())
			return decryptedParams, err
		}
	}
	return decryptedParams, nil
}

func (c *Connector) encryptConfig() bool {
	if c.EncryptedConfig != nil {
		encryptedConfig, err := pgputil.PGPPubEncrypt([]byte(*c.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to encrypt connector config; %s", err.Error())
			c.Errors = append(c.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
			return false
		}
		c.EncryptedConfig = common.StringOrNil(string(encryptedConfig))
	}
	return true
}

func (c *Connector) setConfig(cfg map[string]interface{}) {
	cfgJSON, _ := json.Marshal(cfg)
	_cfgJSON := json.RawMessage(cfgJSON)
	c.Config = &_cfgJSON
}

func (c *Connector) setEncryptedConfig(params map[string]interface{}) {
	paramsJSON, _ := json.Marshal(params)
	_paramsJSON := string(json.RawMessage(paramsJSON))
	c.EncryptedConfig = &_paramsJSON
	c.encryptConfig()
}

func (c *Connector) sanitizeConfig() {
	cfg := c.ParseConfig()

	encryptedCfg, err := c.decryptedConfig()
	if err != nil {
		encryptedCfg = map[string]interface{}{}
	}

	if credentials, credentialsOk := cfg["credentials"]; credentialsOk {
		encryptedCfg["credentials"] = credentials
		delete(cfg, "credentials")
	}

	c.setConfig(cfg)
	c.setEncryptedConfig(encryptedCfg)
}

func (c *Connector) deprovision() error {
	apiClient, err := c.connectorAPI()
	if err != nil {
		return fmt.Errorf("Failed to resolve connector API for %s connector: %s; %s", *c.Type, c.ID, err.Error())
	}
	err = apiClient.Deprovision()
	if err != nil {
		return fmt.Errorf("Failed to deprovision infrastructure for %s connector: %s; %s", *c.Type, c.ID, err.Error())
	}
	return nil
}

func (c *Connector) provision() error {
	apiClient, err := c.connectorAPI()
	if err != nil {
		return fmt.Errorf("Failed to resolve connector API for connector: %s; %s", c.ID, err.Error())
	}
	err = apiClient.Provision()
	if err != nil {
		return fmt.Errorf("Failed to provision infrastructure for %s connector: %s; %s", *c.Type, c.ID, err.Error())
	}
	return nil
}

// Create and persist a new Connector
func (c *Connector) Create() bool {
	db := dbconf.DatabaseConnection()

	if !c.Validate() {
		return false
	}

	c.sanitizeConfig()

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(c) {
			success := rowsAffected > 0
			if success && c.Type != nil {
				runDefaultProvisioner := false
				cfg := c.ParseConfig()

				switch *c.Type {
				case provider.IPFSConnectorProvider:
					_, gatewayURLOk := cfg["gateway_url"].(string)
					_, rpcURLOk := cfg["rpc_url"].(string)
					runDefaultProvisioner = !gatewayURLOk && !rpcURLOk
				default:
					// no-op
				}

				if runDefaultProvisioner {
					c.provision()
				}
			}
			return success
		}
	}
	return false
}

// Validate an Connector for persistence
func (c *Connector) Validate() bool {
	c.Errors = make([]*provide.Error, 0)
	if c.NetworkID == uuid.Nil {
		c.Errors = append(c.Errors, &provide.Error{
			Message: common.StringOrNil("Unable to deploy connector using unspecified network"),
		})
	}
	if c.Type == nil || strings.ToLower(*c.Type) != "ipfs" {
		c.Errors = append(c.Errors, &provide.Error{
			Message: common.StringOrNil("Unable to define connector of invalid type"),
		})
	}
	return len(c.Errors) == 0
}

// Delete a connector
func (c *Connector) Delete() bool {
	db := dbconf.DatabaseConnection()
	result := db.Delete(c)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			c.Errors = append(c.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
	success := len(c.Errors) == 0
	if success {
		c.deprovision()
	}
	return success
}

// connectorAPI returns an instance of the connector's underlying ConnectorAPI
func (c *Connector) connectorAPI() (ConnectorAPI, error) {
	if c.Type == nil {
		return nil, fmt.Errorf("No provider resolved for connector: %s", c.ID)
	}

	db := dbconf.DatabaseConnection()
	var apiClient ConnectorAPI

	switch *c.Type {
	case provider.IPFSConnectorProvider:
		apiClient = provider.InitIPFSProvider(c.ID, db.Model(c), c.ParseConfig())
	default:
		return nil, fmt.Errorf("No provider resolved for connector: %s", c.ID)
	}

	return apiClient, nil
}
