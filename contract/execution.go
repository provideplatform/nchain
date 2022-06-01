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

package contract

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
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
	Gas       *float64      `json:"gas"`
	GasPrice  *float64      `json:"gas_price"`
	Nonce     *uint64       `json:"nonce"`
	Method    string        `json:"method"`
	Params    []interface{} `json:"params"`
	Subsidize bool          `json:"subsidize"`
	Value     *big.Int      `json:"value"`

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
	View        bool        `json:"-"` // used where execution response is read-only and therefore synchronous
}

const natsTxSubject = "nchain.tx"

// ExecuteFromTx allows ephemeral contract execution
func (e *Execution) ExecuteFromTx(
	accountFn func(interface{}, map[string]interface{}) *uuid.UUID,
	walletFn func(interface{}, map[string]interface{}) *uuid.UUID,
	txCreateFn func(*Contract, *network.Network, *uuid.UUID, *uuid.UUID, *Execution, *json.RawMessage) (*ExecutionResponse, error),
) (interface{}, error) {
	network, err := e.Contract.GetNetwork()
	if err != nil {
		common.Log.Warningf("cannot attempt contract execution; %s", err.Error())
		return nil, err
	}

	if network.IsEthereumNetwork() {
		var _abi *abi.ABI
		if execABI, abiOk := e.ABI.(abi.ABI); abiOk {
			_abi = &execABI
		} else if e.Contract != nil {
			execABI, err := e.Contract.ReadEthereumContractAbi()
			if err != nil {
				common.Log.Warningf("cannot attempt EVM-based contract execution without ABI")
				return nil, err
			}
			_abi = execABI
		}

		if _abi != nil {
			if mthd, ok := _abi.Methods[e.Method]; ok {
				if mthd.IsConstant() {
					return e.Contract.ExecuteFromTx(e, accountFn, walletFn, txCreateFn)
				}
			}
		}
	}

	publishedAt := time.Now()
	e.PublishedAt = &publishedAt

	txMsg, _ := json.Marshal(e)
	_, err = natsutil.NatsJetstreamPublish(natsTxSubject, txMsg)
	if err != nil {
		common.Log.Warningf("failed to broadcast EVM-based contract execution message to NATS stream; %s", err.Error())
		return nil, err
	}

	return e, nil
}
