package main

import (
	"errors"
	"fmt"
)

var CurrenctPrices = &Prices{}

func init() {
	CurrenctPrices.PrvdUsdPrice = 0.22 // FIXME-- populate using token sale contract when it exists
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
		return &CurrenctPrices.BtcUsdPrice, nil
	case cp == "ETH-USD":
		return &CurrenctPrices.EthUsdPrice, nil
	case cp == "LTC-USD":
		return &CurrenctPrices.LtcUsdPrice, nil
	case cp == "PRVD-USD":
		return &CurrenctPrices.PrvdUsdPrice, nil
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
	case cp == "PRVD-USD":
		CurrenctPrices.PrvdUsdPrice = price
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return errors.New(msg)
	}
	return nil
}
