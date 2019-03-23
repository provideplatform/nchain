package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	dbconf "github.com/kthomas/go-db-config"
	selfsignedcert "github.com/kthomas/go-self-signed-cert"

	natsutil "github.com/kthomas/go-natsutil"
	"github.com/nats-io/go-nats-streaming"
)

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

func GetDefaultNatsStreamingConnection() stan.Conn {
	conn := natsutil.GetNatsStreamingConnection(func(_ stan.Conn, reason error) {
		if NatsDefaultConnectionLostHandler != nil {
			NatsDefaultConnectionLostHandler()
		}
	})
	if conn == nil {
		return nil
	}
	return *conn
}

func GetNatsStreamingConnection(connectionLostHandler func()) stan.Conn {
	conn := natsutil.GetNatsStreamingConnection(func(_ stan.Conn, reason error) {
		if connectionLostHandler != nil {
			connectionLostHandler()
		} else if NatsDefaultConnectionLostHandler != nil {
			NatsDefaultConnectionLostHandler()
		}
	})
	if conn == nil {
		return nil
	}
	return *conn
}

func shouldServeTLS() bool {
	if requireTLS {
		privKeyPath, certPath, err := selfsignedcert.GenerateToDisk()
		if err != nil {
			Log.Panicf("Failed to generate self-signed certificate; %s", err.Error())
		}
		privateKeyPath = *privKeyPath
		certificatePath = *certPath
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
