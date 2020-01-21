package connector

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/connector/provider"
	"github.com/provideapp/goldmine/network"
	provide "github.com/provideservices/provide-go"
)

// Connector instances represent a logical connection to IPFS or other decentralized filesystem;
// in the future it may represent a logical connection to services of other types
type Connector struct {
	provide.Model
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID       uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name            *string          `sql:"not null" json:"name"`
	Type            *string          `sql:"not null" json:"type"`
	Status          *string          `sql:"not null;default:'init'" json:"status"`
	Description     *string          `json:"description"`
	Config          *json.RawMessage `sql:"type:json" json:"config,omitempty"`
	EncryptedConfig *string          `sql:"type:bytea" json:"-"`
	AccessedAt      *time.Time       `json:"accessed_at,omitempty"`

	Details *Details `sql:"-" json:"details,omitempty"`

	LoadBalancers []network.LoadBalancer `gorm:"many2many:connectors_load_balancers" json:"-"`
	Nodes         []network.Node         `gorm:"many2many:connectors_nodes" json:"-"`
}

// Details is a generic representation for a type-specific enrichment of a described connector;
// the details object may have complexity of its own, such as paginated subresults
type Details struct {
	Page *int64      `json:"page,omitempty"`
	RPP  *int64      `json:"rpp,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func (c *Connector) enrich(enrichment *string, params map[string]interface{}) error {
	apiClient, err := c.connectorAPI()
	if err != nil {
		return fmt.Errorf("failed to resolve connector API for %s connector: %s; %s", *c.Type, c.ID, err.Error())
	}

	switch enrichment {
	case nil:
		// default IPFS enrichment is the ls() command to retrieve directory listing
		// page := int64(1)
		// rpp := int64(25)

		// if pg, pgOk := params["page"].(float64); pgOk {
		// 	page = int64(pg)
		// }
		// if perpg, perpgOk := params["rpp"].(float64); perpgOk {
		// 	rpp = int64(perpg)
		// }

		resp, err := apiClient.List(params)
		if err != nil {
			common.Log.Warningf("failed to enrich connector: %s; %s", c.ID, err.Error())
			return err
		}
		c.Details = &Details{
			// Page: page,
			// RPP: rpp
			Data: resp,
		}
	default:
		common.Log.Warningf("failed to enrich connector: %s; unsupported enrichment", c.ID)
	}
	return nil
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

func (c *Connector) mergedConfig() map[string]interface{} {
	cfg := c.ParseConfig()
	encryptedConfig, err := c.decryptedConfig()
	if err != nil {
		encryptedConfig = map[string]interface{}{}
	}

	for k := range encryptedConfig {
		cfg[k] = encryptedConfig[k]
	}
	return cfg
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
	c.updateStatus(dbconf.DatabaseConnection(), "deprovisioning", nil)

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
	db := dbconf.DatabaseConnection()
	c.updateStatus(db, "provisioning", nil)

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
		apiClient, err := c.connectorAPI()
		if err != nil {
			msg := fmt.Sprintf("Failed to resolve connector API for connector: %s; %s", c.ID, err.Error())
			c.updateStatus(db, "failed", &msg)
			return errors.New(msg)
		}
		err = apiClient.Provision()
		if err != nil {
			msg := fmt.Sprintf("Failed to provision infrastructure for %s connector: %s; %s", *c.Type, c.ID, err.Error())
			c.updateStatus(db, "failed", &msg)
			return errors.New(msg)
		}
	} else {
		common.Log.Debugf("Default provisioner not being run for connector: %s", c.ID)
		c.updateStatus(db, "active", nil)
	}

	return nil
}

func (c *Connector) apiPort() uint {
	cfg := c.ParseConfig()
	port := uint(0)
	if apiPort, apiPortOk := cfg["api_port"].(float64); apiPortOk {
		port = uint(apiPort)
	}
	return port
}

func (c *Connector) apiURL(db *gorm.DB) *string {
	port := c.apiPort()
	if port == 0 {
		return nil
	}

	var host *string
	loadBalancers := make([]*network.LoadBalancer, 0)
	db.Model(c).Association("LoadBalancers").Find(&loadBalancers)
	if len(loadBalancers) == 1 {
		host = loadBalancers[0].Host
	} else if len(loadBalancers) > 1 {
		common.Log.Warningf("Ambiguous loadbalancing configuration for connector: %s; no api url resolved", c.ID)
	}

	if host == nil {
		return nil
	}
	return common.StringOrNil(fmt.Sprintf("https://%s:%d", *host, port)) // FIXME-- allow specification of url scheme in cfg := c.ParseConfig()
}

func (c *Connector) denormalizeConfig() error {
	if c.Type != nil {
		switch *c.Type {
		case provider.IPFSConnectorProvider:
			return c.resolveAPIURL()
		default:
			// no-op
		}
	}
	return nil
}

func (c *Connector) resolveAPIURL() error {
	db := dbconf.DatabaseConnection()
	apiURL := c.apiURL(db)
	if apiURL == nil {
		return fmt.Errorf("Failed to resolve API url for connector: %s", c.ID)
	}
	cfg := c.ParseConfig()
	cfg["api_url"] = apiURL
	c.setConfig(cfg)
	db.Save(&c)
	return nil
}

func (c *Connector) updateStatus(db *gorm.DB, status string, description *string) {
	c.Status = common.StringOrNil(status)
	c.Description = description
	result := db.Save(&c)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			c.Errors = append(c.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
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
			if success {
				msg, _ := json.Marshal(map[string]interface{}{
					"connector_id": c.ID,
				})
				natsutil.NatsPublish(natsConnectorProvisioningSubject, msg)
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
	err := c.deprovision()
	if err != nil {
		return false
	}

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
	return len(c.Errors) == 0
}

// connectorAPI returns an instance of the connector's underlying provider.API
func (c *Connector) connectorAPI() (provider.API, error) {
	if c.Type == nil {
		return nil, fmt.Errorf("No provider resolved for connector: %s", c.ID)
	}

	db := dbconf.DatabaseConnection()
	var apiClient provider.API

	switch *c.Type {
	case provider.IPFSConnectorProvider:
		apiClient = provider.InitIPFSProvider(
			c.ID,
			&c.NetworkID,
			c.ApplicationID,
			db.Model(c),
			c.mergedConfig(),
		)
	default:
		return nil, fmt.Errorf("No provider resolved for connector: %s", c.ID)
	}

	return apiClient, nil
}

// reachable returns true if any of the associated loadbalancers are reachable on the configured port;
// if no loadbalancers are configured, the connector is considered reachable if any configured nodes are
// reachable. reachability for the connector should not be interpreted as high availability. this is useful
// for determining if a connector has transitioned from provisioning -> available...
func (c *Connector) reachable() bool {
	apiClient, err := c.connectorAPI()
	if err != nil {
		common.Log.Warningf("Failed to test connector reachability; unable to resolve connector API for %s connector: %s; %s", *c.Type, c.ID, err.Error())
	}
	return apiClient.Reachable()
}
