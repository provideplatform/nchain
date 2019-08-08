package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	natsutil "github.com/kthomas/go-natsutil"
	pgputil "github.com/kthomas/go-pgputil"
	selfsignedcert "github.com/kthomas/go-self-signed-cert"
	stan "github.com/nats-io/stan.go"
)

var natsStreamingConnectionMutex sync.Mutex

func buildListenAddr() string {
	listenPort := os.Getenv("PORT")
	if listenPort == "" {
		listenPort = "8080"
	}
	return fmt.Sprintf("0.0.0.0:%s", listenPort)
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

// ShouldServeTLS returns true if the API should be served over TLS
func ShouldServeTLS() bool {
	if requireTLS {
		privKeyPath, certPath, err := selfsignedcert.GenerateToDisk()
		if err != nil {
			Log.Panicf("Failed to generate self-signed certificate; %s", err.Error())
		}
		PrivateKeyPath = *privKeyPath
		CertificatePath = *certPath
		return true
	}
	return false
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

// EstablishNATSStreamingConnection establishes (if conn is nil) or reestablishes the given NATS streaming connection
func EstablishNATSStreamingConnection() error {
	natsStreamingConnectionMutex.Lock()
	defer natsStreamingConnectionMutex.Unlock()

	natsConnection, err := natsutil.GetNatsStreamingConnection(30*time.Second, func(conn stan.Conn, err error) {
		EstablishNATSStreamingConnection()
	})
	if err != nil {
		Log.Warningf("Failed to establish NATS connection; %s", err.Error())
		return err
	}
	SharedNatsConnection = natsConnection
	return nil
}

// GetSharedNatsStreamingConnection retrieves the NATS streaming connection
func GetSharedNatsStreamingConnection() (*stan.Conn, error) {
	if SharedNatsConnection != nil {
		conn := (*SharedNatsConnection).NatsConn()
		if conn != nil && conn.IsConnected() {
			return SharedNatsConnection, nil
		}
	}

	err := EstablishNATSStreamingConnection()
	if err != nil {
		Log.Warningf("Failed to establish NATS connection; %s", err.Error())
		return SharedNatsConnection, err
	}
	return SharedNatsConnection, nil
}

// NATSPublish a NATS message to the configured NATS streaming environment
func NATSPublish(subject string, msg []byte) error {
	natsConnection, err := GetSharedNatsStreamingConnection()
	if err != nil {
		Log.Warningf("Failed to retrieve shared NATS streaming connection for Publish; %s", err.Error())
		return err
	}
	return (*natsConnection).Publish(subject, msg)
}

// NATSPublishAsync async publishes a NATS message to the configured NATS streaming environment
func NATSPublishAsync(subject string, msg []byte) (*string, error) {
	natsConnection, err := GetSharedNatsStreamingConnection()
	if err != nil {
		Log.Warningf("Failed to retrieve shared NATS streaming connection for Publish; %s", err.Error())
		return nil, err
	}
	guid, err := (*natsConnection).PublishAsync(subject, msg, func(_ string, err error) {
		if err != nil {
			Log.Warningf("Failed to asynchronously publish %d-byte NATS streaming message; %s", len(msg), err.Error())
		}
	})
	if err != nil {
		Log.Warningf("Failed to asynchronously publish %d-byte NATS streaming message; %s", len(msg), err.Error())
		return nil, err
	}
	return StringOrNil(guid), err
}
