package main

import (
	"sync"

	natsutil "github.com/kthomas/go-natsutil"
	"github.com/nats-io/go-nats-streaming"
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

// RunConsumers launches a goroutine for each data feed
// that has been configured to consume messages
func RunConsumers() {
	go func() {
		waitGroup.Add(1)
		subscribeNatsStreaming()
		for _, currencyPair := range currencyPairs {
			RunExchangeConsumer(currencyPair, &waitGroup)
		}
		waitGroup.Wait()
	}()
}

func getNatsStreamingConnection() stan.Conn {
	return GetNatsStreamingConnection(subscribeNatsStreaming)
}

func subscribeNatsStreaming() {
	if NatsDefaultConnectionLostHandler == nil {
		NatsDefaultConnectionLostHandler = subscribeNatsStreaming
	}

	natsConnection := getNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsTxSubscriptions(natsConnection, &waitGroup)
	createNatsTxReceiptSubscriptions(natsConnection, &waitGroup)
	createNatsBlockFinalizedSubscriptions(natsConnection, &waitGroup)
	createNatsContractCompilerInvocationSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerProvisioningSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection, &waitGroup)
}

func nack(msg *stan.Msg) {
	if msg.Redelivered {
		Log.Warningf("Nacking redelivered %d-byte message without checking subject-specific deadletter business logic on subject: %s", msg.Size(), msg.Subject)
		natsConn := GetDefaultNatsStreamingConnection()
		natsutil.Nack(&natsConn, msg)
	} else {
		Log.Debugf("nack() attempted but given NATS message has not yet been redelivered on subject: %s", msg.Subject)
	}
}
