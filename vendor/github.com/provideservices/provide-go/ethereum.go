package provide

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/scrypt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

// The purpose of this class is to expose generic transactional and ABI-related helper
// methods; ethereum.go is a convenience wrapper around JSON-RPC.

// It also caches JSON-RPC client instances in a few flavors (*ethclient.Client and *ethrpc.Client)
// and maps them to an arbitrary `networkID` after successfully dialing the given RPC URL.

var chainConfigs = map[string]*params.ChainConfig{}        // mapping of network ids to *params.ChainConfig
var ethclientRpcClients = map[string][]*ethclient.Client{} // mapping of network ids to *ethclient.Client instances
var ethrpcClients = map[string][]*ethrpc.Client{}          // mapping of network ids to *ethrpc.Client instances

var evmMutex = &sync.Mutex{}

func evmClearCachedClients(networkID string) {
	evmMutex.Lock()
	delete(chainConfigs, networkID)
	for i := range ethrpcClients[networkID] {
		ethrpcClients[networkID][i].Close()
	}
	for i := range ethclientRpcClients[networkID] {
		ethclientRpcClients[networkID][i].Close()
	}
	ethrpcClients[networkID] = make([]*ethrpc.Client, 0)
	ethclientRpcClients[networkID] = make([]*ethclient.Client, 0)
	evmMutex.Unlock()
}

// EVMDialJsonRpc - dials and caches a new JSON-RPC client instance at the JSON-RPC url and caches it using the given network id
func EVMDialJsonRpc(networkID, rpcURL string) (*ethclient.Client, error) {
	var client *ethclient.Client

	if networkClients, _ := ethclientRpcClients[networkID]; len(networkClients) == 0 {
		rpcClient, err := EVMResolveJsonRpcClient(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		client = ethclient.NewClient(rpcClient)
		evmMutex.Lock()
		ethrpcClients[networkID] = append(ethrpcClients[networkID], rpcClient)
		ethclientRpcClients[networkID] = append(networkClients, client)
		evmMutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethclientRpcClients[networkID][0]
	}

	_, err := EVMGetSyncProgress(client)
	if err != nil {
		evmClearCachedClients(networkID)
		return nil, err
	}

	return client, nil
}

// EVMInvokeJsonRpcClient - invokes the JSON-RPC client for the given network and url
func EVMInvokeJsonRpcClient(networkID, rpcURL, method string, params []interface{}, response interface{}) error {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: time.Second * 60,
	}
	payload := map[string]interface{}{
		"method":  method,
		"params":  params,
		"id":      EVMGetChainID(networkID, rpcURL),
		"jsonrpc": "2.0",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		Log.Warningf("Failed to marshal JSON payload for %s JSON-RPC invocation; %s", method, err.Error())
		return err
	}
	resp, err := client.Post(rpcURL, "application/json", bytes.NewReader(body))
	if err != nil {
		Log.Warningf("Failed to invoke JSON-RPC method: %s; %s", method, err.Error())
		return err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	err = json.Unmarshal(buf.Bytes(), response)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal %s JSON-RPC response: %s; %s", method, buf.Bytes(), err.Error())
	}
	Log.Debugf("Invocation of JSON-RPC method %s succeeded (%v-byte response)", method, buf.Len())
	return nil
}

