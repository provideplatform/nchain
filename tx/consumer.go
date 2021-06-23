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
var natsWG sync.WaitGroup

var txChannels common.ValueDictionary  //(interface is channelPair struct only)
var txRegister common.ValueDictionary  // stores tx references for idempotent processing
var txSequencer common.ValueDictionary // stores identifiers for broadcast sequencing

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
	start := time.Now()
	//common.Log.Debugf("TIMINGNANO: about to process nats sequence %v at %d", msg.Sequence, time.Now().UnixNano())
	common.Log.Debugf("TIMINGNANO: about to process nats sequence %v", msg.Sequence)
	processTxCreateMsg(msg)
	elapsedTime := time.Since(start)
	common.Log.Debugf("TIMINGNANO: processed nats sequence %v in %s", msg.Sequence, elapsedTime)
	//common.Log.Debugf("TIMINGNANO: completed processing nats sequence %v at %d", msg.Sequence, time.Now().UnixNano())

	// TODO
	// push these onto a buffered channel (X txs)
	// get the tx into the database (or pull if reprocessing)
	// get it signed and ready to broadcast (blocking)
	// give the next msg go-ahead to be processed

}

// these need to be processed EXACTLY in order
// which means minimising the processing done before
// it joins an ordered processing queue
func processTxCreateMsg(msg *stan.Msg) {

	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	db := dbconf.DatabaseConnection()

	// process msg data
	tx, contract, err := processNATSTxCreateMsg(msg, db)
	if err != nil {
		common.Log.Debugf("Error processing NATS %v message. Error: %s", msg.Subject, err.Error())
		natsutil.Nack(msg)
		return
	}
	if tx == nil {
		common.Log.Debugf("ACK: Acking previously processed NATS msg seq: %v", msg.Sequence)
		err := msg.Ack()
		if err != nil {
			common.Log.Debugf("ACK: Error acking previously processed NATS msg seq: %v. Error: %s", msg.Sequence, err.Error())
			//common.Log.Debugf("ACK: Error acking tx ref %s. Error: %s", *tx.Ref, err.Error())
			natsutil.AttemptNack(msg, txCreateMsgTimeout)
		}
		return
	}

	//common.Log.Debugf("TIMINGNANO: about to create tx with ref: %s at %d", *tx.Ref, time.Now().UnixNano())
	common.Log.Debugf("XXX: ConsumeTxCreateMsg, about to create tx with ref: %s", *tx.Ref)

	// check if we have this tx ref in the database

	if tx.Create(db) {
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, created tx with ref: %s", *tx.Ref)

		// update the contract with the tx id (unique to contract flow?)
		contract.TransactionID = &tx.ID
		db.Save(&contract)

		//common.Log.Debugf("XXX: ConsumeTxCreateMsg, updated contract with txID %s for tx ref: %s", tx.ID, *tx.Ref)

	} else {

		errmsg := fmt.Sprintf("Failed to execute transaction; tx ref %s failed with %d error(s)", *tx.Ref, len(tx.Errors))
		for _, err := range tx.Errors {
			errmsg = fmt.Sprintf("%s\n\t%s", errmsg, *err.Message)
		}

		common.Log.Debugf("XXX: Tx ref %s failed. Error: %s, Attempting nacking", *tx.Ref, errmsg)
		natsutil.AttemptNack(msg, txCreateMsgTimeout)
	}

	// HACK
	natsutil.AttemptNack(msg, txCreateMsgTimeout)

}

