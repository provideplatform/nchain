package networkfixtures

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/provideapp/goldmine/test/matchers"
)

func NewNetworkFixtureFieldValues(name *string, values []interface{}) *NetworkFixtureFieldValues {
	return &NetworkFixtureFieldValues{
		fieldName: name,
		values:    values,
	}
}

func NetworkFixtureFieldValuesVariety() (NetworkFixtureFieldValuesArray []*NetworkFixtureFieldValues) {

	ptrTrue := ptrToBool(true)
	ptrFalse := ptrToBool(false)
	// emptyConfig := map[string]interface{}{}
	// nilValuesConfig := nilValuesConfig()
	// emptyValuesConfig := emptyValuesConfig()

	chainspecConfig := defaultConfig()
	delete(chainspecConfig, "chainspec_url")
	delete(chainspecConfig, "chainspec_abi_url")
	ch, chAbi := getChainspec()
	chainspecConfig["chainspec"] = ch
	chainspecConfig["chainspec_abi"] = chAbi
	chainspecConfig["json_rpc_url"] = "url"
	chainspecConfig["is_bcoin_network"] = true
	chainspecConfig["is_handshake_network"] = true
	chainspecConfig["is_lcoin_network"] = true
	chainspecConfig["is_quorum_network"] = true

	NetworkFixtureFieldValuesArray = []*NetworkFixtureFieldValues{
		&NetworkFixtureFieldValues{
			fieldName: ptrTo("Name/Prefix"),
			values: []interface{}{
				// ptrTo("   "),
				ptrTo("ETH"),
				// ptrTo("BTC")
			},
		},
		&NetworkFixtureFieldValues{
			fieldName: ptrTo("IsProduction"),
			values: []interface{}{
				// ptrTrue,
				ptrFalse},
		},
		&NetworkFixtureFieldValues{
			fieldName: ptrTo("Cloneable"),
			values: []interface{}{
				ptrTrue,
				// ptrFalse,
			},
		},
		&NetworkFixtureFieldValues{
			fieldName: ptrTo("Enabled"),
			values: []interface{}{
				ptrTrue,
				ptrFalse,
			},
		},
		&NetworkFixtureFieldValues{
			fieldName: ptrTo("Config"),
			values: []interface{}{
				nil,
				// emptyConfig,
				// emptyValuesConfig,
				// nilValuesConfig,
				// chainspecConfig,
			},
		},
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/Skip"), // should be skipped if Config == nil
		// 	values: []interface{}{
		// 		// ptrTrue,
		// 		ptrFalse,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/cloneable_cfg"),
		// 	values: []interface{}{
		// 		nil,
		// 		map[string]interface{}{},
		// 		map[string]interface{}{
		// 			"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/chainspec_url"),
		// 	values: []interface{}{
		// 		// nil,
		// 		// ptrTo(""),
		// 		ptrTo("https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
		// 		// ptrTo("get https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/block_explorer_url"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo(""),
		// 		ptrTo("https://unicorn-explorer.provide.network"),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/chain"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo(""),
		// 		ptrTo("unicorn-v0"),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/engine_id"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo(""),
		// 		ptrTo("authorityRound"),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_ethereum_network"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_bcoin_network"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_handshake_network"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_lcoin_network"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_quorum_network"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/is_load_balanced"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrFalse,
		// 		ptrTrue,
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/native_currency"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo(""),
		// 		ptrTo("PRVD"),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/network_id"),
		// 	values: []interface{}{
		// 		nil,
		// 		// ptrTo(""),
		// 		ptrToInt(0),
		// 		ptrToInt(22),
		// 	},
		// },
		// &NetworkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/protocol_id"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo(""),
		// 		ptrTo("poa"),
		// 	},
		// },
		// 	name := "ETH NonProduction Cloneable Disabled Config w cloneable_cfg w chainspec_url w block_explorer_url w chain w engine_id eth:t lb:t PRVD currency 22 network_id poa protocol_id "

	}

	return
}

