package tx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/contract"
	"github.com/provideapp/nchain/network"
	provideAPI "github.com/provideservices/provide-go/api"
	providecrypto "github.com/provideservices/provide-go/crypto"
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
		resp = contract.ExecutionResponse{
			Response:    response["response"],
			Receipt:     response,
			Traces:      tx.Traces,
			Transaction: tx,
			Ref:         ref,
			View:        readonlyMethod,
		}
		common.Log.Debugf("XXXresp is: %+v", resp)
		common.Log.Debugf("XXX: Execution of tx response for tx ref %s has transaction %v", *resp.Ref, resp.Transaction)
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
				if err != nil {
					return nil, err
				}
				address, err := signer.Address()
				if err != nil {
					return nil, err
				}
				msg := tx.asEthereumCallMsg(*address, 0, 0)
				result, err = client.CallContract(context.TODO(), msg, nil)
				if err != nil {
					err = fmt.Errorf("Failed to read constant method %s on contract: %s; %s", method, c.ID, err.Error())
					return nil, err
				}
			} else {
				params := c.ParseParams()

				value := uint64(0)
				if val, valOk := params["value"].(float64); valOk {
					value = uint64(val)
				}

				var accountID *string
				if acctID, acctIDOk := params["account_id"].(string); acctIDOk {
					accountID = &acctID
				}

				var walletID *string
				var hdDerivationPath *string
				if wlltID, wlltIDOk := params["wallet_id"].(string); wlltIDOk {
					walletID = &wlltID

					if path, pathOk := params["hd_derivation_path"].(string); pathOk {
						hdDerivationPath = &path
					}
				}
				// send as a message to NATS
				txExecutionMsg, _ := json.Marshal(map[string]interface{}{
					"contract_id":        c.ID,
					"data":               data,
					"account_id":         accountID,
					"wallet_id":          walletID,
					"hd_derivation_path": hdDerivationPath,
					"value":              value,
					"params":             params,
					"published_at":       time.Now(),
					"reference":          tx.Ref,
				})

				common.Log.Debugf("XXX: contract.Execute: about to publish exec tx to nats for tx ref: %s", *tx.Ref)
				err = natsutil.NatsStreamingPublish(natsTxSubject, txExecutionMsg)

				if err != nil {
					common.Log.Warningf("Failed to publish contract deployment tx ref %s. Error: %s", *tx.Ref, err.Error())
					c.Errors = append(c.Errors, &provideAPI.Error{
						Message: common.StringOrNil(err.Error()),
					})
					return nil, err
				}

				//otherwise we're good, so return the reference so it can be tracked
				out["reference"] = *tx.Ref
				return out, nil
			}
			// we executed a read-only method
			if result != nil {
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

		}

		err = fmt.Errorf("Failed to execute method %s on contract: %s; method not found in ABI", methodDescriptor, c.ID)
	}

	return nil, err
}
