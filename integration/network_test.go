// +build integration

package integration

import (
	"encoding/json"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

var ropstenNetworkID string = "66d44f30-9092-4182-a3c4-bc02736d6ae5"

var True bool = true
var False bool = false

func TestCreateNetwork(t *testing.T) {
	// let's try it from the docs!

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	//{"block_explorer_url":"https://etherscan.io","chainspec_url":"https://gist.githubusercontent.com/kthomas/3ac2e29ee1b2fb22d501ae7b52884c24/raw/161c6a9de91db7044fb93852aed7b0fa0e78e55f/mainnet.chainspec.json","is_ethereum_network":true,"json_rpc_url":"https://mainnet.infura.io/v3/fde5e81d5d3141a093def423db3eeb33","native_currency":"ETH","network_id":1,"websocket_url":"wss://mainnet.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33","platform":"evm","protocol_id":"pow","engine_id":"ethash","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8050,8051,30300],"udp":[30300]}}}}
	//{"block_explorer_url":"https://ropsten.etherscan.io","client":"geth","engine_id":"ethash","is_ethereum_network":true,"native_currency":"ETH","network_id":3,"platform":"evm","protocol_id":"pow","websocket_url":"wss://ropsten.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33","json_rpc_url":"https://ropsten.infura.io/v3/fde5e81d5d3141a093def423db3eeb33","security":{"egress":"*","ingress":{"0.0.0.0/0":{"tcp":[8545,8546,8547,30303],"udp":[30303]}}}}
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
		Config     chainSpecConfig `json:"config"`
		Alloc      allocation      `json:"alloc"`
		Coinbase   *string         `json:"coinbase"`
		Difficulty *string         `json:"difficulty"`
		ExtraData  *string         `json:"extraData"`
		GasLimit   *string         `json:"gasLimit"`
		Nonce      *string         `json:"nonce"`
		Mixhash    *string         `json:"mixhash"`
		ParentHash *string         `json:"parentHash"`
		Timestamp  *string         `json:"timestamp"`
	}

	type chainConfig struct {
		NativeCurrency *string   `json:"native_currency"`
		Platform       *string   `json:"platform"`
		EngineID       *string   `json:"engine_id"`
		Chain          *string   `json:"chain"`
		ProtocolID     *string   `json:"protocol_id"`
		ChainSpec      chainSpec `json:"chainspec"`
	}

	type chainDef struct {
		Name      *string     `json:"name"`
		Cloneable bool        `json:"cloneable"`
		Config    chainConfig `json:"config"`
	}

	chainySpecConfig := chainSpecConfig{
		0,
		0,
		0,
		0,
		0,
		0,
		0,
	}

	chainyAlloc := allocation{
		common.StringOrNil("this left blank"),
	}

	chainySpec := chainSpec{
		chainySpecConfig,
		chainyAlloc,
		common.StringOrNil("0x0000000000000000000000000000000000000000"),
		common.StringOrNil("0x20000"),
		common.StringOrNil(""),
		common.StringOrNil("0x2fefd8"),
		common.StringOrNil("0x0000000000000042"),
		common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		common.StringOrNil("0x00"),
	}

	chainyConfig := chainConfig{
		common.StringOrNil("TEST"),
		common.StringOrNil("evm"),
		common.StringOrNil("ethash"),
		common.StringOrNil("test"),
		common.StringOrNil("pow"),
		chainySpec,
	}

	chainyChain := chainDef{
		common.StringOrNil("baseline testnet"),
		false,
		chainyConfig,
	}

	t.Logf("chainychain: %+v", chainyChain)
	cc, _ := json.Marshal(chainyChain)
	t.Logf("cc: %s", string(cc))

	config := map[string]interface{}{}
	json.Unmarshal(cc, &config)

	if err != nil {
		t.Errorf("error marshalling config json")
	}
	t.Logf("config: %+v", config)

	status, resp, err := provide.CreateNetwork(*token, map[string]interface{}{
		"name":   testId.String() + " test network",
		"config": &config,
	})

	t.Logf("status: %v", status)
	t.Logf("response %s", resp)
}

