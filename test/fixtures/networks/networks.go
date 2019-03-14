package networkfixtures

import (
	"encoding/json"
	"reflect"

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

func defaultMatcherOptions() map[string]interface{} {
	return map[string]interface{}{
		"channelPolling": false,
		"natsPolling":    false,
	}
}

func defaultNATSMatcherOptions(chNamePtr *string) map[string]interface{} {
	return map[string]interface{}{
		"channelPolling": false,
		"natsPolling":    true,
		"natsChannels":   []*string{chNamePtr},
	}
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
		ethNonCloneableEnabledChainspecNetwork(),
	}
}

func (nf *NetworkFields) clone() *NetworkFields {
	nf2 := &NetworkFields{
		Model:         nf.Model,
		ApplicationID: nf.ApplicationID,
		UserID:        nf.UserID,
		Name:          ptrTo(*nf.Name),
		Description:   nf.Description,
		IsProduction:  nf.IsProduction,
		Cloneable:     nf.Cloneable,
		Enabled:       nf.Enabled,
		ChainID:       nf.ChainID,
		SidechainID:   nf.SidechainID,
		NetworkID:     nf.NetworkID,
		Config:        nf.Config,
		Stats:         nf.Stats,
	}
	return nf2
}

func (nf *NetworkFields) genName(prefix *string) (name *string) {
	production := "Production"
	if !*(nf.IsProduction) {
		production = "Non " + production
	}
	clonable := "Cloneable "
	if !*(nf.Cloneable) {
		clonable = "Non " + clonable
	}

	enabled := "Enabled "
	if !*(nf.Enabled) {
		enabled = "Disabled "
	}

	cfg := "cfg "
	config := map[string]interface{}{}
	if nf.Config != nil {
		json.Unmarshal(*nf.Config, &config)
	}
	if config["cloneable_cfg"] == nil {
		cfg = "nil " + cfg
	} else {
		if reflect.DeepEqual(config["cloneable_cfg"], map[string]interface{}{}) {
			cfg = "empty " + cfg
		}
	}

	name = ptrTo((*prefix) + " " + production + clonable + enabled + cfg)
	return
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
