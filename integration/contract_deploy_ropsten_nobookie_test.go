// +build integration nchain nobookie

package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/ident/common"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestEkhoContractWithSeededHDWallet_Ropsten(t *testing.T) {
	// currently deploys to ropsten
	// simple extensions:
	// - deploy to every chain
	// - deploy multiple contracts to every chain
	// execute a function on each contract
	// execute a read only function on every contract (if available)

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

	// create the deterministic wallet via nchain
	// note there's a question here about the token used to access the wallet
	// everything works, but there's a security cutoff with the vault interaction
	// medium term, this activity should be run on vault and the results passed to nchain
	// i.e. rather than nchain creating the wallet
	// vault is used to create the wallet and nchain builds the transaction and returns the hash to be signed by vault key
	// then the signed transaction is played to the consumer for delivery on chain

	wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
		"mnemonic": "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	})
	if err != nil {
		t.Errorf("error creating wallet: %s", err.Error())
		return
	}
	t.Logf("wallet created: %+v", wallet)
	t.Logf("wallet public key: %s", *wallet.PublicKey)
	t.Logf("vault id for wallet: %s", wallet.VaultID.String())
	t.Logf("key id for wallet: %s", wallet.KeyID.String())

	//"mnemonic":       "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	// create the account for that user, for the Ropsten network
	// account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
	// 	"network_id":     ropstenNetworkID,
	// 	"application_id": app.ID,
	// })

	// if err != nil {
	// 	t.Errorf("error creating user account. Error: %s", err.Error())
	// 	return
	// }

	// t.Logf("account id: %s", account.ID)

	// set up the path we'll use for all of these nchain interactions

	path := `m/44'/60'/2'/0/0`
	t.Logf("test path for hd wallet: %s", path)
	// this describes the test cases, at the moment, we will just deploy a single contract
	// but this can be extended to many contracts when needed, and network ids can be included
	// to hit all the test networks

	// // Shuttle manifest loading and compiledArtifact creation
	// manifest, err := ioutil.ReadFile("provide-capabilities-manifest.json")
	// if err != nil {
	// 	t.Errorf("error loading shuttle manifest. Error: %s", err.Error())
	// 	return
	// }

	// manifestString := fmt.Sprintf("%+v", string(manifest))

	// // convert the json to map[string]interface
	// manifestMap := map[string]interface{}{}
	// err = json.Unmarshal([]byte(manifestString), &manifestMap)
	// if err != nil {
	// 	t.Errorf("error converting json string to map. Error: %s", err.Error())
	// 	return
	// }

	// shuttleCompiledArtifact := nchain.CompiledArtifact{}
	// baseline := manifestMap["baseline"].(map[string]interface{})

	// contractArray := baseline["contracts"].([]interface{})

	// contractRaw, _ := json.Marshal(contractArray[2])
	// _ = json.Unmarshal(contractRaw, &shuttleCompiledArtifact)
	// // end of Shuttle manifest loading

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

	t.Logf("deploying ekho artifact: %s", ekhoArtifact)

	tt := []struct {
		name           string
		derivationPath string
		walletID       string
		artifact       nchain.CompiledArtifact
	}{
		{"ekho", path, wallet.ID.String(), ekhoCompiledArtifact},
		//{"orgRegistry", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %s}`, wallet.ID, path, greeterArtifact)},
		//{"shuttle", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %v}`, wallet.ID, path, compiledArtifact)},
	}

	for _, tc := range tt {

		t.Logf("creating contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)
		contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
			"network_id":     goerliNetworkID,
			"application_id": app.ID.String(),
			"wallet_id":      tc.walletID,
			//"account_id": account.ID.String(),
			"name":    tc.name,
			"address": "0x",
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

		// wait for the contract to be deployed
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

			//t.Logf("%s contract address has not yet been resolved; contract id: %s", tc.name, cntrct.ID.String())
			t.Logf("resolving contract...")
			time.Sleep(contractSleepTime * time.Second)
		}

		// comment this out for the moment to focus on the contract deployment code
		// create a message for ekho
		msg := common.RandomString(118)
		t.Logf("executing contract using wallet id: %s, derivation path: %s", tc.walletID, tc.derivationPath)

		params := map[string]interface{}{}

		parameter := fmt.Sprintf(`{"method":"broadcast", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s"}`, tc.derivationPath, msg, tc.walletID)
		//parameter := fmt.Sprintf(`{"method":"getOrgCount", "hd_derivation_path": "%s", params": [], "value":0, "wallet_id":"%s"}`, path, wallet.ID.String())
		json.Unmarshal([]byte(parameter), &params)

		//t.Logf("params for contract execution: %+v", params)
		// execute the contract method
		execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
		if err != nil {
			t.Errorf("error executing contract. Error: %s", err.Error())
			return
		}
		// t.Logf("contractTx: %+v", execResponse)
		// t.Logf("reference: %s", *execResponse.Reference)
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

			tx, err := nchain.GetTransactionDetails(*appToken.Token, *execResponse.Reference, map[string]interface{}{})
			//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
			if err != nil {
				t.Logf("tx not yet mined...")
				//t.Logf("error fetching transaction; %s", err.Error())
			}

			if err == nil {
				if tx.Hash != nil && *tx.Hash != "0x" {
					t.Logf("tx resolved; tx id: %s; hash: %s", tx.ID.String(), *tx.Hash)
					//t.Logf("tx returned: %+v", tx)
					break
				}

				//t.Logf("transaction has not yet been resolved; tx id: %s", tx.ID.String())
				t.Logf("resolving transaction...")
			}
			time.Sleep(transactionSleepTime * time.Second)
		}

		t.Logf("contract execution successful")
	}
}

// full test (contract deployment, execution and readonly test)
// 1. run through all networks (ropsten, rinkeby, kovan, goerli)
// 2. for each network:
//    - set up a wallet, using a hd derivation path
//    - create a contract
//    - execute a transaction on the contract using an assigned method, and params to be passed into that method
// 3. ensure we have mined all posted transactions

// do the same for an nchain created account (which will post via bookie)
// do the same for an nchain created wallet, but with a 0-balance derivation path (which will post via bookie)
// separate the tests, so we can run
// bookie vs. non-bookie (so fails can determine if bookie is running)
// separate chains?  this way the tests can be aimed at individual chains so we can determine individual failures
// by using a different derivation path for each test, we can run them all in parallel (even for individual chains)
