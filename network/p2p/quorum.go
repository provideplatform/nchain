package p2p

import (
	"errors"

	provide "github.com/provideservices/provide-go"
)

// QuorumP2PProvider is a network.P2PAPI implementing the geth API
type QuorumP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitQuorumP2PProvider initializes and returns the geth p2p provider
func InitQuorumP2PProvider(rpcURL string) *QuorumP2PProvider {
	return &QuorumP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
	}
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *QuorumP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("quorum does not implement AcceptNonReservedPeers()")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *QuorumP2PProvider) DropNonReservedPeers() error {
	return errors.New("quorum does not implement DropNonReservedPeers()")
}

// AddPeer adds a peer by its peer url
func (p *QuorumP2PProvider) AddPeer(peerURL string) error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "admin_addPeer", []interface{}{peerURL}, &resp)
}

// RemovePeer removes a peer by its peer url
func (p *QuorumP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("quorum p2p provider does not impl RemovePeer()")
}

// Upgrade executes a pending upgrade
func (p *QuorumP2PProvider) Upgrade() error {
	return errors.New("quorum p2p provider does not impl Upgrade()")
}
