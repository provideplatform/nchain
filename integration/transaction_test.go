//go:build integration || nchain
// +build integration nchain

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
	"bytes"
	"encoding/json"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

// ExecutionResponse is returned upon successful contract execution
type ExecutionResponse struct {
	Response    interface{} `json:"response"`
	Receipt     interface{} `json:"receipt"`
	Traces      interface{} `json:"traces"`
	Transaction interface{} `json:"transaction"`
	Ref         *string     `json:"ref"`
}

// func TestCreateTransaction(t *testing.T) {
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

// 	// create the account for that user, for the Ropsten network
// 	account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
// 		"network_id":     ropstenNetworkID,
// 		"application_id": app.ID,
// 	})

// 	if err != nil {
// 		t.Errorf("error creating user account. Error: %s", err.Error())
// 	}
// 	t.Logf("account created: %+v", account)

// 	params := map[string]interface{}{}
// 	parameter := fmt.Sprintf(`{"network_id":"%s", "gas_price":%d, "gas":30000}`, ropstenNetworkID, 30)
// 	t.Logf("parameter is %s", parameter)
// 	json.Unmarshal([]byte(parameter), &params)

// 	msg, _ := uuid.NewV4()

// 	tx, err := nchain.CreateTransaction(*userToken, map[string]interface{}{
// 		"network_id": ropstenNetworkID,
// 		"account_id": account.ID.String(),
// 		"to":         "0x31138a6a53141d9b5aa7f313ae0e2ca2fabac602", //ekho contract on ropsten
// 		"data":       []byte(msg.String()),
// 		"value":      big.NewInt(0),
// 		"params":     params,
// 		"gas":        30000,
// 	})
// 	if err != nil {
// 		t.Errorf("error creating transaction: %s", err.Error())
// 		return
// 	}
// 	t.Logf("transaction: %+v", tx)

// }

func TestGetTransactionDetails(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	tx, err := nchain.CreateTransaction(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error creating transaction")
		return
	}

	deets, err := nchain.GetTransactionDetails(*userToken, tx.ID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error getting transaction details")
		return
	}

	if *tx.UserID != *deets.UserID {
		t.Errorf("UserID details do not match. Expected: %s. Got %s", *tx.UserID, *deets.UserID)
		return
	}
	if *tx.ApplicationID != *deets.ApplicationID {
		t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *tx.ApplicationID, *deets.ApplicationID)
		return
	}
	if tx.NetworkID != deets.NetworkID { //CHECKME why isn't this a pointer in the original struct?
		t.Errorf("NetworkID details do not match. Expected: %s. Got %s", tx.NetworkID, deets.NetworkID)
		return
	}
	if *tx.AccountID != *deets.AccountID {
		t.Errorf("AccountID details do not match. Expected: %s. Got %s", *tx.AccountID, *deets.AccountID)
		return
	}
	if *tx.WalletID != *deets.WalletID {
		t.Errorf("WalletID details do not match. Expected: %s. Got %s", *tx.WalletID, *deets.WalletID)
		return
	}
	if *tx.Path != *deets.Path {
		t.Errorf("Path details do not match. Expected: %s. Got %s", *tx.Path, *deets.Path)
		return
	}
	if *tx.Signer != *deets.Signer {
		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
		return
	}
	if *tx.To != *deets.To {
		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
		return
	}
	txValueJSON, _ := json.Marshal(*tx.Value)
	deetsValueJSON, _ := json.Marshal(*deets.Value)
	if bytes.Compare(txValueJSON[:], deetsValueJSON[:]) != 0 {
		t.Errorf("Value details do not match. Expected: %+v. Got %+v", string(txValueJSON), string(deetsValueJSON))
		return
	}
	if *tx.Signer != *deets.Signer {
		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
		return
	}
	if *tx.To != *deets.To {
		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
		return
	}
	if *tx.Data != *deets.Data {
		t.Errorf("Data details do not match. Expected: %s. Got %s", *tx.Data, *deets.Data)
		return
	}
	if *tx.Hash != *deets.Hash {
		t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
		return
	}
	if *tx.Status != *deets.Status {
		t.Errorf("Status details do not match. Expected: %s. Got %s", *tx.Status, *deets.Status)
		return
	}
	// if *tx.Params != *deets.Params {
	// 	t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
	// 	return
	// }
	if *tx.Ref != *deets.Ref {
		t.Errorf("Ref details do not match. Expected: %s. Got %s", *tx.Ref, *deets.Ref)
		return
	}
	if *tx.Description != *deets.Description {
		t.Errorf("Description details do not match. Expected: %s. Got %s", *tx.Description, *deets.Description)
		return
	}

	t.Logf("transaction details %+v", *tx)
}

