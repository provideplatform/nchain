// +build integration nchain nobookie

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

// func TestDeployEkhoContractRopsten(t *testing.T) {
// 	t.Parallel()

// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	testcaseApp := Application{
// 		"app" + testId.String(),
// 		"appdesc " + testId.String(),
// 	}

// 	app, err := appFactory(*userToken, testcaseApp.name, testcaseApp.description)
// 	if err != nil {
// 		t.Errorf("error setting up application. Error: %s", err.Error())
// 		return
// 	}

// 	appToken, err := appTokenFactory(*userToken, app.ID)
// 	if err != nil {
// 		t.Errorf("error getting app token. Error: %s", err.Error())
// 		return
// 	}

// 	wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
// 		"mnemonic": "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
// 	})
// 	if err != nil {
// 		t.Errorf("error creating wallet: %s", err.Error())
// 		return
// 	}
// 	t.Logf("wallet created: %+v", wallet)
// 	t.Logf("wallet id: %s", wallet.ID)
// 	t.Logf("wallet public key: %s", *wallet.PublicKey)

// 	// change this to create an eth address for vault deterministically, using the seed phrase
// 	// then use this keyid for the account.

// 	// let's add the keyspec into the create account, so it can create a bip39 key optionally
// 	// keyspec, seed phrase

// 	//"mnemonic":       "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
// 	// create the account for that user, for the Ropsten network
// 	// account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
// 	// 	"network_id":     ropstenNetworkID,
// 	// 	"application_id": app.ID,
// 	// })

// 	// if err != nil {
// 	// 	t.Errorf("error creating user account. Error: %s", err.Error())
// 	// 	return
// 	// }

// 	// t.Logf("account id: %s", account.ID)

// 	// wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
// 	// 	"network_id":     ropstenNetworkID,
// 	// 	"application_id": app.ID,
// 	// })

// 	// if err != nil {
// 	// 	t.Errorf("error creating wallet. Error: %s", err.Error())
// 	// 	return
// 	// }

// 	hdPath := "m/44'/60'/0'/0"
// 	t.Logf("ekho artifact: %s", ekhoArtifact)
// 	tt := []struct {
// 		name      string
// 		parameter string
// 	}{
// 		{"ekho", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %s}`, wallet.ID, hdPath, ekhoArtifact)},
// 	}

// 	params := map[string]interface{}{}

// 	for _, tc := range tt {
// 		json.Unmarshal([]byte(tc.parameter), &params)

// 		contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
// 			"network_id":     ropstenNetworkID,
// 			"application_id": app.ID.String(),
// 			"wallet_id":      wallet.ID,
// 			//"account_id": account.ID.String(),
// 			"name":    "Ekho",
// 			"address": "0x",
// 			"params":  params,
// 		})
// 		if err != nil {
// 			t.Errorf("error creating contract %s. Error: %s", tc.name, err.Error())
// 			return
// 		}

// 		started := time.Now().Unix()

// 		for {
// 			if time.Now().Unix()-started >= contractTimeout {
// 				t.Errorf("timed out awaiting contract address for %s contract", tc.name)
// 				return
// 			}

// 			cntrct, err := nchain.GetContractDetails(*appToken.Token, contract.ID.String(), map[string]interface{}{})
// 			if err != nil {
// 				t.Errorf("error fetching %s contract details; %s", tc.name, err.Error())
// 				return
// 			}

// 			if cntrct.Address != nil && *cntrct.Address != "0x" {
// 				t.Logf("%s contract address resolved; contract id: %s; address: %s", tc.name, cntrct.ID.String(), *cntrct.Address)
// 				break
// 			}

// 			t.Logf("%s contract address has not yet been resolved; contract id: %s", tc.name, cntrct.ID.String())
// 			time.Sleep(contractSleepTime * time.Second)
// 		}
// 	}
// }

// func TestEkhoTransaction(t *testing.T) {
// 	// ekho address on ropsten: 0x4B8fe5037c5f40070E40De39c2A273B498988ceC
// 	t.Parallel()
// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	testcaseApp := Application{
// 		"app" + testId.String(),
// 		"appdesc " + testId.String(),
// 	}

