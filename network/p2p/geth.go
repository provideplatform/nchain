package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	provide "github.com/provideplatform/provide-go/api/nchain"
	providecrypto "github.com/provideplatform/provide-go/crypto"
)

// GethP2PProvider is a network.p2p.API implementing the parity API
type GethP2PProvider struct {
	rpcClientKey *string
	rpcURL       *string
	network      common.Configurable
	networkID    string
}

// InitGethP2PProvider initializes and returns the parity p2p provider
func InitGethP2PProvider(rpcURL *string, networkID string, ntwrk common.Configurable) *GethP2PProvider {
	return &GethP2PProvider{
		rpcClientKey: rpcURL,
		rpcURL:       rpcURL,
		network:      ntwrk,
		networkID:    networkID,
	}
}

// DefaultEntrypoint returns the default entrypoint to run when starting the container, when one is not otherwise provided
func (p *GethP2PProvider) DefaultEntrypoint() []string {
	cmd := make([]string, 0)

	cfg := p.network.ParseConfig()
	if chainspec, chainspecOk := cfg["chainspec"].(map[string]interface{}); chainspecOk {
		chainspecJSON, _ := json.Marshal(chainspec)
		cmd = append(
			cmd,
			fmt.Sprintf("/bin/sh -c 'tee genesis.json <<<'%s'", string(chainspecJSON)),
			"geth init genesis.json &&",
		)
	}

	cmd = append(
		cmd,
		"geth",
		"--nousb",
		"--nodiscover",
		"--gcmode", "archive",
		"--rpc",
		"--rpcaddr", "0.0.0.0",
		"--rpccorsdomain", "*",
		"--rpcapi", "admin,eth,miner,net,web3,shh",
		"--ws",
		"--wsaddr", "0.0.0.0",
		"--wsapi", "eth,net,web3,shh",
		"--wsorigins", "*",
		"--graphql",
		"--shh",
		"--verbosity", "6",
	)

	return cmd
}

// EnrichStartCommand returns the cmd to append to the command to start the container
func (p *GethP2PProvider) EnrichStartCommand(bootnodes []string) []string {
	cmd := make([]string, 0)
	cfg := p.network.ParseConfig()
	if networkID, networkIDOk := cfg["network_id"].(float64); networkIDOk {
		cmd = append(cmd, "--networkid", fmt.Sprintf("%d", uint64(networkID)))
	}

	cfgBootnodes, cfgBootnodesOk := cfg["bootnodes"].([]string)
	if len(bootnodes) > 0 || (cfgBootnodesOk && len(cfgBootnodes) > 0) {
		_bootnodes := make([]string, 0)
		for i := range bootnodes {
			_bootnodes = append(_bootnodes, bootnodes[i])
		}
		if cfgBootnodesOk {
			for i := range cfgBootnodes {
				_bootnodes = append(_bootnodes, cfgBootnodes[i])
			}
		}
		cmd = append(cmd, "--bootnodes", p.FormatBootnodes(_bootnodes))
	}

	return cmd
}

// FetchTxReceipt fetch a transaction receipt given its hash
func (p *GethP2PProvider) FetchTxReceipt(signerAddress, hash string) (*provide.TxReceipt, error) {
	receipt, err := evmFetchTxReceipt(p.networkID, *p.rpcURL, signerAddress, hash)
	if err != nil {
		return nil, err
	}

	logs := make([]interface{}, 0)
	for _, log := range receipt.Logs {
		logs = append(logs, *log)
	}

	return &provide.TxReceipt{
		TxHash:            receipt.TxHash.Bytes(),
		ContractAddress:   receipt.ContractAddress.Bytes(),
		GasUsed:           receipt.GasUsed,
		BlockHash:         receipt.BlockHash.Bytes(),
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
func (p *GethP2PProvider) FetchTxTraces(hash string) (*provide.TxTrace, error) {
	traces := map[string]interface{}{}
	err := providecrypto.EVMInvokeJsonRpcClient(p.networkID, *p.rpcURL, "debug_traceTransaction", []interface{}{hash}, &traces)
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
	return providecrypto.EVMInvokeJsonRpcClient(*p.rpcClientKey, *p.rpcURL, "admin_addPeer", []interface{}{peerURL}, &resp)
}

// FormatBootnodes formats the given peer urls as a valid bootnodes param
func (p *GethP2PProvider) FormatBootnodes(bootnodes []string) string {
	return strings.Join(bootnodes, ",")
}

// ParsePeerURL parses a peer url from the given raw logs
func (p *GethP2PProvider) ParsePeerURL(msg string) (*string, error) {
	nodeInfo := &provide.EthereumJsonRpcResponse{}
	err := json.Unmarshal([]byte(msg), &nodeInfo)
	if err == nil && nodeInfo != nil {
		result, resultOk := nodeInfo.Result.(map[string]interface{})
		if resultOk {
			if enode, enodeOk := result["enode"].(string); enodeOk {
				peerURL := common.StringOrNil(enode)
				// cfg["peer"] = result
				return peerURL, nil
			}
		}
	} else if err != nil {
		enodeIndex := strings.LastIndex(msg, "enode://")
		if enodeIndex != -1 {
			enode := msg[enodeIndex:]
			peerURL := common.StringOrNil(enode)
			// cfg["peer_url"] = enode
			return peerURL, nil
		}
	}
	return nil, errors.New("geth p2p provider failed to parse peer url")
}

// RemovePeer removes a peer by its peer url
func (p *GethP2PProvider) RemovePeer(peerURL string) error {
	return errors.New("geth p2p provider does not impl RemovePeer()")
}

// ResolvePeerURL attempts to resolve one or more viable peer urls
func (p *GethP2PProvider) ResolvePeerURL() (*string, error) {
	return nil, errors.New("geth p2p provider does not impl ResolvePeerURL()")
}

// ResolveTokenContract attempts to resolve the given token contract details for the contract at a given address
func (p *GethP2PProvider) ResolveTokenContract(signerAddress string, receipt interface{}, artifact *provide.CompiledArtifact) (*string, *string, *big.Int, *string, error) {
	switch receipt.(type) {
	case *types.Receipt:
		contractAddress := receipt.(*types.Receipt).ContractAddress
		return evmResolveTokenContract(*p.rpcClientKey, *p.rpcURL, artifact, contractAddress.Hex(), signerAddress)
	}

	return nil, nil, nil, nil, errors.New("given tx receipt was of invalid type")
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
