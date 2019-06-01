package consumer

import (
	"sync"
	"time"

	natsutil "github.com/kthomas/go-natsutil"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/provideapp/goldmine/common"
)

const natsDefaultClusterID = "provide"

var (
	waitGroup sync.WaitGroup

	currencyPairs = []string{
		// "BTC-USD",
		// "ETH-USD",
		// "LTC-USD",

		// "PRVD-USD", // FIXME-- pull from tokens database
	}
)

func GetNatsStreamingConnection() stan.Conn {
	if common.NatsDefaultConnectionLostHandler == nil {
		common.NatsDefaultConnectionLostHandler = subscribeNatsStreaming
	}

	natsConnection := common.GetNatsStreamingConnection(subscribeNatsStreaming)
	if natsConnection == nil {
		return nil
	}

	return natsConnection
}

func subscribeNatsStreaming() {
	if common.NatsDefaultConnectionLostHandler == nil {
		common.NatsDefaultConnectionLostHandler = subscribeNatsStreaming
	}

	natsConnection := GetNatsStreamingConnection()
	if natsConnection == nil {
		return
	}
}

// AttemptNack tries to Nack the given message if it meets basic time-based deadlettering
func AttemptNack(msg *stan.Msg, timeout int64) {
	if ShouldDeadletter(msg, timeout) {
		common.Log.Debugf("Nacking redelivered %d-byte message after %dms timeout: %s", msg.Size(), timeout, msg.Subject)
		Nack(msg)
	}
}

// Nack the given message
func Nack(msg *stan.Msg) {
	if msg.Redelivered {
		common.Log.Warningf("Nacking redelivered %d-byte message without checking subject-specific deadletter business logic on subject: %s", msg.Size(), msg.Subject)
		natsConn := common.GetDefaultNatsStreamingConnection()
		natsutil.Nack(&natsConn, msg)
	} else {
		common.Log.Debugf("nack() attempted but given NATS message has not yet been redelivered on subject: %s", msg.Subject)
	}
}

// ShouldDeadletter determines if a given message should be deadlettered
func ShouldDeadletter(msg *stan.Msg, deadletterTimeout int64) bool {
	return msg.Redelivered && time.Now().Unix()-msg.Timestamp >= deadletterTimeout
}
