// +build integration nchain failing rinkeby ropsten kovan goerli nobookie basic bookie readonly

package integration

import (
	"encoding/json"
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/ident/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

// ropsten
var ropstenNetworkID string = "66d44f30-9092-4182-a3c4-bc02736d6ae5"
var ropstenNetworkName string = "ropsten"

// rinkeby
var rinkebyNetworkID string = "07102258-5e49-480e-86af-6d0c3260827d"
var rinkebyNetworkName string = "rinkeby"

var gorliNetworkID string = "1b16996e-3595-4985-816c-043345d22f8c"
var gorliNetworkName string = "gorli"

var kovanNetworkID string = "8d31bf48-df6b-4a71-9d7c-3cb291111e27"
var kovanNetworkName string = "kovan"

const contractTimeout = 480
const transactionTimeout = 480

const bulkContractTimeout = 480
const bulkTransactionTimeout = 480

const contractSleepTime = 10

const transactionSleepTime = 10

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
	BlockExplorerUrl  *string    `json:"block_explorer_url"`
	JsonRpcUrl        *string    `json:"json_rpc_url"`
	WebsocketUrl      *string    `json:"websocket_url"`
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

func init() {
	common.Log.Debugf("init")
	// let's enable ropsten and use it as the network id for the moment
	// todo: test enabling all the chains - but need correct chain specs for them all

	_, err := enableNetwork(ropstenNetworkName, ropstenNetworkID)
	if err != nil {
		common.Log.Warningf("error enabling ropsten: Error: %s", err.Error())
	}

	_, err = enableNetwork(rinkebyNetworkName, rinkebyNetworkID)
	if err != nil {
		common.Log.Warningf("error enabling rinkeby: Error: %s", err.Error())
	}

	_, err = enableNetwork(kovanNetworkName, kovanNetworkID)
	if err != nil {
		common.Log.Warningf("error enabling kovan: Error: %s", err.Error())
	}

	_, err = enableNetwork(gorliNetworkName, gorliNetworkID)
	if err != nil {
		common.Log.Warningf("error enabling gorli: Error: %s", err.Error())
	}
}

func enableNetwork(networkName, networkID string) (bool, error) {
	testId, err := uuid.NewV4()
	if err != nil {
		return false, fmt.Errorf("error creating uuid. Error: %s", err.Error())
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		return false, fmt.Errorf("error creating authenticated user. Error: %s", err.Error())
	}

	// network, err := provide.GetNetworkDetails(*token, networkID, map[string]interface{}{})
	// if err != nil {
	// 	return false, fmt.Errorf("error getting %s network details. Error: %s", networkName, err.Error())
	// }

	networkConfig := json.RawMessage("{}")

	switch networkID {
	case ropstenNetworkID:
		networkConfig, err = generateRopstenConfig()
		if err != nil {
			return false, fmt.Errorf("error generating ropsten config. Error: %s", err.Error())
		}
	case rinkebyNetworkID:
		networkConfig, err = generateRinkebyConfig()
		if err != nil {
			return false, fmt.Errorf("error generating rinkeby config. Error: %s", err.Error())
		}
	case kovanNetworkID:
		networkConfig, err = generateKovanConfig()
		if err != nil {
			return false, fmt.Errorf("error generating kovan config. Error: %s", err.Error())
		}
	case gorliNetworkID:
		networkConfig, err = generateGorliConfig()
		if err != nil {
			return false, fmt.Errorf("error generating gorli config. Error: %s", err.Error())
		}
	}

	err = provide.UpdateNetwork(*token, networkID, map[string]interface{}{
		"enabled": true,
		"config":  networkConfig,
	})

	if err != nil {
		return false, fmt.Errorf("error updating %s network details. Error: %s", networkName, err.Error())
	}

	return true, nil
}

func generateRopstenConfig() (json.RawMessage, error) {

	networkSpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	networkAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	networkChainSpec := chainSpec{
		Config:     &networkSpecConfig,
		Alloc:      &networkAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	networkChainConfig := chainConfig{
		NativeCurrency:    common.StringOrNil("TEST"),
		Platform:          common.StringOrNil("evm"),
		EngineID:          common.StringOrNil("ethash"),
		IsEthereumNetwork: true,
		Client:            common.StringOrNil("geth"),
		NetworkID:         3,
		BlockExplorerUrl:  common.StringOrNil("https://ropsten.etherscan.io"),
		JsonRpcUrl:        common.StringOrNil("http://nethermind-ropsten.provide.services:8888"),
		WebsocketUrl:      common.StringOrNil("wss://nethermind-ropsten.provide.services:8888"),
		Chain:             common.StringOrNil("test"),
		ProtocolID:        common.StringOrNil("pow"),
		ChainSpec:         &networkChainSpec,
	}

	// networkChainConfig := chainConfig{
	// 	NativeCurrency:    common.StringOrNil("TEST"),
	// 	Platform:          common.StringOrNil("evm"),
	// 	EngineID:          common.StringOrNil("ethash"),
	// 	IsEthereumNetwork: true,
	// 	Client:            common.StringOrNil("geth"),
	// 	NetworkID:         3,
	// 	BlockExplorerUrl:  common.StringOrNil("https://ropsten.etherscan.io"),
	// 	JsonRpcUrl:        common.StringOrNil("https://ropsten.infura.io/v3/561dda3e54c54188934d2ab95b1910e8"),
	// 	WebsocketUrl:      common.StringOrNil("https://ropsten.infura.io/v3/561dda3e54c54188934d2ab95b1910e8"),
	// 	Chain:             common.StringOrNil("test"),
	// 	ProtocolID:        common.StringOrNil("pow"),
	// 	ChainSpec:         &networkChainSpec,
	// }

	// convert networkChainConfig to a json.RawMessage
	config, err := json.Marshal(networkChainConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshalling config to json. Error: %s", err.Error())
	}

	return config, nil
}

// rinkeby
//{"block_explorer_url":"https://rinkeby.etherscan.io","is_ethereum_network":true,"json_rpc_url":"https://rinkeby.infura.io/v3/561dda3e54c54188934d2ab95b1910e8","native_currency":"ETH","network_id":4,"websocket_url":"wss://rinkeby.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8","platform":"evm","protocol_id":"pow","engine_id":"ethash","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8050,8051,30300],"udp":[30300]}}}}

func generateRinkebyConfig() (json.RawMessage, error) {
	networkSpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	networkAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	networkChainSpec := chainSpec{
		Config:     &networkSpecConfig,
		Alloc:      &networkAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	networkChainConfig := chainConfig{
		NativeCurrency:    common.StringOrNil("TEST"),
		Platform:          common.StringOrNil("evm"),
		EngineID:          common.StringOrNil("ethash"),
		IsEthereumNetwork: true,
		Client:            common.StringOrNil("geth"),
		NetworkID:         4,
		BlockExplorerUrl:  common.StringOrNil("https://rinkeby.etherscan.io"),
		JsonRpcUrl:        common.StringOrNil("https://rinkeby.infura.io/v3/561dda3e54c54188934d2ab95b1910e8"),
		WebsocketUrl:      common.StringOrNil("wss://rinkeby.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8"),
		Chain:             common.StringOrNil("test"),
		ProtocolID:        common.StringOrNil("pow"),
		ChainSpec:         &networkChainSpec,
	}
	// convert networkChainConfig to a json.RawMessage
	config, err := json.Marshal(networkChainConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshalling config to json. Error: %s", err.Error())
	}

	return config, nil
}

//kovan
//{"block_explorer_url":"https://kovan.etherscan.io","is_ethereum_network":true,"json_rpc_url":"https://kovan.infura.io/v3/561dda3e54c54188934d2ab95b1910e8","native_currency":"KETH","network_id":42,"websocket_url":"wss://kovan.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8","platform":"evm","client":"parity","protocol_id":"poa","engine_id":"aura","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8050,8051,30300],"udp":[30300]}}}}

func generateKovanConfig() (json.RawMessage, error) {
	networkSpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	networkAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	networkChainSpec := chainSpec{
		Config:     &networkSpecConfig,
		Alloc:      &networkAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	networkChainConfig := chainConfig{
		NativeCurrency:    common.StringOrNil("KETH"),
		Platform:          common.StringOrNil("evm"),
		EngineID:          common.StringOrNil("aura"),
		IsEthereumNetwork: true,
		Client:            common.StringOrNil("parity"),
		NetworkID:         42,
		BlockExplorerUrl:  common.StringOrNil("https://kovan.etherscan.io"),
		JsonRpcUrl:        common.StringOrNil("https://kovan.infura.io/v3/561dda3e54c54188934d2ab95b1910e8"),
		WebsocketUrl:      common.StringOrNil("wss://kovan.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8"),
		Chain:             common.StringOrNil("test"),
		ProtocolID:        common.StringOrNil("poa"),
		ChainSpec:         &networkChainSpec,
	}
	// convert networkChainConfig to a json.RawMessage
	config, err := json.Marshal(networkChainConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshalling config to json. Error: %s", err.Error())
	}

	return config, nil
}

//gorli
//{"block_explorer_url":"https://goerli.etherscan.io","engine_id":"clique","is_ethereum_network":true,"native_currency":"ETH","network_id":5,"platform":"evm","protocol_id":"poa","websocket_url":"wss://goerli.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8","json_rpc_url":"https://goerli.infura.io/v3/561dda3e54c54188934d2ab95b1910e8","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8545,8546,8547,30303],"udp":[30303]}}}}

func generateGorliConfig() (json.RawMessage, error) {
	networkSpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	networkAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	networkChainSpec := chainSpec{
		Config:     &networkSpecConfig,
		Alloc:      &networkAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	networkChainConfig := chainConfig{
		NativeCurrency:    common.StringOrNil("ETH"),
		Platform:          common.StringOrNil("evm"),
		EngineID:          common.StringOrNil("clique"),
		IsEthereumNetwork: true,
		Client:            common.StringOrNil("parity"),
		NetworkID:         5,
		BlockExplorerUrl:  common.StringOrNil("https://goerli.etherscan.io"),
		JsonRpcUrl:        common.StringOrNil("https://goerli.infura.io/v3/561dda3e54c54188934d2ab95b1910e8"),
		WebsocketUrl:      common.StringOrNil("wss://goerli.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8"),
		Chain:             common.StringOrNil("test"),
		ProtocolID:        common.StringOrNil("poa"),
		ChainSpec:         &networkChainSpec,
	}
	// convert networkChainConfig to a json.RawMessage
	config, err := json.Marshal(networkChainConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshalling config to json. Error: %s", err.Error())
	}

	return config, nil
}

//ropsten
//{"block_explorer_url":"https://ropsten.etherscan.io","client":"geth","engine_id":"ethash","is_ethereum_network":true,"native_currency":"ETH","network_id":3,"platform":"evm","protocol_id":"pow","websocket_url":"wss://ropsten.infura.io/ws/v3/561dda3e54c54188934d2ab95b1910e8","json_rpc_url":"https://ropsten.infura.io/v3/561dda3e54c54188934d2ab95b1910e8","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8545,8546,8547,30303],"udp":[30303]}}}}
