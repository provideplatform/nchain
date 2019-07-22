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

// AddPeer adds a peer by its peer url
func (p *ParityP2PProvider) AddPeer(peerURL string) error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_addReservedPeer", []interface{}{peerURL}, nil)
}

// RemovePeer removes a peer by its peer url
func (p *ParityP2PProvider) RemovePeer(peerURL string) error {
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_removeReservedPeer", []interface{}{peerURL}, nil)
}