// NetworkFixtureGenerator is a thing that generates test fixtures based on set of rules
type NetworkFixtureGenerator struct {
	fieldValuesVariety []*NetworkFixtureFieldValues
	fieldNames         []*string
	fixtures           []*NetworkFields
}

// FieldRegistered returns true if `fieldNames` field has specified string value
func (g *NetworkFixtureGenerator) FieldRegistered(fieldName *string) (b bool) {
	for _, fn := range g.fieldNames {
		// fmt.Printf("fn: %v\n", fn)
		// fmt.Printf("fieldName: %v\n", fieldName)
		if *fn == *fieldName {
			return true
		}
	}
	return false
}

// NewNetworkFixtureGenerator generates NetworkFixtureGenerator with default field-value pairs
func NewNetworkFixtureGenerator(fieldValues []*NetworkFixtureFieldValues) (nfg *NetworkFixtureGenerator) {
	var fieldValuesVariety []*NetworkFixtureFieldValues
	var names []*string

	if fieldValues == nil {
		fieldValuesVariety = NetworkFixtureFieldValuesVariety()
	} else {
		fieldValuesVariety = fieldValues
	}

	names = make([]*string, len(fieldValues))
	for i, nffv := range fieldValues {
		// fmt.Printf("nffv: %v\n", nffv.fieldName)
		names[i] = nffv.fieldName
	}

	fmt.Printf("names: %#v\n", names)
	nfg = &NetworkFixtureGenerator{
		fieldValuesVariety: fieldValuesVariety,
		fieldNames:         names,
	}
	nfg.fixtures = nfg.Generate()
	return
}

// All function returns all fixtures from fixtures field
func (generator *NetworkFixtureGenerator) All() []*NetworkFields {
	return generator.fixtures
}

// AddMatcherCollection to add matcher collection
func (generator *NetworkFixtureGenerator) AddMatcherCollection(mc *matchers.MatcherCollection) bool {

	return true
}

// Select function returns fixtures specified by opts
func (generator *NetworkFixtureGenerator) Select(fieldValues *NetworkFields) (nfs []*NetworkFields, pnfs []*NetworkFields) {
	// fmt.Printf("field values: %v\n", fieldValues)
	for _, f := range generator.fixtures {
		if valEqVal(f, fieldValues) {
			nfs = append(nfs, f)
		}
		if valEqVal(f, fieldValues, "Name") {
			pnfs = append(pnfs, f)
		}
	}
	return
}

// NetworkFixtureFieldValues struct holds field name and array of field values to be iterated during fixture generation.
type NetworkFixtureFieldValues struct {
	fieldName *string
	values    []interface{}
}

// Generate takes default values
func (generator *NetworkFixtureGenerator) Generate() (fields []*NetworkFields) {
	return generator.generate(generator.fieldValuesVariety)
}

func (generator *NetworkFixtureGenerator) generate(fvs []*NetworkFixtureFieldValues) (fields []*NetworkFields) {
	fields = make([]*NetworkFields, 0)
	if len(fvs) == 0 {
		return
	}

	fmt.Printf("Starting fixture generation.\n")
	nf := NetworkFields{}
	config := generator.defaultConfig()
	// fmt.Printf("Initial config: %v\n", config)

	// generate fixtures
	generator.addField(&nf, fvs, 0, &fields, config)

	// generation results
	fmt.Printf("Generated %v fixtures. Starting tests.\n", len(fields))
	for _, f := range fields {
		fmt.Printf("Generated fixture '%v'\n", *f.Name)
		// var config
		c := map[string]interface{}{}
		if f.Config != nil {
			json.Unmarshal(*f.Config, &c)
			fmt.Printf("  config: %v\n", c)
		} else {
			fmt.Printf("  config: %v\n", f.Config)
		}
		// fmt.Printf("  Fields: %v\n", f)
	}
	return
}

