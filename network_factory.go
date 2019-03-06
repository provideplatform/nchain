package main

import (
	"fmt"

	networkfixtures "github.com/provideapp/goldmine/test/fixtures/networks"
)

func testNetworks() (ns []map[string]interface{}) {
	ns = make([]map[string]interface{}, 0)
	for _, nf := range networkfixtures.Networks() {
		n, s := networkFactory(nf.Fixture.(*networkfixtures.NetworkFixture))
		fmt.Printf("%v", n)
		// Log.Debugf("%s", n)

		ns = append(ns, map[string]interface{}{
			"matchers": nf.Matcher,
			"network":  n,
			"name":     s,
		})
	}
	return
}

// NetworkFactory receiving set of fields and putting them to Network object
func networkFactory(nfix *networkfixtures.NetworkFixture) (n *Network, s *string) {
	nf := nfix.Fields

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
	s = nfix.Name
	return
}
