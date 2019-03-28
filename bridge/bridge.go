package bridge

import (
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Bridge{})
	db.Model(&Bridge{}).AddIndex("idx_bridges_application_id", "application_id")
	db.Model(&Bridge{}).AddIndex("idx_bridges_network_id", "network_id")
	db.Model(&Bridge{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// Bridge instances are still in the process of being defined.
type Bridge struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
}
