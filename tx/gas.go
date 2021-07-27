package tx

import (
	"os"
	"strconv"

	"github.com/provideplatform/nchain/common"
)

var minimumGasPrice *float64

const defaultMinimumGasPrice float64 = 6000000000

func MinimumGasPrice() *float64 {

	if minimumGasPrice != nil {
		return minimumGasPrice
	}

	envMinGasPrice := os.Getenv("MINIMUM_GAS_PRICE_IN_WEI")
	if envMinGasPrice == "" {
		gasprice := defaultMinimumGasPrice
		return &gasprice
	}

	customMinGasPrice, err := strconv.ParseFloat(envMinGasPrice, 64)
	if err != nil {
		common.Log.Debugf("Error parsing custom minimum gas price. using default minimum gas price of %v wei. Error: %s", defaultMinimumGasPrice, err.Error())
		gasprice := defaultMinimumGasPrice
		return &gasprice
	}

	minimumGasPrice = &customMinGasPrice
	common.Log.Debugf("Using custom minimum gas price of %v wei", *minimumGasPrice)

	return minimumGasPrice
}
