package main

import (
	"errors"
	"fmt"
)

var CurrenctPrices = &Prices{}

type Prices struct {
	BtcUsdPrice float64 `json:"btcusd"`
	EthUsdPrice float64 `json:"ethusd"`
	LtcUsdPrice float64 `json:"ltcusd"`
}

func CurrentPrice(currencyPair string) (*float64, error) {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		return &CurrenctPrices.BtcUsdPrice, nil
	case cp == "ETH-USD":
		return &CurrenctPrices.EthUsdPrice, nil
	case cp == "LTC-USD":
		return &CurrenctPrices.LtcUsdPrice, nil
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return nil, errors.New(msg)
	}
}

func SetPrice(currencyPair string, price float64) error {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		CurrenctPrices.BtcUsdPrice = price
	case cp == "ETH-USD":
		CurrenctPrices.EthUsdPrice = price
	case cp == "LTC-USD":
		CurrenctPrices.LtcUsdPrice = price
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return errors.New(msg)
	}
	return nil
}
