package p2p

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	provide "github.com/provideplatform/provide-go/api"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

// BaseledgerP2PProvider is a network.p2p.API implementing the baseledger API
type BaseledgerP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
	networkID    string
}

// BaseledgerP2PProvider initializes and returns the baseledger p2p provider
func InitBaseledgerP2PProvider(rpcURL *string, networkID string, ntwrk common.Configurable) *BaseledgerP2PProvider {
	return &BaseledgerP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
		networkID:    networkID,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *BaseledgerP2PProvider) DefaultEntrypoint() []string {
	return []string{}
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *BaseledgerP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)

	return cmd
}

// AcceptNonReservedPeers allows non-reserved peers to connect
func (p *BaseledgerP2PProvider) AcceptNonReservedPeers() error {
	return errors.New("not yet implemented")
}

// DropNonReservedPeers only allows reserved peers to connect; reversed by calling `AcceptNonReservedPeers`
func (p *BaseledgerP2PProvider) DropNonReservedPeers() error {
	return errors.New("not yet implemented")
}

// AddPeer adds a peer by its peer url
func (p *BaseledgerP2PProvider) AddPeer(peerURL string) error {
	return errors.New("not yet implemented")
}

// FetchTxReceipt fetch a transaction receipt given its hash
func (p *BaseledgerP2PProvider) FetchTxReceipt(signerAddress, hash string) (*nchain.TxReceipt, error) {
	httpClient := &provide.Client{
		Host:   *p.rpcURL,
		Scheme: "http",
	}

	status, resp, err := httpClient.Get("tx", map[string]interface{}{"hash": hash})

	if err != nil {
		return nil, err
	}

	if status != 200 {
		respJSON, _ := json.Marshal(resp)
		return nil, errors.New(string(respJSON))
	}

	txEntity := &nchain.TendermintTx{}
	respJSON, _ := json.Marshal(resp)
	json.Unmarshal(respJSON, &txEntity)

	status, resp, err = httpClient.Get("block", map[string]interface{}{"height": txEntity.Result.Height})

	if err != nil {
		return nil, err
	}

	if status != 200 {
		respJSON, _ := json.Marshal(resp)
		return nil, errors.New(string(respJSON))
	}

	blockEntity := &nchain.TendermintBlock{}
	respJSON, _ = json.Marshal(resp)
	json.Unmarshal(respJSON, &blockEntity)

	gasUsed, _ := strconv.Atoi(txEntity.Result.TxResult.GasUsed)
	n := new(big.Int)
	blockNumber, _ := n.SetString(txEntity.Result.Height, 10)
	var logs []interface{}
	json.Unmarshal([]byte(txEntity.Result.TxResult.Log), &logs)
	return &nchain.TxReceipt{
		TxHash:            []byte(txEntity.Result.Hash),
		ContractAddress:   nil,
		GasUsed:           uint64(gasUsed),
		BlockHash:         []byte(blockEntity.Result.BlockID.Hash),
		BlockNumber:       blockNumber,
		TransactionIndex:  0,
		PostState:         nil,
		Status:            uint64(txEntity.Result.TxResult.Code),
		CumulativeGasUsed: uint64(gasUsed),
		Bloom:             nil,
		Logs:              logs,
	}, nil
}

// FetchTxTraces fetch transaction traces given its hash
func (p *BaseledgerP2PProvider) FetchTxTraces(hash string) (*nchain.TxTrace, error) {
	return nil, errors.New("not yet implemented")
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *BaseledgerP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw log string
func (p *BaseledgerP2PProvider) ParsePeerURL(string) (*string, error) {
	return nil, errors.New("not yet implemented")
}

// RemovePeer removes a peer by its peer url
func (p *BaseledgerP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("not yet implemented")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *BaseledgerP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("not yet implemented")
}

// ResolveTokenContract attempts to resolve the given token contract details for the contract at a given address
func (p *BaseledgerP2PProvider) ResolveTokenContract(signerAddress string, receipt interface{}, artifact *nchain.CompiledArtifact) (*string, *string, *big.Int, *string, error) {
	return nil, nil, nil, nil, errors.New("not yet implemented")
}

// RequireBootnodes attempts to resolve the peers to use as bootnodes
func (p *BaseledgerP2PProvider) RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error {
	return errors.New("not yet implemented")
}

// Upgrade executes a pending upgrade
func (p *BaseledgerP2PProvider) Upgrade() error {
	return errors.New("not yet implemented")
}
