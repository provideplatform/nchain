package consumer

import (
	"sync"

	natsutil "github.com/kthomas/go-natsutil"
	stan "github.com/nats-io/stan.go"
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

// AttemptNack tries to Nack the given message if it meets basic time-based deadlettering
func AttemptNack(msg *stan.Msg, timeout int64) {
	if natsutil.ShouldDeadletter(msg, timeout) {
		common.Log.Debugf("Nacking redelivered %d-byte message after %dms timeout: %s", msg.Size(), timeout, msg.Subject)
		Nack(msg)
	}
}

// Nack the given message
func Nack(msg *stan.Msg) {
	if msg.Redelivered {
		common.Log.Warningf("Nacking redelivered %d-byte message without checking subject-specific deadletter business logic on subject: %s", msg.Size(), msg.Subject)
		natsutil.Nack(&common.SharedNatsConnection, msg)
	} else {
		common.Log.Debugf("Nack() attempted but given NATS message has not yet been redelivered on subject: %s", msg.Subject)
	}
}
