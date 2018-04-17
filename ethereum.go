package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/kthomas/go.uuid"

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
type EthereumTxTraceResponse struct {
	Result []struct {
		Action struct {
			CallType string `json:"callType"`
			From     string `json:"from"`
			Gas      string `json:"gas"`
			Init     string `json:"init"`
			Input    string `json:"input"`
			To       string `json:"to"`
			Value    string `json:"value"`
		} `json:"action"`
		BlockHash   string `json:"blockHash"`
		BlockNumber int    `json:"blockNumber"`
		Result      struct {
			Address string `json:"address"`
			Code    string `json:"code"`
			GasUsed string `json:"gasUsed"`
			Output  string `json:"output"`
		} `json:"result"`
		Subtraces           int           `json:"subtraces"`
		TraceAddress        []interface{} `json:"traceAddress"`
		TransactionHash     string        `json:"transactionHash"`
		TransactionPosition int           `json:"transactionPosition"`
		Type                string        `json:"type"`
	} `json:"result"`
}

type ParityJsonRpcErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
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
		return fmt.Errorf("Failed to unmarshal %s parity JSON-RPC response: %s; network: %s; %s", method, buf.Bytes(), *network.Name, err.Error())
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
	Log.Debugf("Attempting to encode %d parameters prior to executing contract method: %s", len(params), methodDescriptor)
	var args []interface{}

	for i := range params {
		input := method.Inputs[i]
		Log.Debugf("Attempting to coerce encoding of %v abi parameter; value: %s", input.Type, params[i])
		param, err := coerceAbiParameter(input.Type, params[i])
		if err != nil {
			Log.Warningf("Failed to encode abi parameter %s in accordance with contract %s; %s", input.Name, methodDescriptor, err.Error())
		} else {
			switch reflect.TypeOf(param).Kind() {
			case reflect.String:
				param = []byte(param.(string))
			default:
				// no-op
			}

			args = append(args, param)
			Log.Debugf("Coerced encoding of %v abi parameter; value: %s", input.Type, param)
		}
	}

	encodedArgs, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	Log.Debugf("Encoded %v abi params prior to executing contract method: %s; abi-encoded arguments %v bytes packed", len(params), methodDescriptor, len(encodedArgs))
	return append(method.Id(), encodedArgs...), nil
}

func TraceTx(network *Network, hash *string) (interface{}, error) {
	var addr = *hash
	if !strings.HasPrefix(addr, "0x") {
		addr = fmt.Sprintf("0x%s", addr)
	}
	params := make([]interface{}, 0)
	params = append(params, addr)
	var result = &EthereumTxTraceResponse{}
	Log.Debugf("Attempting to trace %s tx via trace_transaction method via JSON-RPC; tx hash: %s", *network.Name, addr)
	err := InvokeParityJsonRpcClient(network, "trace_transaction", params, &result)
	if err != nil {
		Log.Warningf("Failed to invoke trace_transaction method via JSON-RPC; %s", err.Error())
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
	case abi.ArrayTy, abi.SliceTy:
		switch v.(type) {
		case []byte:
			return forEachUnpack(t, v.([]byte), 0, len(v.([]interface{}))-1)
		case string:
			return forEachUnpack(t, []byte(v.(string)), 0, len(v.(string)))
		default:
			// HACK-- this fallback for edge case handling isn't the cleanest
			typestr := fmt.Sprintf("%s", t)
			if typestr == "uint256[]" {
				Log.Debugf("Attempting fallback coercion of uint256[] abi parameter")
				vals := make([]*big.Int, t.Size)
				for _, val := range v.([]interface{}) {
					vals = append(vals, big.NewInt(int64(val.(float64))))
				}
				return vals, nil
			}
		}
	case abi.StringTy: // variable arrays are written at the end of the return bytes
		return string(v.([]byte)), nil
	case abi.IntTy, abi.UintTy:
		switch t.Kind {
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
			return readInteger(t.Kind, v.([]byte)), nil
		}
	case abi.BoolTy:
		return v.(bool), nil
	case abi.AddressTy:
		switch v.(type) {
		case string:
			return common.HexToAddress(v.(string)), nil
		default:
			return common.BytesToAddress(v.([]byte)), nil
		}
	case abi.HashTy:
		return common.BytesToHash(v.([]byte)), nil
	case abi.BytesTy:
		return v, nil
	case abi.FixedBytesTy:
		return readFixedBytes(t, []byte(v.(string)))
	case abi.FunctionTy:
		return readFunctionType(t, v.([]byte))
	default:
		// no-op
	}
	return nil, fmt.Errorf("Failed to coerce %s parameter for abi encoding; unhandled type: %v", t.String(), t)
}