// addField function works recursively. First run each field is set to first value of range. With that set, struct is
// cloned, added to list, and then one field is changed. Struct is cloned again, another field is changed and so on until
// the end of each field range.
// The name is generated out of the field values. Thus, the Name/Prefix is used as default value and other fields are added
// as suffixes. The name is set after the last field value is set.
func (generator *NetworkFixtureGenerator) addField(
	nf *NetworkFields,
	fvs []*NetworkFixtureFieldValues,
	fieldIndex int,
	fields *([]*NetworkFields),
	config map[string]interface{}) error {

	fv := fvs[fieldIndex]
	for _, v := range fv.values {
		// for i := 0; i < len(fv.values); i++ {
		// 	v := fv.values[i]

		fmt.Printf("%v                                ", *fv.fieldName)
		if *fv.fieldName == "Name/Prefix" {
			nf.Name = v.(*string)
			fmt.Printf("%v  \n", *nf.Name)
		}
		if *fv.fieldName == "IsProduction" {
			nf.IsProduction = v.(*bool)
			fmt.Printf("%t  \n", *nf.IsProduction)
		}
		if *fv.fieldName == "Cloneable" {
			nf.Cloneable = v.(*bool)
			fmt.Printf("%t  \n", *nf.Cloneable)
		}
		if *fv.fieldName == "Enabled" {
			nf.Enabled = v.(*bool)
			fmt.Printf("%t  \n", *nf.Enabled)
		}
		if *fv.fieldName == "Config" {
			fmt.Printf("%v  \n", v)
			// fmt.Printf("  nf: %v\n", nf)

			// fmt.Printf("\nconfig v: %v\n", v)
			if v == nil {
				// fmt.Printf("config v = nil\n")

				if _, ok := config["nil"]; !ok {
					config = map[string]interface{}{}
					config["nil"] = true // sets config accumulator to nil to prevent other keys being added. but zero map is empty, not nil 8-[]
				}

				if v, ok := config["counter"]; ok {
					config["counter"] = v.(int) + 1
				} else {
					config["counter"] = 0
				}
				fmt.Printf("  nil counter: %v\n", config["counter"])
			} else {
				config = v.(map[string]interface{}) // sets config accumulator to specified value
			}
		}

		nilconfig, ok := config["nil"]
		skipConfigSkip := (ok && nilconfig.(bool))

		fmt.Printf("    config type: %T\n", config)
		fmt.Printf("    config keys:\n")
		for k := range config {
			fmt.Printf("        %v\n", k)
		}
		fmt.Printf("\n")
		fmt.Printf("    skipConfigSkip: %t\n", skipConfigSkip)
		if !skipConfigSkip {
			if *fv.fieldName == "Config/Skip" {
				fmt.Printf("  - config skip value: %v\n", *v.(*bool))
				// fmt.Printf("  config len: %v\n", len(config))
				if *v.(*bool) { // if config set to empty map, that means that nf.Config is expected to be nil
					//nf.Config = marshalConfig(map[string]interface{}{}) // sets Config explicitly.
					//config = map[string]interface{}{}
				} else {
					// if Config isn't skipped we need to assign default value to `config` var

					// if n, ok := config["nil"]; ok && n.(bool) { // old value to be substituted
					config = generator.defaultConfig()
					fmt.Printf("    setting default config\n")
					// }
				}
				// fmt.Printf("nf.Config: %v\n", nf.Config)
				// fmt.Printf("config: %v\n", config)
			}
		}

		skipConfigKeys := skipConfigSkip || len(config) == 0 //(nf.Config != nil && reflect.DeepEqual(*nf.Config, json.RawMessage("{}")))

		fmt.Printf("        skipConfigKeys: %t\n", skipConfigKeys)
		fmt.Printf("            %v %T  value  ", *fv.fieldName, v)
		if skipConfigKeys {
			fmt.Printf("skipped\n")
		} else {
			fmt.Printf("added\n")
		}

		fmt.Printf("      nf.Config before keys: %v\n", nf.Config)
		if nf.Config != nil {
			fmt.Printf("      nf.Config eq {}: %v\n", reflect.DeepEqual(*nf.Config, json.RawMessage("{}")))
			//fmt.Printf("%t\n", reflect.DeepEqual(*nf.Config, json.RawMessage("{}")))
		}
		if !skipConfigKeys { // explicitly set to empty value
			if *fv.fieldName == "Config/cloneable_cfg" {
				config = generator.updConfig(config, ptrTo("cloneable_cfg"), v)
				// nf.Config = generator.updateConfig(nf.Config, ptrTo("cloneable_cfg"), v)
			}
			if *fv.fieldName == "Config/chainspec_url" {

				if v != nil && strings.HasPrefix(*v.(*string), "get ") {
					chainspecJSON, chainspecABIJSON := getChainspec()
					config = generator.updConfig(config, ptrTo("chainspec"), chainspecJSON)
					config = generator.updConfig(config, ptrTo("chainspec_abi"), chainspecABIJSON)
					delete(config, "chainspec_url")
					delete(config, "chainspec_abi_url")
				} else {
					config = generator.updConfig(config, ptrTo("chainspec_url"), v)
					if _, ok := config["chainspec"]; ok {
						delete(config, "chainspec")
						delete(config, "chainspec_abi")
						generator.updConfig(config, ptrTo("chainspec_abi_url"), ptrTo(generator.defaultConfig()["chainspec_abi_url"].(string)))
					}
					// if _, ok := config["chainspec_abi"]; ok {
					// }
				}
				// nf.Config = generator.updateConfig(nf.Config, ptrTo("chainspec_url"), v)
			}
			if *fv.fieldName == "Config/block_explorer_url" {
				config = generator.updConfig(config, ptrTo("block_explorer_url"), v)
			}
			if *fv.fieldName == "Config/chain" {
				config = generator.updConfig(config, ptrTo("chain"), v)

			}
			if *fv.fieldName == "Config/engine_id" {
				config = generator.updConfig(config, ptrTo("engine_id"), v)

			}
			if *fv.fieldName == "Config/is_ethereum_network" {
				config = generator.updConfig(config, ptrTo("is_ethereum_network"), v)

			}
			if *fv.fieldName == "Config/is_load_balanced" {
				config = generator.updConfig(config, ptrTo("is_load_balanced"), v)

			}
			if *fv.fieldName == "Config/native_currency" {
				config = generator.updConfig(config, ptrTo("native_currency"), v)

			}
			if *fv.fieldName == "Config/network_id" {
				config = generator.updConfig(config, ptrTo("network_id"), v)

			}
			if *fv.fieldName == "Config/poa" {
				config = generator.updConfig(config, ptrTo("poa"), v)

			}
			if *fv.fieldName == "Config/protocol_id" {
				config = generator.updConfig(config, ptrTo("protocol_id"), v)

			}
		}

		// fmt.Printf("fv field: %v\n", *fv.fieldName)
		// fmt.Printf("nf: %v\n", nf)
		// fmt.Printf("nf type: %T\n", nf)
		if fieldIndex == len(fvs)-1 { // last index is 1 less
			initialName := *nf.Name

			fmt.Printf("      nf.Config after keys: %v\n", nf.Config)

			configNil := false
			if n, ok := config["nil"]; ok && n.(bool) {
				configNil = true
			}

			if configNil {
				nf.Config = nil // otherwise it gets populated with "{}"
			} else {
				nf.Config = marshalConfig(config)
			}
			nf.Name = nf.genName(nf.Name, generator)
			nfClone := nf.clone()

			alreadyAdded := false
			if nfClone.Config == nil || len(config) == 0 {
				if n, ok := config["nil"]; ok && n.(bool) {
					if v, ok := config["counter"]; ok {
						if v.(int) > 0 {
							alreadyAdded = true
						}
					}
				}
				// fmt.Printf("alreadyAdded: %t\n", alreadyAdded)
				// if !alreadyAdded {
				// 	alreadyAdded = generator.findEqualConfig(*fields, config)
				// }
				// fmt.Printf("alreadyAdded: %t\n", alreadyAdded)
				if !alreadyAdded {
					alreadyAdded = generator.fieldsEqual(*fields, nfClone)
				}
				// fmt.Printf("alreadyAdded: %t\n", alreadyAdded)

			}
			if !alreadyAdded {
				fmt.Printf("FIXTURE ADDED ('%v') \n\n", *nfClone.Name)
				*fields = append(*fields, nfClone)
			}

			// initialize compound values
			nf.Name = ptrTo(initialName)
			nf.Config = nil
			// config = generator.defaultConfig()
			// fmt.Printf("\nfield name: %v\n", *fv.fieldName)
			// fmt.Printf("setting config to default value: %v\n", config)
			// retVal = config
			// use = true
		} else {
			// fmt.Printf("calling addField with config: %v\n", config)
			generator.addField(nf, fvs, fieldIndex+1, fields, config)
		}
	}

	return nil
}

