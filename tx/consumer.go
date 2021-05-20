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
	"github.com/provideapp/nchain/wallet"
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
const txReceiptAckWait = time.Second * 60
const txReceiptMsgTimeout = int64(txReceiptAckWait * 15)

const txInFlightStatus = "IN_FLIGHT"
const txRetryRequiredStatus = "RETRY_REQUIRED"

var waitGroup sync.WaitGroup

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

func consumeTxCreateMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal tx creation message; %s", err.Error())
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
	common.Log.Debugf("XXX: ConsumeTxCreateMsg, about to create tx with ref: %s", *tx.Ref)

	// REDIS locking to prevent dup message processing
	// Objective
	// after the ackwait timeout, NATS redelivers the nchain.tx.create message
	// as the original message might still be processing
	// we want to prevent dupe transactions getting created
	// so we will store 2 nchain.tx.create statuses in redis
	// - IN_FLIGHT (tx is being processed)
	// - COMPLETED (tx is done, either acked or nacked)
	// if we ack or nack, we will set the status to completed
	// based on tx ref as the key
	// and if we haven't seen it before, we will set it to IN_FLIGHT
	// if it's IN_FLIGHT, we will dump the msg, and wait for it to come around again
	// (in case it fails and needs to be reprocessed)
	// as a precursor to a replay-focussed concurrent tx mechanism
	// if it's COMPLETED, we'll log this, as this shouldn't ever happen (should be already acked or nacked)

	txStatus, err := redisutil.Get(*tx.Ref)
	if err != nil {
		common.Log.Debugf("XXX: Error getting tx in flight status for tx ref %s", *tx.Ref)
	}
	if txStatus == nil {
		common.Log.Debugf("XXX: No tx in flight status found for tx ref: %s", *tx.Ref)
		redisutil.WithRedlock(*tx.Ref, func() error {
			redisutil.Set(*tx.Ref, txInFlightStatus, nil)
			return nil
		})
	} else {
		switch *txStatus {
		case txInFlightStatus:
			common.Log.Debugf("XXX: tx ref %s is in flight, waiting for it to finish", *tx.Ref)
			// tx is in flight, so wait for it to finish
			return
		case txRetryRequiredStatus:
			common.Log.Debugf("XXX: tx ref %s retry commencing", *tx.Ref)
			err = redisutil.WithRedlock(*tx.Ref, func() error {
				redisutil.Set(*tx.Ref, txInFlightStatus, nil)
				return nil
			})
			if err != nil {
				common.Log.Debugf("XXX: Error resetting status to retry required. Error: %s", err.Error())
			}
		}
	}

	if tx.Create(db) {
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, created tx with ref: %s", *tx.Ref)
		contract.TransactionID = &tx.ID
		db.Save(&contract)
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, updated contract with txID %s for tx ref: %s", tx.ID, *tx.Ref)
		common.Log.Debugf("Transaction execution successful: %s", *tx.Hash)
		err = msg.Ack()
		if err != nil {
			common.Log.Debugf("XXX: ConsumeTxCreateMsg, error acking tx ref: %s", *tx.Ref)
		}
		common.Log.Debugf("XXX: ConsumeTxCreateMsg, msg acked tx ref: %s", *tx.Ref)
	} else {
		errmsg := fmt.Sprintf("Failed to execute transaction; tx failed with %d error(s)", len(tx.Errors))
		for _, err := range tx.Errors {
			errmsg = fmt.Sprintf("%s\n\t%s", errmsg, *err.Message)
		}
		// getting rid of the subsidize code that's managed by bookie now

		// remove the in-flight status to this can be replayed
		common.Log.Debugf("XXX: Tx ref %s failed... Setting tx status to retry required", *tx.Ref)
		err = redisutil.WithRedlock(*tx.Ref, func() error {
			err := redisutil.Set(*tx.Ref, txRetryRequiredStatus, nil)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			common.Log.Debugf("XXX: Error resetting in flight status for tx ref: %s. Error: %s", *tx.Ref, err.Error())
		}

		common.Log.Debugf("XXX: getting updated status for tx ref: %s", *tx.Ref)
		updatedTxStatus, err := redisutil.Get(*tx.Ref)
		if err != nil {
			common.Log.Debugf("XXX: Error getting tx in flight status for tx ref %s", *tx.Ref)
		}
		common.Log.Debugf("XXX: updatedTxStatus: :%s: for txref: %s", *updatedTxStatus, *tx.Ref)
		common.Log.Debugf("XXX: Tx ref %s failed. Attempting nacking", *tx.Ref)
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

	execution := &contract.Execution{}
	err := json.Unmarshal(msg.Data, execution)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal contract execution during NATS tx message handling")
		natsutil.Nack(msg)
		return
	}

	if execution.ContractID == nil {
		common.Log.Errorf("Invalid tx message; missing contract_id")
		natsutil.Nack(msg)
		return
	}

	if execution.AccountID != nil && *execution.AccountID != uuid.Nil {
		var executionAccountID *uuid.UUID
		if executionAccount, executionAccountOk := execution.Account.(map[string]interface{}); executionAccountOk {
			if executionAccountIDStr, executionAccountIDStrOk := executionAccount["id"].(string); executionAccountIDStrOk {
				execAccountUUID, err := uuid.FromString(executionAccountIDStr)
				if err == nil {
					executionAccountID = &execAccountUUID
				}
			}
		}
		if execution.Account != nil && execution.AccountID != nil && *executionAccountID != *execution.AccountID {
			common.Log.Errorf("Invalid tx message specifying a account_id and account")
			natsutil.Nack(msg)
			return
		}
		account := &wallet.Account{}
		account.SetID(*execution.AccountID)
		execution.Account = account
	}

	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		var executionWalletID *uuid.UUID
		if executionWallet, executionWalletOk := execution.Wallet.(map[string]interface{}); executionWalletOk {
			if executionWalletIDStr, executionWalletIDStrOk := executionWallet["id"].(string); executionWalletIDStrOk {
				execWalletUUID, err := uuid.FromString(executionWalletIDStr)
				if err == nil {
					executionWalletID = &execWalletUUID
				}
			}
		}
		if execution.Wallet != nil && execution.WalletID != nil && *executionWalletID != *execution.WalletID {
			common.Log.Errorf("Invalid tx message specifying a wallet_id and wallet")
			natsutil.Nack(msg)
			return
		}
		wallet := &wallet.Wallet{}
		wallet.SetID(*execution.WalletID)
		execution.Wallet = wallet
	}

	db := dbconf.DatabaseConnection()

	cntract := &contract.Contract{}
	db.Where("id = ?", *execution.ContractID).Find(&cntract)
	if cntract == nil || cntract.ID == uuid.Nil {
		db.Where("address = ?", *execution.ContractID).Find(&cntract)
	}
	if cntract == nil || cntract.ID == uuid.Nil {
		common.Log.Errorf("Unable to execute contract; contract not found: %s", cntract.ID)
		natsutil.Nack(msg)
		return
	}

	executionResponse, err := executeTransaction(cntract, execution)
	if err != nil {
		common.Log.Debugf("contract execution failed; %s", err.Error())

		// CHECKME - this functionality is now in bookie, and shouldn't be replicated here
		// if execution.AccountAddress != nil {
		// 	networkSubsidyFaucetDripValue := int64(100000000000000000) // FIXME-- configurable
		// 	_subsidize := strings.Contains(strings.ToLower(err.Error()), "insufficient funds") && tx.shouldSubsidize()

		// 	if _subsidize {
		// 		common.Log.Debugf("contract execution failed due to insufficient funds but tx subsidize flag is set; requesting subsidized tx funding for target network: %s", cntract.NetworkID)
		// 		faucetBeneficiaryAddress := *execution.AccountAddress
		// 		err = subsidize(
		// 			db,
		// 			cntract.NetworkID,
		// 			faucetBeneficiaryAddress,
		// 			networkSubsidyFaucetDripValue,
		// 			int64(210000*2),
		// 		)
		// 		if err == nil {
		// 			db.Where("ref = ?", execution.Ref).Find(&tx)
		// 			if tx != nil && tx.ID != uuid.Nil {
		// 				db.Delete(&tx) // Drop tx that had insufficient funds so its hash can be rebroadcast...
		// 			}
		// 			common.Log.Debugf("faucet subsidy transaction broadcast; beneficiary: %s", faucetBeneficiaryAddress)
		// 		}
		// 	}

		// } else {
		// 	common.Log.Warningf("Failed to execute contract; %s", err.Error())
		// }

		natsutil.AttemptNack(msg, txMsgTimeout)
	} else {
		logmsg := fmt.Sprintf("Executed contract: %s", *cntract.Address)
		if executionResponse != nil && executionResponse.Response != nil {
			logmsg = fmt.Sprintf("%s; response: %s", logmsg, executionResponse.Response)
		}
		common.Log.Debug(logmsg)

		msg.Ack()
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

// TODO add a channel output for the acking nacking?
func processTxReceipt(msg *stan.Msg, tx *Transaction, db *gorm.DB) {

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
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed tx receipt message; %s", r)
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

	go processTxReceipt(msg, tx, db)

	// signer, err := tx.signerFactory(db)
	// if err != nil {
	// 	desc := "failed to resolve tx signing account or HD wallet"
	// 	common.Log.Warningf(desc)
	// 	tx.updateStatus(db, "failed", common.StringOrNil(desc))
	// 	natsutil.Nack(msg)
	// 	return
	// }

	// err = tx.fetchReceipt(db, signer.Network, signer.Address())
	// if err != nil {
	// 	common.Log.Debugf(fmt.Sprintf("Failed to fetch tx receipt for tx hash %s. Error: %s", *tx.Hash, err.Error()))
	// 	natsutil.AttemptNack(msg, txReceiptMsgTimeout)
	// } else {
	// 	common.Log.Debugf("Fetched tx receipt for hash: %s", *tx.Hash)

	// 	common.Log.Debugf("XXX: receipt is: %+v", tx.Response.Receipt.(*provide.TxReceipt))
	// 	blockNumber := tx.Response.Receipt.(*provide.TxReceipt).BlockNumber
	// 	// if we have a block number in the receipt, and the tx has no block
	// 	// populate the block and finalized timestamp
	// 	if blockNumber != nil && tx.Block == nil {
	// 		receiptBlock := blockNumber.Uint64()
	// 		tx.Block = &receiptBlock
	// 		receiptFinalized := time.Now()
	// 		tx.FinalizedAt = &receiptFinalized
	// 		common.Log.Debugf("*** tx hash %s finalized in block %v at %s", *tx.Hash, blockNumber, receiptFinalized.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	// 	}
	// 	tx.updateStatus(db, "success", nil)
	// 	msg.Ack()
	// }
}
