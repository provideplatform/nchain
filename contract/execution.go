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

// Execution represents a request payload used to execute functionality encapsulated by a contract.
type Execution struct {
	ABI       interface{} `json:"abi"`
	NetworkID *uuid.UUID  `json:"network_id"`

	// Contract fields
	Contract   *Contract  `json:"-"`
	ContractID *uuid.UUID `json:"contract_id"`

	// Account fields
	Account        interface{} `json:"account"`
	AccountID      *uuid.UUID  `json:"account_id"`
	AccountAddress *string     `json:"account_address"`

	// Wallet fields
	Wallet   interface{} `json:"wallet"`
	WalletID *uuid.UUID  `json:"wallet_id"`
	HDPath   *string     `json:"hd_derivation_path"`

	// Tx params
	Gas    *float64      `json:"gas"`
	Nonce  *uint64       `json:"nonce"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	Value  *big.Int      `json:"value"`

	// Tx metadata/instrumentation
	Ref         *string    `json:"ref"`
	PublishedAt *time.Time `json:"published_at"`
}

// ExecutionResponse is returned upon successful contract execution
type ExecutionResponse struct {
	Response    interface{} `json:"response"`
	Receipt     interface{} `json:"receipt"`
	Traces      interface{} `json:"traces"`
	Transaction interface{} `json:"transaction"`
	Ref         *string     `json:"ref"`
}

const natsTxSubject = "goldmine.tx"

// ExecuteFromTx allows ephemeral contract execution
func (e *Execution) ExecuteFromTx(
	accountFn func(interface{}, map[string]interface{}) *uuid.UUID,
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *uuid.UUID, *Execution, *json.RawMessage) (*ExecutionResponse, error),
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
				common.Log.Warningf("Cannot attempt EVM-based contract execution without ABI")
				return nil, err
			}
			_abi = execABI
		}

		if _abi != nil {
			if mthd, ok := _abi.Methods[e.Method]; ok {
				if mthd.Const {
					return e.Contract.ExecuteFromTx(e, accountFn, walletFn, txCreateFn)
				}
			}
		}
	}

	publishedAt := time.Now()
	e.PublishedAt = &publishedAt

	txMsg, _ := json.Marshal(e)
	return e, natsutil.NatsPublish(natsTxSubject, txMsg)
}
