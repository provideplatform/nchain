package networkfixtures

import (
	"encoding/json"
	"reflect"
)

// NetworkFixtureGenerator is a thing that generates test fixtures based on set of rules
type NetworkFixtureGenerator struct {
	fv []*networkFixtureFieldValues
}

// NewNetworkFixtureGenerator generates NetworkFixtureGenerator with default field-value pairs
func NewNetworkFixtureGenerator() *NetworkFixtureGenerator {
	fv := networkFixtureFieldValuesVariety()
	return &NetworkFixtureGenerator{
		fv: fv,
	}
}

type networkFixtureFieldValues struct {
	fieldName *string
	values    []interface{}
}

func networkFixtureFieldValuesVariety() (networkFixtureFieldValuesArray []*networkFixtureFieldValues) {
	networkFixtureFieldValuesArray = []*networkFixtureFieldValues{
		&networkFixtureFieldValues{
			fieldName: ptrTo("Name/Prefix"),
			values: []interface{}{
				ptrTo("   "),
				ptrTo("ETH"),
				ptrTo("BTC")},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("IsProduction"),
			values: []interface{}{
				//ptrToBool(true), // no prod networks yet
				ptrToBool(false)},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Cloneable"),
			values: []interface{}{
				ptrToBool(true),
				ptrToBool(false)},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Enabled"),
			values: []interface{}{
				ptrToBool(true),
				ptrToBool(false)},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config/cloneable_cfg"),
			values: []interface{}{
				nil,
				map[string]interface{}{},
				map[string]interface{}{
					"_security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
			},
		},
		&networkFixtureFieldValues{
			fieldName: ptrTo("Config/chainspec_url"),
			values: []interface{}{
				nil,
				ptrTo("https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.json"),
			},
		},
	}
	return
}

// Generate takes default values
func (generator *NetworkFixtureGenerator) Generate() (fields []*NetworkFields) {
	return generator.generate(generator.fv)
}

func (generator *NetworkFixtureGenerator) generate(fvs []*networkFixtureFieldValues) (fields []*NetworkFields) {
	fields = make([]*NetworkFields, 0)

	nf := NetworkFields{}
	generator.addField(&nf, fvs, 0, &fields)

	return
}

func (generator *NetworkFixtureGenerator) addField(nf *NetworkFields, fvs []*networkFixtureFieldValues, fieldIndex int, fields *([]*NetworkFields)) error {
	fv := fvs[fieldIndex]
	for _, v := range fv.values {
		if *fv.fieldName == "Name/Prefix" {
			nf.Name = v.(*string)
		}
		if *fv.fieldName == "IsProduction" {
			nf.IsProduction = v.(*bool)
		}
		if *fv.fieldName == "Cloneable" {
			nf.Cloneable = v.(*bool)
		}
		if *fv.fieldName == "Enabled" {
			nf.Enabled = v.(*bool)
		}
		if *fv.fieldName == "Config/cloneable_cfg" {
			nf.Config = generator.updateConfig(nf.Config, ptrTo("cloneable_cfg"), v)
		}
		if *fv.fieldName == "Config/chainspec_url" {
			nf.Config = generator.updateConfig(nf.Config, ptrTo("chainspec_url"), v)
		}

		if fieldIndex == len(fvs)-1 { // last index is 1 less
			nf.Name = nf.genName(nf.Name)
			nf2 := nf.clone()
			*fields = append(*fields, nf2)
			nf.Name = ptrTo("")
		} else {
			generator.addField(nf, fvs, fieldIndex+1, fields)
		}
	}

	return nil
}

func (generator *NetworkFixtureGenerator) updateConfig(config *json.RawMessage, key *string, value interface{}) *json.RawMessage {
	c := map[string]interface{}{}
	if config != nil {
		json.Unmarshal(*config, &c)
	} else {
		json.Unmarshal(*(generator.defaultConfig()), &c)
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

func (generator *NetworkFixtureGenerator) defaultConfig() *json.RawMessage {

	c := map[string]interface{}{
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

	return marshalConfig(c)
}
