// +build integration nchain bulk

package integration

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestContractHDWalletKovanBulk(t *testing.T) {

	t.Parallel()

	const numberOfTransactions = 10
	deployedContracts := make([]string, numberOfTransactions)

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

	wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
		"mnemonic": "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	})
	if err != nil {
		t.Errorf("error creating wallet: %s", err.Error())
		return
	}

	// this path produces the ETH address 0x6af845bae76f5cc16bc93f86b83e8928c3dfda19
	path := `m/44'/60'/2'/0/0`

	// load the ekho compiled artifact
	ekhoArtifact, err := ioutil.ReadFile("artifacts/ekho.json")
	if err != nil {
		t.Errorf("error loading ekho artifact. Error: %s", err.Error())
	}

	ekhoCompiledArtifact := nchain.CompiledArtifact{}
	err = json.Unmarshal(ekhoArtifact, &ekhoCompiledArtifact)
	if err != nil {
		t.Errorf("error converting ekho compiled artifact. Error: %s", err.Error())
	}
	tt := []struct {
		network        string
		name           string
		derivationPath string
		walletID       string
		artifact       nchain.CompiledArtifact
	}{
		{kovanNetworkID, "ekho", path, wallet.ID.String(), ekhoCompiledArtifact},
	}

	for _, tc := range tt {
		// we will deploy the contract numberOfTransactions times
		for looper := 0; looper < numberOfTransactions; looper++ {
			t.Logf("creating contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)
			contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
				"network_id":     tc.network,
				"application_id": app.ID.String(),
				"wallet_id":      tc.walletID,
				"name":           tc.name,
				"address":        "0x",
				"params": map[string]interface{}{
					"wallet_id":          tc.walletID,
					"hd_derivation_path": tc.derivationPath,
					"compiled_artifact":  tc.artifact,
				},
			})
			if err != nil {
				t.Errorf("error creating %s contract. Error: %s", tc.name, err.Error())
				return
			}
			deployedContracts[looper] = contract.ID.String()
		} //looper

		t.Logf("deployed %v contracts", len(deployedContracts))
		// wait for the contract to be deployed
		for looper := 0; looper < numberOfTransactions; looper++ {
			started := time.Now().Unix()
			for {
				if time.Now().Unix()-started >= contractTimeout {
					t.Errorf("timed out awaiting contract address for %s contract for %s network", tc.name, tc.network)
					return
				}

				cntrct, err := nchain.GetContractDetails(*appToken.Token, deployedContracts[looper], map[string]interface{}{})
				if err != nil {
					t.Errorf("error fetching %s contract details; %s", tc.name, err.Error())
					return
				}

				if cntrct.Address != nil && *cntrct.Address != "0x" {
					t.Logf("%s contract address resolved; contract id: %s; address: %s", tc.name, cntrct.ID.String(), *cntrct.Address)
					break
				}

				t.Logf("resolving contract id:%v for %s network ...", deployedContracts[looper], tc.network)
				time.Sleep(contractSleepTime * time.Second)
			}
		}

		// 	// create a message for contract
		// 	msg := common.RandomString(118)
		// 	t.Logf("msg: %s", msg)
		// 	t.Logf("executing contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)

		// 	params := map[string]interface{}{}

		// 	switch tc.name {
		// 	case "ekho":
		// 		parameter := fmt.Sprintf(`{"method":"broadcast", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s"}`, tc.derivationPath, msg, tc.walletID)
		// 		json.Unmarshal([]byte(parameter), &params)
		// 	}

		// 	// execute the contract method
		// 	execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
		// 	if err != nil {
		// 		t.Errorf("error executing contract. Error: %s", err.Error())
		// 		return
		// 	}

		// 	if err != nil {
		// 		t.Errorf("error executing contract: %s", err.Error())
		// 		return
		// 	}

		// 	// wait for the transaction to be mined (get a tx hash)
		// 	started = time.Now().Unix()
		// 	for {
		// 		if time.Now().Unix()-started >= transactionTimeout {
		// 			t.Error("timed out awaiting transaction hash")
		// 			return
		// 		}

		// 		tx, err := nchain.GetTransactionDetails(*appToken.Token, *execResponse.Reference, map[string]interface{}{})
		// 		//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
		// 		if err != nil {
		// 			t.Logf("tx not yet mined...")
		// 		}

		// 		if err == nil {
		// 			if tx.Block != nil && *tx.Hash != "0x" {
		// 				t.Logf("tx resolved; tx id: %s; hash: %s; block: %d", tx.ID.String(), *tx.Hash, *tx.Block)
		// 				break
		// 			}
		// 			t.Logf("resolving transaction...")
		// 		}
		// 		time.Sleep(transactionSleepTime * time.Second)
		// 	}

		// 	t.Logf("contract execution successful")
	}
}
