package main

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var EthereumClients = map[string][]*ethclient.Client{}
var EthereumRpcClients = map[string][]*ethrpc.Client{}

func GetJsonRpcUrl(network *Network) *string {
	var url string
	config := network.ParseConfig()
	if jsonRpcUrl, ok := config["json_rpc_url"].(string); ok {
		url = jsonRpcUrl
	} else {
		Log.Warningf("No JSON-RPC url was configured for network: %s (%s)", *network.Name, network.ID)
		url = DefaultEthereumMainnetJsonRpcUrl
	}
	return stringOrNil(url)
}
func DialJsonRpc(network *Network) (*ethclient.Client, error) {
	url := GetJsonRpcUrl(network)
	var client *ethclient.Client

	if networkClients, _ := EthereumClients[network.ID.String()]; len(networkClients) == 0 {
		rpcClient, err := ResolveJsonRpcClient(network)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = ethclient.NewClient(rpcClient)
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

func ResolveJsonRpcClient(network *Network) (*ethrpc.Client, error) {
	url := GetJsonRpcUrl(network)
	var client *ethrpc.Client

	if rpcClients, _ := EthereumRpcClients[network.ID.String()]; len(rpcClients) == 0 {
		erpc, err := ethrpc.Dial(*url)
		if err != nil {
			Log.Warningf("Failed to resolve cached RPC client for %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = erpc
		EthereumRpcClients[network.ID.String()] = append(rpcClients, client)
		Log.Debugf("Dialed %s JSON-RPC host @ %s", *network.Name, url)
	} else {
		client = EthereumRpcClients[network.ID.String()][0]
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

func EncodeABI(method *abi.Method, params ...interface{}) ([]byte, error) {
	var methodDescriptor = fmt.Sprintf("method %s", method.Name)
	Log.Debugf("Attempting to encode %d parameters prior to executing contract %s", len(params), methodDescriptor)
	var args []interface{}

	for i := range params {
		input := method.Inputs[i]
		param, err := coerceAbiParameter(input.Type, params[i])
		if err != nil {
			Log.Warningf("Failed to encode ABI parameter %s in accordance with contract %s; %s", input.Name, methodDescriptor, err.Error())
		} else {
			args = append(args, param)
		}
	}
	encodedArgs, err := method.Inputs.Pack(args...)
	if err != nil {
		Log.Warningf("Failed to encode %d parameters prior to attempting execution of contract %s on contract; %s", len(params), methodDescriptor, err.Error())
		return nil, err
	}
	Log.Debugf("Encoded abi params: %s", encodedArgs)
	return append(method.Id(), encodedArgs...), nil
}

func TraceTx(network *Network, hash *string) (interface{}, error) {
	Log.Warningf("TraceTx not implemented; unable to trace %s tx: %s", *network.Name, hash)
	return nil, nil
}

func coerceAbiParameter(t abi.Type, v interface{}) (interface{}, error) {
	switch t.T {
	case abi.IntTy, abi.UintTy:
		switch kind := t.Kind; kind {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return big.NewInt(int64(v.(int64))), nil
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return big.NewInt(int64(v.(int64))), nil
		case reflect.Float64:
			return big.NewInt(int64(v.(float64))), nil
		case reflect.Ptr:
			switch v.(type) {
			case float64:
				return big.NewInt(int64(v.(float64))), nil
			}
		default:
			return nil, fmt.Errorf("Failed to coerce %s (%s) parameter for ABI encoding", t.String(), kind.String())
		}
	case abi.StringTy:
		return v.(string), nil
	case abi.AddressTy:
		// FIXME... ensure handling of arrays
		return common.HexToAddress(v.(string)), nil
	case abi.BoolTy:
		return v.(bool), nil
	case abi.BytesTy:
		// FIXME... ensure handling of arrays
		return v.([]byte), nil
	case abi.FixedBytesTy, abi.FunctionTy:
		// FIXME... ensure handling of arrays
		return v.([]byte), nil
	default:
		return nil, fmt.Errorf("Failed to coerce %s parameter for ABI encoding", t.String())
	}
	return nil, fmt.Errorf("Failed to coerce %s parameter for ABI encoding", t.String())
}
