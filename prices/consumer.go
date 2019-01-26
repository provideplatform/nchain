package prices

import (
	"strconv"
	"sync"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
	"github.com/provideapp/goldmine/common"
)

func RunExchangeConsumer(currencyPair string, wg sync.WaitGroup) {
	wg.Add(1)
	go func() {
		consumer := exchangeConsumer.GdaxMessageConsumerFactory(common.Log, priceTick, currencyPair)
		err := consumer.Run()
		if err != nil {
			common.Log.Warningf("Consumer exited unexpectedly; %s", err)
		} else {
			common.Log.Infof("Exiting consumer %s", consumer)
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
		common.Log.Debugf("Dropping GDAX message; %s", msg)
	}
	return nil
}
