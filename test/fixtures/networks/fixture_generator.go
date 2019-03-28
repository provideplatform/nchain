package networkfixtures

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/provideapp/goldmine/test/matchers"
)

func networkFixtureFieldValuesVariety() (networkFixtureFieldValuesArray []*networkFixtureFieldValues) {

	ptrTrue := ptrToBool(true)
	ptrFalse := ptrToBool(false)

	networkFixtureFieldValuesArray = []*networkFixtureFieldValues{
		&networkFixtureFieldValues{
			fieldName: ptrTo("Name/Prefix"),
			values: []interface{}{
				// ptrTo("   "),
				ptrTo("ETH"),
				// ptrTo("BTC")
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("IsProduction"),
			values: []interface{}{
				// ptrTrue,
				ptrFalse},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Cloneable"),
			values: []interface{}{
				ptrTrue,
				// ptrFalse,
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Enabled"),
			values: []interface{}{
				ptrTrue,
				ptrFalse,
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config"),
			values: []interface{}{
				nil,
				map[string]interface{}{},
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config/Skip"), // should be skipped if Config == nil
			values: []interface{}{
				ptrTrue,
				ptrFalse,
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config/cloneable_cfg"),
			values: []interface{}{
				nil,
				map[string]interface{}{},
				// map[string]interface{}{
				// "_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config/chainspec_url"),
			values: []interface{}{
				nil,
				// add empty URL
				ptrTo("https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
			},
		},
		// &networkFixtureFieldValues{
		// 	fieldName: ptrTo("Config/block_explorer_url"),
		// 	values: []interface{}{
		// 		nil,
		// 		ptrTo("https://unicorn-explorer.provide.network"),
		// 	},
		// },
	}
	return
}

// NetworkFixtureGenerator is a thing that generates test fixtures based on set of rules
type NetworkFixtureGenerator struct {
	fieldValuesVariety []*networkFixtureFieldValues
	fixtures           []*NetworkFields
}

// NewNetworkFixtureGenerator generates NetworkFixtureGenerator with default field-value pairs
func NewNetworkFixtureGenerator() (nfg *NetworkFixtureGenerator) {
	fieldValuesVariety := networkFixtureFieldValuesVariety()
	nfg = &NetworkFixtureGenerator{
		fieldValuesVariety: fieldValuesVariety,
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
	fmt.Printf("field values: %v\n", fieldValues)
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

type networkFixtureFieldValues struct {
	fieldName *string
	values    []interface{}
}

// Generate takes default values
func (generator *NetworkFixtureGenerator) Generate() (fields []*NetworkFields) {
	return generator.generate(generator.fieldValuesVariety)
}

func (generator *NetworkFixtureGenerator) generate(fvs []*networkFixtureFieldValues) (fields []*NetworkFields) {
	fields = make([]*NetworkFields, 0)

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
	fvs []*networkFixtureFieldValues,
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
				config = generator.updConfig(config, ptrTo("chainspec_url"), v)
				// nf.Config = generator.updateConfig(nf.Config, ptrTo("chainspec_url"), v)
			}
			// if *fv.fieldName == "Config/block_explorer_url" {
			// 	config = generator.updConfig(config, ptrTo("block_explorer_url"), v)
			// }
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
			nf.Name = nf.genName(nf.Name)
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
		if ref.Kind() == reflect.Map {
			c[*key] = value
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

func (generator *NetworkFixtureGenerator) defaultConfigMarshalled() *json.RawMessage {

	c := generator.defaultConfig()

	return marshalConfig(c)
}