// processNATSTxCreateMsg processes a NATS msg into a Transaction object
// TODO wg here?
// TODO name this properly
func processNATSTxCreateMsg(msg *stan.Msg, db *gorm.DB) (*Transaction, *contract.Contract, error) {

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal tx creation message; %s", err.Error())
		err := fmt.Errorf("Error unmarshaling tx creation message from NATS %v message. Error: %s", msg.Subject, err.Error())
		return nil, nil, err
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

	// TODO this reference create needs to be on the nchain side, not the consume side
	// so if this doesn't have a reference, we nack it because something
	// has gone terribly wrong
	if !referenceOk {
		// no reference provided with the contract, so we'll make one
		reference, err = uuid.NewV4()
		if err != nil {
			common.Log.Warningf("Failed to create unique tx ref. Error: %s", err.Error())
			err = fmt.Errorf("Error creating unique ref for tx. Error: %s", err.Error())
			return nil, nil, err
		}
	}

	var ref string
	// get pointer to reference for tx object (HACK, TIDY)
	// this is currently a string, but needs to be a uuid
	// (once the consumer ordering issue is sorted)
	switch reference.(type) {
	case string:
		ref = reference.(string)
	case uuid.UUID:
		ref = reference.(uuid.UUID).String()
	}

	//common.Log.Debugf("TIMING NATS: Processing contract create msg from NATS. Sequence: %v. Tx Ref: %s", msg.Sequence, ref)

	// TODO flag around this as it's only for the contract create message
	if !contractIDOk {
		common.Log.Warningf("Failed to unmarshal contract_id during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling contract_id from NATS %v message", msg.Subject)
		return nil, nil, err
	}
	if !dataOk {
		common.Log.Warningf("Failed to unmarshal data during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling data from NATS %v message", msg.Subject)
		return nil, nil, err
	}
	if !accountIDStrOk && !walletIDStrOk {
		common.Log.Warningf("Failed to unmarshal account_id or wallet_id during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling both accountID and walletID from NATS %v message", msg.Subject)
		return nil, nil, err
	}
	if !valueOk {
		common.Log.Warningf("Failed to unmarshal value during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling value from NATS %v message", msg.Subject)
		return nil, nil, err
	}
	if !paramsOk {
		common.Log.Warningf("Failed to unmarshal params during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling params from NATS %v message", msg.Subject)
		return nil, nil, err
	}
	if !publishedAtOk {
		common.Log.Warningf("Failed to unmarshal published_at during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error unmarshaling published_at from NATS %v message", msg.Subject)
		return nil, nil, err
	}

	// TODO contract flag?
	contract := &contract.Contract{}
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
		err := fmt.Errorf("Error converting accountID and walletID to uuid from NATS %v message", msg.Subject)
		return nil, nil, err
	}

	// ensure we only have EITHER an accountID OR a walletID
	if accountID != nil {
		if walletID != nil {
			common.Log.Warningf("Both account_id and wallet_id present in NATS %v message", msg.Subject)
			err := fmt.Errorf("Error in tx. Both accountID and walletID present in NATS %v message", msg.Subject)
			return nil, nil, err
		}
	}

	// ensure we only have EITHER an accountID OR a walletID
	if walletID != nil {
		if accountID != nil {
			common.Log.Warningf("Both account_id and wallet_id present in NATS %v message", msg.Subject)
			err := fmt.Errorf("Error in tx. Both accountID and walletID present in NATS %v message", msg.Subject)
			return nil, nil, err
		}
	}

	publishedAtTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		common.Log.Warningf("Failed to parse published_at as RFC3339 timestamp during NATS %v message handling; %s", msg.Subject, err.Error())
		err := fmt.Errorf("Error parsing published_at time from NATS %v message", msg.Subject)
		return nil, nil, err
	}

	// convert value to float64 (shouldn't this be uint64?) if it's present
	valueFloat, valueFloatOk := value.(float64)
	if !valueFloatOk {
		common.Log.Warningf("Failed to unmarshal value during NATS %v message handling", msg.Subject)
		err := fmt.Errorf("Error converting value to float64 from NATS %v message", msg.Subject)
		return nil, nil, err
	}

	// handle any provided nonce
	var nonceUint *uint64
	txNonce, txNonceOk := txParams["nonce"].(float64)
	if !txNonceOk {
		nonceUint = nil
	}
	if txNonceOk {
		nonce := uint64(txNonce)
		nonceUint = &nonce
	}

	// handle any provided gas price
	var gasPrice *float64
	gasPriceFloat, gasPriceFloatOk := txParams["gas_price"].(float64)
	if !gasPriceFloatOk {
		gasPrice = nil
	}
	if gasPriceFloatOk {
		gasPrice = &gasPriceFloat
	}

	var parameters Parameters

	parameters.ContractID = contract.ContractID
	parameters.Data = &data
	parameters.AccountID = accountID
	parameters.WalletID = walletID
	parameters.Path = &hdDerivationPath
	parameters.Value = &valueFloat
	parameters.PublishedAt = &publishedAt
	parameters.GasPrice = gasPrice

	var tx Transaction

	// TODO note the contract deps on this tx create
	tx = Transaction{
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
		Parameters:     &parameters,
		Nonce:          nonceUint,
		Message:        msg,
	}

	// replaces this tx with the db tx (if it exists and matches)
	replacementTx, err := tx.replaceWithDatabaseTxIfExists(db, ref)
	if err != nil {
		common.Log.Debugf("Error retrieving tx ref %s from database", *tx.Ref)
		return nil, nil, err
	}

	//common.Log.Debugf("XXX: Finished processing NATS create message. tx ref %s", *tx.Ref)

	if replacementTx != nil {
		common.Log.Debugf("Found tx ref %s in db", *tx.Ref)
		// HACK
		status := replacementTx.Status
		if status != nil {
			if *status == "success" {
				return nil, contract, nil
			}
		}
		tx.ID = replacementTx.ID
		tx.PublishedAt = replacementTx.PublishedAt
		tx.Nonce = replacementTx.Nonce
		tx.Status = replacementTx.Status
		return &tx, contract, nil
	} else {
		common.Log.Debugf("Did not find tx ref %s in db", *tx.Ref)
		return &tx, contract, nil
	}
}

