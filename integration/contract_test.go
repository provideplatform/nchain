// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/ident/common"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestDeployContract(t *testing.T) {
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
	t.Logf("account created: %+v", account)

	params := map[string]interface{}{}

	var contractArtifact map[string]interface{}

	json.Unmarshal([]byte(ekhoArtifact), contractArtifact)

	parameter := fmt.Sprintf(`{
		"account_id": "%s",
		"compiled_artifact": %s
		}`, account.ID, ekhoArtifact)

	json.Unmarshal([]byte(parameter), &params)

	contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID.String(),
		"account_id":     account.ID.String(),
		"name":           "Ekho",
		"address":        "0x",
		"params":         params,
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

		cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
		if err != nil {
			t.Errorf("error fetching contract details; %s", err.Error())
			return
		}

		if cntrct.Address != nil && *cntrct.Address != "0x" {
			t.Logf("contract address resolved; contract id: %s; address: %s", cntrct.ID.String(), *cntrct.Address)
			break
		}

		t.Logf("contract address has not yet been resolved; contract id: %s", cntrct.ID.String())
		time.Sleep(5 * time.Second)
	}

	t.Logf("contract is: %+v", contract)
}

func TestExecuteContract(t *testing.T) {
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
	t.Logf("account created: %+v", account)

	// deploy the contract
	params := map[string]interface{}{}

	var contractArtifact map[string]interface{}

	json.Unmarshal([]byte(ekhoArtifact), contractArtifact)

	parameter := fmt.Sprintf(`{
		"account_id": "%s",
		"compiled_artifact": %s
		}`, account.ID, ekhoArtifact)

	json.Unmarshal([]byte(parameter), &params)

	contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID.String(),
		"account_id":     account.ID.String(),
		"name":           "Ekho",
		"address":        "0x",
		"params":         params,
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

		cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
		if err != nil {
			t.Errorf("error fetching contract details; %s", err.Error())
			return
		}

		if cntrct.Address != nil && *cntrct.Address != "0x" {
			t.Logf("contract address resolved; contract id: %s; address: %s", cntrct.ID.String(), *cntrct.Address)
			break
		}

		t.Logf("contract address has not yet been resolved; contract id: %s", cntrct.ID.String())
		time.Sleep(5 * time.Second)
	}

	t.Logf("contract is: %+v", contract)

	msg := common.RandomString(118)

	params = map[string]interface{}{}
	parameter = fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "account_id":"%s"}`, msg, account.ID.String())

	json.Unmarshal([]byte(parameter), &params)

	execResponse, err := nchain.ExecuteContract(*userToken, contract.ID.String(), params)

	t.Logf("contractTx: %+v", execResponse)
	t.Logf("transaction id: %s", *execResponse.Reference)
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

		tx, err := nchain.GetTransactionDetails(*appToken.Token, *execResponse.Reference, map[string]interface{}{})
		if err != nil {
			t.Errorf("error fetching transaction; %s", err.Error())
			return
		}

		if tx.Hash != nil && *tx.Hash != "0x" {
			t.Logf("tx resolved; tx id: %s; address: %s", tx.ID.String(), *tx.Hash)
			break
		}

		t.Logf("transaction has not yet been resolved; tx id: %s", tx.ID.String())
		time.Sleep(5 * time.Second)
	}

	t.Logf("contract execution successful")
}
