package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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
const txAckWait = time.Second * 15
const txMsgTimeout = int64(txAckWait * 3)

const natsTxCreateSubject = "goldmine.tx.create"
const natsTxCreateMaxInFlight = 2048
const txCreateAckWait = time.Second * 30
const txCreateMsgTimeout = int64(txCreateAckWait * 3)

const natsTxFinalizeSubject = "goldmine.tx.finalize"
const natsTxFinalizeMaxInFlight = 4096
const txFinalizeAckWait = time.Second * 15
const txFinalizedMsgTimeout = int64(txFinalizeAckWait * 6)

const natsTxReceiptSubject = "goldmine.tx.receipt"
const natsTxReceiptMaxInFlight = 2048
const txReceiptAckWait = time.Second * 15
const txReceiptMsgTimeout = int64(txReceiptAckWait * 15)

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
			consumeTxMsg,
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
		ApplicationID: contract.ApplicationID,
		Data:          common.StringOrNil(data),
		NetworkID:     contract.NetworkID,
		AccountID:     accountID,
		WalletID:      walletID,
		Path:          common.StringOrNil(hdDerivationPath),
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
		errmsg := fmt.Sprintf("Failed to execute transaction; tx failed with %d error(s)", len(tx.Errors))
		for _, err := range tx.Errors {
			errmsg = fmt.Sprintf("%s\n\t%s", errmsg, *err.Message)
		}

		params := tx.ParseParams()
		gas, gasOk := params["gas"].(float64)
		if !gasOk {
			gas = float64(100000000000000000)
		} else {
			gas = gas * 1.1
		}

		// FIXME-- read from environment config
		networkSubsidyFaucetApplicationAddressMapping := map[string]interface{}{
			"66d44f30-9092-4182-a3c4-bc02736d6ae5": map[string]interface{}{
				"01554e22-3d7a-44a3-9c65-6bcabaa08c38": "0xdD2F8052bE76FA1456e096526db5C0F12B0af564",
				"146ab73e-b2eb-4386-8c6f-93663792c741": "0x96f1027FEe06A15f42E48180705a2ecB2F846985",
			},
		}
		networkSubsidyFaucetDripValue := int64(gas)
		networkSubsidyFaucets, networkSubsidyFaucetExists := networkSubsidyFaucetApplicationAddressMapping[tx.NetworkID.String()].(map[string]interface{})
		faucetSubsidyEligible := strings.Contains(errmsg, "insufficient funds") && networkSubsidyFaucetExists

		if faucetSubsidyEligible && len(networkSubsidyFaucets) > 0 {
			common.Log.Debugf("Transaction execution failed due to insufficient funds but faucet subsidy exists for network: %s; requesting subsidized tx funding", tx.NetworkID)

			for networkSubsidyFaucetApplicationID, networkSubsidyFaucetAddress := range networkSubsidyFaucets {
				out := []string{}
				db.Table("accounts").Select("id").Where("accounts.application_id = ? AND accounts.address = ?", networkSubsidyFaucetApplicationID, networkSubsidyFaucetAddress).Pluck("id", &out)
				if out == nil || len(out) == 0 || len(out[0]) == 0 {
					common.Log.Warningf("Failed to retrieve configured subsidy faucet signing identity for faucet address: %s; faucet application id: %s", networkSubsidyFaucetAddress, networkSubsidyFaucetApplicationID)
				} else {
					faucetApplicationID, _ := uuid.FromString(networkSubsidyFaucetApplicationID)
					faucetAccountID, _ := uuid.FromString(out[0])
					faucetBeneficiary, _ := tx.signerFactory(db)
					faucetBeneficiaryAddress := faucetBeneficiary.Address()
					faucetGas := int64(210000)
					faucetTx := &Transaction{
						ApplicationID: &faucetApplicationID,
						Data:          common.StringOrNil("0x"),
						NetworkID:     contract.NetworkID,
						AccountID:     &faucetAccountID,
						To:            common.StringOrNil(faucetBeneficiaryAddress),
						Value:         &TxValue{value: big.NewInt(networkSubsidyFaucetDripValue - faucetGas)},
					}

					if faucetTx.Create(db) {
						db.Delete(&tx) // Drop tx that had insufficient funds so its hash can be rebroadcast...

						common.Log.Debugf("Faucet subsidy transaction broadcast; beneficiary: %s", *faucetTx.To)
						contract.TransactionID = &tx.ID
						db.Save(&contract)
						common.Log.Debugf("faucetTx execution successful: %s", *faucetTx.Hash)
					}

					natsutil.AttemptNack(msg, txCreateMsgTimeout)
					break
				}
			}
		} else {
			common.Log.Warning(errmsg)
			natsutil.AttemptNack(msg, txCreateMsgTimeout)
		}
	}
}

