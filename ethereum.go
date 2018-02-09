package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"strings"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// EthereumTxTraceResponse is returned upon successful contract execution
type EthereumTxTraceResponse []struct {
	Action struct {
		CallType string `json:"callType"`
		From     string `json:"from"`
		Gas      string `json:"gas"`
		Input    string `json:"input"`
		To       string `json:"to"`
		Value    string `json:"value"`
	} `json:"action"`
	BlockHash   string `json:"blockHash"`
	BlockNumber int    `json:"blockNumber"`
	Result      struct {
		GasUsed string `json:"gasUsed"`
		Output  string `json:"output"`
	} `json:"result"`
	Subtraces           int           `json:"subtraces"`
	TraceAddress        []interface{} `json:"traceAddress"`
	TransactionHash     string        `json:"transactionHash"`
	TransactionPosition int           `json:"transactionPosition"`
	Type                string        `json:"type"`
}

type ParityJsonRpcResponse struct {
	ID     uint64  `json:"id"`
	Result *string `json:"result"`
}

var EthereumGethClients = map[string][]*ethclient.Client{}
var EthereumGethRpcClients = map[string][]*ethrpc.Client{}

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

func GetParityJsonRpcUrl(network *Network) *string {
	var url string
	config := network.ParseConfig()
	if jsonRpcUrl, ok := config["parity_json_rpc_url"].(string); ok {
		url = jsonRpcUrl
	} else {
		Log.Warningf("No parity JSON-RPC url was configured for network: %s (%s)", *network.Name, network.ID)
		url = DefaultEthereumMainnetJsonRpcUrl
	}
	return stringOrNil(url)
}

