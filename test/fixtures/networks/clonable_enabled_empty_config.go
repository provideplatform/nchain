package networkfixtures

import (
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethClonableEnabledEmptyConfigNetwork() (n *fixtures.FixtureMatcher) {
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
		expectedErrorsCount := 9
		errors := []*string{
			common.StringOrNil("Config should not be empty"),
			common.StringOrNil("Config cloneable_cfg should be present"),
			common.StringOrNil("Config chainspec_url or chainspec should be present"),
			common.StringOrNil("Config block_explorer_url should not be nil"),
			common.StringOrNil("Config chain should not be nil"),
			common.StringOrNil("Config engine_id should not be nil"),
			common.StringOrNil("Config native_currency should not be nil"),
			common.StringOrNil("Config network_id should not be nil"),
			common.StringOrNil("Config protocol_id should not be nil"),
		}

		return matchers.NetworkValidateMatcher(expectedResult, expectedErrorsCount, errors, opts...)
	}, defaultMatcherOptions())
	mc.AddBehavior("ParseConfig", func(opts ...interface{}) types.GomegaMatcher {
		return BeEmpty()
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

	name := "ETH NonProduction Cloneable Enabled Empty Config "
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
				Config: marshalConfig(map[string]interface{}{}),
				// Stats:         nil
			},
			Name: ptrTo(name)},
		Matcher: mc}

	return
}
