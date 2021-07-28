// +build integration nchain rinkeby basic

package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/ident/common"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

func TestContractHDWalletRinkebyApp(t *testing.T) {

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

	// load the readwrite compiled artifact
	rwArtifact, err := ioutil.ReadFile("artifacts/readwritetester.json")
	if err != nil {
		t.Errorf("error loading readwritetester artifact. Error: %s", err.Error())
	}

	rwCompiledArtifact := nchain.CompiledArtifact{}
	err = json.Unmarshal(rwArtifact, &rwCompiledArtifact)
	if err != nil {
		t.Errorf("error converting readwritetester compiled artifact. Error: %s", err.Error())
	}

	tt := []struct {
		network        string
		name           string
		derivationPath string
		walletID       string
		artifact       nchain.CompiledArtifact
	}{
		{rinkebyNetworkID, "ekho", path, wallet.ID.String(), ekhoCompiledArtifact},
		{rinkebyNetworkID, "readwrite", path, wallet.ID.String(), rwCompiledArtifact},
	}

	for _, tc := range tt {
		// create unique contract reference
		contractRef, err := uuid.NewV4()
		if err != nil {
			t.Errorf("error creating unique contract ref. Error: %s", err.Error())
			return
		}

		t.Logf("creating contract using wallet id: %s, derivation path: %s, ref: %s", tc.walletID, tc.derivationPath, contractRef)
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
				"gas_price":          6000000000, //6 GWei
				"ref":                contractRef.String(),
			},
		})
		if err != nil {
			t.Errorf("error creating %s contract. Error: %s", tc.name, err.Error())
			return
		}

		// wait for the contract to be deployed
		started := time.Now().Unix()
		for {
			if time.Now().Unix()-started >= contractTimeout {
				t.Errorf("timed out awaiting contract address for %s contract for %s network", tc.name, tc.network)
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

			t.Logf("resolving contract for %s network ...", tc.network)
			time.Sleep(contractSleepTime * time.Second)
		}

		// create a message for contract
		msg := common.RandomString(118)
		t.Logf("msg: %s", msg)
		t.Logf("executing contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)

		params := map[string]interface{}{}
		var txRef uuid.UUID

		switch tc.name {
		case "ekho":
			txRef, err = uuid.NewV4()
			if err != nil {
				t.Errorf("error creating unique tx ref. Error: %s", err.Error())
				return
			}
			parameter := fmt.Sprintf(`{"method":"broadcast", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s", "ref": "%s", "gas_price": 6000000000}`, tc.derivationPath, msg, tc.walletID, txRef)
			t.Logf("parameter is: %s", parameter)
			json.Unmarshal([]byte(parameter), &params)
		case "readwrite":
			txRef, err = uuid.NewV4()
			if err != nil {
				t.Errorf("error creating unique tx ref. Error: %s", err.Error())
				return
			}
			parameter := fmt.Sprintf(`{"method":"setString", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s", "ref": "%s", "gas_price": 6000000000}`, tc.derivationPath, msg, tc.walletID, txRef)
			t.Logf("parameter is: %s", parameter)
			json.Unmarshal([]byte(parameter), &params)
		}

		// execute the contract method
		t.Logf("%s Executing contract using params %+v", time.Now(), params)
		execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
		if err != nil {
			t.Logf("%s got error %s, and execresponse is: %v", time.Now(), err.Error(), execResponse)
			t.Logf("error executing contract1, but hey, maybe it's not an error Error: %s", err.Error())
		}

		// wait for the transaction to be mined (get a tx hash)
		started = time.Now().Unix()
		for {
			if time.Now().Unix()-started >= transactionTimeout {
				t.Error("timed out awaiting transaction hash")
				return
			}

			tx, err := nchain.GetTransactionDetails(*appToken.Token, txRef.String(), map[string]interface{}{})
			//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
			if err != nil {
				t.Logf("tx not yet mined...")
			}

			if err == nil {
				if tx.Description != nil {
					t.Errorf("tx error; tx id: %s; Error: %s", tx.ID.String(), *tx.Description)
					return
				}
				if tx.Block != nil && *tx.Hash != "0x" {
					t.Logf("tx resolved; tx id: %s; hash: %s; block: %d", tx.ID.String(), *tx.Hash, *tx.Block)
					break
				}
				t.Logf("resolving transaction...")
			}

			time.Sleep(transactionSleepTime * time.Second)
		}

		// now we also run a readonly transaction for some of the contracts
		if tc.name == "readwrite" {
			// HACK, we should only check this when there is a block associated with the tx id!
			time.Sleep(10 * time.Second)
			params := map[string]interface{}{}
			parameter := fmt.Sprintf(`{"method":"getString", "hd_derivation_path": "%s", "params": [""], "value":0, "wallet_id":"%s"}`, tc.derivationPath, tc.walletID)
			json.Unmarshal([]byte(parameter), &params)

			// execute the contract method
			execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
			if err != nil {
				t.Errorf("error executing contract3. Error: %s", err.Error())
				return
			}
			if execResponse.Response != nil {
				t.Logf("execution response: %v", execResponse.Response)
				if execResponse.Response != msg {
					t.Errorf("expected msg %s returned. got %v", msg, execResponse.Response)
					return
				}
			}
			if execResponse.Response == nil {
				t.Errorf("expected msg returned, got nil response")
				return
			}

			if err != nil {
				t.Errorf("error executing contract4: %s", err.Error())
				return
			}
		}

		t.Logf("contract execution successful")
	}
}

