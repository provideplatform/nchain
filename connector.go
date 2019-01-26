package main

import (
	"encoding/json"
	"strings"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
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
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name          *string          `sql:"not null" json:"name"`
	Type          *string          `sql:"not null" json:"type"`
	Config        *json.RawMessage `sql:"type:json" json:"config"`
	AccessedAt    *time.Time       `json:"accessed_at"`
}

// ParseConfig - parse the original JSON params used for Connector creation
func (c *Connector) ParseConfig() map[string]interface{} {
	cfg := map[string]interface{}{}
	if c.Config != nil {
		err := json.Unmarshal(*c.Config, &cfg)
		if err != nil {
			Log.Warningf("Failed to unmarshal connector params; %s", err.Error())
			return nil
		}
	}
	return cfg
}

// Create and persist a new Connector
func (c *Connector) Create() bool {
	db := dbconf.DatabaseConnection()

	if !c.Validate() {
		return false
	}

	if db.NewRecord(c) {
		result := db.Create(&c)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				c.Errors = append(c.Errors, &provide.Error{
					Message: StringOrNil(err.Error()),
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
			Message: StringOrNil("Unable to deploy connector using unspecified network"),
		})
	}
	if c.Type == nil || strings.ToLower(*c.Type) != "ipfs" {
		c.Errors = append(c.Errors, &provide.Error{
			Message: StringOrNil("Unable to define connector of invalid type"),
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
				Message: StringOrNil(err.Error()),
			})
		}
	}
	return len(c.Errors) == 0
}