// EVMResolveEthClient resolves a cached *ethclient.Client client or dials and caches a new instance
func EVMResolveEthClient(networkID, rpcURL string) (*ethclient.Client, error) {
	var client *ethclient.Client
	if networkClients, _ := ethclientRpcClients[networkID]; len(networkClients) == 0 {
		client, err := EVMDialJsonRpc(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial RPC client for JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		evmMutex.Lock()
		ethclientRpcClients[networkID] = append(networkClients, client)
		evmMutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethclientRpcClients[networkID][0]
		Log.Debugf("Resolved cached *ethclient.Client instance for JSON-RPC host @ %s", rpcURL)
	}
	return client, nil
}

// EVMResolveJsonRpcClient resolves a cached *ethclient.Client client or dials and caches a new instance
func EVMResolveJsonRpcClient(networkID, rpcURL string) (*ethrpc.Client, error) {
	var client *ethrpc.Client
	if networkClients, _ := ethrpcClients[networkID]; len(networkClients) == 0 {
		erpc, err := ethrpc.Dial(rpcURL)
		if err != nil {
			Log.Warningf("Failed to dial RPC client for JSON-RPC host: %s", rpcURL)
			return nil, err
		}
		client = erpc
		evmMutex.Lock()
		ethrpcClients[networkID] = append(networkClients, client)
		evmMutex.Unlock()
		Log.Debugf("Dialed JSON-RPC host @ %s", rpcURL)
	} else {
		client = ethrpcClients[networkID][0]
		Log.Debugf("Resolved JSON-RPC host @ %s", rpcURL)
	}
	return client, nil
}

// EVMEncodeABI returns the ABI-encoded calldata for the given method and params
func EVMEncodeABI(method *abi.Method, params ...interface{}) ([]byte, error) {
	var methodDescriptor = fmt.Sprintf("method %s", method.Name)
	defer func() {
		if r := recover(); r != nil {
			Log.Debugf("Failed to encode ABI-compliant calldata for method: %s", methodDescriptor)
		}
	}()

	Log.Debugf("Attempting to encode %d parameters prior to executing contract method: %s", len(params), methodDescriptor)
	var args []interface{}

	for i := range params {
		if i >= len(method.Inputs) {
			break
		}
		input := method.Inputs[i]
		Log.Debugf("Attempting to coerce encoding of %v abi parameter; value: %s", input.Type, params[i])
		param, err := coerceAbiParameter(input.Type, params[i])
		if err != nil {
			Log.Warningf("Failed to encode abi parameter %s in accordance with contract %s; %s", input.Name, methodDescriptor, err.Error())
		} else {
			switch reflect.TypeOf(param).Kind() {
			case reflect.Slice:
				param = []byte(param.(string))
			default:
				// no-op
				Log.Debugf("Unmodified parameter (type: %s); value: %s", input.Type, param)
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

// EVMGenerateKeyPair - creates and returns an ECDSA keypair;
// the returned *ecdsa.PrivateKey can be encoded with: hex.EncodeToString(ethcrypto.FromECDSA(privateKey))
func EVMGenerateKeyPair() (address *string, privateKey *ecdsa.PrivateKey, err error) {
	privateKey, err = ethcrypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}
	address = stringOrNil(ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex())
	return address, privateKey, nil
}

// EVMMarshalKeyPairJSON - returns keystore JSON representation of given private key
func EVMMarshalKeyPairJSON(addr common.Address, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	type keyJSON struct {
		ID         string `json:"id"`
		Address    string `json:"address"`
		PrivateKey string `json:"privatekey"`
		Version    int    `json:"version"`
	}
	keyUUID, _ := generateEVMKeyUUID()
	key := keyJSON{
		ID:         keyUUID,
		Address:    hex.EncodeToString(addr[:]),
		PrivateKey: hex.EncodeToString(ethcrypto.FromECDSA(privateKey)),
		Version:    3,
	}
	return json.Marshal(key)
}

// EVMMarshalEncryptedKey encrypts key as version 3.
func EVMMarshalEncryptedKey(addr common.Address, privateKey *ecdsa.PrivateKey, secret string) ([]byte, error) {
	const (
		// n,r,p = 2^12, 8, 6 uses 4MB memory and approx 100ms CPU time on a modern CPU.
		LightScryptN = 1 << 12
		LightScryptP = 6

		scryptR     = 4
		scryptDKLen = 32
	)

	type cipherparamsJSON struct {
		IV string `json:"iv"`
	}

	type cryptoJSON struct {
		Cipher       string                 `json:"cipher"`
		CipherText   string                 `json:"ciphertext"`
		CipherParams cipherparamsJSON       `json:"cipherparams"`
		KDF          string                 `json:"kdf"`
		KDFParams    map[string]interface{} `json:"kdfparams"`
		MAC          string                 `json:"mac"`
	}

	type web3v3 struct {
		ID      string     `json:"id"`
		Address string     `json:"address"`
		Crypto  cryptoJSON `json:"crypto"`
		Version int        `json:"version"`
	}

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		Log.Errorf("Failed while reading from crypto/rand; %s", err.Error())
		return nil, err
	}

	derivedKey, err := scrypt.Key([]byte(secret), salt, LightScryptN, scryptR, LightScryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := ethcrypto.FromECDSA(privateKey)

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		Log.Errorf("Failed while reading from crypto/rand; %s", err.Error())
		return nil, err
	}

	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := ethcrypto.Keccak256(derivedKey[16:32], cipherText)

	keyUUID, _ := generateEVMKeyUUID()

	return json.Marshal(web3v3{
		ID:      keyUUID,
		Address: hex.EncodeToString(addr[:]),
		Crypto: cryptoJSON{
			Cipher:     "aes-128-ctr",
			CipherText: hex.EncodeToString(cipherText),
			CipherParams: cipherparamsJSON{
				IV: hex.EncodeToString(iv),
			},
			KDF: "scrypt",
			KDFParams: map[string]interface{}{
				"n":     LightScryptN,
				"r":     scryptR,
				"p":     LightScryptP,
				"dklen": scryptDKLen,
				"salt":  hex.EncodeToString(salt),
			},
			MAC: hex.EncodeToString(mac),
		},
		Version: 3,
	})
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

func generateEVMKeyUUID() (string, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:]), nil
}

// EVMHashFunctionSelector returns the first 4 bytes of the Keccak256 hash of the given function selector
func EVMHashFunctionSelector(sel string) string {
	hash := Keccak256(sel)
	return common.Bytes2Hex(hash[0:4])
}

// Keccak256 hash the given string
func Keccak256(str string) []byte {
	return ethcrypto.Keccak256([]byte(str))
}

// Transaction broadcast helpers

// EVMBroadcastTx injects a signed transaction into the pending pool for execution.
func EVMBroadcastTx(ctx context.Context, networkID, rpcURL string, tx *types.Transaction, client *ethclient.Client, result interface{}) error {
	rpcClient, err := EVMResolveJsonRpcClient(networkID, rpcURL)
	if err != nil {
		return err
	}

	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return err
	}

	return rpcClient.CallContext(ctx, result, "eth_sendRawTransaction", common.ToHex(data))
}

// EVMBroadcastSignedTx emits a given signed tx for inclusion in a block
func EVMBroadcastSignedTx(networkID, rpcURL string, signedTx *types.Transaction) error {
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		return fmt.Errorf("Failed to dial JSON-RPC host; %s", err.Error())
	} else if signedTx != nil {
		Log.Debugf("Transmitting signed tx to JSON-RPC host")
		err = EVMBroadcastTx(context.TODO(), networkID, rpcURL, signedTx, client, nil)
		if err != nil {
			return fmt.Errorf("Failed to transmit signed tx to JSON-RPC host; %s", err.Error())
		}
	}
	return nil
}

