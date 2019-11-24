package common

import (
	"os"
	"strings"
	"time"

	awsconf "github.com/kthomas/go-aws-config"
	"github.com/kthomas/go-logger"
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

	ConsumeNATSStreamingSubscriptions = strings.ToLower(os.Getenv("CONSUME_NATS_STREAMING_SUBSCRIPTIONS")) == "true"
}
