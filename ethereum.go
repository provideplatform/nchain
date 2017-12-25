package main

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
)

var EthereumClients = map[string][]*ethclient.Client{}

func DialJsonRpc(network *Network) (*ethclient.Client, error) {
	var url string
	if jsonRpcUrl, ok := network.Config["json_rpc_url"].(string); ok {
		url = jsonRpcUrl
	} else {
		Log.Warningf("No JSON-RPC url was configured for network: %s (%s)", *network.Name, network.Id)
		url = DefaultEthereumJsonRpcUrl
	}

	var client *ethclient.Client

	if networkClients, _ := EthereumClients[network.Id.String()]; len(networkClients) == 0 {
		ec, err := ethclient.Dial(url)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = ec
		EthereumClients[network.Id.String()] = append(networkClients, client)
		Log.Debugf("Dialed %s JSON-RPC host @ %s", *network.Name, url)
	} else {
		client = EthereumClients[network.Id.String()][0]
	}

	progress, _ := client.SyncProgress(context.TODO())
	Log.Debugf("Latest synced block for %s JSON-RPC host @ %s: %v [of %v]", *network.Name, url, progress.CurrentBlock, progress.HighestBlock)

	return client, nil
}

func JsonRpcClient(network *Network) *ethclient.Client {
	if networkClients, ok := EthereumClients[network.Id.String()]; ok {
		if len(networkClients) > 0 {
			return networkClients[0] // FIXME
		}
	}
	return nil
}
