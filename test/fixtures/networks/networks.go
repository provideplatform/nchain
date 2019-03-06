package networkfixtures

import (
	"encoding/json"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideservices/provide-go"
)

func ptrTo(s string) *string {
	return &s
}

func ptrToBool(b bool) *bool {
	return &b
}

func marshalConfig(opts map[string]interface{}) *json.RawMessage {
	cfgJSON, _ := json.Marshal(opts)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}

// NetworkFields is a copy of goldmine Network struct
type NetworkFields struct {
	Model         provide.Model
	ApplicationID *uuid.UUID
	UserID        *uuid.UUID
	Name          *string
	Description   *string
	IsProduction  *bool
	Cloneable     *bool
	Enabled       *bool
	ChainID       *string
	SidechainID   *uuid.UUID
	NetworkID     *uuid.UUID
	Config        *json.RawMessage
	Stats         *provide.NetworkStatus
}

// NetworkFixture combines network fieldset and its name
type NetworkFixture struct {
	Fields *NetworkFields
	Name   *string
}

// Networks function returns all network fixtures: fields and their names
func Networks() []*fixtures.FixtureMatcher {
	//nf := defaultNetwork()
	//fmt.Println("%v", *nf)
	return []*fixtures.FixtureMatcher{
		ethNonCloneableEnabledFullConfigNetwork(), // default
		ethNonProdClonableEnabledNilConfigNetwork(),
		ethNonProdClonableEnabledFullConfigNetwork(),
	}
}

func (nf *NetworkFields) String() string {
	str := ""

	if *nf.IsProduction {
		str += " production: true"
	}
	if *nf.Cloneable {
		str += " cloneable: true"
	}
	if *nf.Enabled {
		str += " enabled: true"
	}
	name := *nf.Name

	return "network: name=" + name + str
}
