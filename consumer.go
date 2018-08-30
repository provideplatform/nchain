package main

import (
	"encoding/json"
	"math/big"
	"strconv"
	"sync"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
	uuid "github.com/kthomas/go.uuid"
	nats "github.com/nats-io/go-nats"
)

const natsTxSubject = "goldmine-tx"

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
		subscribeNats(natsToken)
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

func natsConnection(token string) *nats.Conn {
	conn, err := nats.Connect(natsURL, nats.Token(token))
	if err != nil {
		Log.Warningf("NATS connection failed; %s", err.Error())
	}
	return conn
}

func subscribeNats(token string) {
	natsConnection := natsConnection(token)
	if natsConnection == nil {
		return
	}

	waitGroup.Add(1)
	go func() {
		defer natsConnection.Close()
		_, err := natsConnection.Subscribe(natsTxSubject, consumeTxMsg)
		if err != nil {
			Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
			waitGroup.Done()
			return
		}
		Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)
		waitGroup.Wait()
	}()
}

func consumeTxMsg(msg *nats.Msg) {
	Log.Debugf("Consuming NATS tx message: %s", msg)

	execution := &ContractExecution{}
	err := json.Unmarshal(msg.Data, execution)
	if err != nil {
		Log.Warningf("Failed to unmarshal contract execution during NATS tx message handling")
		return
	}

	if execution.ContractID == nil {
		Log.Errorf("Invalid tx message; missing contract_id")
		return
	}

	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil && execution.Wallet.ID != execution.Wallet.ID {
			Log.Errorf("Invalid tx message specifying a wallet_id and wallet")
			return
		}
		wallet := &Wallet{}
		wallet.setID(*execution.WalletID)
		execution.Wallet = wallet
	}

	contract := &Contract{}
	DatabaseConnection().Where("id = ?", *execution.ContractID).Find(&contract)
	if contract == nil || contract.ID == uuid.Nil {
		Log.Errorf("Unable to execute contract; contract not found: %s", contract.ID)
		return
	}

	gas := execution.Gas
	if gas == nil {
		gas64 := float64(0)
		gas = &gas64
	}
	_gas, _ := big.NewFloat(*gas).Uint64()

	executionResponse, err := contract.Execute(execution.Wallet, execution.Value, execution.Method, execution.Params, _gas)
	if err != nil {
		Log.Warningf("Failed to execute contract")
		return
		// natsConnection := natsConnection(natsToken)
		// if natsConnection == nil {
		// 	Log.Warningf("Unable to nack failed contract execution tx")
		// 	return
		// }
	}

	Log.Debugf("Executed contract; tx: %s", executionResponse)
}