// EVMSignTx signs a transaction using the given private key and calldata;
// providing 0 gas results in the tx attempting to use up to the block
// gas limit for execution
func EVMSignTx(networkID, rpcURL, from, privateKey string, to, data *string, val *big.Int, gasLimit uint64) (*types.Transaction, *string, error) {
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, nil, err
	}
	_, err = EVMGetSyncProgress(client)
	if err == nil {
		cfg := EVMGetChainConfig(networkID, rpcURL)
		blockNumber, err := EVMGetLatestBlockNumber(networkID, rpcURL)
		if err != nil {
			return nil, nil, err
		}
		nonce, err := client.PendingNonceAt(context.TODO(), common.HexToAddress(from))
		if err != nil {
			return nil, nil, err
		}
		gasPrice, _ := client.SuggestGasPrice(context.TODO())
		var _data []byte
		if data != nil {
			_data = common.FromHex(*data)
		}

		var tx *types.Transaction

		if gasLimit == 0 {
			callMsg := asEVMCallMsg(from, data, to, val, gasPrice.Uint64(), gasLimit)
			gasLimit, err = client.EstimateGas(context.TODO(), callMsg)
		}

		if to != nil {
			addr := common.HexToAddress(*to)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to estimate gas for tx; %s", err.Error())
			}
			Log.Debugf("Estimated %d total gas required for tx with %d-byte data payload", gasLimit, len(_data))
			tx = types.NewTransaction(nonce, addr, val, gasLimit, gasPrice, _data)
		} else {
			Log.Debugf("Attempting to deploy contract via tx; network: %s", networkID)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to estimate gas for tx; %s", err.Error())
			}
			Log.Debugf("Estimated %d total gas required for contract deployment tx with %d-byte data payload", gasLimit, len(_data))
			tx = types.NewContractCreation(nonce, val, gasLimit, gasPrice, _data)
		}
		signer := types.MakeSigner(cfg, big.NewInt(int64(blockNumber)))
		hash := signer.Hash(tx).Bytes()
		Log.Debugf("Signing tx on behalf of %s", from)
		_privateKey, err := ethcrypto.HexToECDSA(privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed read private key bytes prior to signing tx; %s", err.Error())
		}
		sig, err := ethcrypto.Sign(hash, _privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to sign tx on behalf of %s; %s", *to, err.Error())
		}
		if err == nil {
			signedTx, _ := tx.WithSignature(signer, sig)
			hash := stringOrNil(fmt.Sprintf("0x%x", signedTx.Hash()))
			signedTxJSON, _ := signedTx.MarshalJSON()
			Log.Debugf("Signed tx for broadcast via JSON-RPC: %s", signedTxJSON)
			return signedTx, hash, nil
		}
		return nil, nil, err
	}
	return nil, nil, err
}

