package main

import (
	"fmt"

	"github.com/provideapp/goldmine/test/fixtures"
)

// DefaultNetwork returns network with following values:
//   production: false
//   cloneable:  false
//   enabled:    true
// func DefaultNetwork() (n *Network) {
// 	return NetworkFactory(defaultNetwork)
// }
// func ETHNonProdClonableEnabledNilConfigNetwork() (n *Network) {
// 	return NetworkFactory(ethNonProdClonableEnabledNilConfigNetwork)
// }

func testNetworks() (ns []map[string]interface{}) {
	ns = make([]map[string]interface{}, 0)
	for _, nf := range fixtures.Networks() {
		n, s := networkFactory(nf)
		fmt.Printf("%v", n)
		// Log.Debugf("%s", n)

		ns = append(ns, map[string]interface{}{
			"network": n,
			"name":    s,
		})
	}
	return
}

// NetworkFactory receiving set of fields and putting them to Network object
func networkFactory(nfix *fixtures.NetworkFixture) (n *Network, s *string) {
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
