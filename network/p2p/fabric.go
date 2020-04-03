package p2p

import (
	"errors"
	"strings"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
)

// HyperledgerFabricP2PProvider is a network.p2p.API implementing the hyperledger fabric API
type HyperledgerFabricP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
}

// InitHyperledgerFabricP2PProvider initializes and returns the parity p2p provider
func InitHyperledgerFabricP2PProvider(rpcURL *string, ntwrk common.Configurable) *HyperledgerFabricP2PProvider {
	return &HyperledgerFabricP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *HyperledgerFabricP2PProvider) DefaultEntrypoint() []string {
	cmd := make([]string, 0)
	cmd = append(
		cmd,
		"peer",
		"node",
		"start",
	)

	return cmd
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *HyperledgerFabricP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)
	return cmd
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *HyperledgerFabricP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("fabric does not implement AcceptNonReservedPeers()")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *HyperledgerFabricP2PProvider) DropNonReservedPeers() error {
	return errors.New("fabric does not implement DropNonReservedPeers()")
}

// AddPeer adds a peer by its peer url
func (p *HyperledgerFabricP2PProvider) AddPeer(peerURL string) error {
	return nil
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *HyperledgerFabricP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw logs
func (p *HyperledgerFabricP2PProvider) ParsePeerURL(msg string) (*string, error) {
	return nil, errors.New("fabric p2p provider failed to parse peer url")
}

// RemovePeer removes a peer by its peer url
func (p *HyperledgerFabricP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("fabric p2p provider does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *HyperledgerFabricP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("fabric p2p provider does not impl ResolvePeerURL()")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *HyperledgerFabricP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	var err error
	common.Log.Debugf("fabric p2p provider RequireBootnodes() no-op")
	return err
}

// Upgrade executes a pending upgrade
func (p *HyperledgerFabricP2PProvider) Upgrade() error {
	if p.rpcURL == nil {
		return errors.New("fabric client unable to invoke admin_addPeer; rpc url unresolved")
	}
	return errors.New("fabric p2p provider does not impl Upgrade()")
}
