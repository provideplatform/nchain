package consumer

import (
	"sync"
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
