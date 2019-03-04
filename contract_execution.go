package main

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	uuid "github.com/kthomas/go.uuid"
)

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	ABI           interface{}   `json:"abi"`
	NetworkID     *uuid.UUID    `json:"network_id"`
	Contract      *Contract     `json:"-"`
	ContractID    *uuid.UUID    `json:"contract_id"`
	WalletID      *uuid.UUID    `json:"wallet_id"`
	WalletAddress *string       `json:"wallet_address"`
	Wallet        *Wallet       `json:"wallet"`
	Gas           *float64      `json:"gas"`
	Nonce         *uint64       `json:"nonce"`
	Method        string        `json:"method"`
	Params        []interface{} `json:"params"`
	Value         *big.Int      `json:"value"`
	Ref           *string       `json:"ref"`
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
	if execABI, abiOk := e.ABI.(abi.ABI); abiOk {
		_abi = &execABI
	} else if e.Contract != nil {
		execABI, err := e.Contract.readEthereumContractAbi()
		if err != nil {
			network, err := e.Contract.GetNetwork()
			if err != nil || network.isEthereumNetwork() {
				Log.Warningf("Cannot attempt EVM-based contract execution without ABI")
				return nil, err
			}
		}
		_abi = execABI
	}

	if _abi != nil {
		if mthd, ok := _abi.Methods[e.Method]; ok {
			if mthd.Const {
				return e.Contract.Execute(e.Ref, e.Wallet, e.Value, e.Method, e.Params, 0, nil)
			}
		}
	}

	txMsg, _ := json.Marshal(e)
	natsConnection := getNatsStreamingConnection()
	return e, natsConnection.Publish(natsTxSubject, txMsg)
}
