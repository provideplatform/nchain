package prices

import (
	"strconv"
	"sync"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
	"github.com/provideapp/goldmine/common"
)

// RunExchangeConsumer runs a real-time consumer of pricing data for the given currency pair
func RunExchangeConsumer(currencyPair string, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		consumer := exchangeConsumer.GdaxMessageConsumerFactory(common.Log, priceTick, currencyPair)
		err := consumer.Run()
		if err != nil {
			common.Log.Warningf("Consumer exited unexpectedly; %s", err)
		} else {
			common.Log.Debug("Exiting exchange consumer...")
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
		common.Log.Debugf("Dropping GDAX message; seq: %d", msg.Sequence)
	}
	return nil
}
