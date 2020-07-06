package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/nchain/test/fixtures"
	"github.com/provideapp/nchain/test/matchers"
)

func ethClonableDisabledConfigNetwork2233222222220() (n *fixtures.FixtureMatcher) {
	mc := &matchers.MatcherCollection{}
	optsNATSCreate := defaultNATSMatcherOptions(ptrTo("network.create"))

	mc.AddBehavior("Create", func(opts ...interface{}) types.GomegaMatcher {
		expectedCreateResult := true
		expectedContractCount := 1
		return matchers.NetworkCreateMatcher(expectedCreateResult, expectedContractCount, opts...)
	}, defaultMatcherOptions())
	mc.AddBehavior("Double Create", func(opts ...interface{}) types.GomegaMatcher {
		return BeFalse()
	}, defaultMatcherOptions())
	mc.AddBehavior("Create with NATS", func(opts ...interface{}) types.GomegaMatcher {
		return BeTrue()
	}, optsNATSCreate)
	mc.AddBehavior("Validate", func(opts ...interface{}) types.GomegaMatcher {
		expectedResult := true
		expectedErrorsCount := 0
		errors := []*string{}

		return matchers.NetworkValidateMatcher(expectedResult, expectedErrorsCount, errors, opts...)
	}, defaultMatcherOptions())
	mc.AddBehavior("ParseConfig", func(opts ...interface{}) types.GomegaMatcher {
		return satisfyAllConfigKeys(false)
	}, defaultMatcherOptions())
	mc.AddBehavior("RpcURL", func(opts ...interface{}) types.GomegaMatcher {
		return Equal("url")
	}, defaultMatcherOptions())
	mc.AddBehavior("NodeCount", func(opts ...interface{}) types.GomegaMatcher {
		return gstruct.PointTo(BeEquivalentTo(0))
	}, defaultMatcherOptions())

	mc.AddBehavior("Network type", func(opts ...interface{}) types.GomegaMatcher {
		if opts[0] == "eth" {
			return BeTrue()
		}
		if opts[0] == "btc" {
			return BeTrue()
		}
		if opts[0] == "handshake" {
			return BeTrue()
		}
		if opts[0] == "ltc" {
			return BeTrue()
		}
		if opts[0] == "quorum" {
			return BeTrue()
		}
		return BeNil()
	}, defaultMatcherOptions())
	name := "ETH NonProduction Cloneable Disabled Config w cloneable_cfg w chainspec "
	chainspecJSON, chainspecABIJSON := getChainspec()

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				// ApplicationID: nil,
				// UserID:        nil,
				Name:         ptrTo(generalName),
				Description:  ptrTo(generalDesc),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(true),
				Enabled:      ptrToBool(false),
				ChainID:      nil,
				// SidechainID:   nil,
				// NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url": "https://unicorn-explorer.provide.network", // required
					"chain":              "unicorn-v0",                               // required
					"chainspec":          chainspecJSON,
					"chainspec_abi":      chainspecABIJSON,
					"cloneable_cfg": map[string]interface{}{
						"security": map[string]interface{}{"egress": "*", "ingress": map[string]interface{}{"0.0.0.0/0": map[string]interface{}{"tcp": []int{5001, 8050, 8051, 8080, 30300}, "udp": []int{30300}}}}}, // If cloneable CFG then security,
					"engine_id":            "aura", // required
					"is_ethereum_network":  true,   // required for ETH
					"is_bcoin_network":     true,
					"is_handshake_network": true,
					"is_lcoin_network":     true,
					"is_quorum_network":    true,
					"is_load_balanced":     true, // implies network load balancer count > 0
					"json_rpc_url":         "url",
					"native_currency":      "PRVD", // required
					"network_id":           22,     // required
					"protocol_id":          "poa",  // required
					"websocket_url":        nil}),
				// Stats: nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}
