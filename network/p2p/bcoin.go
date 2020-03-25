package p2p

import (
	"fmt"

	"github.com/provideapp/goldmine/common"
)

const bcoinPoolIdentitySearchString = "Pool identity key:"

// BcoinP2PProvider is a network.p2p.API implementing the Bcoin API
type BcoinP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitBcoinP2PProvider initializes and returns the bcoin p2p provider
func InitBcoinP2PProvider(host string, port uint, ntwrk common.Configurable) *BcoinP2PProvider {
	return nil
}

// ParsePeerURL parses the peer url from the given logs
func (p *BcoinP2PProvider) ParsePeerURL(msg string) (*string, error) {
	// poolIdentityFoundIndex := strings.LastIndex(msg, bcoinPoolIdentitySearchString)
	// if poolIdentityFoundIndex != -1 {
	// 	defaultPeerListenPort := common.DefaultPeerDiscoveryPort
	// 	poolIdentity := strings.TrimSpace(msg[poolIdentityFoundIndex+len(bcoinPoolIdentitySearchString) : len(msg)-1])
	// 	node := fmt.Sprintf("%s@%s:%v", poolIdentity, *n.IPv4, defaultPeerListenPort)
	// 	peerURL = &node
	// 	cfg["peer_url"] = node
	// 	cfg["peer_identity"] = poolIdentity
	// }

	return nil, fmt.Errorf("bcoin p2p provider does not yet implement ParsePeerURL(")
}
