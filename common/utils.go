package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	pgputil "github.com/kthomas/go-pgputil"
	bookie "github.com/provideservices/provide-go/api/bookie"
)

var natsStreamingConnectionMutex sync.Mutex
var natsStreamingConnectionDrainTimeout = 10 * time.Second

// BroadcastTransaction attempts to broadcast arbitrary calldata to the specified recipient
// using the Provide Payments API
func BroadcastTransaction(to, calldata *string) (*string, error) {
	_calldata := "0x"
	if calldata != nil {
		_calldata = *calldata
	}

	payment, err := bookie.CreatePayment(defaultPaymentsAccessJWT, map[string]interface{}{
		"to":   to,
		"data": _calldata,
	})
	if err != nil {
		return nil, err
	}

	var result *string
	if rslt, rsltOk := payment.Params["result"].(string); rsltOk {
		result = &rslt
		if to == nil {
			Log.Debugf("broadcast %d-byte contract creation transaction using api.providepayments.com; tx hash: %s", len(_calldata), *result)
		} else {
			Log.Debugf("broadcast %d-byte transaction using api.providepayments.com; recipient: %v; tx hash: %s", len(_calldata), *to, *result)
		}
	} else {
		Log.Warningf("failed to broadcast %d-byte transaction using api.providepayments.com", len(_calldata))
	}

	return result, nil
}

// DecryptECDSAPrivateKey - read the wallet-specific ECDSA private key; required for signing transactions on behalf of the wallet
func DecryptECDSAPrivateKey(encryptedKey string) (*ecdsa.PrivateKey, error) {
	result, err := pgputil.PGPPubDecrypt([]byte(encryptedKey))
	if err != nil {
		Log.Warningf("Failed to read ecdsa private key from encrypted storage; %s", err.Error())
		return nil, err
	}
	privateKeyBytes, err := hex.DecodeString(string(result))
	if err != nil {
		Log.Warningf("Failed to decode ecdsa private key after retrieval from encrypted storage; %s", err.Error())
		return nil, err
	}
	return ethcrypto.ToECDSA(privateKeyBytes)
}

// PanicIfEmpty panics if the given string is empty
func PanicIfEmpty(val string, msg string) {
	if val == "" {
		panic(msg)
	}
}

// StringOrNil returns a ptr to a string or nil if the string is empty
func StringOrNil(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}

// BoolOrNil returns a pointer to the given bool
func BoolOrNil(b bool) *bool {
	return &b
}

// PtrToInt returns a pointer to the given int
func PtrToInt(i int) *int {
	return &i
}

// MarshalConfig marshals the given map to raw JSON
func MarshalConfig(opts map[string]interface{}) *json.RawMessage {
	cfgJSON, _ := json.Marshal(opts)
	_cfgJSON := json.RawMessage(cfgJSON)
	return &_cfgJSON
}
