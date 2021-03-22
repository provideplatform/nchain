// +build integration nchain readonly basic

package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestContractHDWallet(t *testing.T) {

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

	// // this path produces the ETH address 0x6af845bae76f5cc16bc93f86b83e8928c3dfda19
	// path := `m/44'/60'/2'/0/0`

	// load the ekho compiled artifact
	// ekhoArtifact, err := ioutil.ReadFile("artifacts/ekho.json")
	// if err != nil {
	// 	t.Errorf("error loading ekho artifact. Error: %s", err.Error())
	// }

	// ekhoCompiledArtifact := nchain.CompiledArtifact{}
	// err = json.Unmarshal(ekhoArtifact, &ekhoCompiledArtifact)
	// if err != nil {
	// 	t.Errorf("error converting ekho compiled artifact. Error: %s", err.Error())
	// }

	// // load the readwrite compiled artifact
	// rwArtifact, err := ioutil.ReadFile("artifacts/readwritetester.json")
	// if err != nil {
	// 	t.Errorf("error loading readwritetester artifact. Error: %s", err.Error())
	// }

	// rwCompiledArtifact := nchain.CompiledArtifact{}
	// err = json.Unmarshal(rwArtifact, &rwCompiledArtifact)
	// if err != nil {
	// 	t.Errorf("error converting readwritetester compiled artifact. Error: %s", err.Error())
	// }

	tt := []struct {
		network string
		name    string
		method  string
		//derivationPath string
		//walletID       string
		//artifact       nchain.CompiledArtifact
		contract       string
		expectedResult string
	}{
		//{kovanNetworkID, "ekho", path, wallet.ID.String(), ekhoCompiledArtifact, "0x5eBe7A42E3496Ed044F9f95A876C8703831598d7"},
		{kovanNetworkID, "readwrite", "getString", "0x5eBe7A42E3496Ed044F9f95A876C8703831598d7", "NfGshn0Uc52U2IDqkfKnhf8yQaRT60lPpkm2xxVRASKWdaXwjx5BBtd3oMUXvJiDRpW4Kw4Xt92mdZ7BTeIQRZ3GA9HfjLPKIZD4Xw2yX1eLUpC7lM1KiI"},
	}

	for _, tc := range tt {

		params := map[string]interface{}{}

		parameter := fmt.Sprintf(`{"method":"%s", "params": [""], "value":0, "wallet_id":"%s"}`, tc.method, wallet.ID)
		json.Unmarshal([]byte(parameter), &params)

		// execute the contract method
		execResponse, err := nchain.ExecuteContract(*appToken.Token, tc.contract, params)
		if err != nil {
			t.Errorf("error executing contract. Error: %s", err.Error())
			return
		}

		if execResponse.Response != nil {
			t.Logf("execution response: %s", *execResponse.Response)
			if *execResponse.Response != tc.expectedResult {
				t.Errorf("expected msg %s returned. got %s", tc.expectedResult, *execResponse.Response)
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

		t.Logf("contract execution successful")
	}
}
