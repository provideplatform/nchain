package common

import (
	"os"
	"strings"
	"time"

	awsconf "github.com/kthomas/go-aws-config"
	"github.com/kthomas/go-logger"
	stan "github.com/nats-io/stan.go"
)

const reachabilityTimeout = time.Millisecond * 2500

var (
	// Log is the default package logger
	Log *logger.Logger

	// DefaultAWSConfig is the default Amazon Web Services API config
	DefaultAWSConfig *awsconf.Config

	// ListenAddr is the http server listen address
	ListenAddr string

	// CertificatePath is the SSL certificate path used by HTTPS listener
	CertificatePath string
	// PrivateKeyPath is the private key used by HTTPS listener
	PrivateKeyPath string

	requireTLS bool

	// GpgPublicKey is the public key used for PGP encryption
	GpgPublicKey string
	// GpgPrivateKey is the private key used for PGP encryption
	GpgPrivateKey string
	// GpgPassword is the password used for PGP encryption
	GpgPassword string

	// EngineToDefaultJSONRPCPortMapping contains a set of sane defaults for signing enginers and default JSON-RPC ports
	EngineToDefaultJSONRPCPortMapping = map[string]uint{"aura": 8050, "handshake": 13037}
	// EngineToDefaultPeerListenPortMapping contains a set of sane defaults for signing enginers and default p2p ports
	EngineToDefaultPeerListenPortMapping = map[string]uint{"aura": 30303, "handshake": 13038}
	// EngineToDefaultWebsocketPortMapping contains a set of sane defaults for signing enginers and default websocket ports
	EngineToDefaultWebsocketPortMapping = map[string]uint{"aura": 8051}

	// TxFilters contains in-memory Filter instances used for real-time stream processing
	TxFilters = map[string][]interface{}{}

	// ConsumeNATSStreamingSubscriptions is a flag the indicates if the goldmine instance is running in API or consumer mode
	ConsumeNATSStreamingSubscriptions bool

	// SharedNatsConnection is a cached connection used by most NATS Publish calls
	SharedNatsConnection *stan.Conn
)

func init() {
	ListenAddr = os.Getenv("LISTEN_ADDR")
	if ListenAddr == "" {
		ListenAddr = buildListenAddr()
	}

	requireTLS = os.Getenv("REQUIRE_TLS") == "true"

	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "INFO"
	}
	Log = logger.NewLogger("goldmine", lvl, true)

	DefaultAWSConfig = awsconf.GetConfig()

	requireGPG()

	err := EstablishNATSStreamingConnection()
	if err != nil {
		Log.Panicf("Failed to established NATS streaming connection; %s", err.Error())
	}

	ConsumeNATSStreamingSubscriptions = strings.ToLower(os.Getenv("CONSUME_NATS_STREAMING_SUBSCRIPTIONS")) == "true"
}

func requireGPG() {
	GpgPublicKey = strings.Replace(os.Getenv("GPG_PUBLIC_KEY"), `\n`, "\n", -1)
	if GpgPublicKey == "" {
		Log.Panicf("Failed to parse GPG public key")
	}

	GpgPrivateKey = strings.Replace(os.Getenv("GPG_PRIVATE_KEY"), `\n`, "\n", -1)
	if GpgPrivateKey == "" {
		Log.Panicf("Failed to parse GPG private key")
	}

	GpgPassword = os.Getenv("GPG_PASSWORD")
	if GpgPassword == "" {
		Log.Panicf("Failed to parse GPG password")
	}
}
