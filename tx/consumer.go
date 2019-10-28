package tx

import (
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
	stan "github.com/nats-io/stan.go"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/wallet"
	provide "github.com/provideservices/provide-go"
)

// TODO: should this be calculated dynamically against average blocktime for the network and subscriptions reestablished?

const natsTxSubject = "goldmine.tx"
const natsTxMaxInFlight = 2048
const txAckWait = time.Second * 10
const txMsgTimeout = int64(txAckWait * 10)

const natsTxCreateSubject = "goldmine.tx.create"
const natsTxCreateMaxInFlight = 2048
const txCreateAckWait = time.Second * 10
const txCreateMsgTimeout = int64(txCreateAckWait * 10)

const natsTxFinalizeSubject = "goldmine.tx.finalize"
const natsTxFinalizeMaxInFlight = 2048
const txFinalizeAckWait = time.Second * 10
const txFinalizedMsgTimeout = int64(txFinalizeAckWait * 10)

const natsTxReceiptSubject = "goldmine.tx.receipt"
const natsTxReceiptMaxInFlight = 2048
const txReceiptAckWait = time.Second * 10
const txReceiptMsgTimeout = int64(txReceiptAckWait * 10)

var waitGroup sync.WaitGroup

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Tx package consumer configured to skip NATS streaming subscription setup")
		return
	}

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
			consumeTxMsg,
			txAckWait,
			natsTxMaxInFlight,
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
	walletID, walletIDOk := params["wallet_id"].(string)
	value, valueOk := params["value"]
	txParams, paramsOk := params["params"].(map[string]interface{})
	publishedAt, publishedAtOk := params["published_at"].(string)

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
	if !walletIDOk {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling", msg.Subject)
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

	walletUUID, err := uuid.FromString(walletID)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling; %s", msg.Subject, err.Error())
		natsutil.Nack(msg)
		return
	}

	publishedAtTime, err := time.Parse(time.RFC3339, publishedAt)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal wallet_id during NATS %v message handling; %s", msg.Subject, err.Error())
		natsutil.Nack(msg)
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

	if tx.Create(db) {
		contract.TransactionID = &tx.ID
		db.Save(&contract)
		common.Log.Debugf("Transaction execution successful: %s", *tx.Hash)
		msg.Ack()
	} else {
		common.Log.Warningf("Failed to execute transaction; tx failed with %d error(s); %s", len(tx.Errors), *tx.Errors[0].Message)
		natsutil.AttemptNack(msg, txCreateMsgTimeout)
	}
}

