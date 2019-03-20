package networkfixtures

import (
	"encoding/json"
	"fmt"
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

func ptrToJRW(jrw json.RawMessage) *json.RawMessage {
	return &jrw
}

func fieldFromValue(v reflect.Value, fieldName string) (r interface{}) {
	e := v.FieldByName(fieldName).Elem()
	if e.IsValid() {
		r = e.Interface()
	} else {
		r = nil
	}
	return
}

func valEqVal(p1 *NetworkFields, p2 *NetworkFields, omitFields ...string) (b bool) {
	if p1 == nil && p2 == nil {
		return true
	}

	defFieldsToCheck := []string{
		"Name", // name should match because it affects logic
		"IsProduction",
		"Cloneable",
		"Enabled",
		"Config",
	}

	fieldsToCheck := []string{}

	var include bool
	for _, f := range defFieldsToCheck {
		include = true
		for _, of := range omitFields {
			if of == f {
				include = false
			}
		}
		if include {
			fieldsToCheck = append(fieldsToCheck, f)
		}
	}
	if p1 != nil && p2 != nil {
		// fmt.Printf("p1 values: %v\n", p1)
		// fmt.Printf("p2 values: %v\n", p2)

		b = true
		for _, ftc := range fieldsToCheck {
			v1 := reflect.ValueOf(p1).Elem()
			v2 := reflect.ValueOf(p2).Elem()
			// fmt.Printf("v1 value: %v\n", v1)
			// fmt.Printf("v2 value: %v\n", v2)
			// fmt.Printf("v1 type: %T\n", v1)
			// fmt.Printf("v2 type: %T\n", v2)

			if !v1.IsValid() && !v2.IsValid() {
				b = b && true
			} else {
				if !v1.IsValid() || !v2.IsValid() {
					b = b && false
				} else {
					f1 := fieldFromValue(v1, ftc)
					f2 := fieldFromValue(v2, ftc)
					// fmt.Printf("f1: %v\n", f1)
					// fmt.Printf("f2: %v\n", f2)
					// fmt.Printf("p1 field %v value: %v\n", ftc, v1)
					// fmt.Printf("p2 field %v value: %v\n", ftc, v2)

					if f1 == nil && f2 == nil {
						b = b && true
					} else {
						if f1 == nil || f2 == nil {
							b = b && false
						} else {
							switch ftc {
							case "Config":
								b = b && reflect.DeepEqual(f1.(json.RawMessage), f2.(json.RawMessage))
							case "Name":
								b = b && reflect.DeepEqual(f1.(string), f2.(string))
							default: // booleans
								b = b && reflect.DeepEqual(f1.(bool), f2.(bool))
							}
							// fmt.Printf("val12 result: %t\n", (v1 == v2))
							// fmt.Printf("temp result: %t\n", b)
							//b = reflect.DeepEqual(v1, v2)
						}
					}
				}
			}
		}
		// fmt.Printf("result: %t\n", b)
		return
		//return reflect.DeepEqual(p1, p2)
	}
	return false
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
		// ethNonCloneableEnabledFullConfigNetwork(), // default, fixed
		// ethNonCloneableEnabledChainspecNetwork(), // TODO: support chainspec text
		// ethClonableDisabledEmptyConfigNetwork(), // TODO: support Config: {}
		// ethClonableDisabledNilConfigNetwork(), // TODO: support Config: nil
		ethClonableDisabledConfigNetwork(),  // fixed
		ethClonableDisabledConfigNetwork1(), // fixed
		// ethClonableDisabledConfigNetwork2(),  // FIXME
		// ethClonableDisabledConfigNetwork3(), // FIXME
		// ethClonableEnabledEmptyConfigNetwork(), // TODO: support Config: {}
		// ethClonableEnabledNilConfigNetwork(), // TODO: support Config: nil
		ethClonableEnabledFullConfigNetwork(), // fixed
		ethClonableEnabledConfigNetwork1(),    // fixed
		// ethClonableEnabledConfigNetwork2(), // FIXME
		ethClonableEnabledConfigNetwork3(), // fixed
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
		Config:        ptrToJRW(*nf.Config),
		Stats:         nf.Stats,
	}
	return nf2
}

func (nf *NetworkFields) genName(prefix *string) (name *string) {
	production := "Production"
	if !*(nf.IsProduction) {
		production = "Non" + production + " "
	}
	clonable := "Cloneable "
	if !*(nf.Cloneable) {
		clonable = "Non" + clonable
	}

	enabled := "Enabled "
	if !*(nf.Enabled) {
		enabled = "Disabled "
	}

	cfg := "Config "
	config := map[string]interface{}{}
	if nf.Config == nil {
		cfg = "Nil " + cfg
	} else {
		if nf.Config != nil {
			json.Unmarshal(*nf.Config, &config)
		}
		if config["cloneable_cfg"] == nil {
			cfg += "nil cloneable_cfg "
		} else {
			if reflect.DeepEqual(config["cloneable_cfg"], map[string]interface{}{}) {
				cfg += "empty cloneable_cfg "
			} else {
				cfg += "w cloneable_cfg "
			}
		}
		if config["chainspec"] == nil && config["chainspec_url"] == nil {
			cfg += "nil chainspec "
		} else {
			if config["chainspec"] != nil {
				cfg += "w chainspec "
			}
			if config["chainspec_url"] != nil {
				cfg += "w chainspec_url "
			}
		}
	}

	fmt.Printf("prefix: %v\n", *prefix)
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
