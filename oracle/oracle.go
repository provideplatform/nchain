package oracle

import (
	"encoding/json"
	"net/url"

	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Oracle{})
	db.Model(&Oracle{}).AddIndex("idx_oracles_application_id", "application_id")
	db.Model(&Oracle{}).AddIndex("idx_oracles_network_id", "network_id")
	db.Model(&Oracle{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Oracle{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
}

// Oracle instances are smart contracts whose terms are fulfilled by writing data from a configured feed onto the blockchain associated with its configured network
type Oracle struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	ContractID    uuid.UUID        `sql:"not null;type:uuid" json:"contract_id"`
	Name          *string          `sql:"not null" json:"name"`
	FeedURL       *url.URL         `sql:"not null" json:"feed_url"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
	AttachmentIds []*uuid.UUID     `sql:"type:uuid[]" json:"attachment_ids"`
}

// Create and persist a new oracle
func (o *Oracle) Create() bool {
	db := DatabaseConnection()

	if !o.Validate() {
		return false
	}

	if db.NewRecord(o) {
		result := db.Create(&o)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				o.Errors = append(o.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(o) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate an oracle for persistence
func (o *Oracle) Validate() bool {
	o.Errors = make([]*provide.Error, 0)
	if o.NetworkID == uuid.Nil {
		o.Errors = append(o.Errors, &provide.Error{
			Message: common.StringOrNil("Unable to deploy oracle using unspecified network"),
		})
	}
	return len(o.Errors) == 0
}

// ParseParams - parse the original JSON params used for oracle creation
func (o *Oracle) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if o.Params != nil {
		err := json.Unmarshal(*o.Params, &params)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal oracle params; %s", err.Error())
			return nil
		}
	}
	return params
}
