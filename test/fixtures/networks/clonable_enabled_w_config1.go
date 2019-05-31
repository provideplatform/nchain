package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethClonableEnabledConfigNetwork1() (n *fixtures.FixtureMatcher) {
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
		expectedResult := false
		expectedErrorsCount := 8
		errors := []*string{
			common.StringOrNil("Config security value should be present for clonable network"),
			common.StringOrNil("Config chainspec_url value should be a valid URL"),
			common.StringOrNil("Config block_explorer_url should not be empty"),
			common.StringOrNil("Config chain should not be empty"),
			common.StringOrNil("Config engine_id should not be empty"),
			common.StringOrNil("Config native_currency should not be empty"),
			common.StringOrNil("Config network_id should not be zero"),
			common.StringOrNil("Config protocol_id should not be empty"),
		}

		return matchers.NetworkValidateMatcher(expectedResult, expectedErrorsCount, errors, opts...)
	}, defaultMatcherOptions())
	mc.AddBehavior("ParseConfig", func(opts ...interface{}) types.GomegaMatcher {
		return satisfyAllConfigKeys(true)
	}, defaultMatcherOptions())
	mc.AddBehavior("RpcURL", func(opts ...interface{}) types.GomegaMatcher {
		return BeEmpty()
	}, defaultMatcherOptions())
	mc.AddBehavior("NodeCount", func(opts ...interface{}) types.GomegaMatcher {
		return gstruct.PointTo(BeEquivalentTo(0))
	}, defaultMatcherOptions())
	mc.AddBehavior("AvailablePeerCount", func(opts ...interface{}) types.GomegaMatcher {
		return BeEquivalentTo(0)
	}, defaultMatcherOptions())
	mc.AddBehavior("Network type", func(opts ...interface{}) types.GomegaMatcher {
		if opts[0] == "eth" {
			return BeFalse()
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
	name := "ETH NonProduction Cloneable Disabled Config empty cloneable_cfg empty chainspec_url empty block_explorer_url empty chain empty engine_id eth:f lb:f empty currency empty network_id empty protocol_id "

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				// ApplicationID: nil,
				// UserID:        nil,
				Name:         ptrTo(generalName),
				Description:  ptrTo(generalDesc),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(true),
				Enabled:      ptrToBool(true),
				ChainID:      nil,
				// SidechainID:   nil,
				// NetworkID:     nil,
				Config: marshalConfig(map[string]interface{}{
					"block_explorer_url":  "", // required
					"chain":               "", // required
					"chainspec_abi_url":   "",
					"chainspec_url":       "",
					"cloneable_cfg":       map[string]interface{}{}, // If cloneable CFG then security,
					"engine_id":           "",                       // required
					"is_ethereum_network": false,                    // required for ETH
					"is_load_balanced":    false,                    // implies network load balancer count > 0
					"json_rpc_url":        "",
					"native_currency":     "", // required
					"network_id":          0,  // required
					"protocol_id":         "", // required
					"websocket_url":       ""}),
				// Stats: nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}