// iteratively unpack elements
func forEachUnpack(t abi.Type, output []byte, start, size int) (interface{}, error) {
	if size < 0 {
		return nil, fmt.Errorf("cannot marshal input to array, size is negative (%d)", size)
	}
	if start+32*size > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go array: offset %d would go over slice boundary (len=%d)", len(output), start+32*size)
	}

	// this value will become our slice or our array, depending on the type
	var refSlice reflect.Value

	if t.T == abi.SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(t.Type, size, size)
	} else if t.T == abi.ArrayTy {
		// declare our array
		refSlice = reflect.New(t.Type).Elem()
	} else {
		return nil, fmt.Errorf("abi: invalid type in array/slice unpacking stage")
	}

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	elemSize := 32
	if t.T == abi.ArrayTy {
		elemSize = getFullElemSize(t.Elem)
	}

	for i, j := start, 0; j < size; i, j = i+elemSize, j+1 {
		inter, err := coerceAbiParameter(t, output)
		if err != nil {
			return nil, err
		}

		// append the item to our reflect slice
		refSlice.Index(j).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

// reads the integer based on its kind
func readInteger(kind reflect.Kind, b []byte) interface{} {
	switch kind {
	case reflect.Uint8:
		return b[len(b)-1]
	case reflect.Uint16:
		return binary.BigEndian.Uint16(b[len(b)-2:])
	case reflect.Uint32:
		return binary.BigEndian.Uint32(b[len(b)-4:])
	case reflect.Uint64:
		return binary.BigEndian.Uint64(b[len(b)-8:])
	case reflect.Int8:
		return int8(b[len(b)-1])
	case reflect.Int16:
		return int16(binary.BigEndian.Uint16(b[len(b)-2:]))
	case reflect.Int32:
		return int32(binary.BigEndian.Uint32(b[len(b)-4:]))
	case reflect.Int64:
		return int64(binary.BigEndian.Uint64(b[len(b)-8:]))
	default:
		return new(big.Int).SetBytes(b)
	}
}

// A function type is simply the address with the function selection signature at the end.
// This enforces that standard by always presenting it as a 24-array (address + sig = 24 bytes)
func readFunctionType(t abi.Type, word []byte) (funcTy [24]byte, err error) {
	if t.T != abi.FunctionTy {
		return [24]byte{}, fmt.Errorf("abi: invalid type in call to make function type byte array")
	}
	if garbage := binary.BigEndian.Uint64(word[24:32]); garbage != 0 {
		err = fmt.Errorf("abi: got improperly encoded function type, got %v", word)
	} else {
		copy(funcTy[:], word[0:24])
	}
	return
}

// through reflection, creates a fixed array to be read from
func readFixedBytes(t abi.Type, word []byte) (interface{}, error) {
	if t.T != abi.FixedBytesTy {
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array")
	}

	Log.Debugf("Attempting to read fixed bytes in accordance with Ethereum contract ABI; type: %v; word: %s", t, word)

	// convert
	array := reflect.New(t.Type).Elem()
	reflect.Copy(array, reflect.ValueOf(word))
	return array.Interface(), nil
}

func requiresLengthPrefix(t *abi.Type) bool {
	return t.T == abi.StringTy || t.T == abi.BytesTy || t.T == abi.SliceTy
}

func getFullElemSize(elem *abi.Type) int {
	//all other should be counted as 32 (slices have pointers to respective elements)
	size := 32
	//arrays wrap it, each element being the same size
	for elem.T == abi.ArrayTy {
		size *= elem.Size
		elem = elem.Elem
	}
	return size
}

func lengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
	bigOffsetEnd := big.NewInt(0).SetBytes(output[index : index+32])
	bigOffsetEnd.Add(bigOffsetEnd, common.Big32)
	outputLength := big.NewInt(int64(len(output)))

	if bigOffsetEnd.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go slice: offset %v would go over slice boundary (len=%v)", bigOffsetEnd, outputLength)
	}

	if bigOffsetEnd.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi offset larger than int64: %v", bigOffsetEnd)
	}

	offsetEnd := int(bigOffsetEnd.Uint64())
	lengthBig := big.NewInt(0).SetBytes(output[offsetEnd-32 : offsetEnd])

	totalSize := big.NewInt(0)
	totalSize.Add(totalSize, bigOffsetEnd)
	totalSize.Add(totalSize, lengthBig)
	if totalSize.BitLen() > 63 {
		return 0, 0, fmt.Errorf("abi length larger than int64: %v", totalSize)
	}

	if totalSize.Cmp(outputLength) > 0 {
		return 0, 0, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %v require %v", outputLength, totalSize)
	}
	start = int(bigOffsetEnd.Uint64())
	length = int(lengthBig.Uint64())
	return
}