func (tx *Transaction) replaceWithDatabaseTxIfExists(db *gorm.DB, ref string) (*Transaction, error) {
	var dbTx Transaction
	var err error
	msg := tx.Message

	db.Where("ref = ?", ref).Find(&dbTx)
	if dbTx.ID != uuid.Nil {
		// we have this tx ref from before
		// check that the parameters are the same and we don't have an attempt to replay a ref
		if dbTx.ApplicationID != nil && tx.ApplicationID != nil {
			if *dbTx.ApplicationID != *tx.ApplicationID {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.OrganizationID != nil && tx.OrganizationID != nil {
			if *dbTx.OrganizationID != *tx.OrganizationID {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.AccountID != nil && tx.AccountID != nil {
			if *dbTx.AccountID != *tx.AccountID {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.WalletID != nil && tx.WalletID != nil {
			if *dbTx.WalletID != *tx.WalletID {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.Path != nil && tx.Path != nil {
			if *dbTx.Path != *tx.Path {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.Value != nil && tx.Value != nil {
			if dbTx.Value.value.Cmp(tx.Value.value) != 0 {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if dbTx.Data != nil && tx.Data != nil {
			if *dbTx.Data != *tx.Data {
				err = fmt.Errorf("Data mismatch in duplicate transaction. Transaction rejected.")
			}
		}

		if err != nil {
			return nil, err
		} else {
			// replace the tx with the one from the db
			dbTx.Message = msg
			// TODO check if anything else is needed other than the ID
			return &dbTx, nil
		}
	}
	return nil, nil
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

	// checks if the message is currently in flight (being procesed)
	err = processMessageStatus(key)
	if err != nil {
		common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
		return
	}

	common.Log.Debugf("XXX: ConsumeTxExecMsg, about to create tx with ref: %s", *tx.Ref)
	if tx.Create(db) {
		common.Log.Debugf("ACK: ConsumeTxExecMsg, about to ack tx with ref: %s", *tx.Ref)
		err = msg.Ack()
		if err != nil {
			common.Log.Debugf("ACK: ConsumeTxExecMsg, error acking tx ref: %s. Error: %s", *tx.Ref, err.Error())
		}
		common.Log.Debugf("ACK: ConsumeTxExecMsg, msg acked tx ref: %s", *tx.Ref)
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
		nack(msg, fmt.Sprintf("Failed to unmarshal tx finalize message; %s", err.Error()), true)
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
	db.Where("hash = ? AND status IN (?, ?, ?, ?)", hash, "ready", "broadcast", "pending", "failed").Find(&tx)
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

	// CHECKME when this is triggered, it populates the stats correctly
	tx.updateStatus(db, "success", nil)
	result := db.Save(&tx)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			tx.Errors = append(tx.Errors, &api.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
		common.Log.Debugf("IDEMPOTENT: error updating tx ref %s. Error: %s", *tx.Ref, err.Error())
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

	address, err := signer.Address()
	if err != nil {
		// let's try processing this one again
		common.Log.Debugf(fmt.Sprintf("Failed to get account address from signer for tx ref %s. Error: %s", *tx.Ref, err.Error()))
		natsutil.AttemptNack(msg, txReceiptMsgTimeout)
		return
	}

	err = tx.fetchReceipt(db, signer.Network, *address)
	if err != nil {
		// TODO got a panic here on *tx.hash (removed temporarily)
		common.Log.Debugf(fmt.Sprintf("Failed to fetch tx receipt for tx ref %s. Error: %s", *tx.Ref, err.Error()))
		natsutil.AttemptNack(msg, txReceiptMsgTimeout)
	} else {
		common.Log.Debugf("Fetched tx receipt for tx ref %s, hash: %s", *tx.Ref, *tx.Hash)

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

	// key = common.StringOrNil(fmt.Sprintf("nchain.tx.receipt%s", transactionID))
	// err = processMessageStatus(*key)
	// if err != nil {
	// 	common.Log.Debugf("Error processing message status for key %s. Error: %s", key, err.Error())
	// 	return
	// }

	common.Log.Debugf("XXX: Starting processTxReceipt for tx ref: %s", *tx.Ref)
	// TODO occasionally throws panic on startup
	// caused by vault not being available, so the signer.address is not populated (:803-ish)
	// process the receipts asynchronously
	go processTxReceipt(msg, tx, key, db)
}
