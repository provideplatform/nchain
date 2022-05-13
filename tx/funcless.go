/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/contract"
	"github.com/provideplatform/nchain/network"
	provide "github.com/provideplatform/provide-go/api/nchain"
	providecrypto "github.com/provideplatform/provide-go/crypto"
)

func executeTransaction(c *contract.Contract, execution *contract.Execution) (*contract.ExecutionResponse, error) {
	db := dbconf.DatabaseConnection()
	hdDerivationPath := execution.HDPath
	publishedAt := execution.PublishedAt
	method := execution.Method
	params := execution.Params
	ref := execution.Ref
	value := execution.Value
	walletID := execution.WalletID
	accountID := execution.AccountID
	gas := execution.Gas
	gasPrice := execution.GasPrice
	nonce := execution.Nonce
	path := execution.HDPath

	// create the tx params
	txParams := map[string]interface{}{}
	if c.Address != nil {
		txParams["to"] = *c.Address
	}

	if gas == nil {
		gas64 := float64(0)
		gas = &gas64
	}
	txParams["gas"] = gas

	if gasPrice != nil {
		txParams["gas_price"] = gasPrice
	}

	if nonce != nil {
		txParams["nonce"] = *nonce
	}

	if path != nil {
		txParams["hd_derivation_path"] = *path
	}

	txParamsJSON, _ := json.Marshal(txParams)
	_txParamsJSON := json.RawMessage(txParamsJSON)

	tx := &Transaction{
		ApplicationID:  c.ApplicationID,
		OrganizationID: c.OrganizationID,
		UserID:         nil,
		NetworkID:      c.NetworkID,
		AccountID:      accountID,
		WalletID:       walletID,
		Path:           hdDerivationPath,
		To:             c.Address,
		Value:          &TxValue{value: value},
		Params:         &_txParamsJSON,
		Ref:            ref,
	}

	if publishedAt != nil {
		tx.PublishedAt = publishedAt
	}

	var err error
	_abi, err := c.ReadEthereumContractAbi()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute contract method %s on contract: %s; no ABI resolved: %s", method, c.ID, err.Error())
	}
	var methodDescriptor = fmt.Sprintf("method %s", method)
	var abiMethod *abi.Method
	if mthd, ok := _abi.Methods[method]; ok {
		abiMethod = &mthd
	} else if method == "" {
		abiMethod = &_abi.Constructor
		methodDescriptor = "constructor"
	}

	// let's take in a network id, and use it to get a network:
	var n *network.Network
	if tx.NetworkID != uuid.Nil {
		n = &network.Network{}
		db.Model(tx).Related(&n)
	}

	var response map[string]interface{}

	if n.IsEthereumNetwork() {
		response, err = getTransactionResponse(tx, c, n, methodDescriptor, method, abiMethod, params)
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

	readonlyMethod := false
	if abiMethod.IsConstant() {
		readonlyMethod = true
	}

	resp := contract.ExecutionResponse{}
	if tx.Response == nil {

		common.Log.Debugf("response is: %+v", response)
		//tx.Response = &contract.ExecutionResponse{
		resp = contract.ExecutionResponse{
			Response:    response["response"],
			Receipt:     response,
			Traces:      tx.Traces,
			Transaction: tx,
			Ref:         ref,
			View:        readonlyMethod,
		}
		common.Log.Debugf("resp is: %+v", resp)
	} else if tx.Response.Transaction == nil {
		//?? when does this get tripped?
		//tx.Response.Transaction = tx
		resp.Transaction = tx
	}

	return &resp, nil
}

func getTransactionResponse(tx *Transaction, c *contract.Contract, network *network.Network, methodDescriptor, method string, abiMethod *abi.Method, params []interface{}) (map[string]interface{}, error) {
	var err error
	result := make([]byte, 32)
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
			invocationSig, err := providecrypto.EVMEncodeABI(abiMethod, params...)
			if err != nil {
				return nil, fmt.Errorf("Failed to encode %d parameters prior to attempting execution of %s on contract: %s; %s", len(params), methodDescriptor, c.ID, err.Error())
			}

			data := fmt.Sprintf("0x%s", ethcommon.Bytes2Hex(invocationSig))
			tx.Data = &data

			if abiMethod.IsConstant() {
				common.Log.Debugf("Attempting to read constant method %s on contract: %s", method, c.ID)
				client, err := providecrypto.EVMDialJsonRpc(network.ID.String(), network.RPCURL())
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
						tx.SignedTx, tx.Hash, err = providecrypto.EVMSignTx(network.ID.String(), network.RPCURL(), publicKey.(string), privateKey.(string), tx.To, tx.Data, tx.Value.BigInt(), nonce, uint64(gas), gasPrice)
						if err != nil {
							err = fmt.Errorf("Unable to broadcast signed tx; typecast failed for signed tx: %s", tx.SignedTx)
							common.Log.Warning(err.Error())
							return nil, err
						}

						if signedTx, ok := tx.SignedTx.(*types.Transaction); ok {
							err = providecrypto.EVMBroadcastSignedTx(network.ID.String(), network.RPCURL(), signedTx)
							return nil, err
						}

						if err != nil {
							err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed using arbitrarily-provided signer: %s; %s", methodDescriptor, c.ID, *tx.Data, publicKey, err.Error())
							common.Log.Warning(err.Error())
							return nil, err
						}
					} else {
						err = fmt.Errorf("Failed to execute %s on contract: %s (signature with encoded parameters: %s); tx broadcast failed", methodDescriptor, c.ID, *tx.Data)
						if len(tx.Errors) > 0 {
							err = fmt.Errorf("%s; %s", err.Error(), *tx.Errors[0].Message)
						}
						common.Log.Warning(err.Error())
						return nil, err
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
