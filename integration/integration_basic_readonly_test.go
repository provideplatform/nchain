//go:build integration || nchain || readonly || basic
// +build integration nchain readonly basic

/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

func TestContractHDWalletReadOnly(t *testing.T) {

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

	// create the predeployed contract in the database, and associate it with the app
	// load the ERC20 compiled artifact
	erc20Artifact, err := ioutil.ReadFile("artifacts/erc20.json")
	if err != nil {
		t.Errorf("error loading erc20 artifact. Error: %s", err.Error())
	}

	erc20CompiledArtifact := nchain.CompiledArtifact{}
	err = json.Unmarshal(erc20Artifact, &erc20CompiledArtifact)
	if err != nil {
		t.Errorf("error converting readwritetester compiled artifact. Error: %s", err.Error())
	}

	//t.Logf("compiled artifact for erc20: %+v", erc20Artifact)
	contractName := "MONEH - erc20 contract"
	contractAddress := "0x45a67Fd75765721D0275d3925a768E86E7a2599c"
	// MONEH contract deployed to Rinekby - 0x45a67Fd75765721D0275d3925a768E86E7a2599c
	contract, err := nchain.CreatePublicContract(*appToken.Token, map[string]interface{}{
		"network_id":     rinkebyNetworkID,
		"application_id": app.ID.String(),
		"name":           contractName,
		"address":        contractAddress,
		"params": map[string]interface{}{
			"compiled_artifact": erc20CompiledArtifact,
		},
	})
	if err != nil {
		t.Errorf("error creating %s contract. Error: %s", contractName, err.Error())
		return
	}

	t.Logf("contract returned: %+v", contract)

	tt := []struct {
		network string
		name    string
		method  string
		//derivationPath string
		//walletID       string
		//artifact       nchain.CompiledArtifact
		contract       string
		expectedResult interface{}
	}{
		{rinkebyNetworkID, "erc20", "symbol", contractAddress, "MONEH"},
		{rinkebyNetworkID, "erc20", "name", contractAddress, "MONEH test token"},
		{rinkebyNetworkID, "erc20", "totalSupply", contractAddress, 1e+06},
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
			t.Logf("execution response: %v", execResponse.Response)
			if execResponse.Response != tc.expectedResult {
				t.Errorf("expected msg %s returned. got %v", tc.expectedResult, execResponse.Response)
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