func txResponsefunc(tx *Transaction, c *contract.Contract, network *network.Network, methodDescriptor, method string, abiMethod *abi.Method, params []interface{}) (map[string]interface{}, error) {
	var err error
	var result []byte
	var receipt map[string]interface{}
	out := map[string]interface{}{}

	db := dbconf.DatabaseConnection()

	signer, err := tx.signerFactory(db)
	if err != nil {
		err = fmt.Errorf("failed to resolve tx signer for contract: %s; %s", c.ID, err.Error())
		return nil, err
	}

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
				msg := tx.asEthereumCallMsg(signer.Address(), 0, 0)
				result, err = client.CallContract(context.TODO(), msg, nil)
				if err != nil {
					err = fmt.Errorf("Failed to read constant method %s on contract: %s; %s", method, c.ID, err.Error())
					return nil, err
				}
			} else {
				var txResponse *contract.ExecutionResponse
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
					var gasPrice *uint64
					gp, gpOk := txParams["gas_price"].(float64)
					if gpOk {
						_gasPrice := uint64(gp)
						gasPrice = &_gasPrice
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
						tx.SignedTx, tx.Hash, err = provide.EVMSignTx(network.ID.String(), network.RPCURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), nonce, uint64(gas), gasPrice)
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
							txReceipt := txResponse.Receipt.(*types.Receipt)
							txReceiptJSON, _ := json.Marshal(txReceipt)
							json.Unmarshal(txReceiptJSON, &receipt)
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

			outptr, err := abiMethod.Outputs.UnpackValues(result)
			if err != nil {
				return nil, fmt.Errorf("Failed to unpack contract execution response for contract: %s; method: %s; signature with encoded parameters: %s; %s", c.ID, methodDescriptor, *tx.Data, err.Error())
			}
			if len(outptr) == 1 {
				out["response"] = outptr[0]
			} else {
				out["response"] = outptr
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

func txCreatefunc(tx *Transaction, c *contract.Contract, n *network.Network, accountID *uuid.UUID, walletID *uuid.UUID, execution *contract.Execution, _txParamsJSON *json.RawMessage) (*contract.ExecutionResponse, error) {
	db := dbconf.DatabaseConnection()
	hdDerivationPath := execution.HDPath
	publishedAt := execution.PublishedAt
	method := execution.Method
	params := execution.Params
	ref := execution.Ref
	value := execution.Value

	tx = &Transaction{
		ApplicationID: c.ApplicationID,
		UserID:        nil,
		NetworkID:     c.NetworkID,
		AccountID:     accountID,
		WalletID:      walletID,
		Path:          hdDerivationPath,
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
		tx.Response = &contract.ExecutionResponse{
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

func afunc(a interface{}, txParams map[string]interface{}) *uuid.UUID {
	tmpAccount := &wallet.Account{}
	var address *string
	var privateKey *string

	switch a.(type) {
	case *wallet.Account:
		acct := a.(*wallet.Account)
		accountID := acct.ID
		if accountID != uuid.Nil {
			return &accountID
		}
		if acct.Address != "" {
			address = &acct.Address
		}
		privateKey = acct.PrivateKey
	case map[string]interface{}:
		accountMap := a.(map[string]interface{})
		if addr, addrOk := accountMap["address"].(string); addrOk {
			address = &addr
		}
		if privKey, privKeyOk := accountMap["private_key"].(string); privKeyOk {
			privateKey = &privKey
		}
	default:
		common.Log.Debugf("no valid signing identity uuid resolved during attempted contract execution; contract execution address: %s", txParams["to"])
	}

	if address != nil {
		db := dbconf.DatabaseConnection()
		db.Where("address = ?", *address).Find(&tmpAccount)
		if tmpAccount != nil && tmpAccount.ID != uuid.Nil {
			return &tmpAccount.ID
		}
		common.Log.Warningf("Failed to resolve managed wallet for address: %s", *address)

		if privateKey != nil {
			txParams["public_key"] = address
			txParams["private_key"] = privateKey
		}
	}

	return &uuid.Nil
}

func wfunc(w interface{}, txParams map[string]interface{}) *uuid.UUID {
	switch w.(type) {
	case *wallet.Wallet:
		wallet := w.(*wallet.Wallet)
		walletID := wallet.ID
		if walletID != uuid.Nil {
			return &walletID
		}
	case map[string]interface{}:
		walletMap := w.(map[string]interface{})
		if walletIDStr, walletIDStrOk := walletMap["wallet_id"].(string); walletIDStrOk {
			common.Log.Debugf("resolved wallet id for deterministic tx signer: %s", walletIDStr)
			walletID, err := uuid.FromString(walletIDStr)
			if err != nil {
				common.Log.Warningf("failed to parse wallet uuid for deterministic tx signer; %s", err.Error())
				return nil
			}
			return &walletID
		}

	default:
		common.Log.Debugf("no HD wallet uuid resolved during attempted contract execution; contract execution address: %s", txParams["to"])
	}

	return &uuid.Nil
}

func consumeTxMsg(msg *stan.Msg) {
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

	var tx Transaction
	txCreateFn := func(c *contract.Contract, network *network.Network, accountID *uuid.UUID, walletID *uuid.UUID, execution *contract.Execution, _txParamsJSON *json.RawMessage) (*contract.ExecutionResponse, error) {
		return txCreatefunc(&tx, c, network, accountID, walletID, execution, _txParamsJSON)
	}

	executionResponse, err := cntract.ExecuteFromTx(execution, afunc, wfunc, txCreateFn)

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
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("Recovered from failed tx receipt message; %s", r)
			natsutil.AttemptNack(msg, txReceiptMsgTimeout)
		}
	}()

	common.Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal load balancer provisioning message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	transactionID, transactionIDOk := params["transaction_id"].(string)
	if !transactionIDOk {
		common.Log.Warningf("Failed to consume NATS tx receipt message; no transaction id provided")
		natsutil.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	tx := &Transaction{}
	db.Where("id = ?", transactionID).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		common.Log.Warningf("Failed to fetch tx receipt; no tx resolved for id: %s", transactionID)
		natsutil.Nack(msg)
		return
	}

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
		common.Log.Warningf(fmt.Sprintf("Failed to fetch tx receipt; %s", err.Error()))
		natsutil.AttemptNack(msg, txReceiptMsgTimeout)
	} else {
		common.Log.Debugf("fetched tx receipt for hash: %s", *tx.Hash)
		tx.updateStatus(db, "success", nil)
		msg.Ack()
	}
}
