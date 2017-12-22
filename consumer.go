package main

import (
	"sync"

	. "github.com/kthomas/exchange-consumer"
)

var (
	waitGroup sync.WaitGroup

	currencyPairs = []string{
		"BTC-USD",
		"ETH-USD",
		"LTC-USD",
	}
)

func RunConsumers() {

	go func() {
		waitGroup.Add(1)
		for _, currencyPair := range currencyPairs {
			runConsumer(currencyPair)
		}
		waitGroup.Wait()
	}()
}

func runConsumer(currencyPair string) {
	waitGroup.Add(1)
	go func() {
		consumer := GdaxMessageConsumerFactory(Log, priceTick, currencyPair)
		err := consumer.Run()
		if err != nil {
			Log.Warningf("Consumer exited unexpectedly; %s", err)
		} else {
			Log.Infof("Exiting consumer %s", consumer)
		}
	}()
}

func priceTick(msg *GdaxMessage) error {
	if msg.Type == "done" && msg.Reason == "filled" && msg.Price != "" {
		Log.Infof("Price ticked; %s", msg)
		// SetPrice(msg.ProductId)
	} else {
		Log.Debugf("Dropping GDAX message; %s", msg)
	}
	return nil
}
