package bridge

import (
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go/api"
)

// Bridge instances are still in the process of being defined.
type Bridge struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
}