func TestGetNetworkDetails(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	networks := []struct {
		name string
		ID   string
	}{
		{"Ethereum mainnet", "deca2436-21ba-4ff5-b225-ad1b0b2f5c59"},
		{"Ethereum Rinkeby testnet", "07102258-5e49-480e-86af-6d0c3260827d"},
		{"Ethereum Ropsten testnet", "66d44f30-9092-4182-a3c4-bc02736d6ae5"},
		{"Ethereum Kovan testnet", "8d31bf48-df6b-4a71-9d7c-3cb291111e27"},
		{"Ethereum Görli Testnet", "1b16996e-3595-4985-816c-043345d22f8c"},
	}

	for _, network := range networks {
		_, err := provide.GetNetworkDetails(*token, network.ID, map[string]interface{}{})
		if err != nil {
			t.Errorf("error getting network details for %s. Error: %s", network.name, err.Error())
			return
		}
	}
}

func TestEnableRopsten(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	ropsten, err := provide.GetNetworkDetails(*token, ropstenNetworkID, map[string]interface{}{})
	if err != nil {
		t.Errorf("error getting network details for network %s. Error: %s", ropstenNetworkID, err.Error())
		return
	}

	err = provide.UpdateNetwork(*token, ropstenNetworkID, map[string]interface{}{
		"enabled": true,
		"config":  ropsten.Config,
	})

	if err != nil {
		t.Errorf("error enabling ropsten network. Error: %s", err.Error())
		return
	}
}

// Note: this fails because none of the default networks are enabled
// and when attempting to enable ropsten, I hit an issue in the config where
// it was missing a chainspec, which is where I stopped on that rabbit hole
func TestListNetworks(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	networks, err := provide.ListNetworks(*token, &True, &False, map[string]interface{}{
		"public": true,
	})
	if err != nil {
		t.Errorf("error listing networks %s", err.Error())
		return
	}

	t.Logf("networks returned: %+v", networks)
}

func TestListNetworkAddresses(t *testing.T) {
	t.Logf("not implemented")
}

func TestListNetworkBlocks(t *testing.T) {
	t.Logf("not implemented")
}

func TestListNetworkBridges(t *testing.T) {
	t.Logf("not implemented")
}

func TestListNetworkConnectors(t *testing.T) {
	t.Logf("not implemented")
}

func TestNetworkStatus(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	networks := []struct {
		name    string
		ID      string
		chainID string
	}{
		{"Ethereum mainnet", "deca2436-21ba-4ff5-b225-ad1b0b2f5c59", "1"},
		{"Ethereum Rinkeby testnet", "07102258-5e49-480e-86af-6d0c3260827d", "4"},
		{"Ethereum Ropsten testnet", "66d44f30-9092-4182-a3c4-bc02736d6ae5", "3"},
		{"Ethereum Kovan testnet", "8d31bf48-df6b-4a71-9d7c-3cb291111e27", "42"},
		{"Ethereum Görli Testnet", "1b16996e-3595-4985-816c-043345d22f8c", "5"},
	}

	for _, network := range networks {
		status, err := provide.GetNetworkStatusMeta(*token, network.ID, map[string]interface{}{})
		if err != nil {
			t.Errorf("error getting network status for %s network", network.name)
			return
		}

		if *status.ChainID != network.chainID {
			t.Errorf("incorrect chainID returned. expected %s, got %s", network.chainID, *status.ChainID)
			return
		}
	}
}

func TestCreateLoadBalancer(t *testing.T) {
	t.Errorf("no handler to create load balancer")
	return
}

func TestListLoadBalancers(t *testing.T) {
	t.Logf("test to be completed")
}

func TestUpdateLoadBalancer(t *testing.T) {
	t.Logf("test to be completed")
}

func TestRemoveLoadBalancer(t *testing.T) {
	t.Logf("no handler for this purpose - is it needed")
}

func TestListNodes(t *testing.T) {
	t.Logf("test to be completed")
}

func TestCreateNode(t *testing.T) {
	// needs an application user token...
	// maybe stay away from this one until I get a walkthrough
	// passes in aws credentials and don't want to stand up a node just yet...
}

func TestNodeDetails(t *testing.T) {
	t.Logf("test to be completed")
}

func TestUpdateNode(t *testing.T) {
	t.Logf("test to be completed")
}

func TestDeleteNode(t *testing.T) {
	t.Logf("test to be completed")
}

func TestNodesLogs(t *testing.T) {
	t.Logf("test to be completed")
}
