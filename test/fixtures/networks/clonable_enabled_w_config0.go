package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethClonableEnabledConfigNetwork0() (n *fixtures.FixtureMatcher) {
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
			common.StringOrNil("Config cloneable_cfg should not be null"),
			common.StringOrNil("Config chainspec_url value should not be empty"),
			common.StringOrNil("Config block_explorer_url should not be empty"),
			common.StringOrNil("Config chain should not be empty"),
			common.StringOrNil("Config engine_id should not be empty"),
			common.StringOrNil("Config native_currency should not be empty"),
			common.StringOrNil("Config network_id should not be empty"),
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
	name := "ETH NonProduction Cloneable Disabled Config nil cloneable_cfg nil chainspec_url nil block_explorer_url nil chain nil engine_id nil eth nil lb nil currency nil network_id nil protocol_id "

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
					"block_explorer_url":  nil, // required
					"chain":               nil, // required
					"chainspec_abi_url":   nil,
					"chainspec_url":       nil,
					"cloneable_cfg":       nil, // If cloneable CFG then security,
					"engine_id":           nil, // required
					"is_ethereum_network": nil, // required for ETH
					"is_load_balanced":    nil, // implies network load balancer count > 0
					"json_rpc_url":        nil,
					"native_currency":     nil, // required
					"network_id":          nil, // required
					"protocol_id":         nil, // required
					"websocket_url":       nil}),
				// Stats: nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}
