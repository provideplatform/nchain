package main

import (
	"sync"

	. "github.com/kthomas/exchange-consumer"
	amqputil "github.com/kthomas/go-amqputil"
)

var (
	waitGroup sync.WaitGroup
)

func Consume() {
	consumers := []*amqputil.Consumer{
		GdaxMessageConsumerFactory(Log, priceTick, "BTC-USD"),
		GdaxMessageConsumerFactory(Log, priceTick, "ETH-USD"),
		GdaxMessageConsumerFactory(Log, priceTick, "LTC-USD"),
	}

	for _, consumer := range consumers {
		waitGroup.Add(1)
		go func() {
			consumer.Run()
			Log.Infof("Exiting ticker message consumer %s", consumer)
		}()
	}

	waitGroup.Wait()
	Log.Infof("Exiting consumer")
}

func priceTick(msg *GdaxMessage) {
	Log.Infof("Price ticked; %s", msg)
}