func readBool(word []byte) (bool, error) {
	for _, b := range word[:31] {
		if b != 0 {
			return false, errors.New("abi: improperly encoded boolean value")
		}
	}
	switch word[31] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errors.New("abi: improperly encoded boolean value")
	}
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
		Log.Debugf("Attempting to encode %d parameters [ %s ] prior to executing contract %s on contract: %s", len(params), params, methodDescriptor, c.ID)
		invocationSig, err := EncodeABI(abiMethod, params...)
		if err != nil {
			Log.Warningf("Failed to encode %d parameters prior to attempting execution of contract %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
			return nil, err
		}

		data := common.Bytes2Hex(invocationSig)
		tx.Data = &data

		if abiMethod.Const {
			Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
			network, _ := tx.GetNetwork()
			client, err := DialJsonRpc(network)
			gasPrice, _ := client.SuggestGasPrice(context.TODO())
			msg := tx.asEthereumCallMsg(gasPrice.Uint64(), 0)
			result, _ := client.CallContract(context.TODO(), msg, nil)
			var out interface{}
			err = abiMethod.Outputs.Unpack(&out, result)
			if len(abiMethod.Outputs) == 1 {
				err = abiMethod.Outputs.Unpack(&out, result)
			} else if len(abiMethod.Outputs) > 1 {
				// handle tuple
				vals := make([]interface{}, len(abiMethod.Outputs))
				for i := range abiMethod.Outputs {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[i].Type)
					Log.Debugf("Reflectively adding type hint for unpacking %s in return values slot %v", typestr, i)
					typ, err := abi.NewType(typestr)
					if err != nil {
						err = fmt.Errorf("Failed to reflectively add appropriately-typed %s value for in return values slot %v); %s", typestr, i, err.Error())
						Log.Warning(err.Error())
						return nil, err
					}
					vals[i] = reflect.New(typ.Type).Interface()
				}
				err = abiMethod.Outputs.Unpack(&vals, result)
				out = vals
				Log.Debugf("Unpacked %v returned values from read of constant %s on contract: %s; values: %s", len(vals), methodDescriptor, c.ID, vals)
			}
			if err != nil {
				err = fmt.Errorf("Failed to read constant %s on contract: %s (signature with encoded parameters: %s); %s", methodDescriptor, c.ID, *tx.Data, err.Error())
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
		Value:    t.Value.BigInt(),
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
			Log.Debugf("%s contract created by broadcast tx: %s; address must be retrieved from pending tx receipt", *network.Name, txHash)
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
			tx = types.NewTransaction(nonce, addr, t.Value.BigInt(), gasLimit, gasPrice, data)
		} else {
			Log.Debugf("Attempting to deploy %s contract via tx; estimating total gas requirements", *network.Name)
			callMsg := t.asEthereumCallMsg(gasPrice.Uint64(), 0)
			gasLimit, err := client.EstimateGas(context.TODO(), callMsg)
			if err != nil {
				Log.Warningf("Failed to estimate gas for %s tx; %s", *network.Name, err.Error())
				return nil, err
			}
			Log.Debugf("Estimated %d total gas required for %s contract deployment tx with %d-byte data payload", gasLimit, *network.Name, len(data))
			tx = types.NewContractCreation(nonce, t.Value.BigInt(), gasLimit, gasPrice, data)
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
			Log.Debugf("Signed %s tx for broadcast via JSON-RPC: %s", *network.Name, signedTx
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

// Retrieve a wallet's native currency balance
func ethereumNativeBalance(network *Network, addr string) (*big.Int, error) {
	client, err := DialJsonRpc(network)
	if err != nil {
		return nil, err
	}
	return client.BalanceAt(context.TODO(), common.HexToAddress(addr), nil)
}

// Retrieve a wallet's token balance for a given token id
func ethereumTokenBalance(network *Network, token *Token, addr string) (*big.Int, error) {
	var balance *big.Int
	if token.ID == uuid.Nil {
		return nil, errors.New("Unable to read balance of nil token contract")
	}
	abi, err := token.readEthereumContractAbi()
	if err != nil {
		return nil, err
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
	if method, ok := abi.Methods["balanceOf"]; ok {
		method.Outputs.Unpack(&balance, result)
		if balance != nil {
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
