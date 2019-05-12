package contract

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
)

// ContractExecution represents a request payload used to execute functionality encapsulated by a contract.
type ContractExecution struct {
	ABI           interface{}   `json:"abi"`
	NetworkID     *uuid.UUID    `json:"network_id"`
	Contract      *Contract     `json:"-"`
	ContractID    *uuid.UUID    `json:"contract_id"`
	WalletID      *uuid.UUID    `json:"wallet_id"`
	WalletAddress *string       `json:"wallet_address"`
	Wallet        interface{}   `json:"wallet"`
	Gas           *float64      `json:"gas"`
	Nonce         *uint64       `json:"nonce"`
	Method        string        `json:"method"`
	Params        []interface{} `json:"params"`
	Value         *big.Int      `json:"value"`
	Ref           *string       `json:"ref"`
	PublishedAt   *time.Time    `json:"published_at"`
}

// ContractExecutionResponse is returned upon successful contract execution
type ContractExecutionResponse struct {
	Response    interface{} `json:"response"`
	Receipt     interface{} `json:"receipt"`
	Traces      interface{} `json:"traces"`
	Transaction interface{} `json:"transaction"`
	Ref         *string     `json:"ref"`
}

const natsTxSubject = "goldmine.tx"

// Execute an ephemeral ContractExecution
// func (e *ContractExecution) Execute() (interface{}, error) {
// 	var _abi *abi.ABI
// 	if execABI, abiOk := e.ABI.(abi.ABI); abiOk {
// 		_abi = &execABI
// 	} else if e.Contract != nil {
// 		execABI, err := e.Contract.readEthereumContractAbi()
// 		if err != nil {
// 			network, err := e.Contract.GetNetwork()
// 			if err != nil || network.isEthereumNetwork() {
// 				common.Log.Warningf("Cannot attempt EVM-based contract execution without ABI")
// 				return nil, err
// 			}
// 		}
// 		_abi = execABI
// 	}

// 	if _abi != nil {
// 		if mthd, ok := _abi.Methods[e.Method]; ok {
// 			if mthd.Const {
// 				return e.Contract.Execute(e)
// 			}
// 		}
// 	}

// 	publishedAt := time.Now()
// 	e.PublishedAt = &publishedAt

// 	txMsg, _ := json.Marshal(e)
// 	natsConnection := getNatsStreamingConnection()
// 	return e, natsConnection.Publish(natsTxSubject, txMsg)
// }

// Execute an ephemeral ContractExecution in Transaction context
func (e *ContractExecution) ExecuteFromTx(
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *ContractExecution, *json.RawMessage) (*ContractExecutionResponse, error),
) (interface{}, error) {
	var _abi *abi.ABI
	if execABI, abiOk := e.ABI.(abi.ABI); abiOk {
		_abi = &execABI
	} else if e.Contract != nil {
		execABI, err := e.Contract.ReadEthereumContractAbi()
		if err != nil {
			network, err := e.Contract.GetNetwork()
			if err != nil || network.IsEthereumNetwork() {
				common.Log.Warningf("Cannot attempt EVM-based contract execution without ABI")
				return nil, err
			}
		}
		_abi = execABI
	}

	if _abi != nil {
		if mthd, ok := _abi.Methods[e.Method]; ok {
			if mthd.Const {
				return e.Contract.ExecuteFromTx(e, walletFn, txCreateFn)
			}
		}
	}

	publishedAt := time.Now()
	e.PublishedAt = &publishedAt

	txMsg, _ := json.Marshal(e)
	natsConnection := common.GetDefaultNatsStreamingConnection()
	return e, natsConnection.Publish(natsTxSubject, txMsg)
}