// ABI-related helpers

func coerceAbiParameter(t abi.Type, v interface{}) (interface{}, error) {
	typestr := fmt.Sprintf("%s", t)
	defer func() {
		if r := recover(); r != nil {
			Log.Debugf("Failed to coerce ABI parameter of type: %s; value: %s", typestr, v)
		}
	}()
	switch t.T {
	case abi.ArrayTy, abi.SliceTy:
		switch v.(type) {
		case []byte:
			return evmForEachUnpack(t, v.([]byte), 0, len(v.([]interface{}))-1)
		case string:
			return evmForEachUnpack(t, []byte(v.(string)), 0, len(v.(string)))
		default:
			// HACK-- this fallback for edge case handling isn't the cleanest
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
		if val, valOk := v.(string); valOk {
			return val, nil
		}
		return string(v.([]byte)), nil
	case abi.IntTy, abi.UintTy:
		switch t.Kind {
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var intval *big.Int
			if valInt64, valInt64Ok := v.(int64); valInt64Ok {
				intval = big.NewInt(int64(valInt64))
			} else if valFloat64, valFloat64Ok := v.(float64); valFloat64Ok {
				intval = big.NewInt(int64(valFloat64))
			}
			if intval != nil {
				return evmReadInteger(t.Kind, intval.Bytes()), nil
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val, valOk := v.(string); valOk {
				intval, err := strconv.Atoi(val)
				if err != nil {
					Log.Warningf("Failed to coerce string val %s to integer type in accordance with abi; %s", v, err.Error())
					return nil, err
				}
				return big.NewInt(int64(intval)), nil
			}
			return big.NewInt(int64(v.(int64))), nil
		case reflect.Float64:
			return big.NewInt(int64(v.(float64))), nil
		case reflect.Ptr:
			switch v.(type) {
			case float64:
				return big.NewInt(int64(v.(float64))), nil
			}
		default:
			return evmReadInteger(t.Kind, v.([]byte)), nil
		}
	case abi.BoolTy:
		if boolstr, ok := v.(string); ok {
			if strings.ToLower(boolstr) == "true" {
				return true, nil
			}
			return false, nil
		}
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
		return evmReadFixedBytes(t, []byte(v.(string)))
	case abi.FunctionTy:
		return evmReadFunctionType(t, v.([]byte))
	default:
		// no-op
	}
	return nil, fmt.Errorf("Failed to coerce %s parameter for abi encoding; unhandled type: %v", t.String(), t)
}

// iteratively unpack elements
func evmForEachUnpack(t abi.Type, output []byte, start, size int) (interface{}, error) {
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
		elemSize = evmGetFullElemSize(t.Elem)
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
func evmReadInteger(kind reflect.Kind, b []byte) interface{} {
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
func evmReadFunctionType(t abi.Type, word []byte) (funcTy [24]byte, err error) {
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
func evmReadFixedBytes(t abi.Type, word []byte) (interface{}, error) {
	if t.T != abi.FixedBytesTy {
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array")
	}

	Log.Debugf("Attempting to read fixed bytes in accordance with Ethereum contract ABI; type: %v; word: %s", t, word)

	// convert
	array := reflect.New(t.Type).Elem()
	reflect.Copy(array, reflect.ValueOf(word))
	return array.Interface(), nil
}

func evmRequiresLengthPrefix(t *abi.Type) bool {
	return t.T == abi.StringTy || t.T == abi.BytesTy || t.T == abi.SliceTy
}

func evmGetFullElemSize(elem *abi.Type) int {
	//all other should be counted as 32 (slices have pointers to respective elements)
	size := 32
	//arrays wrap it, each element being the same size
	for elem.T == abi.ArrayTy {
		size *= elem.Size
		elem = elem.Elem
	}
	return size
}

func evmLengthPrefixPointsTo(index int, output []byte) (start int, length int, err error) {
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

func evmReadBool(word []byte) (bool, error) {
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

// More calldata construction related items

func asEVMCallMsg(from string, data, to *string, val *big.Int, gasPrice, gasLimit uint64) ethereum.CallMsg {
	var _to *common.Address
	var _data []byte
	if to != nil {
		addr := common.HexToAddress(*to)
		_to = &addr
	}
	if data != nil {
		_data = common.FromHex(*data)
	}
	return ethereum.CallMsg{
		From:     common.HexToAddress(from),
		To:       _to,
		Gas:      gasLimit,
		GasPrice: big.NewInt(int64(gasPrice)),
		Value:    val,
		Data:     _data,
	}
}

func parseContractABI(contractAbi interface{}) (*abi.ABI, error) {
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

	return &abival, nil
}

// EVMEthCall invokes eth_call manually via JSON-RPC
func EVMEthCall(networkID, rpcURL string, params []interface{}) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_call", params, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// EVMGetBlockNumber retrieves the latest block known to the JSON-RPC client
func EVMGetBlockNumber(networkID, rpcURL string) *uint64 {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch latest block number via JSON-RPC eth_blockNumber method")
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_blockNumber", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_blockNumber method via JSON-RPC; %s", err.Error())
		return nil
	}
	blockNumber, err := hexutil.DecodeBig(resp.Result.(string))
	if err != nil {
		return nil
	}
	_blockNumber := blockNumber.Uint64()
	return &_blockNumber
}

// EVMGetChainConfig parses the cached network config mapped to the given
// `networkID`, if one exists; otherwise, the mainnet chain config is returned.
func EVMGetChainConfig(networkID, rpcURL string) *params.ChainConfig {
	if cfg, ok := chainConfigs[networkID]; ok {
		return cfg
	}
	cfg := params.MainnetChainConfig
	chainID, err := strconv.ParseUint(networkID, 10, 0)
	if err != nil {
		cfg.ChainID = big.NewInt(int64(chainID))
		chainConfigs[networkID] = cfg
	}
	return cfg
}

// EVMGetChainID retrieves the current chainID via JSON-RPC
func EVMGetChainID(networkID, rpcURL string) *big.Int {
	ethClient, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		Log.Warningf("Failed to read network id for *ethclient.Client instance: %s; %s", ethClient, err.Error())
		return nil
	}
	if ethClient == nil {
		Log.Warningf("Failed to read network id for unresolved *ethclient.Client instance; network id: %s; JSON-RPC URL: %s", networkID, rpcURL)
		return nil
	}
	chainID, err := ethClient.NetworkID(context.TODO())
	if err != nil {
		Log.Warningf("Failed to read network id for *ethclient.Client instance: %s; %s", ethClient, err.Error())
		return nil
	}
	if chainID != nil {
		Log.Debugf("Received chain id from *ethclient.Client instance: %s", ethClient, chainID)
	}
	return chainID
}

// EVMGetGasPrice returns the gas price
func EVMGetGasPrice(networkID, rpcURL string) *string {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch gas price via JSON-RPC eth_gasPrice method")
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_gasPrice", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_gasPrice method via JSON-RPC; %s", err.Error())
		return nil
	}
	return stringOrNil(resp.Result.(string))
}

// EVMGetLatestBlock retrieves the latsest block
func EVMGetLatestBlock(networkID, rpcURL string) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_getBlockByNumber", []interface{}{"latest", true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// EVMGetLatestBlockNumber retrieves the latest block number
func EVMGetLatestBlockNumber(networkID, rpcURL string) (uint64, error) {
	resp, err := EVMGetLatestBlock(networkID, rpcURL)
	if err != nil {
		return 0, err
	}
	blockNumberStr, blockNumberStrOk := resp.Result.(map[string]interface{})["number"].(string)
	if !blockNumberStrOk {
		return 0, errors.New("Unable to parse block number from JSON-RPC response")
	}
	blockNumber, err := hexutil.DecodeUint64(blockNumberStr)
	if err != nil {
		return 0, fmt.Errorf("Unable to decode block number hex; %s", err.Error())
	}
	return blockNumber, nil
}

// EVMGetBlockGasLimit retrieves the latest block gas limit
func EVMGetBlockGasLimit(networkID, rpcURL string) (uint64, error) {
	resp, err := EVMGetLatestBlock(networkID, rpcURL)
	if err != nil {
		return 0, err
	}
	blockGasLimitStr, blockGasLimitStrOk := resp.Result.(map[string]interface{})["gasLimit"].(string)
	if !blockGasLimitStrOk {
		return 0, errors.New("Unable to parse block gas limit from JSON-RPC response")
	}
	blockGasLimit, err := hexutil.DecodeUint64(blockGasLimitStr)
	if err != nil {
		return 0, fmt.Errorf("Unable to decode block gas limit hex; %s", err.Error())
	}
	return blockGasLimit, nil
}

// EVMGetBlockByNumber retrieves a given block by number
func EVMGetBlockByNumber(networkID, rpcURL string, blockNumber uint64) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_getBlockByNumber", []interface{}{hexutil.EncodeUint64(blockNumber), true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// EVMGetHeaderByNumber retrieves a given block header by number
func EVMGetHeaderByNumber(networkID, rpcURL string, blockNumber uint64) (*EthereumJsonRpcResponse, error) {
	var jsonRpcResponse = &EthereumJsonRpcResponse{}
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_getHeaderByNumber", []interface{}{hexutil.EncodeUint64(blockNumber), true}, &jsonRpcResponse)
	return jsonRpcResponse, err
}

// EVMGetNativeBalance retrieves a wallet's native currency balance
func EVMGetNativeBalance(networkID, rpcURL, addr string) (*big.Int, error) {
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, err
	}
	return client.BalanceAt(context.TODO(), common.HexToAddress(addr), nil)
}

// EVMGetNetworkStatus retrieves current metadata from the JSON-RPC client;
// returned struct includes block height, chainID, number of connected peers,
// protocol version, and syncing state.
func EVMGetNetworkStatus(networkID, rpcURL string) (*NetworkStatus, error) {
	ethClient, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil || rpcURL == "" || ethClient == nil {
		meta := map[string]interface{}{
			"error": nil,
		}
		if err != nil {
			Log.Warningf("Failed to dial JSON-RPC host: %s; %s", rpcURL, err.Error())
			meta["error"] = err.Error()
		} else if rpcURL == "" {
			meta["error"] = "No 'full-node' JSON-RPC URL configured or resolvable"
		} else if ethClient == nil {
			meta["error"] = "Configured 'full-node' JSON-RPC client not resolved"
		}
		return &NetworkStatus{
			State: stringOrNil("configuring"),
			Meta:  meta,
		}, nil
	}

	defer func() {
		if r := recover(); r != nil {
			Log.Debugf("Recovered from failed attempt to retrieve network sync progress from JSON-RPC host: %s", rpcURL)
			evmClearCachedClients(networkID)
		}
	}()

	syncProgress, err := EVMGetSyncProgress(ethClient)
	if err != nil {
		Log.Warningf("Failed to read network sync progress using JSON-RPC host; %s", err.Error())
		evmClearCachedClients(networkID)
		return nil, err
	}
	var state string
	var block uint64        // current block; will be less than height while syncing in progress
	var height *uint64      // total number of blocks
	var lastBlockAt *uint64 // unix timestamp of last block
	chainID := EVMGetChainID(networkID, rpcURL)
	peers := EVMGetPeerCount(networkID, rpcURL)
	protocolVersion := EVMGetProtocolVersion(networkID, rpcURL)
	meta := map[string]interface{}{}
	var syncing = false
	if syncProgress == nil {
		state = "synced"
		resp, err := EVMGetLatestBlock(networkID, rpcURL)
		if err != nil {
			Log.Warningf("Failed to read latest block for %s using JSON-RPC host; %s", rpcURL, err.Error())
			return nil, err
		}
		hdr := resp.Result.(map[string]interface{})
		delete(hdr, "transactions") // HACK
		delete(hdr, "uncles")       // HACK

		meta["last_block_header"] = hdr
		block, err = hexutil.DecodeUint64(hdr["number"].(string))
		if err != nil {
			return nil, fmt.Errorf("Unable to decode block number hex; %s", err.Error())
		}

		_lastBlockAt, err := hexutil.DecodeUint64(hdr["timestamp"].(string))
		if err != nil {
			return nil, fmt.Errorf("Unable to decode block timestamp hex; %s", err.Error())
		}
		lastBlockAt = &_lastBlockAt
	} else {
		block = syncProgress.CurrentBlock
		height = &syncProgress.HighestBlock
		syncing = true
	}
	return &NetworkStatus{
		Block:           block,
		Height:          height,
		ChainID:         stringOrNil(hexutil.EncodeBig(chainID)),
		PeerCount:       peers,
		LastBlockAt:     lastBlockAt,
		ProtocolVersion: protocolVersion,
		State:           stringOrNil(state),
		Syncing:         syncing,
		Meta:            meta,
	}, nil
}

// EVMGetPeerCount returns the number of peers currently connected to the JSON-RPC client
func EVMGetPeerCount(networkID, rpcURL string) uint64 {
	var peerCount uint64
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch peer count via net_peerCount method via JSON-RPC")
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "net_peerCount", params, &resp)
	if err != nil {
		Log.Debugf("Attempting to fetch peer count via parity_netPeers method via JSON-RPC")
		err := EVMInvokeJsonRpcClient(networkID, rpcURL, "parity_netPeers", params, &resp)
		Log.Warningf("Failed to invoke parity_netPeers method via JSON-RPC; %s", err.Error())
		return 0
	}
	if peerCountStr, ok := resp.Result.(string); ok {
		peerCount, err = hexutil.DecodeUint64(peerCountStr)
		if err != nil {
			return 0
		}
	}
	return peerCount
}

