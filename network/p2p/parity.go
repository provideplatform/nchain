package p2p

import (
	provide "github.com/provideservices/provide-go"
)

// ParityP2PProvider is a network.P2PAPI implementing the parity API
type ParityP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitParityP2PProvider initializes and returns the parity p2p provider
func InitParityP2PProvider(rpcURL string) *ParityP2PProvider {
	return &ParityP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
	}
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *ParityP2PProvider) AcceptNonReservedPeers() error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_acceptNonReservedPeers", []interface{}{}, nil)
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *ParityP2PProvider) DropNonReservedPeers(peerURL string) error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_dropNonReservedPeers", []interface{}{}, nil)
}

// AddPeer adds a peer by its peer url
func (p *ParityP2PProvider) AddPeer(peerURL string) error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_addReservedPeer", []interface{}{peerURL}, nil)
}

// RemovePeer removes a peer by its peer url
func (p *ParityP2PProvider) RemovePeer(peerURL string) error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_removeReservedPeer", []interface{}{peerURL}, nil)
}

// Upgrade executes a pending upgrade
func (p *ParityP2PProvider) Upgrade() error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_executeUpgrade", []interface{}{}, nil)
}
