package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethClonableEnabledNilConfigNetwork() (n *fixtures.FixtureMatcher) {
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
		return BeEmpty()
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

	namePtr := ptrTo("ETH NonProduction Cloneable Enabled Nil Config ")

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				// ApplicationID: nil,
				// UserID:        nil,
				Name:         namePtr,
				Description:  ptrTo("Ethereum Network"),
				IsProduction: ptrToBool(false),
				Cloneable:    ptrToBool(true),
				Enabled:      ptrToBool(true),
				ChainID:      nil,
				// SidechainID:   nil,
				// NetworkID:     nil,
				Config: nil,
				// Stats:  nil
			},
			Name: namePtr},
		Matcher: mc}

	return
}