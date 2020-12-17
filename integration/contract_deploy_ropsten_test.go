// +build integration nchain ropsten

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestDeployEkhoContractRopsten(t *testing.T) {
	t.Parallel()

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	testcaseApp := Application{
		"app" + testId.String(),
		"appdesc " + testId.String(),
	}

	app, err := appFactory(*userToken, testcaseApp.name, testcaseApp.description)
	if err != nil {
		t.Errorf("error setting up application. Error: %s", err.Error())
		return
	}

	appToken, err := appTokenFactory(*userToken, app.ID)
	if err != nil {
		t.Errorf("error getting app token. Error: %s", err.Error())
		return
	}

	// create the account for that user, for the Ropsten network
	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID,
	})

	if err != nil {
		t.Errorf("error creating user account. Error: %s", err.Error())
	}

	tt := []struct {
		name      string
		parameter string
	}{
		{"ekho", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, ekhoArtifact)},
	}

	params := map[string]interface{}{}

	for _, tc := range tt {
		json.Unmarshal([]byte(tc.parameter), &params)

		contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
			"network_id":     ropstenNetworkID,
			"application_id": app.ID.String(),
			"account_id":     account.ID.String(),
			"name":           "Ekho",
			"address":        "0x",
			"params":         params,
		})
		if err != nil {
			t.Errorf("error creating contract %s. Error: %s", tc.name, err.Error())
			return
		}

		started := time.Now().Unix()

		for {
			if time.Now().Unix()-started >= contractTimeout {
				t.Errorf("timed out awaiting contract address for %s contract", tc.name)
				return
			}

			cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
			if err != nil {
				t.Errorf("error fetching %s contract details; %s", tc.name, err.Error())
				return
			}

			if cntrct.Address != nil && *cntrct.Address != "0x" {
				t.Logf("%s contract address resolved; contract id: %s; address: %s", tc.name, cntrct.ID.String(), *cntrct.Address)
				break
			}

			t.Logf("%s contract address has not yet been resolved; contract id: %s", tc.name, cntrct.ID.String())
			time.Sleep(contractSleepTime * time.Second)
		}
	}
}

func TestDeployContractsRopsten(t *testing.T) {
	t.Parallel()

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	testcaseApp := Application{
		"app" + testId.String(),
		"appdesc " + testId.String(),
	}

	app, err := appFactory(*userToken, testcaseApp.name, testcaseApp.description)
	if err != nil {
		t.Errorf("error setting up application. Error: %s", err.Error())
		return
	}

	appToken, err := appTokenFactory(*userToken, app.ID)
	if err != nil {
		t.Errorf("error getting app token. Error: %s", err.Error())
		return
	}

	// create the account for that user, for the Ropsten network
	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID,
	})

	if err != nil {
		t.Errorf("error creating user account. Error: %s", err.Error())
	}

	tt := []struct {
		name      string
		parameter string
	}{
		{"ekho", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, ekhoArtifact)},
		{"ERC1820", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, ERC1820RegistryArtifact)},
		// {"Shuttle", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, ShuttleArtifact)},
		{"Registry", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, RegistryArtifact)},
		//{"Shield", fmt.Sprintf(`{"account_id": "%s","compiled_artifact": %s}`, account.ID, ShieldArtifact)},
	}

	params := map[string]interface{}{}

	for _, tc := range tt {
		json.Unmarshal([]byte(tc.parameter), &params)

		contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
			"network_id":     ropstenNetworkID,
			"application_id": app.ID.String(),
			"account_id":     account.ID.String(),
			"name":           "Ekho",
			"address":        "0x",
			"params":         params,
		})
		if err != nil {
			t.Errorf("error creating contract %s. Error: %s", tc.name, err.Error())
			return
		}

		started := time.Now().Unix()

		for {
			if time.Now().Unix()-started >= contractTimeout {
				t.Errorf("timed out awaiting contract address for %s contract", tc.name)
				return
			}

			cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
			if err != nil {
				t.Errorf("error fetching %s contract details; %s", tc.name, err.Error())
				return
			}

			if cntrct.Address != nil && *cntrct.Address != "0x" {
				t.Logf("%s contract address resolved; contract id: %s; address: %s", tc.name, cntrct.ID.String(), *cntrct.Address)
				break
			}

			t.Logf("%s contract address has not yet been resolved; contract id: %s", tc.name, cntrct.ID.String())
			time.Sleep(contractSleepTime * time.Second)
		}

	}
}
