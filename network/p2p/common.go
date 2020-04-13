package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// PlatformBcoin bcoin platform
const PlatformBcoin = "bcoin"

// PlatformEVM evm platform
const PlatformEVM = "evm"

// PlatformHandshake handshake platform
const PlatformHandshake = "handshake"

// PlatformHyperledgerBesu hyperledger besu platform
const PlatformHyperledgerBesu = "hyperledger_besu"

// PlatformHyperledgerFabric hyperledger fabric platform
const PlatformHyperledgerFabric = "hyperledger_fabric"

// PlatformQuorum quorum platform
const PlatformQuorum = "quorum"

// ProviderBcoin bcoin p2p provider
const ProviderBcoin = "bcoin"

// ProviderGeth geth p2p provider
const ProviderGeth = "geth"

// ProviderHyperledgerBesu besu p2p provider
const ProviderHyperledgerBesu = "hyperledger_besu"

// ProviderHyperledgerFabric fabric p2p provider
const ProviderHyperledgerFabric = "hyperledger_fabric"

// ProviderNethermind nethermind p2p provider
const ProviderNethermind = "nethermind"

// ProviderParity parity p2p provider
const ProviderParity = "parity"

// ProviderQuorum quorum p2p provider
const ProviderQuorum = "quorum"

// API defines an interface for p2p network implementations
type API interface {
	AcceptNonReservedPeers() error
	DropNonReservedPeers() error
	AddPeer(string) error
	RemovePeer(string) error
	ParsePeerURL(string) (*string, error)
	FetchTxReceipt(signerAddress, hash string) (*provide.TxReceipt, error)
	FetchTxTraces(hash string) (*provide.TxTrace, error)
	FormatBootnodes([]string) string
	RequireBootnodes(db *gorm.DB, userID *uuid.UUID, networkID *uuid.UUID, n common.Configurable) error
	ResolvePeerURL() (*string, error)
	ResolveTokenContract(string, interface{}, *provide.CompiledArtifact) (*string, *big.Int, *string, error) // name, decimals, symbol, error
	Upgrade() error

	DefaultEntrypoint() []string
	EnrichStartCommand(bootnodes []string) []string
}

func evmFetchTxReceipt(rpcClientKey, rpcURL, signerAddress, hash string) (*types.Receipt, error) {
	receipt, err := provide.EVMGetTxReceipt(rpcClientKey, rpcURL, hash, signerAddress)
	if err != nil {
		common.Log.Warningf("failed to fetch tx receipt; %s", err.Error())
		return nil, err
	}
	return receipt, nil
}

func evmFetchTxTraces(rpcClientKey, rpcURL, hash string) (*provide.EthereumTxTraceResponse, error) {
	traces, err := provide.EVMTraceTx(rpcClientKey, rpcURL, &hash)
	if err != nil {
		common.Log.Warningf("failed to fetch tx traces; %s", err.Error())
		return nil, err
	}
	return traces.(*provide.EthereumTxTraceResponse), nil
}

func evmResolveTokenContract(
	rpcClientKey, rpcURL string,
	artifact *provide.CompiledArtifact,
	contractAddress,
	signerAddress string,
) (*string, *big.Int, *string, error) {
	if artifact.ABI != nil {
		return nil, nil, nil, errors.New("given artifact does not contain ABI")
	}

	contractAddr := ethcommon.HexToAddress(contractAddress)
	client, err := provide.EVMDialJsonRpc(rpcClientKey, rpcURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize eth client; %s", err.Error())
	}

	abistr, err := json.Marshal(artifact.ABI)
	if err != nil {
		common.Log.Warningf("Failed to marshal contract abi to json...  %s", err.Error())
	}
	_abi, err := abi.JSON(strings.NewReader(string(abistr)))
	if err == nil {
		msg := ethereum.CallMsg{
			From:     ethcommon.HexToAddress(signerAddress),
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("name()")),
		}

		result, _ := client.CallContract(context.TODO(), msg, nil)
		var name string
		if method, ok := _abi.Methods["name"]; ok {
			err = method.Outputs.Unpack(&name, result)
			if err != nil {
				common.Log.Warningf("Failed to read contract name from deployed contract %s; %s", contractAddress, err.Error())
			}
		}

		msg = ethereum.CallMsg{
			From:     ethcommon.HexToAddress(signerAddress),
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("decimals()")),
		}
		result, _ = client.CallContract(context.TODO(), msg, nil)
		var decimals *big.Int
		if method, ok := _abi.Methods["decimals"]; ok {
			err = method.Outputs.Unpack(&decimals, result)
			if err != nil {
				common.Log.Warningf("Failed to read contract decimals from deployed contract %s; %s", contractAddress, err.Error())
			}
		}

		msg = ethereum.CallMsg{
			From:     ethcommon.HexToAddress(signerAddress),
			To:       &contractAddr,
			Gas:      0,
			GasPrice: big.NewInt(0),
			Value:    nil,
			Data:     ethcommon.FromHex(provide.EVMHashFunctionSelector("symbol()")),
		}
		result, _ = client.CallContract(context.TODO(), msg, nil)
		var symbol string
		if method, ok := _abi.Methods["symbol"]; ok {
			err = method.Outputs.Unpack(&symbol, result)
			if err != nil {
				common.Log.Warningf("Failed to read contract symbol from deployed contract %s; %s", contractAddress, err.Error())
			}
		}

		if name != "" && decimals != nil && symbol != "" { // isERC20Token
			return common.StringOrNil(name), decimals, common.StringOrNil(symbol), nil
		}
	}

	return nil, nil, nil, fmt.Errorf("failed to resolve token contract at contract address: %s", contractAddress)
}