func txResponsefunc(tx *Transaction, c *contract.Contract, network *network.Network, methodDescriptor, method string, abiMethod *abi.Method, params []interface{}) (map[string]interface{}, error) {
	var err error
	var result []byte
	var receipt map[string]interface{}
	var response map[string]interface{}
	out := map[string]interface{}{}

	if network.IsEthereumNetwork() {
		if abiMethod != nil {

			common.Log.Debugf("Attempting to encode %d parameters %s prior to executing method %s on contract: %s", len(params), params, methodDescriptor, c.ID)
			invocationSig, err := provide.EVMEncodeABI(abiMethod, params...)
			if err != nil {
				return nil, fmt.Errorf("Failed to encode %d parameters prior to attempting execution of %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
			}

			data := fmt.Sprintf("0x%s", ethcommon.Bytes2Hex(invocationSig))
			tx.Data = &data

			if abiMethod.Const {
				common.Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
				client, err := provide.EVMDialJsonRpc(network.ID.String(), network.RPCURL())
				msg := tx.asEthereumCallMsg(0, 0)
				result, err = client.CallContract(context.TODO(), msg, nil)
				if err != nil {
					err = fmt.Errorf("Failed to read constant method %s on contract: %s; %s", method, c.ID, err.Error())
					return nil, err
				}
			} else {
				common.Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
				db := dbconf.DatabaseConnection()

				var txResponse *contract.ContractExecutionResponse
				if tx.Create(db) {
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
						tx.SignedTx, tx.Hash, err = provide.EVMSignTx(network.ID.String(), network.RPCURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), nonce, uint64(gas))
						if err == nil {
							if signedTx, ok := tx.SignedTx.(*types.Transaction); ok {
								err = provide.EVMBroadcastSignedTx(network.ID.String(), network.RPCURL(), signedTx)
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

					if txResponse.Traces != nil {
						if traces, tracesOk := txResponse.Traces.(*provide.EthereumTxTraceResponse); tracesOk {
							common.Log.Debugf("EVM tracing included in tx response")
							if len(traces.Result) > 0 {
								traceResult := traces.Result[0].Result
								if traceResult.Output != nil {
									result = []byte(*traceResult.Output)
								}
							}
						}
					} else {
						common.Log.Debugf("Received response to tx broadcast attempt calling method %s on contract: %s", methodDescriptor, c.ID)
						switch (txResponse.Receipt).(type) {
						case []byte:
							result = (txResponse.Receipt).([]byte)
							json.Unmarshal(result, &receipt)
						case types.Receipt:
							// client, _ := provide.EVMDialJsonRpc(network.ID.String(), network.RPCURL())
							txReceipt := txResponse.Receipt.(*types.Receipt)
							txReceiptJSON, _ := json.Marshal(txReceipt)
							json.Unmarshal(txReceiptJSON, &receipt)
							// txdeets, _, err := client.TransactionByHash(context.TODO(), txReceipt.TxHash)
							// if err != nil {
							// 	err = fmt.Errorf("Failed to retrieve %s transaction by tx hash: %s", *network.Name, *tx.Hash)
							// 	common.Log.Warning(err.Error())
							// 	return nil, err
							// }
							// txdeetsJSON, _ := json.Marshal(txdeets)
							// json.Unmarshal(txdeetsJSON, &out)
							out["receipt"] = receipt
							return out, nil
						default:
							// no-op
							common.Log.Warningf("Unhandled transaction receipt type; %s", tx.Response.Receipt)
						}
					}

					return out, nil
				}
			}

			if len(abiMethod.Outputs) == 1 {
				// var outptr interface{}
				typestr := fmt.Sprintf("%s", abiMethod.Outputs[0].Type)
				common.Log.Debugf("Reflectively adding type hint for unpacking %s in return value", typestr)
				// isTuple := strings.Index(typestr, "(") == 0
				// if isTuple {
				// 	common.Log.Debugf("Processing tuple for type-hinted retval: %s", typestr)
				// 	tupleLen := len(strings.Split(typestr, ","))
				// 	outptr = make([]interface{}, tupleLen)
				// }
				typ, _ := abi.NewType(typestr, nil)
				outptr := reflect.New(typ.Type).Interface()

				err = abiMethod.Outputs.Unpack(&outptr, result)
				if err == nil {
					common.Log.Debugf("Attempting to marshal %s result of constant contract execution of %s on contract: %s", typestr, methodDescriptor, c.ID)
					switch outptr.(type) {
					case [32]byte:
						arrbytes, _ := outptr.([32]byte)
						out["response"] = string(arrbytes[:]) //string(bytes.Trim(arrbytes[:], "\x00"))
					case [][32]byte:
						arrbytesarr, _ := outptr.([][32]byte)
						vals := make([]string, len(arrbytesarr))
						for i, item := range arrbytesarr {
							vals[i] = string(item[:]) //string(bytes.Trim(item[:], "\x00"))
						}
						out["response"] = vals
					default:
						common.Log.Debugf("Noop during marshaling of constant contract execution of %s on contract: %s", methodDescriptor, c.ID)
						out["response"] = outptr
					}
				}
			} else if len(abiMethod.Outputs) > 1 {
				response := map[string]interface{}{}
				// err = abiMethod.Outputs.UnpackIntoMap(response, result)
				// common.Log.Debugf("unpacked map: %s", response)

				vals := make([]interface{}, len(abiMethod.Outputs))
				for i := range abiMethod.Outputs {
					typestr := fmt.Sprintf("%s", abiMethod.Outputs[i].Type)
					common.Log.Debugf("Reflectively adding type hint for unpacking %s in return values slot %v", typestr, i)
					typ, err := abi.NewType(typestr, nil)
					if err != nil {
						return nil, fmt.Errorf("Failed to reflectively add appropriately-typed %s value for in return values slot %v); %s", typestr, i, err.Error())
					}
					vals[i] = reflect.New(typ.Type).Interface()
				}
				// err = abiMethod.Outputs.Unpack(&vals, result)
				// if err == nil {
				// 	for i := range vals {
				// 		response[abiMethod.Outputs[i].Name] = vals[i]
				// 	}
				// }
				for i := range vals {
					response[abiMethod.Outputs[i].Name] = vals[i]
				}
				out["response_map"] = response
				out["response"] = vals
				common.Log.Debugf("Unpacked %v returned values from execution of method %s on contract: %s; values: %s", len(vals), methodDescriptor, c.ID, vals)
				if vals != nil && len(vals) == abiMethod.Outputs.LengthNonIndexed() {
					err = nil
				}
			}

			if err != nil {
				return nil, fmt.Errorf("Failed to unpack contract execution response for contract: %s; method: %s; signature with encoded parameters: %s; %s", c.ID, methodDescriptor, *tx.Data, err.Error())
			}

			if response != nil {
				out["response"] = response
			}
			if receipt != nil {
				out["receipt"] = receipt
			}

			return out, nil
		}

		err = fmt.Errorf("Failed to execute method %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
	}

	return nil, err
}

func txCreatefunc(tx *Transaction, c *contract.Contract, n *network.Network, walletID *uuid.UUID, execution *contract.ContractExecution, _txParamsJSON *json.RawMessage) (*contract.ContractExecutionResponse, error) {
	db := dbconf.DatabaseConnection()
	publishedAt := execution.PublishedAt
	method := execution.Method
	params := execution.Params
	ref := execution.Ref
	value := execution.Value

	network.RequireNetworkStatsDaemon(n)

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

	var response map[string]interface{}

	txResponseCallback := func(c *contract.Contract, network *network.Network, methodDescriptor, method string, abiMethod *abi.Method, params []interface{}) (map[string]interface{}, error) {
		return txResponsefunc(tx, c, network, methodDescriptor, method, abiMethod, params)
	}

	var err error
	if n.IsEthereumNetwork() {
		response, err = c.ExecuteEthereumContract(n, txResponseCallback, method, params)
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
			Receipt:     response,
			Traces:      tx.Traces,
			Transaction: tx,
			Ref:         ref,
		}
	} else if tx.Response.Transaction == nil {
		tx.Response.Transaction = tx
	}

	return tx.Response, nil
}

func wfunc(w interface{}, txParams map[string]interface{}) *uuid.UUID {
	db := dbconf.DatabaseConnection()
	tmpWallet := &wallet.Wallet{}
	wallet := w.(*wallet.Wallet)
	// params := txParams
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
			txParams["public_key"] = wallet.Address
			txParams["private_key"] = wallet.PrivateKey
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
		natsutil.Nack(msg)
		return
	}

	if execution.ContractID == nil {
		common.Log.Errorf("Invalid tx message; missing contract_id")
		natsutil.Nack(msg)
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

	var tx Transaction
	txCreateFn := func(c *contract.Contract, network *network.Network, walletID *uuid.UUID, execution *contract.ContractExecution, _txParamsJSON *json.RawMessage) (*contract.ContractExecutionResponse, error) {
		return txCreatefunc(&tx, c, network, walletID, execution, _txParamsJSON)
	}

	walletFn := func(w interface{}, txParams map[string]interface{}) *uuid.UUID {
		return wfunc(w.(*wallet.Wallet), txParams)
	}

	executionResponse, err := cntract.ExecuteFromTx(execution, walletFn, txCreateFn)

	if err != nil {
		common.Log.Warningf("Failed to execute contract; %s", err.Error())
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
	common.Log.Debugf("Consuming NATS tx finalize message: %s", msg)

	var params map[string]interface{}

	nack := func(msg *stan.Msg, errmsg string, dropPacket bool) {
		common.Log.Warningf("Failed to handle NATS tx finalize message; %s", errmsg)
		if dropPacket {
			common.Log.Debugf("Dropping tx packet (seq: %d) on the floor; %s", msg.Sequence, errmsg)
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
			publishLatency := uint64(tx.BroadcastAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.PublishLatency = &publishLatency

			e2eLatency := uint64(tx.FinalizedAt.Sub(*tx.PublishedAt)) / uint64(time.Millisecond)
			tx.E2ELatency = &e2eLatency
		}

		broadcastLatency := uint64(tx.FinalizedAt.Sub(*tx.BroadcastAt)) / uint64(time.Millisecond)
		tx.BroadcastLatency = &broadcastLatency
	}

	tx.updateStatus(db, "success", nil)
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
		nack(msg, fmt.Sprintf("Failed to set block and finalized_at timestamp on tx: %s; error: %s", hash, *tx.Errors[0].Message), false)
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
		natsutil.Nack(msg)
		return
	}

	tx.Reload()

	network, err := tx.GetNetwork()
	if err != nil {
		desc := fmt.Sprintf("Failed to resolve tx network; %s", err.Error())
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		natsutil.Nack(msg)
		return
	}

	wallet, err := tx.GetWallet()
	if err != nil {
		desc := fmt.Sprintf("Failed to resolve tx wallet; %s", err.Error())
		common.Log.Warningf(desc)
		tx.updateStatus(db, "failed", common.StringOrNil(desc))
		natsutil.Nack(msg)
		return
	}

	err = tx.fetchReceipt(db, network, wallet)
	if err != nil {
		if msg.Redelivered { // FIXME-- implement proper dead-letter logic; only set tx to failed upon deadletter
			desc := fmt.Sprintf("Failed to fetch tx receipt; %s", err.Error())
			common.Log.Warningf(desc)
			tx.updateStatus(db, "failed", common.StringOrNil(desc))
		}

		natsutil.AttemptNack(msg, txReceiptMsgTimeout)
	} else {
		msg.Ack()
	}
}
