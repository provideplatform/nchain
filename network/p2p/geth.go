package p2p

import (
	"errors"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// GethP2PProvider is a network.P2PAPI implementing the parity API
type GethP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
}

// InitGethP2PProvider initializes and returns the parity p2p provider
func InitGethP2PProvider(rpcURL *string, ntwrk common.Configurable) *GethP2PProvider {
	return &GethP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
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
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "admin_addPeer", []interface{}{peerURL}, &resp)
}

// RemovePeer removes a peer by its peer url
func (p *GethP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("geth p2p provider does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *GethP2PProvider) ResolvePeerURL() error {
	return errors.New("geth p2p provider does not impl ResolvePeerURL()")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *GethP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	var err error
	common.Log.Debugf("geth p2p provider RequireBootnodes() no-op")
	return err
}

// Upgrade executes a pending upgrade
func (p *GethP2PProvider) Upgrade() error {
	if p.rpcURL == nil {
		return errors.New("geth client unable to invoke admin_addPeer; rpc url unresolved")
	}
	return errors.New("geth p2p provider does not impl Upgrade()")
}
