package main

import (
	"encoding/json"
	"math/big"
	"sync"
	"time"

	"github.com/nats-io/go-nats-streaming"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
)

const natsTxSubject = "goldmine.tx"
const natsTxMaxInFlight = 128
const natsTxReceiptSubject = "goldmine.tx.receipt"
const natsTxReceiptMaxInFlight = 64

func createNatsTxSubscriptions(natsConnection stan.Conn, wg sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txSubscription, err := natsConnection.QueueSubscribe(natsTxSubject, natsTxSubject, consumeTxMsg, stan.SetManualAckMode(), stan.AckWait(time.Millisecond*10000), stan.MaxInflight(natsTxMaxInFlight), stan.DurableName(natsTxSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				wg.Done()
				return
			}
			defer txSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			wg.Wait()
		}()
	}
}

func createNatsTxReceiptSubscriptions(natsConnection stan.Conn, wg sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txReceiptSubscription, err := natsConnection.QueueSubscribe(natsTxReceiptSubject, natsTxReceiptSubject, consumeTxReceiptMsg, stan.SetManualAckMode(), stan.AckWait(receiptTickerTimeout), stan.MaxInflight(natsTxReceiptMaxInFlight), stan.DurableName(natsTxReceiptSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxReceiptSubject)
				wg.Done()
				return
			}
			defer txReceiptSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsTxReceiptSubject)

			wg.Wait()
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
	dbconf.DatabaseConnection().Where("id = ?", *execution.ContractID).Find(&contract)
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
	} else {
		Log.Debugf("Executed contract; tx: %s", executionResponse)
	}

	msg.Ack()
}

func consumeTxReceiptMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	db := dbconf.DatabaseConnection()

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