// 	app, err := appFactory(*userToken, testcaseApp.name, testcaseApp.description)
// 	if err != nil {
// 		t.Errorf("error setting up application. Error: %s", err.Error())
// 		return
// 	}

// 	appToken, err := appTokenFactory(*userToken, app.ID)
// 	if err != nil {
// 		t.Errorf("error getting app token. Error: %s", err.Error())
// 		return
// 	}

// 	wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
// 		"network_id":     ropstenNetworkID,
// 		"application_id": app.ID,
// 	})

// 	if err != nil {
// 		t.Errorf("error creating wallet. Error: %s", err.Error())
// 		return
// 	}

// 	//hdPath := "m/44'/60'/0'/0"
// 	// tt := []struct {
// 	// 	name      string
// 	// 	parameter string
// 	// }{
// 	// 	{"ekho", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %s}`, wallet.ID, hdPath, ekhoArtifact)},
// 	// }

// 	// generate a random string from bytes
// 	msg := common.RandomString(118)

// 	params := map[string]interface{}{}
// 	parameter := fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "wallet_id":"%s"}`, msg, wallet.ID.String())

// 	json.Unmarshal([]byte(parameter), &params)

// 	//using fixed contract id: 1084255a-410e-4a1b-b72d-9182767ffc9d
// 	// which is a contract at address:
// 	execResponse, err := nchain.ExecuteContract(*appToken.Token, "1084255a-410e-4a1b-b72d-9182767ffc9d", params)
// 	if err != nil {
// 		t.Errorf("error executing contract. Error: %s", err.Error())
// 		return
// 	}
// 	t.Logf("contractTx: %+v", execResponse)
// 	t.Logf("reference: %s", *execResponse.Reference)
// 	if err != nil {
// 		t.Errorf("error executing contract: %s", err.Error())
// 		return
// 	}

// 	started := time.Now().Unix()

// 	for {
// 		if time.Now().Unix()-started >= transactionTimeout {
// 			t.Error("timed out awaiting transaction hash")
// 			return
// 		}

// 		tx, err := nchain.GetTransactionDetails(*appToken.Token, *execResponse.Reference, map[string]interface{}{})
// 		//this is populated by nchain consumer, so it can take a moment to appear, so we won't quit right away on a 404
// 		if err != nil {
// 			t.Logf("error fetching transaction; %s", err.Error())
// 		}

// 		if err == nil {
// 			if tx.Hash != nil && *tx.Hash != "0x" {
// 				t.Logf("tx resolved; tx id: %s; hash: %s", tx.ID.String(), *tx.Hash)
// 				break
// 			}

// 			t.Logf("transaction has not yet been resolved; tx id: %s", tx.ID.String())
// 		}
// 		time.Sleep(transactionSleepTime * time.Second)
// 	}

// 	t.Logf("contract execution successful")

// }

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

	path := `m/44'/60'/2'/0/0`
	t.Logf("ekho artifact: %s", ekhoArtifact)
	tt := []struct {
		name      string
		parameter string
	}{
		{"ekho", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %s}`, wallet.ID, path, ekhoArtifact)},
	}

	params := map[string]interface{}{}

	for _, tc := range tt {
		json.Unmarshal([]byte(tc.parameter), &params)

		contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
			"network_id":     ropstenNetworkID,
			"application_id": app.ID.String(),
			"wallet_id":      wallet.ID,
			//"account_id": account.ID.String(),
			"name":    "Ekho",
			"address": "0x",
			"params":  params,
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

		msg := common.RandomString(118)

		// hd path opts from vault tests
		// opts := map[string]interface{}{}
		//path := `m/44'/60'/2'/0/0`
		// options := fmt.Sprintf(`{"hdwallet":{"hd_derivation_path":"%s"}}`, path)
		// json.Unmarshal([]byte(options), &opts)

		t.Logf("executing contract using wallet id: %s", wallet.ID.String())
		params := map[string]interface{}{}
		parameter := fmt.Sprintf(`{"method":"broadcast", "hd_derivation_path": "%s", "params": ["%s"], "value":0, "wallet_id":"%s"}`, path, msg, wallet.ID.String())

		json.Unmarshal([]byte(parameter), &params)

		execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
		if err != nil {
			t.Errorf("error executing contract. Error: %s", err.Error())
			return
		}
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
}
