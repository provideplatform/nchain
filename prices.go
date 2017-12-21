package main

import (
	"errors"
	"fmt"
)

var prices = &Prices{}

type Prices struct {
	BtcUsdPrice float64
	EthUsdPrice float64
	LtcUsdPrice float64
}

func CurrentPrice(currencyPair string) (*float64, error) {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		return &prices.BtcUsdPrice, nil
	case cp == "ETH-USD":
		return &prices.EthUsdPrice, nil
	case cp == "LTC-USD":
		return &prices.LtcUsdPrice, nil
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return nil, errors.New(msg)
	}
}

func SetPrice(currencyPair string, price float64) error {

	switch cp := currencyPair; cp != "" {
	case cp == "BTC-USD":
		prices.BtcUsdPrice = price
	case cp == "ETH-USD":
		prices.EthUsdPrice = price
	case cp == "LTC-USD":
		prices.LtcUsdPrice = price
	default:
		msg := fmt.Sprintf("Attempted lookup for unsupported or invalid currency pair: %s", cp)
		Log.Warning(msg)
		return errors.New(msg)
	}
	return nil
}
