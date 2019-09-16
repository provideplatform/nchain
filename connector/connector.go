package connector

import (
	"encoding/json"
	"strings"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Connector{})
	db.Model(&Connector{}).AddIndex("idx_connectors_application_id", "application_id")
	db.Model(&Connector{}).AddIndex("idx_connectors_network_id", "network_id")
	db.Model(&Connector{}).AddIndex("idx_connectors_type", "type")
	db.Model(&Connector{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// Connector instances represent a logical connection to IPFS or other decentralized filesystem;
// in the future it may represent a logical connection to services of other types
type Connector struct {
	provide.Model
	ApplicationID   *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID       uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name            *string          `sql:"not null" json:"name"`
	Type            *string          `sql:"not null" json:"type"`
	Config          *json.RawMessage `sql:"type:json" json:"config"`
	EncryptedConfig *string          `sql:"type:bytea" json:"-"`
	AccessedAt      *time.Time       `json:"accessed_at"`
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
			common.Log.Warningf("Failed to decrypt encrypted load balancer config; %s", err.Error())
			return decryptedParams, err
		}

		err = json.Unmarshal(encryptedConfigJSON, &decryptedParams)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal decrypted load balancer config; %s", err.Error())
			return decryptedParams, err
		}
	}
	return decryptedParams, nil
}

func (c *Connector) encryptConfig() bool {
	if c.EncryptedConfig != nil {
		encryptedConfig, err := pgputil.PGPPubEncrypt([]byte(*c.EncryptedConfig))
		if err != nil {
			common.Log.Warningf("Failed to encrypt load balancer config; %s", err.Error())
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
			return rowsAffected > 0
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
	return len(c.Errors) == 0
}
