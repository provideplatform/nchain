package n_test

import (
	"github.com/provideapp/goldmine/network"
	networkfixtures "github.com/provideapp/goldmine/test/fixtures/networks"
	"github.com/provideapp/goldmine/test/matchers"
)

func TestNetworks() (nf []*networkFactory, nc []*networkfixtures.NetworkFixture) {
	// ns = make([]map[string]interface{}, 0)
	// for _, nf := range networkfixtures.Networks() {
	// 	n, s := networkFactory(nf.Fixture.(*networkfixtures.NetworkFixture))
	// 	fmt.Printf("%v", n)
	// 	// common.Log.Debugf("%s", n)

	// 	ns = append(ns, map[string]interface{}{
	// 		"matchers": nf.Matcher,
	// 		"network":  n,
	// 		"name":     s,
	// 	})
	// }
	// return

	networkFixtureGenerator := networkfixtures.NewNetworkFixtureGenerator()
	dispatcher := networkfixtures.NewNetworkFixtureDispatcher(networkFixtureGenerator)

	networks := dispatcher.Networks()
	count := len(networks)
	nf = make([]*networkFactory, count)
	nc = dispatcher.NotCovered()

	for i, n := range networks {
		fixture := n.Fixture.(*networkfixtures.NetworkFixture)
		nf[i] = &networkFactory{
			fixture:  fixture,
			Name:     fixture.Name,
			Matchers: n.Matcher,
		}
	}
	return
}

type networkFactory struct {
	fixture  *networkfixtures.NetworkFixture
	Name     *string
	Matchers *matchers.MatcherCollection
}

func (factory *networkFactory) Network() (n *network.Network) {
	nf := factory.fixture.Fields
	n = &network.Network{
		// ApplicationID: nf.ApplicationID,
		// UserID:        nf.UserID,
		Name:         nf.Name,
		Description:  nf.Description,
		IsProduction: nf.IsProduction,
		Cloneable:    nf.Cloneable,
		Enabled:      nf.Enabled,
		ChainID:      nf.ChainID,
		// SidechainID:   nf.SidechainID,
		// NetworkID:     nf.NetworkID,
		Config: nf.Config,
		// Stats:         nf.Stats,
	}
	return
}
