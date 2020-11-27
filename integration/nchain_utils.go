// +build integration

package integration

import (
	"encoding/json"
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

var ropstenNetworkID string = "66d44f30-9092-4182-a3c4-bc02736d6ae5"
var ropstenNetworkName string = "Ethereum Ropsten testnet"

var True bool = true
var False bool = false

type chainSpecConfig struct {
	HomesteadBlock      int `json:"homesteadBlock"`
	Eip150Block         int `json:"eip150Block"`
	Eip155Block         int `json:"eip155Block"`
	Eip158Block         int `json:"eip158Block"`
	ByzantiumBlock      int `json:"byzantiumBlock"`
	ConstantinopleBlock int `json:"constantinopleBlock"`
	PetersburgBlock     int `json:"petersburgBlock"`
}

type allocation struct {
	notexportedhack *string `json:"hackyhack,omitempty"`
}

type chainSpec struct {
	Config     *chainSpecConfig `json:"config"`
	Alloc      *allocation      `json:"alloc"`
	Coinbase   *string          `json:"coinbase"`
	Difficulty *string          `json:"difficulty"`
	ExtraData  *string          `json:"extraData"`
	GasLimit   *string          `json:"gasLimit"`
	Nonce      *string          `json:"nonce"`
	Mixhash    *string          `json:"mixhash"`
	ParentHash *string          `json:"parentHash"`
	Timestamp  *string          `json:"timestamp"`
}

type chainConfig struct {
	NativeCurrency    *string    `json:"native_currency"`
	IsEthereumNetwork bool       `json:"is_ethereum_network"`
	Client            *string    `json:"client"`
	JsonRpcUrl        *string    `json:"json_rpc_url"`
	Platform          *string    `json:"platform"`
	EngineID          *string    `json:"engine_id"`
	Chain             *string    `json:"chain"`
	ProtocolID        *string    `json:"protocol_id"`
	NetworkID         int        `json:"network_id"`
	ChainSpec         *chainSpec `json:"chainspec"`
}

type chainDef struct {
	Name      *string      `json:"name"`
	Cloneable bool         `json:"cloneable"`
	Config    *chainConfig `json:"config"`
}

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
