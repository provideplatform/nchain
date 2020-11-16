// +build integration

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go/api/nchain"
)

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

func TestEnableRopsten(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
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
