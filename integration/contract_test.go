// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
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
	account, err := GoCreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID,
	})

	if err != nil {
		t.Errorf("error creating user account. Error: %s", err.Error())
	}
	t.Logf("account created: %+v", account)

	params := map[string]interface{}{}
	//contractName := fmt.Sprintf("ekho(%s)", testId.String())

	// parameter := fmt.Sprintf(`{
	// "account_id": "%s",
	// "lang":"solidity",
	// "name":"%s",
	// "raw_source": "pragma solidity >=0.4.22 <0.7.5;contract ekhoprotocol {event ekho(bytes message);function broadcast(bytes memory message) public {emit ekho(message);}}"
	// }`, account.ID.String(), contractName)

	//unmarshal the unstructured json into mapstringinterface

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

	for {
		cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
		if err != nil {
			t.Errorf("error fetching contract details; %s", err.Error())
			return
		}

		if cntrct.Address != nil {
			t.Logf("contract address resolved; contract id: %s; address: %s", cntrct.ID.String(), *cntrct.Address)
			break
		}

		t.Logf("contract address has not yet been resolved; contract id: %s", cntrct.ID.String())
		time.Sleep(5)
	}

	t.Logf("contract is: %+v", contract)
}
