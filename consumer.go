package main

import (
	"strconv"
	"sync"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
)

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
		for _, currencyPair := range currencyPairs {
			runConsumer(currencyPair)
		}
		waitGroup.Wait()
	}()
}

func runConsumer(currencyPair string) {
	waitGroup.Add(1)
	go func() {
		consumer := exchangeConsumer.GdaxMessageConsumerFactory(Log, priceTick, currencyPair)
		err := consumer.Run()
		if err != nil {
			Log.Warningf("Consumer exited unexpectedly; %s", err)
		} else {
			Log.Infof("Exiting consumer %s", consumer)
		}
	}()
}

func priceTick(msg *exchangeConsumer.GdaxMessage) error {
	if msg.Type == "match" && msg.Price != "" {
		price, err := strconv.ParseFloat(msg.Price, 64)
		if err == nil {
			SetPrice(msg.ProductId, msg.Sequence, price)
		}
	} else {
		Log.Debugf("Dropping GDAX message; %s", msg)
	}
	return nil
}
