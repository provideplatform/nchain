//go:build integration || nchain || ropsten || rinkeby || kovan || goerli || basic || bookie || readonly
// +build integration nchain ropsten rinkeby kovan goerli basic bookie readonly

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

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideplatform/provide-go/api/nchain"
)

// Note: this will fail if the db volume isn't removed, as this uses an existing chain_id
// which has a unique index
func TestCreateNetwork(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Logf("no handler to create load balancer")
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
