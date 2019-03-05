package fixtures

import (
	"encoding/json"

	"github.com/kthomas/go-logger"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideservices/provide-go"
)

var log *logger.Logger

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
func Networks() []*NetworkFixture {
	//nf := defaultNetwork()
	//fmt.Println("%v", *nf)
	return []*NetworkFixture{
		&NetworkFixture{
			Fields: defaultNetwork(),
			Name:   ptrTo("ETH NonProd Clonable Enabled Full Config")},
		&NetworkFixture{
			Fields: ethNonProdClonableEnabledNilConfigNetwork(),
			Name:   ptrTo("ETH NonProd Clonable Enabled Nil Config")},
	}
}

func (nf *NetworkFields) String() string {
	str := ""

	if *nf.IsProduction {
		str += " production: true"
	}
	log.Debugf("nil isp")

	if *nf.Cloneable {
		str += " cloneable: true"
	}
	log.Debugf("nil clo")

	if *nf.Enabled {
		str += " enabled: true"
	}
	log.Debugf("nil ena")

	name := *nf.Name

	log.Debugf("nil name")

	return "network: name=" + name + str
}

// DefaultNetwork returns network with following values:
//   production: false
//   cloneable:  false
//   enabled:    true
//   config:
func defaultNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Enabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(true),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config: marshalConfig(map[string]interface{}{
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
			"websocket_url":       nil}),
		Stats: nil}

	return n
}

func ethNonProdClonableEnabledNilConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Enabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(true),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        nil,
		Stats:         nil}
	return
}

func ethNonProdClonableEnabledEmptyConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Enabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(true),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        marshalConfig(map[string]interface{}{}),
		Stats:         nil}
	return
}

func ethNonProdClonableDisabledNilConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Disabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(false),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        nil,
		Stats:         nil}
	return
}

func ethNonProdClonableDisabledEmptyConfigNetwork() (n *NetworkFields) {
	n = &NetworkFields{
		ApplicationID: nil,
		UserID:        nil,
		Name:          ptrTo("Name ETH non-Cloneable Disabled"),
		Description:   ptrTo("Ethereum Network"),
		IsProduction:  ptrToBool(false),
		Cloneable:     ptrToBool(false),
		Enabled:       ptrToBool(false),
		ChainID:       nil,
		SidechainID:   nil,
		NetworkID:     nil,
		Config:        marshalConfig(map[string]interface{}{}),
		Stats:         nil}
	return
}
