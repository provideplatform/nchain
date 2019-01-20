package natsutil

import (
	"os"
	"strconv"
	"sync"

	"github.com/kthomas/go-logger"
	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
)

var (
	log *logger.Logger

	natsClientPrefix        string
	natsClusterID           string
	natsConsumerConcurrency uint64
	natsConnection          *nats.Conn
	natsStreamingConnection *stan.Conn
	natsToken               string
	natsURL                 string
	natsStreamingURL        string

	bootstrapOnce sync.Once
)

func init() {
	bootstrapOnce.Do(func() {
		lvl := os.Getenv("LOG_LEVEL")
		if lvl == "" {
			lvl = "INFO"
		}
		log = logger.NewLogger("go-natsutil", lvl, true)

		if os.Getenv("NATS_TOKEN") != "" {
			natsToken = os.Getenv("NATS_TOKEN")
		}

		if os.Getenv("NATS_URL") != "" {
			natsURL = os.Getenv("NATS_URL")
		}

		if os.Getenv("NATS_CLIENT_PREFIX") != "" {
			natsClientPrefix = os.Getenv("NATS_CLIENT_PREFIX")
		} else {
			natsClientPrefix = "go-natsutil"
		}

		if os.Getenv("NATS_CLUSTER_ID") != "" {
			natsClusterID = os.Getenv("NATS_CLUSTER_ID")
		}

		if os.Getenv("NATS_STREAMING_URL") != "" {
			natsStreamingURL = os.Getenv("NATS_STREAMING_URL")

			if os.Getenv("NATS_STREAMING_CONCURRENCY") != "" {
				concurrency, err := strconv.ParseUint(os.Getenv("NATS_STREAMING_CONCURRENCY"), 10, 8)
				if err == nil {
					natsConsumerConcurrency = concurrency
				} else {
					natsConsumerConcurrency = 1
				}
			}
		}
	})
}
