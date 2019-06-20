package common

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
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
func DecryptECDSAPrivateKey(encryptedKey, gpgPrivateKey, gpgEncryptionKey string) (*ecdsa.PrivateKey, error) {
	results := make([]byte, 1)
	db := dbconf.DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as private_key", encryptedKey, gpgPrivateKey, gpgEncryptionKey).Rows()
	defer rows.Close()
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

// PGPPubDecrypt decrypts data previously encrypted using pgp_pub_encrypt
func PGPPubDecrypt(encryptedVal, gpgPrivateKey, gpgPassword string) ([]byte, error) {
	results := make([]byte, 1)
	db := dbconf.DatabaseConnection()
	rows, err := db.Raw("SELECT pgp_pub_decrypt(?, dearmor(?), ?) as val", encryptedVal, gpgPrivateKey, gpgPassword).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		rows.Scan(&results)
		return results, nil
	}
	return nil, errors.New("Failed to decrypt record from encrypted storage")
}

// PGPPubEncrypt encrypts data using using pgp_pub_encrypt
func PGPPubEncrypt(unencryptedVal, gpgPublicKey string) (*string, error) {
	out := []string{}
	db := dbconf.DatabaseConnection()
	db.Raw("SELECT pgp_pub_encrypt(?, dearmor(?))", unencryptedVal, gpgPublicKey).Pluck("val", &out)
	return StringOrNil(out[0]), nil
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
		if conn != nil && !conn.IsClosed() && !conn.IsDraining() && !conn.IsReconnecting() {
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
