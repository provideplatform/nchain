package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethClonableEnabledConfigNetwork20() (n *fixtures.FixtureMatcher) {
	mc := &matchers.MatcherCollection{}
	optsNATSCreate := defaultNATSMatcherOptions(ptrTo("network.create"))

	mc.AddBehavior("Create", func(opts ...interface{}) types.GomegaMatcher {
		expectedCreateResult := false
		expectedContractCount := 0
		return matchers.NetworkCreateMatcher(expectedCreateResult, expectedContractCount, opts...)
	}, defaultMatcherOptions())
	mc.AddBehavior("Double Create", func(opts ...interface{}) types.GomegaMatcher {
		return BeFalse()
	}, defaultMatcherOptions())
	mc.AddBehavior("Create with NATS", func(opts ...interface{}) types.GomegaMatcher {
		return BeFalse()
	}, optsNATSCreate)
	mc.AddBehavior("Validate", func(opts ...interface{}) types.GomegaMatcher {
		return BeFalse()
	}, defaultMatcherOptions())
	mc.AddBehavior("ParseConfig", func(opts ...interface{}) types.GomegaMatcher {
		return satisfyAllConfigKeys(true)
	}, defaultMatcherOptions())
	mc.AddBehavior("Network type", func(opts ...interface{}) types.GomegaMatcher {
		if opts[0] == "eth" {
			return BeTrue()
		}
		if opts[0] == "btc" {
			return BeFalse()
		}
		if opts[0] == "handshake" {
			return BeFalse()
		}
		if opts[0] == "ltc" {
			return BeFalse()
		}
		if opts[0] == "quorum" {
			return BeFalse()
		}
		return BeNil()
	}, defaultMatcherOptions())
	name := "ETH NonProduction Cloneable Enabled Config w cloneable_cfg nil chainspec "

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				// ApplicationID: nil,
				// UserID:        nil,
				Name:         ptrTo(name),
				Description:  ptrTo("Ethereum Network"),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(true),
				Enabled:      ptrToBool(true),
				ChainID:      nil,
				// SidechainID:   nil,
				// NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network", // required
					"chain":              "unicorn-v0",                               // required
					"chainspec_abi_url":  "https://raw.githubusercontent.com/providenetwork/chain-spec/unicorn-v0/spec.abi.json",
					"chainspec_url":      nil,
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
				// Stats: nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}
