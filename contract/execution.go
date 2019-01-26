package contract

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
)

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	ABI        interface{}    `json:"abi"`
	NetworkID  *uuid.UUID     `json:"network_id"`
	Contract   *Contract      `json:"-"`
	ContractID *uuid.UUID     `json:"contract_id"`
	WalletID   *uuid.UUID     `json:"wallet_id"`
	Wallet     *wallet.Wallet `json:"wallet"`
	Gas        *float64       `json:"gas"`
	Method     string         `json:"method"`
	Params     []interface{}  `json:"params"`
	Value      *big.Int       `json:"value"`
	Ref        *string        `json:"ref"`
}

// ContractExecutionResponse is returned upon successful contract execution
type ContractExecutionResponse struct {
	Response    interface{}  `json:"response"`
	Receipt     interface{}  `json:"receipt"`
	Traces      interface{}  `json:"traces"`
	Transaction *Transaction `json:"transaction"`
	Ref         *string      `json:"ref"`
}

// Execute an ephemeral ContractExecution
func (e *ContractExecution) Execute() (interface{}, error) {
	var _abi *abi.ABI
	if __abi, abiOk := e.ABI.(abi.ABI); abiOk {
		_abi = &__abi
	} else if e.Contract != nil {
		__abi, err := e.Contract.readEthereumContractAbi()
		if err != nil {
			common.Log.Warningf("Cannot attempt contract execution without ABI")
			return nil, err
		}
		_abi = __abi
	}

	if _abi != nil {
		if mthd, ok := _abi.Methods[e.Method]; ok {
			if mthd.Const {
				return e.Contract.Execute(e.Ref, e.Wallet, e.Value, e.Method, e.Params, 0)
			}
		}
	}

	txMsg, _ := json.Marshal(e)
	natsConnection := getNatsStreamingConnection()
	return e, natsConnection.Publish(natsTxSubject, txMsg)
}
