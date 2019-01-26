package main

import (
	"sync"

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
			RunExchangeConsumer(currencyPair, waitGroup)
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

	createNatsTxSubscriptions(natsConnection, waitGroup)
	createNatsTxReceiptSubscriptions(natsConnection, waitGroup)
	createNatsBlockFinalizedSubscriptions(natsConnection, waitGroup)
	createNatsContractCompilerInvocationSubscriptions(natsConnection, waitGroup)
	createNatsLoadBalancerProvisioningSubscriptions(natsConnection, waitGroup)
	createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection, waitGroup)
	createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection, waitGroup)
	createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection, waitGroup)
}
