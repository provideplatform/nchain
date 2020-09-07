package p2p

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

// NethermindP2PProvider is a network.p2p.API implementing the nethermind API
type NethermindP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
	networkID    string
}

// InitNethermindP2PProvider initializes and returns the parity p2p provider
func InitNethermindP2PProvider(rpcURL *string, networkID string, ntwrk common.Configurable) *NethermindP2PProvider {
	return &NethermindP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
		networkID:    networkID,
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

// FetchTxReceipt fetch a transaction receipt given its hash
func (p *NethermindP2PProvider) FetchTxReceipt(signerAddress, hash string) (*provide.TxReceipt, error) {
	receipt, err := evmFetchTxReceipt(p.networkID, *p.rpcURL, signerAddress, hash)
	if err != nil {
		return nil, err
	}

	logs := make([]interface{}, 0)
	for _, log := range receipt.Logs {
		logs = append(logs, *log)
	}

	return &provide.TxReceipt{
		TxHash:            receipt.TxHash,
		ContractAddress:   receipt.ContractAddress,
		GasUsed:           receipt.GasUsed,
		BlockHash:         receipt.BlockHash,
		BlockNumber:       receipt.BlockNumber,
		TransactionIndex:  receipt.TransactionIndex,
		PostState:         receipt.PostState,
		Status:            receipt.Status,
		CumulativeGasUsed: receipt.CumulativeGasUsed,
		Bloom:             receipt.Bloom,
		Logs:              logs,
	}, nil
}

// FetchTxTraces fetch transaction traces given its hash
func (p *NethermindP2PProvider) FetchTxTraces(hash string) (*provide.TxTrace, error) {
	traces, err := evmFetchTxTraces(p.networkID, *p.rpcURL, hash)
	if err != nil {
		return nil, err
	}

	// HACK!!!
	prvdTraces := &provide.TxTrace{}
	rawTraces, _ := json.Marshal(traces)
	json.Unmarshal(rawTraces, &prvdTraces)

	return prvdTraces, nil
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

// ResolveTokenContract attempts to resolve the given token contract details for the contract at a given address
func (p *NethermindP2PProvider) ResolveTokenContract(signerAddress string, receipt interface{}, artifact *provide.CompiledArtifact) (*string, *big.Int, *string, error) {
	switch receipt.(type) {
	case *types.Receipt:
		contractAddress := receipt.(*types.Receipt).ContractAddress
		return evmResolveTokenContract(*p.rpcClientKey, *p.rpcURL, artifact, contractAddress.Hex(), signerAddress)
	}

	return nil, nil, nil, errors.New("given tx receipt was of invalid type")
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
