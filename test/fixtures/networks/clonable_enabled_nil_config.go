package networkfixtures

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/provideapp/goldmine/test/fixtures"
	"github.com/provideapp/goldmine/test/matchers"
)

func ethNonProdClonableEnabledNilConfigNetwork() (n *fixtures.FixtureMatcher) {
	mc := &matchers.MatcherCollection{}
	mc.AddBehavior("Create", func() types.GomegaMatcher {
		return BeFalse()
	})
	mc.AddBehavior("Validate", func() types.GomegaMatcher {
		return BeFalse()
	})
	mc.AddBehavior("ParseConfig", func() types.GomegaMatcher {
		return BeTrue()
	})

	namePtr := ptrTo("ETH NonProd Clonable Enabled Nil Config")

	n = &fixtures.FixtureMatcher{
		Fixture: &NetworkFixture{
			Fields: &NetworkFields{
				ApplicationID: nil,
				UserID:        nil,
				Name:          namePtr,
				Description:   ptrTo("Ethereum Network"),
				IsProduction:  ptrToBool(false),
				Cloneable:     ptrToBool(true),
				Enabled:       ptrToBool(true),
				ChainID:       nil,
				SidechainID:   nil,
				NetworkID:     nil,
				Config:        nil,
				Stats:         nil},
			Name: namePtr},
		Matcher: mc}

	return
}
