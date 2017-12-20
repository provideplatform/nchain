package main

import (
	"sync"

	. "github.com/kthomas/exchange-consumer"
	amqputil "github.com/kthomas/go-amqputil"
)

var (
	waitGroup sync.WaitGroup
)

func RunConsumer() {
	consumers := []*amqputil.Consumer{
		GdaxMessageConsumerFactory(Log, priceTick, "BTC-USD"),
		GdaxMessageConsumerFactory(Log, priceTick, "ETH-USD"),
		GdaxMessageConsumerFactory(Log, priceTick, "LTC-USD"),
	}

	for _, consumer := range consumers {
		waitGroup.Add(1)
		go func() {
			consumer.Run()
			waitGroup.Done()
			Log.Infof("Exiting ticker message consumer %s", consumer)
		}()
	}
}

func priceTick(msg *GdaxMessage) error {
	if msg.Type == "done" && msg.Reason == "filled" && msg.Price != "" {
		Log.Infof("Price ticked; %s", msg)
	} else {
		Log.Debugf("Dropping GDAX message; %s", msg)
	}
	return nil
}
