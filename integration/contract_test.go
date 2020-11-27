// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideservices/provide-go/api"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

// Contract instances must be associated with an application identifier.
type Contract struct {
	api.Model
	ApplicationID *uuid.UUID       `json:"application_id"`
	NetworkID     uuid.UUID        `json:"network_id"`
	ContractID    *uuid.UUID       `json:"contract_id"`    // id of the contract which created the contract (or null)
	TransactionID *uuid.UUID       `json:"transaction_id"` // id of the transaction which deployed the contract (or null)
	Name          *string          `json:"name"`
	Address       *string          `json:"address"`
	Type          *string          `json:"type"`
	Params        *json.RawMessage `json:"params,omitempty"`
	AccessedAt    *time.Time       `json:"accessed_at"`
	PubsubPrefix  *string          `json:"pubsub_prefix,omitempty"`
}

// TODO move this to provide-go when it's working
// GoCreateContract
func GoCreateContract(token string, params map[string]interface{}) (*Contract, error) {
	// return InitNChainService(token).Post("contracts", params)
	uri := "contracts"
	status, resp, err := nchain.InitNChainService(token).Post(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 201 && status != 202 {
		return nil, fmt.Errorf("failed to create contract. status: %v. resp: %v", status, resp)
	}

	contract := &Contract{}
	contractRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(contractRaw, &contract)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract. status: %v; %s", status, err.Error())
	}

	return contract, nil

}

// TODO move this to provide-go when it's working
// GoExecuteContract
func GoExecuteContract(token, contractID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("contracts/%s/execute", contractID)
	return nchain.InitNChainService(token).Post(uri, params)
}

// TODO move this to provide-go when it's working
// GoListContracts
func GoListContracts(token string, params map[string]interface{}) (int, interface{}, error) {
	return nchain.InitNChainService(token).Get("contracts", params)
}

// TODO move this to provide-go when it's working
// GoGetContractDetails
func GoGetContractDetails(token, contractID string, params map[string]interface{}) (int, interface{}, error) {
	uri := fmt.Sprintf("contracts/%s", contractID)
	return nchain.InitNChainService(token).Get(uri, params)
}

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

	contract, err := GoCreateContract(*appToken.Token, map[string]interface{}{
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

	t.Logf("contract is: %+v", contract)
}
