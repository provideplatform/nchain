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

// +build integration nchain failing rinkeby ropsten kovan goerli nobookie basic bookie readonly bulk

package integration

import (
	"encoding/json"
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/ident/common"
	provide "github.com/provideplatform/provide-go/api/nchain"
)

func NetworkFactory(token string, testId uuid.UUID) (*provide.Network, error) {

	// set up the chain specification
	chainySpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	chainyAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	chainySpec := chainSpec{
		Config:     &chainySpecConfig,
		Alloc:      &chainyAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	chainyConfig := chainConfig{
		NativeCurrency:    common.StringOrNil("TEST"),
		Platform:          common.StringOrNil("evm"),
		EngineID:          common.StringOrNil("ethash"),
		Client:            common.StringOrNil("geth"),
		NetworkID:         3,
		IsEthereumNetwork: true,
		Chain:             common.StringOrNil("test"),
		ProtocolID:        common.StringOrNil("pow"),
		ChainSpec:         &chainySpec,
	}

	chainName := fmt.Sprintf("Ethereum Testnet %s", testId.String())

	chainyChain := chainDef{
		Name:      common.StringOrNil(chainName),
		Cloneable: false,
		Config:    &chainyConfig,
	}

	chainyChainJSON, _ := json.Marshal(chainyChain)

	params := map[string]interface{}{}
	json.Unmarshal(chainyChainJSON, &params)

	network, err := provide.CreateNetwork(token, params)
	if err != nil {
		return nil, err
	}

	return network, nil
}