// EVMGetProtocolVersion returns the JSON-RPC client protocol version
func EVMGetProtocolVersion(networkID, rpcURL string) *string {
	params := make([]interface{}, 0)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch protocol version via JSON-RPC eth_protocolVersion method")
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_protocolVersion", params, &resp)
	if err != nil {
		Log.Debugf("Attempting to fetch protocol version via JSON-RPC net_version method")
		err := EVMInvokeJsonRpcClient(networkID, rpcURL, "net_version", params, &resp)

		Log.Warningf("Failed to invoke eth_protocolVersion method via JSON-RPC; %s", err.Error())
		return nil
	}
	return stringOrNil(resp.Result.(string))
}

// EVMGetCode retrieves the code stored at the named address in the given scope;
// scope can be a block number, latest, earliest or pending
func EVMGetCode(networkID, rpcURL, addr, scope string) (*string, error) {
	params := make([]interface{}, 0)
	params = append(params, addr)
	params = append(params, scope)
	var resp = &EthereumJsonRpcResponse{}
	Log.Debugf("Attempting to fetch code from %s via eth_getCode JSON-RPC method", addr)
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "eth_getCode", params, &resp)
	if err != nil {
		Log.Warningf("Failed to invoke eth_getCode method via JSON-RPC; %s", err.Error())
		return nil, err
	}
	return stringOrNil(resp.Result.(string)), nil
}

