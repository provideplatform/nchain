package p2p

import (
	"errors"
	"strings"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
)

// NethermindP2PProvider is a network.p2p.API implementing the nethermind API
type NethermindP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
}

// InitNethermindP2PProvider initializes and returns the parity p2p provider
func InitNethermindP2PProvider(rpcURL *string, ntwrk common.Configurable) *NethermindP2PProvider {
	return &NethermindP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *NethermindP2PProvider) DefaultEntrypoint() []string {
	return []string{}
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *NethermindP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)
	return cmd
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *NethermindP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("nethermind p2p client does not impl AcceptNonReservedPeers()")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *NethermindP2PProvider) DropNonReservedPeers() error {
	return errors.New("nethermind p2p client does not impl DropNonReservedPeers()")
}

// AddPeer adds a peer by its peer url
func (p *NethermindP2PProvider) AddPeer(peerURL string) error {
	return errors.New("nethermind p2p client does not impl AddPeer()")
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *NethermindP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw log string
func (p *NethermindP2PProvider) ParsePeerURL(string) (*string, error) {
	return nil, errors.New("nethermind p2p client does not impl ParsePeerURL()")
}

// RemovePeer removes a peer by its peer url
func (p *NethermindP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("nethermind p2p client does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *NethermindP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("nethermind p2p client does not impl ResolvePeerURL()")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *NethermindP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	var err error
	common.Log.Debugf("nethermind p2p provider RequireBootnodes() no-op")
	return err
}

// Upgrade executes a pending upgrade
func (p *NethermindP2PProvider) Upgrade() error {
	return errors.New("nethermind p2p client does not impl Upgrade()")
}
