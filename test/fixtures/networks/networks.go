package networkfixtures

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/provideapp/nchain/test/fixtures"
	"github.com/provideservices/provide-go"
)

func ptrTo(s string) *string {
	return &s
}

func ptrToInt(i int) *int {
	return &i
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
		// "Name", // name should not match because it doesn't affects logic
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
								// fmt.Printf("f1: %v\n", string(f1.(json.RawMessage)))
								// fmt.Printf("f2: %v\n", string(f2.(json.RawMessage)))
								// fmt.Printf("equal: %t\n", reflect.DeepEqual(f1.(json.RawMessage), f2.(json.RawMessage)))

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

// NetworkFields is a copy of nchain Network struct
type NetworkFields struct {
	provide.Model
	// ApplicationID *uuid.UUID
	// UserID        *uuid.UUID
	Name         *string
	Description  *string
	IsProduction *bool
	Cloneable    *bool
	Enabled      *bool
	ChainID      *string
	// SidechainID   *uuid.UUID
	// NetworkID     *uuid.UUID
	Config *json.RawMessage
	// Stats         *provide.NetworkStatus
}

// NetworkFixture combines network fieldset and its name
type NetworkFixture struct {
	Fields *NetworkFields
	Name   *string
}

// Networks function returns all network fixtures: fields and their names
// the network filters have following naming rules:
// - eth
// - Clonable/NonClonable - filter Cloneable field state
// - Disabled/Enabled     - filter Enabled field state
// - EmptyConfig          - filter Config has value {}
// - NilConfig            - filter Config has value nil
// - ConfigN              - equal to ConfigNNNNNNNNNNNNN with same N: Config0 = Config00000000000000
// - ConfigNNNNNNNNNNNNN  - filter Config has value with json fields of certain values.
//    Each N reflects 1 field in the next order:
//    - "block_explorer_url"
//    - "chain"
//    - "chainspec_abi_url"
//    - "chainspec_url"
//    - "cloneable_cfg"
//    - "engine_id"
//    - "is_ethereum_network"
//    - "is_load_balanced"
//    - "json_rpc_url"
//    - "native_currency"
//    - "network_id"
//    - "protocol_id"
//    - "websocket_url
//    N follows the next rules:
//    - equals 0 if the correspoding value is nil.
//    - equals 1 if the corresponding value is empty or 0.
//    - equals 2 if the corresponding value is default value for the field.
//    - equals 3 for special case of chainspec data.
func Networks() []*fixtures.FixtureMatcher {
	//nf := defaultNetwork()
	//fmt.Println("%v", *nf)
	return []*fixtures.FixtureMatcher{
		// ethNonCloneableEnabledFullConfigNetwork(), // default
		// ethNonCloneableEnabledChainspecNetwork(), // TODO: support chainspec text
		ethClonableDisabledEmptyConfigNetwork(),
		ethClonableDisabledConfigNetwork0(),
		ethClonableDisabledConfigNetwork1(),
		// ethClonableDisabledConfigNetwork0222222202220(),
		// ethClonableDisabledConfigNetwork1222222202220(),
		// ethClonableDisabledConfigNetwork2022222202220(),
		// ethClonableDisabledConfigNetwork2122222202220(),
		// ethClonableDisabledConfigNetwork2220022202220(),
		// ethClonableDisabledConfigNetwork2220122202220(),
		// ethClonableDisabledConfigNetwork2220222202220(),
		// ethClonableDisabledConfigNetwork2221022202220(),
		// ethClonableDisabledConfigNetwork2221122202220(),
		// ethClonableDisabledConfigNetwork2221222202220(),
		// ethClonableDisabledConfigNetwork2222022202220(),
		// ethClonableDisabledConfigNetwork2222122202220(),
		// ethClonableDisabledConfigNetwork2222202202220(),
		// ethClonableDisabledConfigNetwork2222212202220(),
		// ethClonableDisabledConfigNetwork2222220202220(),
		// ethClonableDisabledConfigNetwork2222221202220(),
		// ethClonableDisabledConfigNetwork2222222002220(),
		// ethClonableDisabledConfigNetwork2222222102220(),
		// ethClonableDisabledConfigNetwork2222222200220(),
		// ethClonableDisabledConfigNetwork2222222201220(),
		// ethClonableDisabledConfigNetwork2222222202020(),
		// ethClonableDisabledConfigNetwork2222222202120(),
		// ethClonableDisabledConfigNetwork2222222202200(),
		// ethClonableDisabledConfigNetwork2222222202210(),
		// ethClonableDisabledConfigNetwork2222222202220(),
		// ethClonableDisabledConfigNetwork2233022202220(),
		// ethClonableDisabledConfigNetwork2233122202220(),
		// ethClonableDisabledConfigNetwork2233222202220(),

		ethClonableDisabledConfigNetwork2233222222220(),
		ethClonableEnabledEmptyConfigNetwork(),
		ethClonableEnabledConfigNetwork0(),
		ethClonableEnabledConfigNetwork1(),
		ethClonableEnabledConfigNetwork2233222222220(),

		// ethClonableEnabledEmptyConfigNetwork(),
		// ethClonableEnabledNilConfigNetwork(),
		// ethClonableEnabledConfigNetwork00(),
		// ethClonableEnabledConfigNetwork01(),
		// ethClonableEnabledConfigNetwork02(),
		// ethClonableEnabledConfigNetwork10(),
		// ethClonableEnabledConfigNetwork11(),
		// ethClonableEnabledConfigNetwork12(),
		// ethClonableEnabledConfigNetwork20(),
		// ethClonableEnabledConfigNetwork21(),
		// ethClonableEnabledConfigNetwork22(),
	}
}

func (nf *NetworkFields) clone() (nf2 *NetworkFields) {
	config := nf.Config
	if config == nil {
		nf2 = &NetworkFields{
			Model: nf.Model,
			// ApplicationID: nf.ApplicationID,
			// UserID:        nf.UserID,
			Name:         ptrTo(*nf.Name),
			Description:  nf.Description,
			IsProduction: nf.IsProduction,
			Cloneable:    nf.Cloneable,
			Enabled:      nf.Enabled,
			ChainID:      nf.ChainID,
			// SidechainID:   nf.SidechainID,
			// NetworkID:     nf.NetworkID,
			Config: nil,
			// Stats:         nf.Stats,
		}
	} else {
		nf2 = &NetworkFields{
			Model: nf.Model,
			// ApplicationID: nf.ApplicationID,
			// UserID:        nf.UserID,
			Name:         ptrTo(*nf.Name),
			Description:  nf.Description,
			IsProduction: nf.IsProduction,
			Cloneable:    nf.Cloneable,
			Enabled:      nf.Enabled,
			ChainID:      nf.ChainID,
			// SidechainID:   nf.SidechainID,
			// NetworkID:     nf.NetworkID,
			Config: ptrToJRW(*nf.Config),
			// Stats:         nf.Stats,
		}
	}

	return
}

func (nf *NetworkFields) genName(prefix *string, nfg *NetworkFixtureGenerator) (name *string) {
	production := "Production"
	if nf.IsProduction == nil || !*(nf.IsProduction) {
		production = "Non" + production + " "
	}
	clonable := "Cloneable "
	if nf.Cloneable == nil || !*(nf.Cloneable) {
		clonable = "Non" + clonable
	}

	enabled := "Enabled "
	if nf.Enabled == nil || !*(nf.Enabled) {
		enabled = "Disabled "
	}

	cfg := "Config "
	if nf.Config == nil {
		cfg = "Nil " + cfg
	} else {
		config := map[string]interface{}{}
		if nf.Config != nil {
			json.Unmarshal(*nf.Config, &config)
		}
		if len(config) == 0 {
			cfg = "Empty " + cfg
		} else {

			// if nfg.FieldRegistered(ptrTo("Config/cloneable_cfg")) {
			if config["cloneable_cfg"] == nil {
				cfg += "nil cloneable_cfg "
			} else {
				if reflect.DeepEqual(config["cloneable_cfg"], map[string]interface{}{}) {
					cfg += "empty cloneable_cfg "
				} else {
					// cloneable_cfg := config["cloneable_cfg"].(map[string]interface{})
					// if _, secOk := cloneable_cfg["security"]; secOk {
					// 	cfg += "w cloneable_cfg w security "
					// } else {
					cfg += "w cloneable_cfg "
					// }
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/chainspec_url")) {
			if config["chainspec"] == nil && config["chainspec_url"] == nil {
				cfg += "nil chainspec "
			} else {
				if config["chainspec"] != nil {
					cfg += "w chainspec "
				}
				if config["chainspec_url"] != nil {
					if config["chainspec_url"] == "" {
						cfg += "empty chainspec_url "
					} else {
						cfg += "w chainspec_url "
					}
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/block_explorer_url")) {
			if config["block_explorer_url"] == nil {
				cfg += "nil block_explorer_url "
			} else {
				if reflect.DeepEqual(config["block_explorer_url"], map[string]interface{}{}) {
					cfg += "empty block_explorer_url "
				} else {
					cfg += "w block_explorer_url "
				}
			}
			// }

			// if config["block_explorer_url"] == nil {
			// 	cfg += "nil block_explorer_url "
			// } else {
			// 	if reflect.DeepEqual(config["block_explorer_url"], map[string]interface{}{}) {
			// 		cfg += "empty block_explorer_url "
			// 	} else {
			// 		cfg += "w block_explorer_url "
			// 	}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/chain")) {
			if config["chain"] == nil {
				cfg += "nil chain "
			} else {
				if reflect.DeepEqual(config["chain"], map[string]interface{}{}) {
					cfg += "empty chain "
				} else {
					cfg += "w chain "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/engine_id")) {
			if config["engine_id"] == nil {
				cfg += "nil engine_id "
			} else {
				if reflect.DeepEqual(config["engine_id"], map[string]interface{}{}) {
					cfg += "empty engine_id "
				} else {
					cfg += "w engine_id "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/is_ethereum_network")) {
			if config["is_ethereum_network"] == nil {
				cfg += "nil eth "
			} else {
				fmt.Printf("is eth: %v\n", config["is_ethereum_network"].(bool))
				if config["is_ethereum_network"].(bool) {
					cfg += "eth:t "
				} else {
					cfg += "eth:f "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/is_load_balanced")) {
			if config["is_load_balanced"] == nil {
				cfg += "nil lb "
			} else {
				if config["is_load_balanced"].(bool) {
					cfg += "lb:t "
				} else {
					cfg += "lb:f "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/native_currency")) {
			if config["native_currency"] == nil {
				cfg += "nil currency "
			} else {
				if nc := config["native_currency"]; nc == "" {
					cfg += "empty currency "
				} else {
					cfg += nc.(string) + " currency "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/network_id")) {
			if config["network_id"] == nil {
				cfg += "nil network_id "
			} else {
				if nc := config["network_id"]; nc == "" {
					cfg += "empty network_id "
				} else {
					fmt.Printf("network id: %v\n", nc.(float64))
					cfg += strconv.Itoa(int(nc.(float64))) + " network_id "
				}
			}
			// }

			// if nfg.FieldRegistered(ptrTo("Config/protocol_id")) {
			if config["protocol_id"] == nil {
				cfg += "nil protocol_id "
			} else {
				if nc := config["protocol_id"]; nc == "" {
					cfg += "empty protocol_id "
				} else {
					cfg += nc.(string) + " protocol_id "
				}
			}
			// }

			// name := "ETH NonProduction Cloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id "

		}
	}

	name = ptrTo((*prefix) + " " + production + clonable + enabled + cfg)
	return
}

//