// EVMGetSyncProgress retrieves the status of the current network sync
func EVMGetSyncProgress(client *ethclient.Client) (*ethereum.SyncProgress, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	progress, err := client.SyncProgress(ctx)
	if err != nil {
		Log.Warningf("Failed to read sync progress for *ethclient.Client instance: %s; %s", client, err.Error())
		cancel()
		return nil, err
	}
	if progress != nil {
		Log.Debugf("Latest synced block reported by *ethclient.Client instance: %v [of %v]", client, progress.CurrentBlock, progress.HighestBlock)
	}
	cancel()
	return progress, nil
}

// EVMGetTokenBalance retrieves a token balance for a specific token contract and network address
func EVMGetTokenBalance(networkID, rpcURL, tokenAddr, addr string, contractABI interface{}) (*big.Int, error) {
	var balance *big.Int
	abi, err := parseContractABI(contractABI)
	if err != nil {
		return nil, err
	}
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	gasPrice, _ := client.SuggestGasPrice(context.TODO())
	to := common.HexToAddress(tokenAddr)
	msg := ethereum.CallMsg{
		From:     common.HexToAddress(addr),
		To:       &to,
		Gas:      0,
		GasPrice: gasPrice,
		Value:    nil,
		Data:     common.FromHex(EVMHashFunctionSelector("balanceOf(address)")),
	}
	result, _ := client.CallContract(context.TODO(), msg, nil)
	if method, ok := abi.Methods["balanceOf"]; ok {
		method.Outputs.Unpack(&balance, result)
		if balance != nil {
			symbol, _ := EVMGetTokenSymbol(networkID, rpcURL, addr, tokenAddr, contractABI)
			Log.Debugf("Read %s token balance (%v) from token contract address: %s", symbol, balance, addr)
		}
	} else {
		Log.Warningf("Unable to read balance of unsupported token contract address: %s", tokenAddr)
	}
	return balance, nil
}

