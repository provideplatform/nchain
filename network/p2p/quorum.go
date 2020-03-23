package p2p

import (
	"errors"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// QuorumP2PProvider is a network.P2PAPI implementing the geth API
type QuorumP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
}

// InitQuorumP2PProvider initializes and returns the geth p2p provider
func InitQuorumP2PProvider(rpcURL *string, ntwrk common.Configurable) *QuorumP2PProvider {
	return &QuorumP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
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
	return provide.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "admin_addPeer", []interface{}{peerURL}, &resp)
}

// RemovePeer removes a peer by its peer url
func (p *QuorumP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("quorum p2p provider does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *QuorumP2PProvider) ResolvePeerURL() error {
	return errors.New("quorum p2p provider does not impl ResolvePeerURL()")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *QuorumP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	var err error
	common.Log.Debugf("quorum p2p provider RequireBootnodes() no-op")
	return err
}

// Upgrade executes a pending upgrade
func (p *QuorumP2PProvider) Upgrade() error {
	return errors.New("quorum p2p provider does not impl Upgrade()")
}