func (generator *NetworkFixtureGenerator) findEqualConfig(fields []*NetworkFields, config map[string]interface{}) bool {
	for _, f := range fields {
		c := map[string]interface{}{}
		if f.Config != nil {
			json.Unmarshal(*f.Config, &c)
			eq := reflect.DeepEqual(c, config)
			// fmt.Printf("      unmarshaled config: %v\n", c)
			// fmt.Printf("      tested config: %v", config)
			// fmt.Printf("      eq: %t\n", eq)
			if eq {
				return true
			}
		}
	}
	return false
}

func (generator *NetworkFixtureGenerator) fieldsEqual(fields []*NetworkFields, nf *NetworkFields) bool {
	for _, f := range fields {
		if reflect.DeepEqual(f, nf) {
			return true
		}
	}
	return false
}

func (generator *NetworkFixtureGenerator) updConfig(c map[string]interface{}, key *string, value interface{}) map[string]interface{} {
	// fmt.Printf("\nc: %v\n", c)
	// fmt.Printf("key: %v\n", *key)
	// fmt.Printf("value: %v\n", value)
	if len(c) == 0 { // zero map is empty, not nil 8-[]
		return c
	}

	if value == nil {
		c[*key] = nil
	} else {
		ref := reflect.ValueOf(value)
		// fmt.Printf("key: %v\n", *key)
		if *key != "cloneable_cfg" {
			// fmt.Printf("ref: %v\n", ref)
			// fmt.Printf("ref kind: %v\n", ref.Kind())
			// fmt.Printf("ref type: %v\n", ref.Type())
			// fmt.Printf("ref type eq typeof true: %t\n", ref.Type() == reflect.TypeOf(ptrToBool(true)))
			// fmt.Printf("ref elem: %v\n", ref.Elem())
		}

		if ref.Kind() == reflect.Map {
			c[*key] = value
		} else if ref.Type() == reflect.TypeOf(ptrToBool(true)) {
			// fmt.Printf("ref bool: %v\n", *ref.Interface().(*bool))
			c[*key] = *ref.Interface().(*bool)
		} else if ref.Type() == reflect.TypeOf(ptrToInt(1)) {
			// fmt.Printf("ref int: %v\n", *ref.Interface().(*int))
			c[*key] = *ref.Interface().(*int)
		} else {
			// fmt.Printf("elem type: %T\n", ref.Elem().Type()) // *reflect.rtype
			c[*key] = ref.Elem().String()
		}
	}
	// fmt.Printf("new c: %v\n", c)

	return c
}

