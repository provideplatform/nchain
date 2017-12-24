package main

import (
	"errors"
	"fmt"
)

var CurrentPrices = &Prices{}

func init() {
	CurrentPrices.PrvdUsdPrice = 0.22 // FIXME-- populate using token sale contract when it exists
}

type Prices struct {
	BtcUsdPrice float64 `json:"btcusd"`
	EthUsdPrice float64 `json:"ethusd"`
	LtcUsdPrice float64 `json:"ltcusd"`

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
		Log.Warning(msg)
		return nil, errors.New(msg)
	}
}

func SetPrice(currencyPair string, price float64) error {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		CurrentPrices.BtcUsdPrice = price
	case cp == "ETH-USD":
		CurrentPrices.EthUsdPrice = price
	case cp == "LTC-USD":
		CurrentPrices.LtcUsdPrice = price
	case cp == "PRVD-USD":
		CurrentPrices.PrvdUsdPrice = price
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return errors.New(msg)
	}
	return nil
}
