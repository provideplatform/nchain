package tx

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	"github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/stan.go"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/contract"
	api "github.com/provideservices/provide-go/api"
	bookie "github.com/provideservices/provide-go/api/bookie"
	provide "github.com/provideservices/provide-go/api/nchain"
	util "github.com/provideservices/provide-go/common/util"
)

// TODO: should this be calculated dynamically against average blocktime for the network and subscriptions reestablished?

const natsTxSubject = "nchain.tx"
const natsTxMaxInFlight = 2048
const txAckWait = time.Second * 60
const txMsgTimeout = int64(txAckWait * 5)

const natsTxCreateSubject = "nchain.tx.create"
const natsTxCreateMaxInFlight = 2048
const txCreateAckWait = time.Second * 60
const txCreateMsgTimeout = int64(txCreateAckWait * 5)

const natsTxFinalizeSubject = "nchain.tx.finalize"
const natsTxFinalizeMaxInFlight = 4096
const txFinalizeAckWait = time.Second * 15
const txFinalizedMsgTimeout = int64(txFinalizeAckWait * 6)

const natsTxReceiptSubject = "nchain.tx.receipt"
const natsTxReceiptMaxInFlight = 2048
const txReceiptAckWait = time.Second * 15
const txReceiptMsgTimeout = int64(txReceiptAckWait * 15)

const msgInFlight = "IN_FLIGHT"
const msgRetryRequired = "RETRY_REQUIRED"

var waitGroup sync.WaitGroup

var ch chan BroadcastConfirmation

// incoming and outgoing channels for transactions
type channelPair struct {
	incoming chan BroadcastConfirmation
	outgoing chan BroadcastConfirmation
}

// dictionary for channels (interface is channelPair struct only)
var channelPairs map[string]interface{}

type ValueDictionary struct {
	channels map[string]channelPair
	lock     sync.RWMutex
}

var dict ValueDictionary

// Set adds a new item to the dictionary
func (d *ValueDictionary) Set(k string, v channelPair) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.channels == nil {
		d.channels = make(map[string]channelPair)
	}
	d.channels[k] = v
}

// Get returns the value associated with the key
func (d *ValueDictionary) Get(k string) channelPair {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.channels[k]
}

// Has returns true if the key exists in the dictionary
func (d *ValueDictionary) Has(k string) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()
	_, ok := d.channels[k]
	return ok
}

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Tx package consumer configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsStreamingConnection(nil)

	createNatsTxSubscriptions(&waitGroup)
	createNatsTxCreateSubscriptions(&waitGroup)
	createNatsTxFinalizeSubscriptions(&waitGroup)
	createNatsTxReceiptSubscriptions(&waitGroup)

	// HACK
	// need a buffered channel (per address, but ignore that for the moment)
	ch = make(chan BroadcastConfirmation)

	// change required
	// so as not to be sharing one channel between everything (teh horror for sc4ling)
	// we'll have a new channel pair for every transaction
	// channel 1 is incoming (and will tell it to go)
	// channel 2 is outgoing (and will tell the next one to go)
	// once the incoming channel is read, it will close
	// once the outgoing channel is read, it will close
	// it will make the incoming channel and outgoing channel
	// indexed by the nonce selected for that account and network (network:account:nonce)
	// in a dictionary kv store. key: channel1 & 2 struct)
	// creating a tx creates both channels
	// the creating goroutine will take both channels
	// read on 1
	// write on 2
	// when I create the tx
	//  - check if there's a channel available (in memory dictionary)
	//  - for both the current nonce and the previous
	//  - if there's one for the previous, use its outgoing channel for incoming (by reference copy)
	//  - if there isn't, we make a new incoming channel

}

func createNatsTxSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			txAckWait,
			natsTxSubject,
			natsTxSubject,
			consumeTxExecutionMsg,
			txAckWait,
			natsTxMaxInFlight,
			nil,
		)
	}
}

func createNatsTxCreateSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			txCreateAckWait,
			natsTxCreateSubject,
			natsTxCreateSubject,
			consumeTxCreateMsg,
			txCreateAckWait,
			natsTxCreateMaxInFlight,
			nil,
		)
	}
}

func createNatsTxFinalizeSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			txFinalizeAckWait,
			natsTxFinalizeSubject,
			natsTxFinalizeSubject,
			consumeTxFinalizeMsg,
			txFinalizeAckWait,
			natsTxFinalizeMaxInFlight,
			nil,
		)
	}
}

func createNatsTxReceiptSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			txReceiptAckWait,
			natsTxReceiptSubject,
			natsTxReceiptSubject,
			consumeTxReceiptMsg,
			txReceiptAckWait,
			natsTxReceiptMaxInFlight,
			nil,
		)
	}
}

func getWithLock(key string) (*string, error) {
	var status *string

	lockErr := redisutil.WithRedlock(key, func() error {
		var err error
		status, err = redisutil.Get(key)
		if err != nil {
			return err
		}
		return nil
	})
	if lockErr != nil {
		return nil, lockErr
	}
	common.Log.Debugf("Got message key %s status %s", key, *status)
	return status, nil
}

func setWithLock(key, status string) error {
	lockErr := redisutil.WithRedlock(key, func() error {
		err := redisutil.Set(key, status, nil)
		if err != nil {
			return err
		}
		return nil
	})
	if lockErr != nil {
		return lockErr
	}
	common.Log.Debugf("Set message key %s to status %s", key, status)
	return nil
}

// processMessage checks if the message is already in flight
// with a long running process, in which case, wait for it to finish.
// If the message has failed, processMessage will allow it to be reprocessed
func processMessageStatus(key string) error {
	// need to add a lock to this
	// got a subscribe locked panic ?? or maybe that was for the cached network??  must repro

	status, _ := redisutil.Get(key)

	if status == nil {
		common.Log.Debugf("Setting message key %s status to in flight", key)
		lockErr := setWithLock(key, msgInFlight)
		if lockErr != nil {
			err := fmt.Errorf("Error setting message key %s to in flight status. Error: %s", key, lockErr.Error())
			return err
		}
	}

	if status != nil {
		switch *status {
		case msgInFlight:
			err := fmt.Errorf("Message key %s is in flight", key)
			return err
		case msgRetryRequired:
			lockErr := setWithLock(key, msgInFlight)
			if lockErr != nil {
				err := fmt.Errorf("Error resetting message key %s back to in flight status", key)
				return err
			}
		}
	}
	return nil
}

func consumeTxCreateMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal tx creation message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	contractID, contractIDOk := params["contract_id"]
	data, dataOk := params["data"].(string)
	accountIDStr, accountIDStrOk := params["account_id"].(string)
	walletIDStr, walletIDStrOk := params["wallet_id"].(string)
	hdDerivationPath, _ := params["hd_derivation_path"].(string)
	value, valueOk := params["value"]
	txParams, paramsOk := params["params"].(map[string]interface{})
	publishedAt, publishedAtOk := params["published_at"].(string)

	reference, referenceOk := txParams["ref"]

	if !referenceOk {
		// no reference provided with the contract, so we'll make one
		reference, err = uuid.NewV4()
		if err != nil {
			common.Log.Warningf("Failed to create unique tx ref. Error: %s", err.Error())
			natsutil.Nack(msg)
			return
		}
	}

	var ref string
	// get pointer to reference for tx object (HACK, TIDY)
	//HACK until I find where this is getting set incorrectly
	switch reference.(type) {
	case string:
		ref = reference.(string)
	case uuid.UUID:
		ref = reference.(uuid.UUID).String()
	}

	if !contractIDOk {
		common.Log.Warningf("Failed to unmarshal contract_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !dataOk {
		common.Log.Warningf("Failed to unmarshal data during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !accountIDStrOk && !walletIDStrOk {
		common.Log.Warningf("Failed to unmarshal account_id or wallet_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !valueOk {
		common.Log.Warningf("Failed to unmarshal value during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !paramsOk {
		common.Log.Warningf("Failed to unmarshal params during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !publishedAtOk {
		common.Log.Warningf("Failed to unmarshal published_at during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}

	contract := &contract.Contract{}
	db := dbconf.DatabaseConnection()
	db.Where("id = ?", contractID).Find(&contract)

	var accountID *uuid.UUID
	var walletID *uuid.UUID

	accountUUID, accountUUIDErr := uuid.FromString(accountIDStr)
	if accountUUIDErr == nil {
		accountID = &accountUUID
	}

	walletUUID, walletUUIDErr := uuid.FromString(walletIDStr)
	if walletUUIDErr == nil {
		walletID = &walletUUID
	}

	if accountID == nil && walletID == nil {
		common.Log.Warningf("Failed to unmarshal account_id or wallet_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}

	publishedAtTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		common.Log.Warningf("Failed to parse published_at as RFC3339 timestamp during NATS %v message handling; %s", msg.Subject, err.Error())
		natsutil.Nack(msg)
		return
	}

	tx := &Transaction{
		ApplicationID:  contract.ApplicationID,
		OrganizationID: contract.OrganizationID,
		Data:           common.StringOrNil(data),
		NetworkID:      contract.NetworkID,
		AccountID:      accountID,
		WalletID:       walletID,
		Path:           common.StringOrNil(hdDerivationPath),
		To:             nil,
		Value:          &TxValue{value: big.NewInt(int64(value.(float64)))},
		PublishedAt:    &publishedAtTime,
		Ref:            &ref,
	}

	tx.setParams(txParams)

	key := fmt.Sprintf("nchain.tx.create.%s", *tx.Ref)
	err = processMessageStatus(key)
	if err != nil {
		common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
		return
	}

	common.Log.Debugf("XXX: ConsumeTxCreateMsg, about to create tx with ref: %s", *tx.Ref)
	if tx.Create(db) {
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, created tx with ref: %s", *tx.Ref)
		contract.TransactionID = &tx.ID
		db.Save(&contract)
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, updated contract with txID %s for tx ref: %s", tx.ID, *tx.Ref)
		//		common.Log.Debugf("Transaction execution successful: %s", *tx.Hash)
		err = msg.Ack()
		if err != nil {
			common.Log.Debugf("XXX: ConsumeTxCreateMsg, error acking tx ref: %s", *tx.Ref)
		}
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, msg acked tx ref: %s", *tx.Ref)
	} else {
		errmsg := fmt.Sprintf("Failed to execute transaction; tx ref %s failed with %d error(s)", *tx.Ref, len(tx.Errors))
		for _, err := range tx.Errors {
			errmsg = fmt.Sprintf("%s\n\t%s", errmsg, *err.Message)
		}
		// got rid of the subsidize code that's managed by bookie now

		// remove the in-flight status to this can be replayed
		lockErr := setWithLock(key, msgRetryRequired)
		if lockErr != nil {
			common.Log.Debugf("XXX: Error resetting in flight status for tx ref: %s. Error: %s", *tx.Ref, lockErr.Error())
			// TODO what to do if this fails????
		}
		//TODO this is not showing the errors, instead showing the tx ref
		common.Log.Debugf("XXX: Tx ref %s failed. Error: %s, Attempting nacking", errmsg, *tx.Ref)
		natsutil.AttemptNack(msg, txCreateMsgTimeout)
	}
}

// subsidize the given beneficiary with a drip equal to the given val
func subsidize(db *gorm.DB, networkID uuid.UUID, beneficiary string, val, gas int64) error {
	payment, err := bookie.CreatePayment(util.DefaultVaultAccessJWT, map[string]interface{}{
		"to":   beneficiary,
		"data": common.StringOrNil("0x"),
	})
	if err != nil {
		return err
	}

	common.Log.Debugf("subsidized transaction using api.providepayments.com; beneficiary: %s; tx hash: %s", beneficiary, payment.Params["result"].(string))
	return nil
}

func consumeTxExecutionMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal tx execution message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	contractID, contractIDOk := params["contract_id"]
	data, dataOk := params["data"].(string)
	accountIDStr, accountIDStrOk := params["account_id"].(string)
	walletIDStr, walletIDStrOk := params["wallet_id"].(string)
	hdDerivationPath, _ := params["hd_derivation_path"].(string)
	value, valueOk := params["value"]
	txParams, paramsOk := params["params"].(map[string]interface{})
	publishedAt, publishedAtOk := params["published_at"].(string)

	reference, referenceOk := params["reference"]

	if !referenceOk {
		// no reference provided with the contract, so we'll make one
		reference, err = uuid.NewV4()
		if err != nil {
			common.Log.Warningf("Failed to create unique tx ref. Error: %s", err.Error())
			natsutil.Nack(msg)
			return
		}
	}

	var ref string
	// get pointer to reference for tx object (HACK, TIDY)
	//HACK until I find where this is getting set incorrectly
	switch reference.(type) {
	case string:
		ref = reference.(string)
	case uuid.UUID:
		ref = reference.(uuid.UUID).String()
	}

	if !contractIDOk {
		common.Log.Warningf("Failed to unmarshal contract_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !dataOk {
		common.Log.Warningf("Failed to unmarshal data during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !accountIDStrOk && !walletIDStrOk {
		common.Log.Warningf("Failed to unmarshal account_id or wallet_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !valueOk {
		common.Log.Warningf("Failed to unmarshal value during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !paramsOk {
		common.Log.Warningf("Failed to unmarshal params during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}
	if !publishedAtOk {
		common.Log.Warningf("Failed to unmarshal published_at during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}

	contract := &contract.Contract{}
	db := dbconf.DatabaseConnection()
	db.Where("id = ?", contractID).Find(&contract)

	var accountID *uuid.UUID
	var walletID *uuid.UUID

	accountUUID, accountUUIDErr := uuid.FromString(accountIDStr)
	if accountUUIDErr == nil {
		accountID = &accountUUID
	}

	walletUUID, walletUUIDErr := uuid.FromString(walletIDStr)
	if walletUUIDErr == nil {
		walletID = &walletUUID
	}

	if accountID == nil && walletID == nil {
		common.Log.Warningf("Failed to unmarshal account_id or wallet_id during NATS %v message handling", msg.Subject)
		natsutil.Nack(msg)
		return
	}

	publishedAtTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		common.Log.Warningf("Failed to parse published_at as RFC3339 timestamp during NATS %v message handling; %s", msg.Subject, err.Error())
		natsutil.Nack(msg)
		return
	}

	tx := &Transaction{
		ApplicationID:  contract.ApplicationID,
		OrganizationID: contract.OrganizationID,
		Data:           common.StringOrNil(data),
		NetworkID:      contract.NetworkID,
		AccountID:      accountID,
		WalletID:       walletID,
		Path:           common.StringOrNil(hdDerivationPath),
		To:             contract.Address,
		Value:          &TxValue{value: big.NewInt(int64(value.(float64)))},
		PublishedAt:    &publishedAtTime,
		Ref:            &ref,
	}

	tx.setParams(txParams)

	key := fmt.Sprintf("nchain.tx.%s", *tx.Ref)
	err = processMessageStatus(key)
	if err != nil {
		common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
		return
	}

	common.Log.Debugf("XXX: ConsumeTxExecMsg, about to create tx with ref: %s", *tx.Ref)
	if tx.Create(db) {
		common.Log.Debugf("XXX: ConsumeTxExecMsg, created tx with ref: %s", *tx.Ref)
		err = msg.Ack()
		if err != nil {
			common.Log.Debugf("XXX: ConsumeTxExecMsg, error acking tx ref: %s", *tx.Ref)
		}
		common.Log.Debugf("XXX: ConsumeTxExecMsg, msg acked tx ref: %s", *tx.Ref)
	} else {
		errmsg := fmt.Sprintf("Failed to execute transaction; tx failed with %d error(s)", len(tx.Errors))
		for _, err := range tx.Errors {
			errmsg = fmt.Sprintf("%s\n\t%s", errmsg, *err.Message)
		}
		// got rid of the subsidize code that's managed by bookie now

		// remove the in-flight status to this can be replayed
		lockErr := setWithLock(key, msgRetryRequired)
		if lockErr != nil {
			common.Log.Debugf("XXX: Error resetting in flight status for tx ref: %s. Error: %s", *tx.Ref, lockErr.Error())
			// TODO what to do if this fails????
		}
		common.Log.Debugf("XXX: Tx ref %s failed. Attempting nacking", *tx.Ref)
		natsutil.AttemptNack(msg, txMsgTimeout)
	}
}

// func consumeTxExecutionMsg(msg *stan.Msg) {
// 	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

// 	execution := &contract.Execution{}
// 	err := json.Unmarshal(msg.Data, execution)
// 	if err != nil {
// 		common.Log.Warningf("Failed to unmarshal contract execution during NATS tx message handling")
// 		natsutil.Nack(msg)
// 		return
// 	}

// 	if execution.ContractID == nil {
// 		common.Log.Errorf("Invalid tx message; missing contract_id")
// 		natsutil.Nack(msg)
// 		return
// 	}

// 	if execution.AccountID != nil && *execution.AccountID != uuid.Nil {
// 		var executionAccountID *uuid.UUID
// 		if executionAccount, executionAccountOk := execution.Account.(map[string]interface{}); executionAccountOk {
// 			if executionAccountIDStr, executionAccountIDStrOk := executionAccount["id"].(string); executionAccountIDStrOk {
// 				execAccountUUID, err := uuid.FromString(executionAccountIDStr)
// 				if err == nil {
// 					executionAccountID = &execAccountUUID
// 				}
// 			}
// 		}
// 		if execution.Account != nil && execution.AccountID != nil && *executionAccountID != *execution.AccountID {
// 			common.Log.Errorf("Invalid tx message specifying a account_id and account")
// 			natsutil.Nack(msg)
// 			return
// 		}
// 		account := &wallet.Account{}
// 		account.SetID(*execution.AccountID)
// 		execution.Account = account
// 	}

// 	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
// 		var executionWalletID *uuid.UUID
// 		if executionWallet, executionWalletOk := execution.Wallet.(map[string]interface{}); executionWalletOk {
// 			if executionWalletIDStr, executionWalletIDStrOk := executionWallet["id"].(string); executionWalletIDStrOk {
// 				execWalletUUID, err := uuid.FromString(executionWalletIDStr)
// 				if err == nil {
// 					executionWalletID = &execWalletUUID
// 				}
// 			}
// 		}
// 		if execution.Wallet != nil && execution.WalletID != nil && *executionWalletID != *execution.WalletID {
// 			common.Log.Errorf("Invalid tx message specifying a wallet_id and wallet")
// 			natsutil.Nack(msg)
// 			return
// 		}
// 		wallet := &wallet.Wallet{}
// 		wallet.SetID(*execution.WalletID)
// 		execution.Wallet = wallet
// 	}

// 	db := dbconf.DatabaseConnection()

// 	cntract := &contract.Contract{}
// 	db.Where("id = ?", *execution.ContractID).Find(&cntract)
// 	if cntract == nil || cntract.ID == uuid.Nil {
// 		db.Where("address = ?", *execution.ContractID).Find(&cntract)
// 	}
// 	if cntract == nil || cntract.ID == uuid.Nil {
// 		common.Log.Errorf("Unable to execute contract; contract not found: %s", cntract.ID)
// 		natsutil.Nack(msg)
// 		return
// 	}

// 	key := fmt.Sprintf("nchain.tx.%s", *execution.Ref)
// 	err = processMessageStatus(key)
// 	if err != nil {
// 		common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
// 		return
// 	}

// 	executionResponse, err := executeTransaction(cntract, execution)
// 	if err != nil {
// 		common.Log.Debugf("contract execution failed; %s", err.Error())
// 		// remove the in-flight status to this can be replayed
// 		lockErr := setWithLock(key, msgRetryRequired)
// 		if lockErr != nil {
// 			common.Log.Debugf("XXX: Error resetting in flight status for key: %s. Error: %s", key, lockErr.Error())
// 			// TODO what to do if this fails????
// 		}
// 		natsutil.AttemptNack(msg, txMsgTimeout)
// 	} else {
// 		logmsg := fmt.Sprintf("Executed contract: %s", *cntract.Address)
// 		if executionResponse != nil && executionResponse.Response != nil {
// 			logmsg = fmt.Sprintf("%s; response: %s", logmsg, executionResponse.Response)
// 		}
// 		common.Log.Debug(logmsg)

// 		msg.Ack()
// 	}
// }

// TODO: consider batching this using a buffered channel for high-volume networks
func consumeTxFinalizeMsg(msg *stan.Msg) {
	common.Log.Tracef("Consuming NATS tx finalize message: %s", msg)

	var params map[string]interface{}

	nack := func(msg *stan.Msg, errmsg string, dropPacket bool) {
		if dropPacket {
			common.Log.Tracef("Dropping tx packet (seq: %d) on the floor; %s", msg.Sequence, errmsg)
			msg.Ack()
			return
		}
		natsutil.Nack(msg)
	}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		nack(msg, fmt.Sprintf("Failed to umarshal tx finalize message; %s", err.Error()), true)
		return
	}

	block, blockOk := params["block"].(float64)
	blockTimestampStr, blockTimestampStrOk := params["block_timestamp"].(string)
	finalizedAtStr, finalizedAtStrOk := params["finalized_at"].(string)
	hash, hashOk := params["hash"].(string)

	if !blockOk {
		nack(msg, "Failed to finalize tx; no block provided", true)
		return
	}
	if !blockTimestampStrOk {
		nack(msg, "Failed to finalize tx; no block timestamp provided", true)

		return
	}
	if !finalizedAtStrOk {
		nack(msg, "Failed to finalize tx; no finalized at timestamp provided", true)
		return
	}
	if !hashOk {
		nack(msg, "Failed to finalize tx; no hash provided", true)
		return
	}

	blockTimestamp, err := time.Parse(time.RFC3339, blockTimestampStr)
	if err != nil {
		nack(msg, fmt.Sprintf("Failed to unmarshal block_timestamp during NATS %v message handling; %s", msg.Subject, err.Error()), true)
		return
	}

	finalizedAt, err := time.Parse(time.RFC3339, finalizedAtStr)
	if err != nil {
		nack(msg, fmt.Sprintf("Failed to unmarshal finalized_at during NATS %v message handling; %s", msg.Subject, err.Error()), true)
		return
	}

	tx := &Transaction{}
	db := dbconf.DatabaseConnection()
	common.Log.Tracef("checking local db for tx status; tx hash: %s", hash)
	db.Where("hash = ? AND status IN (?, ?)", hash, "pending", "failed").Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		// TODO: this is integration point to upsert Wallet & Transaction... need to think thru performance implications & implementation details
		nack(msg, fmt.Sprintf("Failed to mark block and finalized_at timestamp on tx: %s; tx not found for given hash", hash), true)
		return
	}

	blockNumber := uint64(block)

	tx.Block = &blockNumber
	tx.BlockTimestamp = &blockTimestamp
	tx.FinalizedAt = &finalizedAt
	if tx.BroadcastAt != nil {
		if tx.PublishedAt != nil {
			queueLatency := uint64(tx.BroadcastAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.QueueLatency = &queueLatency

			e2eLatency := uint64(tx.FinalizedAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.E2ELatency = &e2eLatency
		}

		networkLatency := uint64(tx.FinalizedAt.Sub(*tx.BroadcastAt)) / uint64(time.Millisecond)
		tx.NetworkLatency = &networkLatency
	}

	tx.updateStatus(db, "success", nil)
	result := db.Save(&tx)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			tx.Errors = append(tx.Errors, &api.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
	if len(tx.Errors) > 0 {
		nack(msg, fmt.Sprintf("Failed to set block and finalized_at timestamp on tx: %s; error: %s", hash, *tx.Errors[0].Message), false)
		return
	}

	msg.Ack()
}

func processTxReceipt(msg *stan.Msg, tx *Transaction, key *string, db *gorm.DB) {

	signer, err := tx.signerFactory(db)
	if err != nil {
		desc := "failed to resolve tx signing account or HD wallet"
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		natsutil.Nack(msg)
		return
	}

	err = tx.fetchReceipt(db, signer.Network, signer.Address())
	if err != nil {
		common.Log.Debugf(fmt.Sprintf("Failed to fetch tx receipt for tx hash %s. Error: %s", *tx.Hash, err.Error()))
		// remove the in-flight status to this can be replayed
		lockErr := setWithLock(*key, msgRetryRequired)
		if lockErr != nil {
			common.Log.Debugf("XXX: Error resetting in flight status for tx ref: %s. Error: %s", *tx.Ref, err.Error())
			// TODO what to do if this fails????
		}
		natsutil.AttemptNack(msg, txReceiptMsgTimeout)
	} else {
		common.Log.Debugf("Fetched tx receipt for hash: %s", *tx.Hash)

		common.Log.Debugf("XXX: receipt is: %+v", tx.Response.Receipt.(*provide.TxReceipt))
		blockNumber := tx.Response.Receipt.(*provide.TxReceipt).BlockNumber
		// if we have a block number in the receipt, and the tx has no block
		// populate the block and finalized timestamp
		if blockNumber != nil && tx.Block == nil {
			receiptBlock := blockNumber.Uint64()
			tx.Block = &receiptBlock
			receiptFinalized := time.Now()
			tx.FinalizedAt = &receiptFinalized
			common.Log.Debugf("*** tx hash %s finalized in block %v at %s", *tx.Hash, blockNumber, receiptFinalized.Format("Mon, 02 Jan 2006 15:04:05 MST"))
		}
		tx.updateStatus(db, "success", nil)
		msg.Ack()
	}
}

func consumeTxReceiptMsg(msg *stan.Msg) {
	var key *string

	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed tx receipt message; %s", r)
			if key != nil {
				lockErr := setWithLock(*key, msgRetryRequired)
				if lockErr != nil {
					common.Log.Debugf("XXX: Error resetting in flight status for key: %s. Error: %s", *key, lockErr.Error())
					// TODO what to do if this fails????
				}
			}
			natsutil.AttemptNack(msg, txReceiptMsgTimeout)
		}
	}()

	common.Log.Debugf("consuming NATS tx receipt message: %s", msg)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to unmarshal tx receipt message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	transactionID, transactionIDOk := params["transaction_id"].(string)
	if !transactionIDOk {
		common.Log.Warningf("failed to consume NATS tx receipt message; no transaction id provided")
		natsutil.Nack(msg)
		return
	}

	common.Log.Debugf("XXX: Consuming tx receipt for tx id: %s", transactionID)
	db := dbconf.DatabaseConnection()

	tx := &Transaction{}
	db.Where("id = ?", transactionID).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		common.Log.Tracef("failed to fetch tx receipt; no tx resolved for id: %s", transactionID)
		natsutil.Nack(msg)
		return
	}

	key = common.StringOrNil(fmt.Sprintf("nchain.tx.receipt%s", transactionID))
	err = processMessageStatus(*key)
	if err != nil {
		common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
		return
	}

	// process the receipts
	go processTxReceipt(msg, tx, key, db)
}
