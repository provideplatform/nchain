package main

import (
	networkfixtures "github.com/provideapp/goldmine/test/fixtures/networks"
	"github.com/provideapp/goldmine/test/matchers"
)

func testNetworks() (nf []*networkFactory) {
	// ns = make([]map[string]interface{}, 0)
	// for _, nf := range networkfixtures.Networks() {
	// 	n, s := networkFactory(nf.Fixture.(*networkfixtures.NetworkFixture))
	// 	fmt.Printf("%v", n)
	// 	// Log.Debugf("%s", n)

	// 	ns = append(ns, map[string]interface{}{
	// 		"matchers": nf.Matcher,
	// 		"network":  n,
	// 		"name":     s,
	// 	})
	// }
	// return

	networks := networkfixtures.Networks()
	count := len(networks)
	nf = make([]*networkFactory, count)

	for i, n := range networks {
		fixture := n.Fixture.(*networkfixtures.NetworkFixture)
		nf[i] = &networkFactory{
			fixture:  fixture,
			name:     fixture.Name,
			matchers: n.Matcher,
		}
	}
	return
}

type networkFactory struct {
	fixture  *networkfixtures.NetworkFixture
	name     *string
	matchers *matchers.MatcherCollection
}

func (factory *networkFactory) network() (n *Network) {
	nf := factory.fixture.Fields
	n = &Network{
		Model:         nf.Model,
		ApplicationID: nf.ApplicationID,
		UserID:        nf.UserID,
		Name:          nf.Name,
		Description:   nf.Description,
		IsProduction:  nf.IsProduction,
		Cloneable:     nf.Cloneable,
		Enabled:       nf.Enabled,
		ChainID:       nf.ChainID,
		SidechainID:   nf.SidechainID,
		NetworkID:     nf.NetworkID,
		Config:        nf.Config,
		Stats:         nf.Stats,
	}
	return
}
