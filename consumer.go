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
const natsTxReceiptSubject = "goldmine-tx-receipt"

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
		subscribeNats()
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

func getNatsConnection() *nats.Conn {
	if natsConnection == nil {
		conn, err := nats.Connect(natsURL, nats.Token(natsToken))
		if err == nil {
			natsConnection = conn
		} else {
			Log.Warningf("NATS connection failed; %s", err.Error())
		}
	}

	return natsConnection
}

func subscribeNats() {
	natsConnection := getNatsConnection()
	if natsConnection == nil {
		return
	}

	for i := uint64(0); i < natsConsumerConcurrency; i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txSubscription, err := natsConnection.QueueSubscribe(natsTxSubject, natsTxSubject, consumeTxMsg)
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()

			txSubscription.Unsubscribe()
			txSubscription.Drain()
		}()

		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txReceiptSubscription, err := natsConnection.QueueSubscribe(natsTxReceiptSubject, natsTxReceiptSubject, consumeTxReceiptMsg)
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()

			txReceiptSubscription.Unsubscribe()
			txReceiptSubscription.Drain()
		}()
	}
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

	executionResponse, err := contract.Execute(execution.Ref, execution.Wallet, execution.Value, execution.Method, execution.Params, _gas)
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

func consumeTxReceiptMsg(msg *nats.Msg) {
	Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	db := DatabaseConnection()

	var tx *Transaction
	var wallet *Wallet

	err := json.Unmarshal(msg.Data, &tx)
	if err != nil {
		Log.Warningf("Failed to umarshal tx receipt message; %s", err.Error())
		return
	}

	network, err := tx.GetNetwork()
	if err != nil {
		Log.Warningf("Failed to resolve tx network; %s", err.Error())
	}

	db.Model(&Wallet{}).Where("id = ?", tx.WalletID).Find(&wallet)
	if wallet != nil {
		Log.Warningf("Failed to resolve tx wallet")
	}

	tx.fetchReceipt(db, network, wallet)
}
