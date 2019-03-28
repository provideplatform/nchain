package tx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/consumer"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/wallet"
	provide "github.com/provideservices/provide-go"
)

const natsTxSubject = "goldmine.tx"
const natsTxMaxInFlight = 2048
const natsTxCreateSubject = "goldmine.tx.create"
const natsTxCreateMaxInFlight = 2048
const natsTxFinalizeSubject = "goldmine.tx.finalize"
const natsTxFinalizeMaxInFlight = 2048
const natsTxReceiptSubject = "goldmine.tx.receipt"
const natsTxReceiptMaxInFlight = 2048

const txAckWait = time.Second * 10
const txCreateAckWait = time.Second * 10
const txFinalizeAckWait = time.Second * 10
const txReceiptAckWait = time.Second * 10

var waitGroup sync.WaitGroup

func init() {
	natsConnection := consumer.GetNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsTxSubscriptions(natsConnection, &waitGroup)
	createNatsTxCreateSubscriptions(natsConnection, &waitGroup)
	createNatsTxFinalizeSubscriptions(natsConnection, &waitGroup)
	createNatsTxReceiptSubscriptions(natsConnection, &waitGroup)
}

func createNatsTxSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txSubscription, err := natsConnection.QueueSubscribe(natsTxSubject, natsTxSubject, consumeTxMsg, stan.SetManualAckMode(), stan.AckWait(txAckWait), stan.MaxInflight(natsTxMaxInFlight), stan.DurableName(natsTxSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				wg.Done()
				return
			}
			defer txSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			wg.Wait()
		}()
	}
}

func createNatsTxCreateSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txFinalizeSubscription, err := natsConnection.QueueSubscribe(natsTxCreateSubject, natsTxCreateSubject, consumeTxCreateMsg, stan.SetManualAckMode(), stan.AckWait(txCreateAckWait), stan.MaxInflight(natsTxCreateMaxInFlight), stan.DurableName(natsTxCreateSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", txFinalizeSubscription)
				wg.Done()
				return
			}
			defer txFinalizeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", txFinalizeSubscription)

			wg.Wait()
		}()
	}
}

func createNatsTxFinalizeSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txFinalizeSubscription, err := natsConnection.QueueSubscribe(natsTxFinalizeSubject, natsTxFinalizeSubject, consumeTxFinalizeMsg, stan.SetManualAckMode(), stan.AckWait(txFinalizeAckWait), stan.MaxInflight(natsTxFinalizeMaxInFlight), stan.DurableName(natsTxFinalizeSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", txFinalizeSubscription)
				wg.Done()
				return
			}
			defer txFinalizeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", txFinalizeSubscription)

			wg.Wait()
		}()
	}
}

func createNatsTxReceiptSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			txReceiptSubscription, err := natsConnection.QueueSubscribe(natsTxReceiptSubject, natsTxReceiptSubject, consumeTxReceiptMsg, stan.SetManualAckMode(), stan.AckWait(txReceiptAckWait), stan.MaxInflight(natsTxReceiptMaxInFlight), stan.DurableName(natsTxReceiptSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxReceiptSubject)
				wg.Done()
				return
			}
			defer txReceiptSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsTxReceiptSubject)

			wg.Wait()
		}()
	}
}

func consumeTxCreateMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal tx finalize message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	contractID, contractIDOk := params["contract_id"]
	data, dataOk := params["data"].(string)
	walletID, walletIDOk := params["wallet_id"].(string)
	value, valueOk := params["value"]
	txParams, paramsOk := params["params"].(map[string]interface{})
	publishedAt, publishedAtOk := params["published_at"].(string)

	if !contractIDOk {
		common.Log.Warningf("Failed to unmarshal contract_id during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}
	if !dataOk {
		common.Log.Warningf("Failed to unmarshal data during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}
	if !walletIDOk {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}
	if !valueOk {
		common.Log.Warningf("Failed to unmarshal value during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}
	if !paramsOk {
		common.Log.Warningf("Failed to unmarshal params during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}
	if !publishedAtOk {
		common.Log.Warningf("Failed to unmarshal published_at during NATS %v message handling", msg.Subject)
		consumer.Nack(msg)
		return
	}

	contract := &contract.Contract{}
	db := dbconf.DatabaseConnection()
	db.Where("id = ?", contractID).Find(&contract)

	walletUUID, err := uuid.FromString(walletID)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling; %s", msg.Subject, err.Error())
		consumer.Nack(msg)
		return
	}

	publishedAtTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling; %s", msg.Subject, err.Error())
		consumer.Nack(msg)
		return
	}

	tx := &Transaction{
		ApplicationID: contract.ApplicationID,
		Data:          common.StringOrNil(data),
		NetworkID:     contract.NetworkID,
		WalletID:      &walletUUID,
		To:            nil,
		Value:         &TxValue{value: big.NewInt(int64(value.(float64)))},
		PublishedAt:   &publishedAtTime,
	}
	tx.setParams(txParams)

	if tx.Create() {
		contract.TransactionID = &tx.ID
		db.Save(&contract)
		common.Log.Debugf("Contract compiled from source and deployed via tx: %s", *tx.Hash)
		msg.Ack()
	} else {
		common.Log.Warningf("Failed to deploy compiled contract; tx failed with %d error(s); %s", len(tx.Errors), *tx.Errors[0].Message)
		consumer.Nack(msg)
	}
}

func txResponsefunc(tx *Transaction, c *contract.Contract, network *network.Network, methodDescriptor string, method string, abiMethod *abi.Method, params []interface{}) (*interface{}, *interface{}, error) {
	var err error
	if abiMethod != nil {
		common.Log.Debugf("Attempting to encode %d parameters %s prior to executing method %s on contract: %s", len(params), params, methodDescriptor, c.ID)
		invocationSig, err := provide.EVMEncodeABI(abiMethod, params...)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to encode %d parameters prior to attempting execution of %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
		}

		data := fmt.Sprintf("0x%s", ethcommon.Bytes2Hex(invocationSig))
		tx.Data = &data

		if abiMethod.Const {
			common.Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
			network, _ := tx.GetNetwork()
			client, err := provide.EVMDialJsonRpc(network.ID.String(), network.RpcURL())
			msg := tx.asEthereumCallMsg(0, 0)
			result, err := client.CallContract(context.TODO(), msg, nil)
			if err != nil {
				err = fmt.Errorf("Failed to read constant method %s on contract: %s; %s", method, c.ID, err.Error())
				return nil, nil, err
			}
			var out interface{}
			if len(abiMethod.Outputs) == 1 {
				err = abiMethod.Outputs.Unpack(&out, result)
				if err == nil {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[0].Type)
					common.Log.Debugf("Attempting to marshal %s result of constant contract execution of %s on contract: %s", typestr, methodDescriptor, c.ID)
					switch out.(type) {
					case [32]byte:
						arrbytes, _ := out.([32]byte)
						out = string(bytes.Trim(arrbytes[:], "\x00"))
					case [][32]byte:
						arrbytesarr, _ := out.([][32]byte)
						vals := make([]string, len(arrbytesarr))
						for i, item := range arrbytesarr {
							vals[i] = string(bytes.Trim(item[:], "\x00"))
						}
						out = vals
					default:
						common.Log.Debugf("Noop during marshaling of constant contract execution of %s on contract: %s", methodDescriptor, c.ID)
					}
				}
			} else if len(abiMethod.Outputs) > 1 {
				// handle tuple
				vals := make([]interface{}, len(abiMethod.Outputs))
				for i := range abiMethod.Outputs {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[i].Type)
					common.Log.Debugf("Reflectively adding type hint for unpacking %s in return values slot %v", typestr, i)
					typ, err := abi.NewType(typestr)
					if err != nil {
						return nil, nil, fmt.Errorf("Failed to reflectively add appropriately-typed %s value for in return values slot %v); %s", typestr, i, err.Error())
					}
					vals[i] = reflect.New(typ.Type).Interface()
				}
				err = abiMethod.Outputs.Unpack(&vals, result)
				out = vals
				common.Log.Debugf("Unpacked %v returned values from read of constant %s on contract: %s; values: %s", len(vals), methodDescriptor, c.ID, vals)
				if vals != nil && len(vals) == abiMethod.Outputs.LengthNonIndexed() {
					err = nil
				}
			}
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to read constant %s on contract: %s (signature with encoded parameters: %s); %s", methodDescriptor, c.ID, *tx.Data, err.Error())
			}
			return nil, &out, nil
		}

		var txResponse *contract.ContractExecutionResponse
		if tx.Create() {
			common.Log.Debugf("Executed %s on contract: %s", methodDescriptor, c.ID)
			if tx.Response != nil {
				txResponse = tx.Response
			}
		} else {
			common.Log.Debugf("Failed tx errors: %s", *tx.Errors[0].Message)
			txParams := tx.ParseParams()
			publicKey, publicKeyOk := txParams["public_key"].(interface{})
			privateKey, privateKeyOk := txParams["private_key"].(interface{})
			gas, gasOk := txParams["gas"].(float64)
			if !gasOk {
				gas = float64(0)
			}
			var nonce *uint64
			if nonceFloat, nonceOk := txParams["nonce"].(float64); nonceOk {
				nonceUint := uint64(nonceFloat)
				nonce = &nonceUint
			}
			delete(txParams, "private_key")
			tx.setParams(txParams)

			if publicKeyOk && privateKeyOk {
				common.Log.Debugf("Attempting to execute %s on contract: %s; arbitrarily-provided signer for tx: %s; gas supplied: %v", methodDescriptor, c.ID, publicKey, gas)
				tx.SignedTx, tx.Hash, err = provide.EVMSignTx(network.ID.String(), network.RpcURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), nonce, uint64(gas))
				if err == nil {
					if signedTx, ok := tx.SignedTx.(*types.Transaction); ok {
						err = provide.EVMBroadcastSignedTx(network.ID.String(), network.RpcURL(), signedTx)
					} else {
						err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", tx.SignedTx)
						common.Log.Warning(err.Error())
					}
				}

				if err != nil {
					err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed using arbitrarily-provided signer: %s; %s", methodDescriptor, c.ID, *tx.Data, publicKey, err.Error())
					common.Log.Warning(err.Error())
				}
			} else {
				err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", methodDescriptor, c.ID, *tx.Data)
				common.Log.Warning(err.Error())
			}
		}

		if txResponse != nil {
			common.Log.Debugf("Received response to tx broadcast attempt calling method %s on contract: %s", methodDescriptor, c.ID)

			var out interface{}
			switch (txResponse.Receipt).(type) {
			case []byte:
				out = (txResponse.Receipt).([]byte)
				common.Log.Debugf("Received response: %s", out)
			case types.Receipt:
				client, _ := provide.EVMDialJsonRpc(network.ID.String(), network.RpcURL())
				receipt := txResponse.Receipt.(*types.Receipt)
				txdeets, _, err := client.TransactionByHash(context.TODO(), receipt.TxHash)
				if err != nil {
					err = fmt.Errorf("Failed to retrieve %s transaction by tx hash: %s", *network.Name, *tx.Hash)
					common.Log.Warning(err.Error())
					return nil, nil, err
				}
				out = txdeets
			default:
				// no-op
				common.Log.Warningf("Unhandled transaction receipt type; %s", tx.Response.Receipt)
			}
			return &out, nil, nil
		}
	} else {
		err = fmt.Errorf("Failed to execute method %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
	}
	return nil, nil, err
}

func txCreatefunc(tx *Transaction, c *contract.Contract, n *network.Network, walletID *uuid.UUID, execution *contract.ContractExecution, _txParamsJSON *json.RawMessage) (*contract.ContractExecutionResponse, error) {
	db := dbconf.DatabaseConnection()
	publishedAt := execution.PublishedAt
	method := execution.Method
	params := execution.Params
	ref := execution.Ref
	value := execution.Value

	tx = &Transaction{
		ApplicationID: c.ApplicationID,
		UserID:        nil,
		NetworkID:     c.NetworkID,
		WalletID:      walletID,
		To:            c.Address,
		Value:         &TxValue{value: value},
		Params:        _txParamsJSON,
		Ref:           ref,
	}

	if publishedAt != nil {
		tx.PublishedAt = publishedAt
	}

	var receipt *interface{}
	var response *interface{}

	txResponseCallback := func(c *contract.Contract, network *network.Network, methodDescriptor string, method string, abiMethod *abi.Method, params []interface{}) (*interface{}, *interface{}, error) {
		return txResponsefunc(tx, c, network, methodDescriptor, method, abiMethod, params)
	}

	var err error
	if n.IsEthereumNetwork() {
		receipt, response, err = c.ExecuteEthereumContract(n, txResponseCallback, method, params)
	} else {
		err = fmt.Errorf("unsupported network: %s", *n.Name)
	}

	if err != nil {
		desc := err.Error()
		tx.updateStatus(db, "failed", &desc)
		return nil, fmt.Errorf("Unable to execute %s contract; %s", *n.Name, err.Error())
	}

	accessedAt := time.Now()
	go func() {
		c.AccessedAt = &accessedAt
		db.Save(c)
	}()

	if tx.Response == nil {
		tx.Response = &contract.ContractExecutionResponse{
			Response:    response,
			Receipt:     receipt,
			Traces:      tx.Traces,
			Transaction: tx,
			Ref:         ref,
		}
	} else if tx.Response.Transaction == nil {
		tx.Response.Transaction = tx
	}

	return tx.Response, nil
}

func wfunc(w interface{}, txParams *map[string]interface{}) *uuid.UUID {
	db := dbconf.DatabaseConnection()
	tmpWallet := &wallet.Wallet{}
	wallet := w.(*wallet.Wallet)
	params := *txParams
	if wallet != nil {
		// need reflection to work with wallet here, or...
		if wallet.ID != uuid.Nil {
			return &wallet.ID
		} else if wallet.Address != "" {
			db.Where("address = ?", wallet.Address).Find(&tmpWallet)
			if tmpWallet != nil && tmpWallet.ID != uuid.Nil {
				return &tmpWallet.ID
			}
		}
		if common.StringOrNil(wallet.Address) != nil && wallet.PrivateKey != nil {
			params["public_key"] = wallet.Address
			params["private_key"] = wallet.PrivateKey
		}
	}
	return &uuid.Nil
}

func consumeTxMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming %d-byte NATS tx message on subject: %s", msg.Size(), msg.Subject)

	execution := &contract.ContractExecution{}
	err := json.Unmarshal(msg.Data, execution)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal contract execution during NATS tx message handling")
		consumer.Nack(msg)
		return
	}

	if execution.ContractID == nil {
		common.Log.Errorf("Invalid tx message; missing contract_id")
		consumer.Nack(msg)
		return
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
		if execution.Wallet != nil && *executionWalletID != *execution.WalletID {
			common.Log.Errorf("Invalid tx message specifying a wallet_id and wallet")
			consumer.Nack(msg)
			return
		}
		wallet := &wallet.Wallet{}
		wallet.SetID(*execution.WalletID)
		execution.Wallet = wallet
	}

	kontract := &contract.Contract{}
	dbconf.DatabaseConnection().Where("id = ?", *execution.ContractID).Find(&kontract)
	if kontract == nil || kontract.ID == uuid.Nil {
		common.Log.Errorf("Unable to execute contract; contract not found: %s", kontract.ID)
		consumer.Nack(msg)
		return
	}

	var tx Transaction
	txCreateFn := func(c *contract.Contract, network *network.Network, walletID *uuid.UUID, execution *contract.ContractExecution, _txParamsJSON *json.RawMessage) (*contract.ContractExecutionResponse, error) {
		return txCreatefunc(&tx, c, network, walletID, execution, _txParamsJSON)
	}

	walletFn := func(w interface{}, txParams *map[string]interface{}) *uuid.UUID {
		return wfunc(w.(*wallet.Wallet), txParams)
	}

	executionResponse, err := kontract.ExecuteFromTx(execution, walletFn, txCreateFn)

	if err != nil {
		common.Log.Warningf("Failed to execute contract; %s", err.Error())
		consumer.Nack(msg)
	} else {
		common.Log.Debugf("Executed contract; tx: %s", executionResponse)
		msg.Ack()
	}
}

func consumeTxFinalizeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS tx finalize message: %s", msg)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal tx finalize message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	block, blockOk := params["block"].(float64)
	blockTimestampStr, blockTimestampStrOk := params["block_timestamp"].(string)
	finalizedAtStr, finalizedAtStrOk := params["finalized_at"].(string)
	hash, hashOk := params["hash"].(string)

	if !blockOk {
		common.Log.Warningf("Failed to finalize tx; no block provided")
		consumer.Nack(msg)
		return
	}
	if !blockTimestampStrOk {
		common.Log.Warningf("Failed to finalize tx; no block timestamp provided")
		consumer.Nack(msg)
		return
	}
	if !finalizedAtStrOk {
		common.Log.Warningf("Failed to finalize tx; no finalized at timestamp provided")
		consumer.Nack(msg)
		return
	}
	if !hashOk {
		common.Log.Warningf("Failed to finalize tx; no hash provided")
		consumer.Nack(msg)
		return
	}

	blockTimestamp, err := time.Parse(time.RFC3339, blockTimestampStr)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal block_timestamp during NATS %v message handling; %s", msg.Subject, err.Error())
		consumer.Nack(msg)
		return
	}

	finalizedAt, err := time.Parse(time.RFC3339, finalizedAtStr)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal finalized_at during NATS %v message handling; %s", msg.Subject, err.Error())
		consumer.Nack(msg)
		return
	}

	tx := &Transaction{}
	db := dbconf.DatabaseConnection()
	db.Where("hash = ?", hash).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		common.Log.Warningf("Failed to set block and finalized_at timestamp on tx: %s", hash)
		consumer.Nack(msg)
		return
	}

	blockNumber := uint64(block)

	tx.Block = &blockNumber
	tx.BlockTimestamp = &blockTimestamp
	tx.FinalizedAt = &finalizedAt
	if tx.BroadcastAt != nil {
		if tx.PublishedAt != nil {
			publishLatency := uint64(tx.BroadcastAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.PublishLatency = &publishLatency

			e2eLatency := uint64(tx.FinalizedAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.E2ELatency = &e2eLatency
		}

		broadcastLatency := uint64(tx.FinalizedAt.Sub(*tx.BroadcastAt)) / uint64(time.Millisecond)
		tx.BroadcastLatency = &broadcastLatency
	}

	tx.Status = common.StringOrNil("success")
	result := db.Save(&tx)
	errors := result.GetErrors()
	if len(errors) > 0 {
		for _, err := range errors {
			tx.Errors = append(tx.Errors, &provide.Error{
				Message: common.StringOrNil(err.Error()),
			})
		}
	}
	if len(tx.Errors) > 0 {
		common.Log.Warningf("Failed to set block and finalized_at timestamp on tx: %s; error: %s", hash, tx.Errors[0].Message)
		consumer.Nack(msg)
		return
	}

	msg.Ack()
}

func consumeTxReceiptMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	db := dbconf.DatabaseConnection()

	var tx *Transaction

	err := json.Unmarshal(msg.Data, &tx)
	if err != nil {
		desc := fmt.Sprintf("Failed to umarshal tx receipt message; %s", err.Error())
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		consumer.Nack(msg)
		return
	}

	tx.Reload()

	network, err := tx.GetNetwork()
	if err != nil {
		desc := fmt.Sprintf("Failed to resolve tx network; %s", err.Error())
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		consumer.Nack(msg)
		return
	}

	wallet, err := tx.GetWallet()
	if err != nil {
		desc := fmt.Sprintf("Failed to resolve tx wallet; %s", err.Error())
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		consumer.Nack(msg)
		return
	}

	err = tx.fetchReceipt(db, network, wallet)
	if err != nil {
		if msg.Redelivered { // FIXME-- implement proper dead-letter logic; only set tx to failed upon deadletter
			desc := fmt.Sprintf("Failed to fetch tx receipt; %s", err.Error())
			common.Log.Warningf(desc)
			tx.updateStatus(db, "failed", common.StringOrNil(desc))
		}

		consumer.Nack(msg)
	} else {
		msg.Ack()
	}
}
