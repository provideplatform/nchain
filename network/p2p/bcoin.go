package p2p

import "github.com/provideapp/goldmine/common"

// BcoinP2PProvider is a network.P2PAPI implementing the Bcoin API
type BcoinP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitBcoinP2PProvider initializes and returns the bcoin p2p provider
func InitBcoinP2PProvider(host string, port uint, ntwrk common.Configurable) *BcoinP2PProvider {
	return nil
}
