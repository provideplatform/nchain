package p2p

import (
	"errors"
	"math/big"
	"strings"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

// HederaP2PProvider is a network.p2p.API implementing the hedera hashgraph API
type HederaP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
	networkID    string
}

// InitHederaP2PProvider initializes and returns the parity p2p provider
func InitHederaP2PProvider(rpcURL *string, networkID string, ntwrk common.Configurable) *HederaP2PProvider {
	return &HederaP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
		networkID:    networkID,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *HederaP2PProvider) DefaultEntrypoint() []string {
	return []string{}
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *HederaP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)
	return cmd
}

// FetchTxReceipt fetch a transaction receipt given its hash
func (p *HederaP2PProvider) FetchTxReceipt(signerAddress, hash string) (*provide.TxReceipt, error) {
	return nil, errors.New("hedera p2p client does not impl FetchTxReceipt()")
}

// FetchTxTraces fetch transaction traces given its hash
func (p *HederaP2PProvider) FetchTxTraces(hash string) (*provide.TxTrace, error) {
	return nil, errors.New("hedera p2p client does not impl FetchTxTraces()")
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *HederaP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("hedera p2p client does not impl AcceptNonReservedPeers()")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *HederaP2PProvider) DropNonReservedPeers() error {
	return errors.New("hedera p2p client does not impl DropNonReservedPeers()")
}

// AddPeer adds a peer by its peer url
func (p *HederaP2PProvider) AddPeer(peerURL string) error {
	return errors.New("hedera p2p client does not impl AddPeer()")
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *HederaP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw log string
func (p *HederaP2PProvider) ParsePeerURL(string) (*string, error) {
	return nil, errors.New("hedera p2p client does not impl ParsePeerURL()")
}

// RemovePeer removes a peer by its peer url
func (p *HederaP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("hedera p2p client does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *HederaP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("hedera p2p client does not impl ResolvePeerURL()")
}

// ResolveTokenContract attempts to resolve the given token contract details for the contract at a given address
func (p *HederaP2PProvider) ResolveTokenContract(signerAddress string, receipt interface{}, artifact *provide.CompiledArtifact) (*string, *big.Int, *string, error) {
	return nil, nil, nil, errors.New("hedera p2p client does not impl ResolveTokenContract()")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *HederaP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	return errors.New("hedera p2p client does not impl RequireBootnodes()")
}

// Upgrade executes a pending upgrade
func (p *HederaP2PProvider) Upgrade() error {
	return errors.New("hedera p2p client does not impl Upgrade()")
}
