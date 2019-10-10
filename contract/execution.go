package contract

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	natsutil "github.com/kthomas/go-natsutil"
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

// ExecuteFromTx allows ephemeral contract execution
func (e *ContractExecution) ExecuteFromTx(
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *ContractExecution, *json.RawMessage) (*ContractExecutionResponse, error),
) (interface{}, error) {
	network, err := e.Contract.GetNetwork()
	if err != nil {
		common.Log.Warningf("Cannot attempt contract execution; %s", err.Error())
		return nil, err
	}

	if network.IsEthereumNetwork() {
		var _abi *abi.ABI
		if execABI, abiOk := e.ABI.(abi.ABI); abiOk {
			_abi = &execABI
		} else if e.Contract != nil {
			execABI, err := e.Contract.ReadEthereumContractAbi()
			if err != nil {

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
	}

	publishedAt := time.Now()
	e.PublishedAt = &publishedAt

	txMsg, _ := json.Marshal(e)
	return e, natsutil.NatsPublish(natsTxSubject, txMsg)
}
