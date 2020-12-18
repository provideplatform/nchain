// +build integration nchain rinkeby

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/vault/common"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestExecuteContractRinkeby(t *testing.T) {
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

	// create the account for that user, for the Rinkeby network
	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     rinkebyNetworkID,
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
		"network_id":     rinkebyNetworkID,
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
		if time.Now().Unix()-started >= contractTimeout {
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
		time.Sleep(contractSleepTime * time.Second)
	}

	t.Logf("contract is: %+v", contract)

	// generate a random string from bytes
	msg := common.RandomString(118)

	params = map[string]interface{}{}
	parameter = fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "account_id":"%s"}`, msg, account.ID.String())

	json.Unmarshal([]byte(parameter), &params)

	execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)

	t.Logf("contractTx: %+v", execResponse)
	t.Logf("reference: %s", *execResponse.Reference)
	if err != nil {
		t.Errorf("error executing contract: %s", err.Error())
		return
	}

	started = time.Now().Unix()

	for {
		if time.Now().Unix()-started >= transactionTimeout {
			t.Error("timed out awaiting transaction hash")
			return
		}

		tx, err := nchain.GetTransactionDetails(*appToken.Token, *execResponse.Reference, map[string]interface{}{})
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
		time.Sleep(transactionSleepTime * time.Second)
	}

	t.Logf("contract execution successful")
}

func TestBulkExecuteContractRinkeby(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	// number of transactions to execute against the deployed contract
	const numberofTransactions = 50

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

	// create the account for that user, for the Rinkeby network
	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     rinkebyNetworkID,
		"application_id": app.ID,
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

	contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
		"network_id":     rinkebyNetworkID,
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
		if time.Now().Unix()-started >= contractTimeout {
			t.Error("timed out awaiting contract address")
			return
		}

		deployedContract, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
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

	var references [numberofTransactions]string //we'll store the returned references in here

	// send a bunch of transactions at the contract
	for execloop := 0; execloop < numberofTransactions; execloop++ {

		msg := common.RandomString(118)

		params = map[string]interface{}{}
		parameter = fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "account_id":"%s"}`, msg, account.ID.String())

		json.Unmarshal([]byte(parameter), &params)

		execResponse, err := nchain.ExecuteContract(*userToken, contract.ID.String(), params)
		if err != nil {
			t.Errorf("error executing contract: %s", err.Error())
			return
		}
		references[execloop] = *execResponse.Reference
	}

	// let's give the nchain-consumer a chance to get started on the account funding
	time.Sleep(numberofTransactions * time.Second)

	// ok, now we'll confirm that each of the transactions has succeeded
	// (or at least hit the mempool and has a transaction hash )
	started = time.Now().Unix()
	mined := 0

	for txloop := 0; txloop < numberofTransactions; txloop++ {
		for {
			if time.Now().Unix()-started >= transactionTimeout {
				t.Error("timed out awaiting transaction hash")
				return
			}

			tx, err := nchain.GetTransactionDetails(*appToken.Token, references[txloop], map[string]interface{}{})
			//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
			if err != nil {
				t.Logf("error fetching transaction; %s", err.Error())
			}

			if err == nil {
				if tx.Hash != nil && *tx.Hash != "0x" {
					t.Logf("tx resolved; tx id: %s; hash: %s", tx.ID.String(), *tx.Hash)
					mined++
					break
				}

				t.Logf("transaction has not yet been resolved; tx id: %s", tx.ID.String())
			}
			time.Sleep(transactionSleepTime * time.Second)
		}
	}

	if mined != numberofTransactions {
		t.Errorf("not all transactions were mined. expected %d, mined %d", numberofTransactions, mined)
		return
	}

	t.Logf("%d transactions mined successfully", mined)
}
