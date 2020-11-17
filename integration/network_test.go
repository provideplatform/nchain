// +build integration

package integration

import (
	"encoding/json"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/api/nchain"
)

func init() {
	// let's enable ropsten and use it as the network id for the moment
	// todo: test enabling all the chains - but need correct chain specs for them all

	testId, err := uuid.NewV4()
	if err != nil {
		common.Log.Debugf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		common.Log.Debugf("user authentication failed. Error: %s", err.Error())
	}

	ropsten, err := provide.GetNetworkDetails(*token, ropstenNetworkID, map[string]interface{}{})
	if err != nil {
		common.Log.Debugf("error getting network details for network %s. Error: %s", ropstenNetworkID, err.Error())
	}

	// let's try marshalling the ropsten config into my objects
	config := &chainConfig{}
	configRaw, _ := json.Marshal(ropsten.Config)
	err = json.Unmarshal(configRaw, &config)
	if err != nil {
		common.Log.Debugf("failed to marshal ropsten config. Error: %s", err.Error())
	}

	common.Log.Debugf("chain config from db: %+v", *config)
	// we'll add values the config is missing in order to enable ropsten
	ropstenSpecConfig := chainSpecConfig{
		HomesteadBlock:      0,
		Eip150Block:         0,
		Eip155Block:         0,
		Eip158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
	}

	ropstenAlloc := allocation{
		notexportedhack: common.StringOrNil("this left blank"),
	}

	ropstenChainSpec := chainSpec{
		Config:     &ropstenSpecConfig,
		Alloc:      &ropstenAlloc,
		Coinbase:   common.StringOrNil("0x0000000000000000000000000000000000000000"),
		Difficulty: common.StringOrNil("0x20000"),
		ExtraData:  common.StringOrNil(""),
		GasLimit:   common.StringOrNil("0x2fefd8"),
		Nonce:      common.StringOrNil("0x0000000000000042"),
		Mixhash:    common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		ParentHash: common.StringOrNil("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Timestamp:  common.StringOrNil("0x00"),
	}

	ropstenChainConfig := chainConfig{
		NativeCurrency: common.StringOrNil("TEST"),
		Platform:       common.StringOrNil("evm"),
		EngineID:       common.StringOrNil("ethash"),
		Chain:          common.StringOrNil("test"),
		ProtocolID:     common.StringOrNil("pow"),
		ChainSpec:      &ropstenChainSpec,
	}

	config.ChainSpec = &ropstenChainSpec
	err = provide.UpdateNetwork(*token, ropstenNetworkID, map[string]interface{}{
		"enabled": true,
		"config":  ropstenChainConfig,
	})

	if err != nil {
		common.Log.Debugf("error enabling ropsten network. Error: %s", err.Error())
	}

	common.Log.Debugf("ropsten enabled")
}

// Note: this will fail if the db volume isn't removed, as this uses an existing chain_id
// which has a unique index
func TestCreateNetwork(t *testing.T) {
	// let's try it from the docs!

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	_, err = NetworkFactory(*token, testId)
	if err != nil {
		t.Errorf("Error creating network. Error: %s", err.Error())
		return
	}

}

func TestGetNetworkDetails(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
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

// Note: this fails because none of the default networks are enabled
// and when attempting to enable ropsten, I hit an issue in the config where
// it was missing a chainspec, which is where I stopped on that rabbit hole
func TestListNetworks(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
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

	// we have only enabled Ropsten, so let's fix to that for the moment
	ropstenFound := false
	for counter, network := range networks {
		t.Logf("network %v returned: %s", counter, *network.Name)
		if *network.Name == ropstenNetworkName {
			ropstenFound = true
		}
	}

	if ropstenFound != true {
		t.Errorf("ropsten network not found in network listing")
		return
	}
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

	token, err := UserAndTokenFactory(testId)
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
