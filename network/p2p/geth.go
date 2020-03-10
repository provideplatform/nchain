package p2p

import (
	"errors"

	provide "github.com/provideservices/provide-go"
)

// GethP2PProvider is a network.P2PAPI implementing the parity API
type GethP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitGethP2PProvider initializes and returns the parity p2p provider
func InitGethP2PProvider(rpcURL string) *GethP2PProvider {
	return &GethP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
	}
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *GethP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("geth does not implement AcceptNonReservedPeers()")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *GethP2PProvider) DropNonReservedPeers() error {
	return errors.New("geth does not implement DropNonReservedPeers()")
}

// AddPeer adds a peer by its peer url
func (p *GethP2PProvider) AddPeer(peerURL string) error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "admin_addPeer", []interface{}{peerURL}, &resp)
}

// RemovePeer removes a peer by its peer url
func (p *GethP2PProvider) RemovePeer(peerURL string) error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "admin_removePeer", []interface{}{peerURL}, &resp)
}

// Upgrade executes a pending upgrade
func (p *GethP2PProvider) Upgrade() error {
	var resp interface{}
	return provide.EVMInvokeJsonRpcClient(p.rpcClientKey, p.rpcURL, "parity_executeUpgrade", []interface{}{}, &resp)
}