func (generator *NetworkFixtureGenerator) updateConfig(config *json.RawMessage, key *string, value interface{}) *json.RawMessage {
	c := map[string]interface{}{}
	if config != nil {
		json.Unmarshal(*config, &c)
	} else {
		json.Unmarshal(*(generator.defaultConfigMarshalled()), &c)
	}

	// fmt.Printf("key: %v\n", *key)
	// fmt.Printf("value: %v\n", value)
	// fmt.Printf("reflection: %v\n", reflect.ValueOf(value))
	// fmt.Printf("kind: %v\n", reflect.ValueOf(value).Kind())
	if value == nil {
		c[*key] = nil
	} else {
		ref := reflect.ValueOf(value)
		if ref.Kind() == reflect.Map {
			c[*key] = value
		} else {
			// fmt.Printf("elem type: %T\n", ref.Elem().Type()) // *reflect.rtype
			c[*key] = ref.Elem().String()
		}
	}
	return marshalConfig(c)
}

func (generator *NetworkFixtureGenerator) defaultConfig() map[string]interface{} {
	return defaultConfig()
}

func defaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"block_explorer_url": "https://unicorn-explorer.provide.network", // required
		"chain":              "unicorn-v0",                               // required
		"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
		"chainspec_url":      "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json", // required If ethereum network
		"cloneable_cfg": map[string]interface{}{
			"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security
		"engine_id":           "authorityRound", // required
		"is_ethereum_network": true,             // required for ETH
		"is_load_balanced":    true,             // implies network load balancer count > 0
		"json_rpc_url":        nil,
		"native_currency":     "PRVD", // required
		"network_id":          22,     // required
		"protocol_id":         "poa",  // required
		"websocket_url":       nil}
}