// EVMGetTokenSymbol attempts to retrieve the symbol of a token presumed to be deployed at the given token contract address
func EVMGetTokenSymbol(networkID, rpcURL, from, tokenAddr string, contractABI interface{}) (*string, error) {
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		return nil, err
	}
	_abi, err := parseContractABI(contractABI)
	if err != nil {
		return nil, err
	}
	to := common.HexToAddress(tokenAddr)
	msg := ethereum.CallMsg{
		From:     common.HexToAddress(from),
		To:       &to,
		Gas:      0,
		GasPrice: big.NewInt(0),
		Value:    nil,
		Data:     common.FromHex(EVMHashFunctionSelector("symbol()")),
	}
	result, _ := client.CallContract(context.TODO(), msg, nil)
	var symbol string
	if method, ok := _abi.Methods["symbol"]; ok {
		err = method.Outputs.Unpack(&symbol, result)
		if err != nil {
			Log.Warningf("Failed to read token symbol from deployed token contract %s; %s", tokenAddr, err.Error())
		}
	}
	return stringOrNil(symbol), nil
}

// EVMTraceTx returns the VM traces; requires parity JSON-RPC client and the node must
// be configured with `--fat-db on --tracing on --pruning archive`
func EVMTraceTx(networkID, rpcURL string, hash *string) (interface{}, error) {
	var addr = *hash
	if !strings.HasPrefix(addr, "0x") {
		addr = fmt.Sprintf("0x%s", addr)
	}
	params := make([]interface{}, 0)
	params = append(params, addr)
	var result = &EthereumTxTraceResponse{}
	Log.Debugf("Attempting to trace tx via trace_transaction method via JSON-RPC; tx hash: %s", addr)
	err := EVMInvokeJsonRpcClient(networkID, rpcURL, "trace_transaction", params, &result)
	if err != nil {
		Log.Warningf("Failed to invoke trace_transaction method via JSON-RPC; %s", err.Error())
		return nil, err
	}
	return result, nil
}

// EVMGetTxReceipt retrieves the full transaction receipt via JSON-RPC given the transaction hash
func EVMGetTxReceipt(networkID, rpcURL, txHash, from string) (*types.Receipt, error) {
	client, err := EVMDialJsonRpc(networkID, rpcURL)
	if err != nil {
		Log.Warningf("Failed to retrieve tx receipt for broadcast tx: %s; %s", txHash, err.Error())
		return nil, err
	}
	Log.Debugf("Attempting to retrieve tx receipt for broadcast tx: %s", txHash)
	return client.TransactionReceipt(context.TODO(), common.HexToHash(txHash))
}
