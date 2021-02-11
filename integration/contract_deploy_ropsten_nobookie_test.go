// +build integration nchain nobookie

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

	wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
		"mnemonic": "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	})
	if err != nil {
		t.Errorf("error creating wallet: %s", err.Error())
		return
	}
	t.Logf("wallet created: %+v", wallet)
	t.Logf("wallet public key: %s", *wallet.PublicKey)

	// change this to create an eth address for vault deterministically, using the seed phrase
	// then use this keyid for the account.

	// let's add the keyspec into the create account, so it can create a bip39 key optionally
	// keyspec, seed phrase

	//"mnemonic":       "traffic charge swing glimpse will citizen push mutual embrace volcano siege identify gossip battle casual exit enrich unlock muscle vast female initial please day",
	// create the account for that user, for the Ropsten network
	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
		"network_id":     ropstenNetworkID,
		"application_id": app.ID,
	})

	if err != nil {
		t.Errorf("error creating user account. Error: %s", err.Error())
		return
	}

	t.Logf("account id: %s", account.ID)

	// wallet, err := nchain.CreateWallet(*appToken.Token, map[string]interface{}{
	// 	"network_id":     ropstenNetworkID,
	// 	"application_id": app.ID,
	// })

	// if err != nil {
	// 	t.Errorf("error creating wallet. Error: %s", err.Error())
	// 	return
	// }

	hdPath := "m/44'/60'/0'/0"
	t.Logf("ekho artifact: %s", ekhoArtifact)
	tt := []struct {
		name      string
		parameter string
	}{
		{"ekho", fmt.Sprintf(`{"wallet_id": "%s","hd_derivation_path": "%s","compiled_artifact": %s}`, wallet.ID, hdPath, ekhoArtifact)},
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
	}
}
