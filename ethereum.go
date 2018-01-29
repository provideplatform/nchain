package main

import (
	"context"
	"fmt"
	"strings"

	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var EthereumClients = map[string][]*ethclient.Client{}

func DialJsonRpc(network *Network) (*ethclient.Client, error) {
	var url string
	config := network.ParseConfig()
	if jsonRpcUrl, ok := config["json_rpc_url"].(string); ok {
		url = jsonRpcUrl
	} else {
		Log.Warningf("No JSON-RPC url was configured for network: %s (%s)", *network.Name, network.ID)
		url = DefaultEthereumMainnetJsonRpcUrl
	}

	var client *ethclient.Client

	if networkClients, _ := EthereumClients[network.ID.String()]; len(networkClients) == 0 {
		ec, err := ethclient.Dial(url)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = ec
		EthereumClients[network.ID.String()] = append(networkClients, client)
		Log.Debugf("Dialed %s JSON-RPC host @ %s", *network.Name, url)
	} else {
		client = EthereumClients[network.ID.String()][0]
	}

	progress, err := client.SyncProgress(context.TODO())
	if err != nil {
		Log.Warningf("Failed to read sync progress for %s JSON-RPC host: %s; %s", *network.Name, url, err.Error())
		return nil, err
	} else if progress != nil {
		Log.Debugf("Latest synced block for %s JSON-RPC host @ %s: %v [of %v]", *network.Name, url, progress.CurrentBlock, progress.HighestBlock)
	}

	return client, nil
}

func GetChainConfig(network *Network) *params.ChainConfig {
	config := network.ParseConfig()
	if testnet, ok := config["testnet"].(string); ok {
		if strings.ToLower(testnet) == "ropsten" {
			return params.TestnetChainConfig
		} else if strings.ToLower(testnet) == "rinkeby" {
			return params.RinkebyChainConfig
		}
	}
	return params.MainnetChainConfig
}

func JsonRpcClient(network *Network) *ethclient.Client {
	if networkClients, ok := EthereumClients[network.ID.String()]; ok {
		if len(networkClients) > 0 {
			return networkClients[0] // FIXME
		}
	}
	return nil
}

func EncodeFunctionSignature(funcsig string) []byte {
	return ethcrypto.Keccak256([]byte(funcsig))[0:4]
}

func EncodeABI(abi *ethabi.ABI, method string, params ...interface{}) ([]byte, error) {
	var methodDescriptor = fmt.Sprintf("method %s", method)
	Log.Debugf("Attempting to encode %d parameters prior to executing contract %s", len(params), methodDescriptor)
	invocationSig, err := abi.Pack(method, params...)
	return invocationSig, err
}