func nilValuesConfig() map[string]interface{} {
	return map[string]interface{}{
		"block_explorer_url":  nil, // required
		"chain":               nil, // required
		"chainspec_abi_url":   nil,
		"chainspec_url":       nil, // required If ethereum network
		"cloneable_cfg":       nil, // If cloneable CFG then security
		"engine_id":           nil, // required
		"is_ethereum_network": nil, // required for ETH
		"is_load_balanced":    nil, // implies network load balancer count > 0
		"json_rpc_url":        nil,
		"native_currency":     nil, // required
		"network_id":          nil, // required
		"protocol_id":         nil, // required
		"websocket_url":       nil}
}

func emptyValuesConfig() map[string]interface{} {
	return map[string]interface{}{
		"block_explorer_url":  "", // required
		"chain":               "", // required
		"chainspec_abi_url":   "",
		"chainspec_url":       "",                       // required If ethereum network
		"cloneable_cfg":       map[string]interface{}{}, // If cloneable CFG then security
		"engine_id":           "",                       // required
		"is_ethereum_network": false,                    // required for ETH
		"is_load_balanced":    false,                    // implies network load balancer count > 0
		"json_rpc_url":        "",
		"native_currency":     "", // required
		"network_id":          0,  // required
		"protocol_id":         "", // required
		"websocket_url":       ""}
}

func (generator *NetworkFixtureGenerator) defaultConfigMarshalled() *json.RawMessage {

	c := generator.defaultConfig()

	return marshalConfig(c)
}

func getChainspec() (chainspecJSON map[string]interface{}, chainspecABIJSON map[string]interface{}) {
	ethChainspecFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn/spec.json"
	ethChainspecAbiFileurl := "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json"
	response, err := http.Get(ethChainspecFileurl)
	//chainspec_text := ""
	// chainspec_abi_text := ""
	chainspecJSON = map[string]interface{}{}
	chainspecABIJSON = map[string]interface{}{}

	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		// fmt.Printf("%s\n", string(contents))
		//chainspec_text = string(contents)
		json.Unmarshal(contents, &chainspecJSON)
		// common.Log.Debugf("error parsing chainspec: %v", errJSON)

	}

	responseAbi, err := http.Get(ethChainspecAbiFileurl)

	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		defer responseAbi.Body.Close()
		contents, err := ioutil.ReadAll(responseAbi.Body)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		// fmt.Printf("%s\n", string(contents))
		// chainspec_abi_text = string(contents)
		json.Unmarshal(contents, &chainspecABIJSON)
		// common.Log.Debugf("error parsing chainspec: %v", errJSON)
	}

	return
}
