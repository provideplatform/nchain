package prices

import (
	"errors"
	"fmt"

	"github.com/provideapp/nchain/common"
)

var CurrentPrices = &Prices{}

func init() {
	CurrentPrices.PrvdUsdPrice = 0.0 // PRVD tokens are worthless until the mainnet launches; TODO-- determine appropriate strategy to oracalize this data
}

type Prices struct {
	BtcUsdPrice    float64 `json:"btcusd"`
	BtcUsdPriceSeq uint64  `json:"-"`

	EthUsdPrice    float64 `json:"ethusd"`
	EthUsdPriceSeq uint64  `json:"-"`

	LtcUsdPrice    float64 `json:"ltcusd"`
	LtcUsdPriceSeq uint64  `json:"-"`

	PrvdUsdPrice float64 `json:"prvdusd"`
}

func CurrentPrice(currencyPair string) (*float64, error) {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		return &CurrentPrices.BtcUsdPrice, nil
	case cp == "ETH-USD":
		return &CurrentPrices.EthUsdPrice, nil
	case cp == "LTC-USD":
		return &CurrentPrices.LtcUsdPrice, nil
	case cp == "PRVD-USD":
		return &CurrentPrices.PrvdUsdPrice, nil
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		common.Log.Warning(msg)
		return nil, errors.New(msg)
	}
}

func CurrentPriceSeq(currencyPair string) (*uint64, error) {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		return &CurrentPrices.BtcUsdPriceSeq, nil
	case cp == "ETH-USD":
		return &CurrentPrices.EthUsdPriceSeq, nil
	case cp == "LTC-USD":
		return &CurrentPrices.LtcUsdPriceSeq, nil
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		common.Log.Warning(msg)
		return nil, errors.New(msg)
	}
}

func SetPrice(currencyPair string, seq uint64, price float64) error {

	_seq, err := CurrentPriceSeq(currencyPair)
	if err != nil {
		return err
	}

	if seq < *_seq {
		msg := fmt.Sprintf("Attempted to update price using stale message for currency pair: %s", currencyPair)
		common.Log.Warning(msg)
		return errors.New(msg)
	}

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		CurrentPrices.BtcUsdPrice = price
		CurrentPrices.BtcUsdPriceSeq = seq
	case cp == "ETH-USD":
		CurrentPrices.EthUsdPrice = price
		CurrentPrices.EthUsdPriceSeq = seq
	case cp == "LTC-USD":
		CurrentPrices.LtcUsdPrice = price
		CurrentPrices.LtcUsdPriceSeq = seq
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		common.Log.Warning(msg)
		return errors.New(msg)
	}
	return nil
}
