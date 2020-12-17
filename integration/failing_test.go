// +build failing

package integration

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
	provide "github.com/provideservices/provide-go/api/nchain"
)

// Note: this fails because none of the default networks are enabled
// and when attempting to enable ropsten, I hit an issue in the config where
// it was missing a chainspec, which is where I stopped on that rabbit hole
func TestListNetworks(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	networks, err := provide.ListNetworks(*token, map[string]interface{}{
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

// likely fails because orgs are not fully integrated with nchain
func TestExecuteContract_Organization(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	testcaseOrg := Organization{
		"org" + testId.String(),
		"orgdesc " + testId.String(),
	}

	org, err := orgFactory(*userToken, testcaseOrg.name, testcaseOrg.description)
	if err != nil {
		t.Errorf("error setting up organization. Error: %s", err.Error())
		return
	}

	orgToken, err := orgTokenFactory(*userToken, org.ID)
	if err != nil {
		t.Errorf("error getting org token. Error: %s", err.Error())
		return
	}

	// create the account for that user, for the Ropsten network
	account, err := nchain.CreateAccount(*orgToken.Token, map[string]interface{}{
		"network_id":      ropstenNetworkID,
		"organization_id": org.ID,
	})

	if err != nil {
		t.Errorf("error creating user account. Error: %s", err.Error())
	}

	// deploy the contract
	params := map[string]interface{}{}

	var contractArtifact map[string]interface{}

	json.Unmarshal([]byte(ekhoArtifact), contractArtifact)

	parameter := fmt.Sprintf(`{
		"account_id": "%s",
		"compiled_artifact": %s
		}`, account.ID, ekhoArtifact)

	json.Unmarshal([]byte(parameter), &params)

	contract, err := nchain.CreateContract(*orgToken.Token, map[string]interface{}{
		"network_id":      ropstenNetworkID,
		"organization_id": org.ID.String(),
		"account_id":      account.ID.String(),
		"name":            "Ekho",
		"address":         "0x",
		"params":          params,
	})
	if err != nil {
		t.Errorf("error creating contract. Error: %s", err.Error())
		return
	}

	started := time.Now().Unix()

	for {
		if time.Now().Unix()-started >= 60 {
			t.Error("timed out awaiting contract address")
			return
		}

		deployedContract, err := nchain.GetContractDetails(*orgToken.Token, contract.ID.String(), map[string]interface{}{})
		if err != nil {
			t.Errorf("error fetching contract details; %s", err.Error())
			return
		}

		if deployedContract.Address != nil && *deployedContract.Address != "0x" {
			t.Logf("contract address resolved; contract id: %s; address: %s", deployedContract.ID.String(), *deployedContract.Address)
			break
		}

		t.Logf("contract address has not yet been resolved; contract id: %s", deployedContract.ID.String())
		time.Sleep(contractSleepTime * time.Second)
	}

	// generate a random string from bytes
	randomBytes := make([]byte, 118)
	_, err = rand.Read(randomBytes)
	msg := string(randomBytes)

	params = map[string]interface{}{}
	parameter = fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "account_id":"%s"}`, msg, account.ID.String())

	json.Unmarshal([]byte(parameter), &params)

	execResponse, err := nchain.ExecuteContract(*orgToken.Token, contract.ID.String(), params)

	if err != nil {
		t.Errorf("error executing contract: %s", err.Error())
		return
	}

	started = time.Now().Unix()

	for {
		if time.Now().Unix()-started >= 60 {
			t.Error("timed out awaiting transaction hash")
			return
		}

		tx, err := nchain.GetTransactionDetails(*orgToken.Token, *execResponse.Reference, map[string]interface{}{})
		//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
		if err != nil {
			t.Logf("error fetching transaction; %s", err.Error())
		}

		if err == nil {
			if tx.Hash != nil && *tx.Hash != "0x" {
				t.Logf("tx resolved; tx id: %s; hash: %s", tx.ID.String(), *tx.Hash)
				break
			}

			t.Logf("transaction has not yet been resolved; tx id: %s", tx.ID.String())
		}
		time.Sleep(contractSleepTime * time.Second)
	}

	t.Logf("contract execution successful")
}