func InvokeParityJsonRpcClient(network *Network, method string, params []interface{}, response interface{}) error {
	url := GetParityJsonRpcUrl(network)
	if url == nil {
		return fmt.Errorf("No parity JSON-RPC url was configured for network: %s (%s)", *network.Name, network.ID)
	}
	client := &http.Client{
		Transport: &http.Transport{}, // FIXME-- support self-signed certs here
		Timeout:   time.Second * 10,
	}
	payload := map[string]interface{}{
		"method":  method,
		"params":  params,
		"id":      GetChainConfig(network).ChainId,
		"jsonrpc": "2.0",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		Log.Warningf("Failed to marshal JSON payload for %s parity JSON-RPC invocation; network: %s; %s", method, *network.Name, err.Error())
		return err
	}
	resp, err := client.Post(*url, "application/json", bytes.NewReader(body))
	if err != nil {
		Log.Warningf("Failed to invoke %s parity JSON-RPC method: %s; %s", *network.Name, method, err.Error())
		return err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	err = json.Unmarshal(buf.Bytes(), response)
	if err != nil {
		Log.Warningf("Failed to unmarshal %s parity JSON-RPC response; network: %s; %s", method, *network.Name, err.Error())
		return err
	}
	Log.Debugf("Parity invocation of %s JSON-RPC method %s succeeded (%v-byte response)", *network.Name, method, buf.Len())
	return nil
}

func DialJsonRpc(network *Network) (*ethclient.Client, error) {
	url := GetJsonRpcUrl(network)
	var client *ethclient.Client

	if networkClients, _ := EthereumGethClients[network.ID.String()]; len(networkClients) == 0 {
		rpcClient, err := ResolveJsonRpcClient(network)
		if err != nil {
			Log.Warningf("Failed to dial %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = ethclient.NewClient(rpcClient)
		EthereumGethClients[network.ID.String()] = append(networkClients, client)
		Log.Debugf("Dialed %s JSON-RPC host @ %s", *network.Name, *url)
	} else {
		client = EthereumGethClients[network.ID.String()][0]
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

	if rpcClients, _ := EthereumGethRpcClients[network.ID.String()]; len(rpcClients) == 0 {
		erpc, err := ethrpc.Dial(*url)
		if err != nil {
			Log.Warningf("Failed to resolve cached RPC client for %s JSON-RPC host: %s", *network.Name, url)
			return nil, err
		}
		client = erpc
		EthereumGethRpcClients[network.ID.String()] = append(rpcClients, client)
		Log.Debugf("Dialed %s JSON-RPC host @ %s", *network.Name, *url)
	} else {
		client = EthereumGethRpcClients[network.ID.String()][0]
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
	} else if networkID, ok := config["network_id"].(int64); ok {
		return &params.ChainConfig{
			ChainId: big.NewInt(networkID),
		}
	}
	return params.MainnetChainConfig
}

func JsonRpcClient(network *Network) *ethclient.Client {
	if networkClients, ok := EthereumGethClients[network.ID.String()]; ok {
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
	params := make([]interface{}, 0)
	params = append(params, hash)
	var result = &EthereumTxTraceResponse{}
	err := InvokeParityJsonRpcClient(network, "trace_replayTransaction", append(params, []string{"vmTrace"}), &result)
	if err != nil {
		Log.Warningf("Failed to invoke trace_replayTransaction method via JSON-RPC; %s", err.Error())
		return nil, err
	}
	return result, nil
}

// broadcastEthereumTx injects a signed transaction into the pending pool for execution.
//
// If the transaction was a contract creation use the TransactionReceipt method to get the
// contract address after the transaction has been mined.
func broadcastEthereumTx(ctx context.Context, network *Network, tx *types.Transaction, client *ethclient.Client, result interface{}) error {
	rpcClient, err := ResolveJsonRpcClient(network)
	if err != nil {
		return err
	}

	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}

	return rpcClient.CallContext(ctx, result, "eth_sendRawTransaction", common.ToHex(data))
}

func (n *Network) ethereumNetworkLatestBlock() (uint64, error) {
	status, err := n.ethereumNetworkStatus()
	if err != nil {
		return 0, err
	}
	return *status.Block, nil
}

func (n *Network) ethereumNetworkStatus() (*NetworkStatus, error) {
	client, err := DialJsonRpc(n)
	if err != nil {
		Log.Warningf("Failed to dial %s JSON-RPC host; %s", *n.Name, err.Error())
		return nil, err
	}

	syncProgress, err := client.SyncProgress(context.TODO())
	if err != nil {
		Log.Warningf("Failed to read %s sync progress using JSON-RPC host; %s", *n.Name, err.Error())
		return nil, err
	}
	var state string
	var block *uint64  // current block; will be less than height while syncing in progress
	var height *uint64 // total number of blocks
	var syncing = false
	if syncProgress == nil {
		hdr, err := client.HeaderByNumber(context.TODO(), nil)
		if err != nil && hdr == nil {
			Log.Warningf("Failed to read latest block header for %s using JSON-RPC host; %s", *n.Name, err.Error())
			var parityResponse = &ParityJsonRpcResponse{}
			err = InvokeParityJsonRpcClient(n, "eth_getBlockByNumber", []interface{}{}, &parityResponse)
			if err != nil {
				Log.Warningf("Failed to read latest block header for %s using JSON-RPC host; %s", *n.Name, err.Error())
				return nil, err // TODO: make this internal error type indicating failure
			}
			if parityResponse.Result != nil {
				Log.Debugf("Got parity fallback response; %s", *parityResponse.Result)
			}
		}
		hdrUint64 := hdr.Number.Uint64()
		block = &hdrUint64
	} else {
		block = &syncProgress.CurrentBlock
		height = &syncProgress.HighestBlock
		syncing = true
	}
	return &NetworkStatus{
		Block:   block,
		Height:  height,
		State:   stringOrNil(state),
		Syncing: syncing,
		Meta:    map[string]interface{}{},
	}, nil
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

func (c *Contract) executeEthereumContract(network *Network, tx *Transaction, method string, params []interface{}) (*interface{}, error) { // given tx has been built but broadcast has not yet been attempted
	var err error
	_abi, err := c.readEthereumContractAbi()
	if err != nil {
		err := fmt.Errorf("Failed to execute contract method %s on contract: %s; no ABI resolved: %s", method, c.ID, err.Error())
		return nil, err
	}
	var methodDescriptor = fmt.Sprintf("method %s", method)
	var abiMethod *abi.Method
	if mthd, ok := _abi.Methods[method]; ok {
		abiMethod = &mthd
	} else if method == "" {
		abiMethod = &_abi.Constructor
		methodDescriptor = "constructor"
	}
	if abiMethod != nil {
		Log.Debugf("Attempting to encode %d parameters prior to executing contract %s on contract: %s", len(params), methodDescriptor, c.ID)
		invocationSig, err := EncodeABI(abiMethod, params...)
		if err != nil {
			Log.Warningf("Failed to encode %d parameters prior to attempting execution of contract %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
			return nil, err
		}

		data := common.Bytes2Hex(invocationSig)
		tx.Data = &data

		if abiMethod.Const {
			Log.Debugf("Attempting to execute constant method %s on contract: %s", method, c.ID)
			network, _ := tx.GetNetwork()
			client, err := DialJsonRpc(network)
			gasPrice, _ := client.SuggestGasPrice(context.TODO())
			msg := tx.asEthereumCallMsg(gasPrice.Uint64(), 0)
			result, _ := client.CallContract(context.TODO(), msg, nil)
			var out interface{}
			err = abiMethod.Outputs.Unpack(&out, result)
			if err != nil {
				err = fmt.Errorf("Failed to execute constant %s on contract: %s (signature with encoded parameters: %s)", methodDescriptor, c.ID, *tx.Data)
				Log.Warning(err.Error())
				return nil, err
			}
			return &out, nil
		}
		if tx.Create() {
			Log.Debugf("Executed contract %s on contract: %s", methodDescriptor, c.ID)

			if tx.Response != nil {
				Log.Debugf("Received response to tx broadcast attempt calling contract %s on contract: %s", methodDescriptor, c.ID)

				var out interface{}
				switch (tx.Response.Result).(type) {
				case []byte:
					out = (tx.Response.Result).([]byte)
					Log.Debugf("Received response: %s", out)
				case types.Receipt:
					client, _ := DialJsonRpc(network)
					receipt := tx.Response.Result.(*types.Receipt)
					txdeets, _, err := client.TransactionByHash(context.TODO(), receipt.TxHash)
					if err != nil {
						err = fmt.Errorf("Failed to retrieve %s transaction by tx hash: %s", *network.Name, *tx.Hash)
						Log.Warning(err.Error())
						return nil, err
					}
					out = txdeets
				default:
					// no-op
				}
				return &out, nil
			}
		} else {
			err = fmt.Errorf("Failed to execute contract %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", methodDescriptor, c.ID, *tx.Data)
			Log.Warning(err.Error())
		}
	} else {
		err = fmt.Errorf("Failed to execute contract %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
	}
	return nil, err
}

func (c *Contract) readEthereumContractAbi() (*abi.ABI, error) {
	var _abi *abi.ABI
	params := c.ParseParams()
	if contractAbi, ok := params["abi"]; ok {
		abistr, err := json.Marshal(contractAbi)
		if err != nil {
			Log.Warningf("Failed to marshal ABI from contract params to json; %s", err.Error())
			return nil, err
		}

		abival, err := abi.JSON(strings.NewReader(string(abistr)))
		if err != nil {
			Log.Warningf("Failed to initialize ABI from contract  params to json; %s", err.Error())
			return nil, err
		}

		_abi = &abival
	} else {
		return nil, fmt.Errorf("Failed to read ABI from params for contract: %s", c.ID)
	}
	return _abi, nil
}

func (t *Token) readEthereumContractAbi() (*abi.ABI, error) {
	contract, err := t.GetContract()
	if err != nil {
		return nil, err
	}
	return contract.readEthereumContractAbi()
}

func (t *Transaction) asEthereumCallMsg(gasPrice, gasLimit uint64) ethereum.CallMsg {
	db := DatabaseConnection()
	var wallet = &Wallet{}
	db.Model(t).Related(&wallet)
	var to *common.Address
	var data []byte
	if t.To != nil {
		addr := common.HexToAddress(*t.To)
		to = &addr
	}
	if t.Data != nil {
		data = common.FromHex(*t.Data)
	}
	return ethereum.CallMsg{
		From:     common.HexToAddress(wallet.Address),
		To:       to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    big.NewInt(int64(t.Value)),
		Data:     data,
	}
}

func (t *Transaction) fetchEthereumTxReceipt(network *Network, wallet *Wallet) (*types.Receipt, error) {
	var err error
	var receipt *types.Receipt
	client, err := DialJsonRpc(network)
	gasPrice, _ := client.SuggestGasPrice(context.TODO())
	txHash := fmt.Sprintf("0x%s", *t.Hash)
	Log.Debugf("Attempting to retrieve %s tx receipt for broadcast tx: %s", *network.Name, txHash)
	err = ethereum.NotFound
	for receipt == nil && err == ethereum.NotFound {
		Log.Debugf("Retrieving tx receipt for %s contract creation tx: %s", *network.Name, txHash)
		receipt, err = client.TransactionReceipt(context.TODO(), common.HexToHash(txHash))
		if err != nil && err == ethereum.NotFound {
			Log.Warningf("%s contract created by broadcast tx: %s; address must be retrieved from tx receipt", *network.Name, txHash)
		} else {
			Log.Debugf("Retrieved tx receipt for %s contract creation tx: %s; deployed contract address: %s", *network.Name, txHash, receipt.ContractAddress.Hex())
			params := t.ParseParams()
			contractName := fmt.Sprintf("Contract %s", *stringOrNil(receipt.ContractAddress.Hex()))
			if name, ok := params["name"].(string); ok {
				contractName = name
			}
			contract := &Contract{
				ApplicationID: t.ApplicationID,
				NetworkID:     t.NetworkID,
				TransactionID: &t.ID,
				Name:          stringOrNil(contractName),
				Address:       stringOrNil(receipt.ContractAddress.Hex()),
				Params:        t.Params,
			}
			if contract.Create() {
				Log.Debugf("Created contract %s for %s contract creation tx: %s", contract.ID, *network.Name, txHash)

				if contractAbi, ok := params["abi"]; ok {
					abistr, err := json.Marshal(contractAbi)
					if err != nil {
						Log.Warningf("failed to marshal abi to json...  %s", err.Error())
					}
					_abi, err := abi.JSON(strings.NewReader(string(abistr)))
					if err == nil {
						msg := ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: gasPrice,
							Value:    nil,
							Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("name()"))),
						}

						result, _ := client.CallContract(context.TODO(), msg, nil)
						var name string
						if method, ok := _abi.Methods["name"]; ok {
							err = method.Outputs.Unpack(&name, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract name from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: gasPrice,
							Value:    nil,
							Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("decimals()"))),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var decimals *big.Int
						if method, ok := _abi.Methods["decimals"]; ok {
							err = method.Outputs.Unpack(&decimals, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract decimals from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
							}
						}

						msg = ethereum.CallMsg{
							From:     common.HexToAddress(wallet.Address),
							To:       &receipt.ContractAddress,
							Gas:      0,
							GasPrice: gasPrice,
							Value:    nil,
							Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("symbol()"))),
						}
						result, _ = client.CallContract(context.TODO(), msg, nil)
						var symbol string
						if method, ok := _abi.Methods["symbol"]; ok {
							err = method.Outputs.Unpack(&symbol, result)
							if err != nil {
								Log.Warningf("Failed to read %s, contract symbol from deployed contract %s; %s", *network.Name, contract.ID, err.Error())
							}
						}

						if name != "" && decimals != nil && symbol != "" { // isERC20Token
							Log.Debugf("Resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)
							token := &Token{
								ApplicationID: contract.ApplicationID,
								NetworkID:     contract.NetworkID,
								ContractID:    &contract.ID,
								Name:          stringOrNil(name),
								Symbol:        stringOrNil(symbol),
								Decimals:      decimals.Uint64(),
								Address:       stringOrNil(receipt.ContractAddress.Hex()),
							}
							if token.Create() {
								Log.Debugf("Created token %s for associated %s contract creation tx: %s", token.ID, *network.Name, txHash)
							} else {
								Log.Warningf("Failed to create token for associated %s contract creation tx %s; %d errs: %s", *network.Name, txHash, len(token.Errors), *stringOrNil(*token.Errors[0].Message))
							}
						}
					} else {
						Log.Warningf("Failed to parse JSON ABI for %s contract; %s", *network.Name, err.Error())
					}
				}
			} else {
				Log.Warningf("Failed to create contract for %s contract creation tx %s", *network.Name, txHash)
			}
		}
	}
	return receipt, err
}

func signAndBroadcastEthereumTx(t *Transaction, network *Network, wallet *Wallet) (response *ContractExecutionResponse, err error) {
	client, err := DialJsonRpc(network)
	if err != nil {
		Log.Warningf("Failed to dial %s JSON-RPC host; %s", *network.Name, err.Error())
	} else {
		cfg := GetChainConfig(network)
		tx, err := t.signEthereumTx(network, wallet, cfg)
		if err == nil {
			Log.Debugf("Transmitting signed %s tx to JSON-RPC host", *network.Name)
			var out interface{}
			err := broadcastEthereumTx(context.TODO(), network, tx, client, &out)
			if err != nil {
				Log.Warningf("Failed to transmit signed %s tx to JSON-RPC host; %s", *network.Name, err.Error())
			} else if out == nil { // FIXME-- out pointer not yet handled by broadcastEthereumTx
				receipt, err := t.fetchEthereumTxReceipt(network, wallet)
				if err != nil {
					Log.Warningf("Failed to fetch ethereum tx receipt with tx hash: %s; %s", *t.Hash, err.Error())
				} else {
					Log.Debugf("Fetched ethereum tx receipt with tx hash: %s; receipt: %s", *t.Hash, receipt)
					response = &ContractExecutionResponse{
						Result:      receipt,
						Transaction: t,
					}
				}
			} else {
				trace, err := TraceTx(network, stringOrNil(out.(string)))
				if err == nil && trace != nil {
					response = &ContractExecutionResponse{
						Result:      trace,
						Transaction: t,
					}
				} else if trace == nil {
					response = &ContractExecutionResponse{
						Result:      out,
						Transaction: t,
					}
				}
			}
		} else {
			err = fmt.Errorf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
		}
	}
	return response, err
}

func (t *Transaction) signEthereumTx(network *Network, wallet *Wallet, cfg *params.ChainConfig) (*types.Transaction, error) {
	client := JsonRpcClient(network)
	syncProgress, err := client.SyncProgress(context.TODO())
	if err == nil {
		blockNumber, err := network.ethereumNetworkLatestBlock()
		if err != nil {
			return nil, err
		}
		nonce, err := client.PendingNonceAt(context.TODO(), common.HexToAddress(wallet.Address))
		if err != nil {
			return nil, err
		}
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		var data []byte
		if t.Data != nil {
			data = common.FromHex(*t.Data)
		}
		var tx *types.Transaction
		if t.To != nil {
			addr := common.HexToAddress(*t.To)
			callMsg := t.asEthereumCallMsg(gasPrice.Uint64(), 0)
			gasLimit, err := client.EstimateGas(context.TODO(), callMsg)
			if err != nil {
				Log.Warningf("Failed to estimate gas for %s tx; %s", *network.Name, err.Error())
				return nil, err
			}
			Log.Debugf("Estimated %d total gas required for %s tx with %d-byte data payload", gasLimit, *network.Name, len(data))
			tx = types.NewTransaction(nonce, addr, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		} else {
			Log.Debugf("Attempting to deploy %s contract via tx; estimating total gas requirements", *network.Name)
			callMsg := t.asEthereumCallMsg(gasPrice.Uint64(), 0)
			gasLimit, err := client.EstimateGas(context.TODO(), callMsg)
			if err != nil {
				Log.Warningf("Failed to estimate gas for %s tx; %s", *network.Name, err.Error())
				return nil, err
			}
			Log.Debugf("Estimated %d total gas required for %s contract deployment tx with %d-byte data payload", gasLimit, *network.Name, len(data))
			tx = types.NewContractCreation(nonce, big.NewInt(int64(t.Value)), gasLimit, gasPrice, data)
		}
		signer := types.MakeSigner(cfg, big.NewInt(int64(blockNumber)))
		hash := signer.Hash(tx).Bytes()
		privateKey, err := decryptECDSAPrivateKey(*wallet.PrivateKey, GpgPrivateKey, WalletEncryptionKey)
		if err != nil {
			Log.Warningf("Failed to sign %s tx using wallet: %s", *network.Name, wallet.ID)
			return nil, err
		}
		Log.Debugf("Signing %s tx using wallet: %s", *network.Name, wallet.ID)
		sig, err := ethcrypto.Sign(hash, privateKey)
		if err != nil {
			Log.Warningf("Failed to sign %s tx using wallet: %s; %s", *network.Name, wallet.ID, err.Error())
			return nil, err
		}
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			t.Hash = stringOrNil(fmt.Sprintf("%x", signedTx.Hash()))
			Log.Debugf("Signed %s tx for broadcast via JSON-RPC: %s", *network.Name, signedTx.String())
			return signedTx, nil
		}
		return nil, err
	} else if syncProgress == nil {
		Log.Debugf("%s JSON-RPC is in sync with the network", *network.Name)
	}
	return nil, err
}

// isEthereumNetwork returns true if the network is a public or private blockchain network based on the Ethereum protocol
func (n *Network) isEthereumNetwork() bool {
	if strings.HasPrefix(strings.ToLower(*n.Name), "eth") {
		return true
	}

	cfg := n.ParseConfig()
	if cfg != nil {
		if isEthereumNetwork, ok := cfg["is_ethereum_network"]; ok {
			if _ok := isEthereumNetwork.(bool); _ok {
				return isEthereumNetwork.(bool)
			}
		}
		if _, ok := cfg["parity_json_rpc_url"]; ok {
			return true
		}
	}
	return false
}

// TokenBalance
// Retrieve a wallet's token balance for a given token id
func ethereumTokenBalance(network *Network, token *Token, addr string) (uint64, error) {
	balance := uint64(0)
	abi, err := token.readEthereumContractAbi()
	if err != nil {
		return 0, err
	}
	client, err := DialJsonRpc(network)
	gasPrice, _ := client.SuggestGasPrice(context.TODO())
	to := common.HexToAddress(*token.Address)
	msg := ethereum.CallMsg{
		From:     common.HexToAddress(addr),
		To:       &to,
		Gas:      0,
		GasPrice: gasPrice,
		Value:    nil,
		Data:     common.FromHex(common.Bytes2Hex(EncodeFunctionSignature("balanceOf(address)"))),
	}
	result, _ := client.CallContract(context.TODO(), msg, nil)
	var out *big.Int
	if method, ok := abi.Methods["balanceOf"]; ok {
		method.Outputs.Unpack(&out, result)
		if out != nil {
			balance = out.Uint64()
			Log.Debugf("Read %s %s token balance (%v) from token contract address: %s", *network.Name, token.Symbol, balance, token.Address)
		}
	} else {
		Log.Warningf("Unable to read balance of unsupported %s token contract address: %s", *network.Name, token.Address)
	}
	return balance, nil
}

// decryptECDSAPrivateKey - read the wallet-specific ECDSA private key; required for signing transactions on behalf of the wallet
func decryptECDSAPrivateKey(encryptedKey, gpgPrivateKey, gpgEncryptionKey string) (*ecdsa.PrivateKey, error) {
	results := make([]byte, 1)
	db := DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as private_key", encryptedKey, gpgPrivateKey, gpgEncryptionKey).Rows()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&results)
		privateKeyBytes, err := hex.DecodeString(string(results))
		if err != nil {
			Log.Warningf("Failed to read ecdsa private key from encrypted storage; %s", err.Error())
			return nil, err
		}
		return ethcrypto.ToECDSA(privateKeyBytes)
	}
	return nil, errors.New("Failed to decode ecdsa private key after retrieval from encrypted storage")
}

// generateEthereumKeyPair - generate an Ethereum address and private key
func generateEthereumKeyPair() (address, encodedPrivateKey *string, err error) {
	privateKey, err := ethcrypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	address = stringOrNil(ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	encodedPrivateKey = stringOrNil(hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
	return address, encodedPrivateKey, nil
}