// XXX temporarily disabled to reduce test noise
func TestContractHDWalletRinkebyOrg(t *testing.T) {

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

	wallet, err := nchain.CreateWallet(*orgToken.Token, map[string]interface{}{
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

	// load the readwrite compiled artifact
	rwArtifact, err := ioutil.ReadFile("artifacts/readwritetester.json")
	if err != nil {
		t.Errorf("error loading readwritetester artifact. Error: %s", err.Error())
	}

	rwCompiledArtifact := nchain.CompiledArtifact{}
	err = json.Unmarshal(rwArtifact, &rwCompiledArtifact)
	if err != nil {
		t.Errorf("error converting readwritetester compiled artifact. Error: %s", err.Error())
	}

	tt := []struct {
		network        string
		name           string
		derivationPath string
		walletID       string
		artifact       nchain.CompiledArtifact
	}{
		{rinkebyNetworkID, "ekho", path, wallet.ID.String(), ekhoCompiledArtifact},
		{rinkebyNetworkID, "readwrite", path, wallet.ID.String(), rwCompiledArtifact},
	}

	for _, tc := range tt {

		// create unique contract reference
		contractRef, err := uuid.NewV4()
		if err != nil {
			t.Errorf("error creating unique contract ref. Error: %s", err.Error())
			return
		}
		//TODO add a test that doesn't add a ref or gas price
		// currently refs are getting repeated (likely not getting created properly on nchain, before nats)
		t.Logf("creating contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)
		contract, err := nchain.CreateContract(*orgToken.Token, map[string]interface{}{
			"network_id":      tc.network,
			"organization_id": org.ID.String(),
			"wallet_id":       tc.walletID,
			"name":            tc.name,
			"address":         "0x",
			"params": map[string]interface{}{
				"wallet_id":          tc.walletID,
				"hd_derivation_path": tc.derivationPath,
				"gas_price":          6000000000, //6 GWei
				"compiled_artifact":  tc.artifact,
				"ref":                contractRef.String(),
			},
		})
		if err != nil {
			t.Errorf("error creating %s contract. Error: %s", tc.name, err.Error())
			return
		}

		// wait for the contract to be deployed
		started := time.Now().Unix()
		for {
			if time.Now().Unix()-started >= contractTimeout {
				t.Errorf("timed out awaiting contract address for %s contract for %s network", tc.name, tc.network)
				return
			}

			cntrct, err := nchain.GetContractDetails(*orgToken.Token, contract.ID.String(), map[string]interface{}{})
			if err != nil {
				t.Errorf("error fetching %s contract details; %s", tc.name, err.Error())
				return
			}

			if cntrct.Address != nil && *cntrct.Address != "0x" {
				t.Logf("%s contract address resolved; contract id: %s; address: %s", tc.name, cntrct.ID.String(), *cntrct.Address)
				break
			}

			t.Logf("resolving contract for %s network ...", tc.network)
			time.Sleep(contractSleepTime * time.Second)
		}

		// create a message for contract
		msg := common.RandomString(118)
		t.Logf("msg: %s", msg)
		t.Logf("executing contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)

		params := map[string]interface{}{}
		var txRef uuid.UUID

		switch tc.name {
		case "ekho":
			txRef, err = uuid.NewV4()
			if err != nil {
				t.Errorf("error creating unique tx ref. Error: %s", err.Error())
				return
			}
			parameter := fmt.Sprintf(`{"method":"broadcast", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s", "ref": "%s", "gas_price": 6000000000}`, tc.derivationPath, msg, tc.walletID, txRef)
			t.Logf("parameter is: %s", parameter)
			json.Unmarshal([]byte(parameter), &params)
		case "readwrite":
			txRef, err = uuid.NewV4()
			if err != nil {
				t.Errorf("error creating unique tx ref. Error: %s", err.Error())
				return
			}
			parameter := fmt.Sprintf(`{"method":"setString", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s", "ref": "%s", "gas_price": 6000000000}`, tc.derivationPath, msg, tc.walletID, txRef)
			t.Logf("parameter is: %s", parameter)
			json.Unmarshal([]byte(parameter), &params)
		}

		// execute the contract method
		execResponse, err := nchain.ExecuteContract(*orgToken.Token, contract.ID.String(), params)
		if err != nil {
			t.Errorf("error executing contract. Error: %s", err.Error())
			return
		}

		if err != nil {
			t.Errorf("error executing contract: %s", err.Error())
			return
		}

		// wait for the transaction to be mined (get a tx hash)
		started = time.Now().Unix()
		for {
			if time.Now().Unix()-started >= transactionTimeout {
				t.Error("timed out awaiting transaction hash")
				return
			}

			tx, err := nchain.GetTransactionDetails(*orgToken.Token, *execResponse.Reference, map[string]interface{}{})
			//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
			if err != nil {
				t.Logf("tx not yet mined...")
			}

			if err == nil {
				if tx.Block != nil && *tx.Hash != "0x" {
					t.Logf("tx resolved; tx id: %s; hash: %s; block: %d", tx.ID.String(), *tx.Hash, *tx.Block)
					break
				}
				t.Logf("resolving transaction...")
			}
			time.Sleep(transactionSleepTime * time.Second)
		}

		// now we also run a readonly transaction for some of the contracts
		if tc.name == "readwrite" {
			params := map[string]interface{}{}
			parameter := fmt.Sprintf(`{"method":"getString", "hd_derivation_path": "%s", "params": [""], "value":0, "wallet_id":"%s"}`, tc.derivationPath, tc.walletID)
			json.Unmarshal([]byte(parameter), &params)

			// execute the contract method
			execResponse, err := nchain.ExecuteContract(*orgToken.Token, contract.ID.String(), params)
			if err != nil {
				t.Errorf("error executing contract. Error: %s", err.Error())
				return
			}
			if execResponse.Response != nil {
				t.Logf("execution response: %v", execResponse.Response)
				if execResponse.Response != msg {
					t.Errorf("expected msg %s returned. got %v", msg, execResponse.Response)
					return
				}
			}
			if execResponse.Response == nil {
				t.Errorf("expected msg returned, got nil response")
				return
			}

			if err != nil {
				t.Errorf("error executing contract: %s", err.Error())
				return
			}
		}

		t.Logf("contract execution successful")
	}
}
