package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
	uuid "github.com/kthomas/go.uuid"
	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
)

const natsDefaultClusterID = "provide"
const natsContractCompilerInvocationSubject = "goldmine-contract-compiler-invocation"
const natsContractCompilerInvocationMaxInFlight = 32
const natsStreamingTxFilterSubject = "streaming-tx-filter"
const natsTxSubject = "goldmine-tx"
const natsTxMaxInFlight = 128
const natsTxReceiptSubject = "goldmine-tx-receipt"
const natsTxReceiptMaxInFlight = 64

var (
	waitGroup sync.WaitGroup

	currencyPairs = []string{
		// "BTC-USD",
		// "ETH-USD",
		// "LTC-USD",

		// "PRVD-USD", // FIXME-- pull from tokens database
	}
)

// runConsumers launches a goroutine for each data feed
// that has been configured to consume messages
func runConsumers() {
	go func() {
		waitGroup.Add(1)
		subscribeNatsStreaming()
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

func cacheTxFilters() {
	db := DatabaseConnection()
	var filters []Filter
	db.Find(&filters)
	for _, filter := range filters {
		appFilters := txFilters[filter.ApplicationID.String()]
		if appFilters == nil {
			appFilters = make([]*Filter, 0)
			txFilters[filter.ApplicationID.String()] = appFilters
		}
		appFilters = append(appFilters, &filter)
	}
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

func getNatsStreamingConnection() *stan.Conn {
	if natsStreamingConnection == nil {
		clientID, err := uuid.NewV4()
		if err != nil {
			Log.Warningf("Failed to generate client id for NATS streaming connection; %s", err.Error())
			return nil
		}

		natsConn, err := nats.Connect(natsStreamingURL, nats.Token(natsToken))
		if err != nil {
			Log.Warningf("NATS connection failed; %s", err.Error())
			return nil
		}

		conn, err := stan.Connect(natsDefaultClusterID, fmt.Sprintf("goldmine-%s", clientID.String()), stan.NatsConn(natsConn), stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			natsStreamingConnection = nil
			subscribeNatsStreaming()
		}))
		if err == nil {
			natsStreamingConnection = &conn
		} else {
			Log.Warningf("NATS streaming connection failed; %s", err.Error())
		}
	}

	return natsStreamingConnection
}

func subscribeNatsStreaming() {
	natsConnection := getNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsTxSubscriptions(*natsConnection)
	createNatsTxReceiptSubscriptions(*natsConnection)
	createNatsContractCompilerInvocationSubscriptions(*natsConnection)
}

func createNatsTxSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsConsumerConcurrency; i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txSubscription, err := natsConnection.QueueSubscribe(natsTxSubject, natsTxSubject, consumeTxMsg, stan.SetManualAckMode(), stan.AckWait(time.Millisecond*10000), stan.MaxInflight(natsTxMaxInFlight), stan.DurableName(natsTxSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()

			txSubscription.Unsubscribe()
		}()
	}
}

func createNatsTxReceiptSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsConsumerConcurrency; i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txReceiptSubscription, err := natsConnection.QueueSubscribe(natsTxReceiptSubject, natsTxReceiptSubject, consumeTxReceiptMsg, stan.SetManualAckMode(), stan.AckWait(receiptTickerTimeout), stan.MaxInflight(natsTxReceiptMaxInFlight), stan.DurableName(natsTxReceiptSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()

			txReceiptSubscription.Unsubscribe()
		}()
	}
}

func createNatsContractCompilerInvocationSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsConsumerConcurrency; i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			contractCompilerInvocationSubscription, err := natsConnection.QueueSubscribe(natsContractCompilerInvocationSubject, natsContractCompilerInvocationSubject, consumeContractCompilerInvocationMsg, stan.SetManualAckMode(), stan.AckWait(receiptTickerTimeout), stan.MaxInflight(natsContractCompilerInvocationMaxInFlight), stan.DurableName(natsContractCompilerInvocationSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()

			contractCompilerInvocationSubscription.Unsubscribe()
		}()
	}
}

func consumeTxMsg(msg *stan.Msg) {
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
		Log.Warningf("Failed to execute contract; %s", err.Error())
		Log.Warningf("NATS message dropped: %s", msg)
		// FIXME-- Augment NATS support and Nack?
	}

	Log.Debugf("Executed contract; tx: %s", executionResponse)
	msg.Ack()
}

func consumeTxReceiptMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	db := DatabaseConnection()

	var tx *Transaction

	err := json.Unmarshal(msg.Data, &tx)
	if err != nil {
		Log.Warningf("Failed to umarshal tx receipt message; %s", err.Error())
		return
	}

	network, err := tx.GetNetwork()
	if err != nil {
		Log.Warningf("Failed to resolve tx network; %s", err.Error())
	}

	wallet, err := tx.GetWallet()
	if err != nil {
		Log.Warningf("Failed to resolve tx wallet")
	}

	tx.fetchReceipt(db, network, wallet)
	msg.Ack()
}

func consumeContractCompilerInvocationMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS contract compiler invocation message: %s", msg)
	var contract *Contract

	err := json.Unmarshal(msg.Data, &contract)
	if err != nil {
		Log.Warningf("Failed to umarshal contract compiler invocation message; %s", err.Error())
		return
	}

	_, err = contract.Compile()
	if err != nil {
		Log.Warningf("Failed to compile contract; %s", err.Error())
	}

	msg.Ack()
}