func TestListTransactions(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	// let's get a single transaction in here to get the test rolling
	_, err = nchain.CreateTransaction(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error creating transaction")
		return
	}

	transactions, err := nchain.ListTransactions(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing transactions")
		return
	}

	if len(transactions) != 1 {
		t.Errorf("incorrect number of transactions returned. expected 1, got %d", len(transactions))
		return
	}
	t.Logf("transactions: %+v", transactions)
}

func TestListNetworkTransactions(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	// let's get a single transaction in here to get the test rolling
	tx, err := nchain.CreateTransaction(*userToken, map[string]interface{}{
		"network_id": ropstenNetworkID,
	})
	if err != nil {
		t.Errorf("error creating transaction")
		return
	}

	deets, err := nchain.GetTransactionDetails(*userToken, tx.ID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error getting transaction details")
		return
	}

	if *tx.UserID != *deets.UserID {
		t.Errorf("UserID details do not match. Expected: %s. Got %s", *tx.UserID, *deets.UserID)
		return
	}
	if *tx.ApplicationID != *deets.ApplicationID {
		t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *tx.ApplicationID, *deets.ApplicationID)
		return
	}
	if tx.NetworkID != deets.NetworkID { //CHECKME why isn't this a pointer in the original struct?
		t.Errorf("NetworkID details do not match. Expected: %s. Got %s", tx.NetworkID, deets.NetworkID)
		return
	}
	if *tx.AccountID != *deets.AccountID {
		t.Errorf("AccountID details do not match. Expected: %s. Got %s", *tx.AccountID, *deets.AccountID)
		return
	}
	if *tx.WalletID != *deets.WalletID {
		t.Errorf("WalletID details do not match. Expected: %s. Got %s", *tx.WalletID, *deets.WalletID)
		return
	}
	if *tx.Path != *deets.Path {
		t.Errorf("Path details do not match. Expected: %s. Got %s", *tx.Path, *deets.Path)
		return
	}
	if *tx.Signer != *deets.Signer {
		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
		return
	}
	if *tx.To != *deets.To {
		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
		return
	}
	txValueJSON, _ := json.Marshal(*tx.Value)
	deetsValueJSON, _ := json.Marshal(*deets.Value)
	if bytes.Compare(txValueJSON[:], deetsValueJSON[:]) != 0 {
		t.Errorf("Value details do not match. Expected: %+v. Got %+v", string(txValueJSON), string(deetsValueJSON))
		return
	}
	if *tx.Signer != *deets.Signer {
		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
		return
	}
	if *tx.To != *deets.To {
		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
		return
	}
	if *tx.Data != *deets.Data {
		t.Errorf("Data details do not match. Expected: %s. Got %s", *tx.Data, *deets.Data)
		return
	}
	if *tx.Hash != *deets.Hash {
		t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
		return
	}
	if *tx.Status != *deets.Status {
		t.Errorf("Status details do not match. Expected: %s. Got %s", *tx.Status, *deets.Status)
		return
	}
	// if *tx.Params != *deets.Params {
	// 	t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
	// 	return
	// }
	if *tx.Ref != *deets.Ref {
		t.Errorf("Ref details do not match. Expected: %s. Got %s", *tx.Ref, *deets.Ref)
		return
	}
	if *tx.Description != *deets.Description {
		t.Errorf("Description details do not match. Expected: %s. Got %s", *tx.Description, *deets.Description)
		return
	}

	t.Logf("transaction details %+v", *tx)
}

func TestGetNetworkTransactionDetails(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	// let's get a single transaction in here to get the test rolling
	_, err = nchain.CreateTransaction(*userToken, map[string]interface{}{
		"network_id": ropstenNetworkID,
	})
	if err != nil {
		t.Errorf("error creating transaction")
		return
	}

	transactions, err := nchain.ListNetworkTransactions(*userToken, ropstenNetworkID, map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing transactions")
		return
	}

	if len(transactions) != 1 {
		t.Errorf("incorrect number of transactions returned. expected 1, got %d", len(transactions))
		return
	}
	t.Logf("transactions: %+v", transactions)
}
