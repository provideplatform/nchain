// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/ident/common"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

// ExecutionResponse is returned upon successful contract execution
type ExecutionResponse struct {
	Response    interface{} `json:"response"`
	Receipt     interface{} `json:"receipt"`
	Traces      interface{} `json:"traces"`
	Transaction interface{} `json:"transaction"`
	Ref         *string     `json:"ref"`
}

// TxValue provides JSON marshaling and gorm driver support for wrapping/unwrapping big.Int
type TxValue struct {
	value *big.Int
}

type Transaction struct {
	ID        uuid.UUID `json:"transaction_id,omitempty"`
	NetworkID uuid.UUID `json:"network_id,omitempty"`

	// Application or user id, if populated, is the entity for which the transaction was custodially signed and broadcast
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`

	// Account or HD wallet which custodially signed the transaction; when an HD wallet is used, if no HD derivation path is provided,
	// the most recently derived non-zero account is used to sign
	AccountID *uuid.UUID `json:"account_id,omitempty"`
	WalletID  *uuid.UUID `json:"wallet_id,omitempty"`
	Path      *string    `json:"hd_derivation_path,omitempty"`

	// Network-agnostic tx fields
	Signer      *string          `json:"signer,omitempty"`
	To          *string          `json:"to"`
	Value       *TxValue         `json:"value"`
	Data        *string          `json:"data"`
	Hash        *string          `json:"hash"`
	Status      *string          `default:'pending'" json:"status"`
	Params      *json.RawMessage `json:"params,omitempty"`
	Ref         *string          `json:"ref"`
	Description *string          `json:"description"`

	// Ephemeral fields for managing the tx/rx and tracing lifecycles
	Response *ExecutionResponse `json:"-"`
	SignedTx interface{}        `json:"-"`
	Traces   interface{}        `json:"traces,omitempty"`

	// Transaction metadata/instrumentation
	Block          *uint64    `json:"block"`
	BlockTimestamp *time.Time `json:"block_timestamp,omitempty"` // timestamp when the tx was finalized on-chain, according to its tx receipt
	BroadcastAt    *time.Time `json:"broadcast_at,omitempty"`    // timestamp when the tx was broadcast to the network
	FinalizedAt    *time.Time `json:"finalized_at,omitempty"`    // timestamp when the tx was finalized on-platform
	PublishedAt    *time.Time `json:"published_at,omitempty"`    // timestamp when the tx was published to NATS cluster
	QueueLatency   *uint64    `json:"queue_latency,omitempty"`   // broadcast_at - published_at (in millis) -- the amount of time between when a message is enqueued to the NATS broker and when it is broadcast to the network
	NetworkLatency *uint64    `json:"network_latency,omitempty"` // finalized_at - broadcast_at (in millis) -- the amount of time between when a message is broadcast to the network and when it is finalized on-chain
	E2ELatency     *uint64    `json:"e2e_latency,omitempty"`     // finalized_at - published_at (in millis) -- the amount of time between when a message is published to the NATS broker and when it is finalized on-chain
}

func GoCreateTransaction(token string, params map[string]interface{}) (*Transaction, error) {
	uri := "transactions"
	status, resp, err := nchain.InitNChainService(token).Post(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 201 {
		return nil, fmt.Errorf("failed to create transaction. status: %v", status)
	}

	tx := &Transaction{}
	txRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(txRaw, &tx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction. status: %v; %s", status, err.Error())
	}

	return tx, nil
}

func GoGetNetworkTransactionDetails(token, networkID, transactionID string, params map[string]interface{}) (*Transaction, error) {
	uri := fmt.Sprintf("networks/%s/transactions/%s", networkID, transactionID)
	status, resp, err := nchain.InitNChainService(token).Get(uri, params)

	if status != 201 {
		return nil, fmt.Errorf("failed to get transaction details. status: %v", status)
	}

	tx := &Transaction{}
	txRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(txRaw, &tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction details. status: %v; %s", status, err.Error())
	}

	return tx, nil
}

func GoGetTransactionDetails(token string, transactionID string, params map[string]interface{}) (*Transaction, error) {
	uri := fmt.Sprintf("transactions/%s", transactionID)
	status, resp, err := nchain.InitNChainService(token).Get(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 201 {
		return nil, fmt.Errorf("failed to get transaction details. status: %v", status)
	}

	tx := &Transaction{}
	txRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(txRaw, &tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction details. status: %v; %s", status, err.Error())
	}

	return tx, nil
}

func GoListTransactions(token string, params map[string]interface{}) ([]*Transaction, error) {
	uri := "transactions"
	status, resp, err := nchain.InitNChainService(token).Get(uri, params)
	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to list transactions. status: %v", status)
	}

	transactions := make([]*Transaction, 0)
	for _, item := range resp.([]interface{}) {
		tx := &Transaction{}
		txRaw, _ := json.Marshal(item)
		json.Unmarshal(txRaw, &tx)
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func GoListNetworkTransactions(token, networkID string, params map[string]interface{}) ([]*Transaction, error) {
	uri := fmt.Sprintf("/networks/%s/transactions", networkID)

	status, resp, err := nchain.InitNChainService(token).Get(uri, params)
	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to list network transactions. status: %v", status)
	}

	transactions := make([]*Transaction, 0)
	for _, item := range resp.([]interface{}) {
		tx := &Transaction{}
		txRaw, _ := json.Marshal(item)
		json.Unmarshal(txRaw, &tx)
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

func TestCreateTransaction(t *testing.T) {
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
	parameter := fmt.Sprintf(`{"network_id":"%s", "gas_price":%d, "gas":30000}`, ropstenNetworkID, 30)
	t.Logf("parameter is %s", parameter)
	json.Unmarshal([]byte(parameter), &params)

	msg := common.RandomString(118)

	tx, err := GoCreateTransaction(*userToken, map[string]interface{}{
		"network_id": ropstenNetworkID,
		"account_id": account.ID.String(),
		"to":         "0x31138a6a53141d9b5aa7f313ae0e2ca2fabac602", //ekho contract on ropsten
		"data":       []byte(msg),
		"value":      big.NewInt(0),
		"params":     params,
		"gas":        30000,
	})
	if err != nil {
		t.Errorf("error creating transaction: %s", err.Error())
		return
	}
	t.Logf("transaction: %+v", tx)

}

// func TestGetTransactionDetails(t *testing.T) {
// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	tx, err := GoCreateTransaction(*userToken, map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error creating transaction")
// 		return
// 	}

// 	deets, err := GoGetTransactionDetails(*userToken, tx.ID.String(), map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error getting transaction details")
// 		return
// 	}

// 	if *tx.UserID != *deets.UserID {
// 		t.Errorf("UserID details do not match. Expected: %s. Got %s", *tx.UserID, *deets.UserID)
// 		return
// 	}
// 	if *tx.ApplicationID != *deets.ApplicationID {
// 		t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *tx.ApplicationID, *deets.ApplicationID)
// 		return
// 	}
// 	if tx.NetworkID != deets.NetworkID { //CHECKME why isn't this a pointer in the original struct?
// 		t.Errorf("NetworkID details do not match. Expected: %s. Got %s", tx.NetworkID, deets.NetworkID)
// 		return
// 	}
// 	if *tx.AccountID != *deets.AccountID {
// 		t.Errorf("AccountID details do not match. Expected: %s. Got %s", *tx.AccountID, *deets.AccountID)
// 		return
// 	}
// 	if *tx.WalletID != *deets.WalletID {
// 		t.Errorf("WalletID details do not match. Expected: %s. Got %s", *tx.WalletID, *deets.WalletID)
// 		return
// 	}
// 	if *tx.Path != *deets.Path {
// 		t.Errorf("Path details do not match. Expected: %s. Got %s", *tx.Path, *deets.Path)
// 		return
// 	}
// 	if *tx.Signer != *deets.Signer {
// 		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
// 		return
// 	}
// 	if *tx.To != *deets.To {
// 		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
// 		return
// 	}
// 	txValueJSON, _ := json.Marshal(*tx.Value)
// 	deetsValueJSON, _ := json.Marshal(*deets.Value)
// 	if bytes.Compare(txValueJSON[:], deetsValueJSON[:]) != 0 {
// 		t.Errorf("Value details do not match. Expected: %+v. Got %+v", string(txValueJSON), string(deetsValueJSON))
// 		return
// 	}
// 	if *tx.Signer != *deets.Signer {
// 		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
// 		return
// 	}
// 	if *tx.To != *deets.To {
// 		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
// 		return
// 	}
// 	if *tx.Data != *deets.Data {
// 		t.Errorf("Data details do not match. Expected: %s. Got %s", *tx.Data, *deets.Data)
// 		return
// 	}
// 	if *tx.Hash != *deets.Hash {
// 		t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
// 		return
// 	}
// 	if *tx.Status != *deets.Status {
// 		t.Errorf("Status details do not match. Expected: %s. Got %s", *tx.Status, *deets.Status)
// 		return
// 	}
// 	// if *tx.Params != *deets.Params {
// 	// 	t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
// 	// 	return
// 	// }
// 	if *tx.Ref != *deets.Ref {
// 		t.Errorf("Ref details do not match. Expected: %s. Got %s", *tx.Ref, *deets.Ref)
// 		return
// 	}
// 	if *tx.Description != *deets.Description {
// 		t.Errorf("Description details do not match. Expected: %s. Got %s", *tx.Description, *deets.Description)
// 		return
// 	}

// 	t.Logf("transaction details %+v", *tx)
// }

// func TestListTransactions(t *testing.T) {
// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	// let's get a single transaction in here to get the test rolling
// 	_, err = GoCreateTransaction(*userToken, map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error creating transaction")
// 		return
// 	}

// 	transactions, err := GoListTransactions(*userToken, map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error listing transactions")
// 		return
// 	}

// 	if len(transactions) != 1 {
// 		t.Errorf("incorrect number of transactions returned. expected 1, got %d", len(transactions))
// 		return
// 	}
// 	t.Logf("transactions: %+v", transactions)
// }

// func TestListNetworkTransactions(t *testing.T) {
// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	// let's get a single transaction in here to get the test rolling
// 	tx, err := GoCreateTransaction(*userToken, map[string]interface{}{
// 		"network_id": ropstenNetworkID,
// 	})
// 	if err != nil {
// 		t.Errorf("error creating transaction")
// 		return
// 	}

// 	deets, err := GoGetTransactionDetails(*userToken, tx.ID.String(), map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error getting transaction details")
// 		return
// 	}

// 	if *tx.UserID != *deets.UserID {
// 		t.Errorf("UserID details do not match. Expected: %s. Got %s", *tx.UserID, *deets.UserID)
// 		return
// 	}
// 	if *tx.ApplicationID != *deets.ApplicationID {
// 		t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *tx.ApplicationID, *deets.ApplicationID)
// 		return
// 	}
// 	if tx.NetworkID != deets.NetworkID { //CHECKME why isn't this a pointer in the original struct?
// 		t.Errorf("NetworkID details do not match. Expected: %s. Got %s", tx.NetworkID, deets.NetworkID)
// 		return
// 	}
// 	if *tx.AccountID != *deets.AccountID {
// 		t.Errorf("AccountID details do not match. Expected: %s. Got %s", *tx.AccountID, *deets.AccountID)
// 		return
// 	}
// 	if *tx.WalletID != *deets.WalletID {
// 		t.Errorf("WalletID details do not match. Expected: %s. Got %s", *tx.WalletID, *deets.WalletID)
// 		return
// 	}
// 	if *tx.Path != *deets.Path {
// 		t.Errorf("Path details do not match. Expected: %s. Got %s", *tx.Path, *deets.Path)
// 		return
// 	}
// 	if *tx.Signer != *deets.Signer {
// 		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
// 		return
// 	}
// 	if *tx.To != *deets.To {
// 		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
// 		return
// 	}
// 	txValueJSON, _ := json.Marshal(*tx.Value)
// 	deetsValueJSON, _ := json.Marshal(*deets.Value)
// 	if bytes.Compare(txValueJSON[:], deetsValueJSON[:]) != 0 {
// 		t.Errorf("Value details do not match. Expected: %+v. Got %+v", string(txValueJSON), string(deetsValueJSON))
// 		return
// 	}
// 	if *tx.Signer != *deets.Signer {
// 		t.Errorf("Signer details do not match. Expected: %s. Got %s", *tx.Signer, *deets.Signer)
// 		return
// 	}
// 	if *tx.To != *deets.To {
// 		t.Errorf("To details do not match. Expected: %s. Got %s", *tx.To, *deets.To)
// 		return
// 	}
// 	if *tx.Data != *deets.Data {
// 		t.Errorf("Data details do not match. Expected: %s. Got %s", *tx.Data, *deets.Data)
// 		return
// 	}
// 	if *tx.Hash != *deets.Hash {
// 		t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
// 		return
// 	}
// 	if *tx.Status != *deets.Status {
// 		t.Errorf("Status details do not match. Expected: %s. Got %s", *tx.Status, *deets.Status)
// 		return
// 	}
// 	// if *tx.Params != *deets.Params {
// 	// 	t.Errorf("Hash details do not match. Expected: %s. Got %s", *tx.Hash, *deets.Hash)
// 	// 	return
// 	// }
// 	if *tx.Ref != *deets.Ref {
// 		t.Errorf("Ref details do not match. Expected: %s. Got %s", *tx.Ref, *deets.Ref)
// 		return
// 	}
// 	if *tx.Description != *deets.Description {
// 		t.Errorf("Description details do not match. Expected: %s. Got %s", *tx.Description, *deets.Description)
// 		return
// 	}

// 	t.Logf("transaction details %+v", *tx)
// }

// func TestGetNetworkTransactionDetails(t *testing.T) {
// 	testId, err := uuid.NewV4()
// 	if err != nil {
// 		t.Logf("error creating new UUID")
// 	}

// 	userToken, err := UserAndTokenFactory(testId)
// 	if err != nil {
// 		t.Errorf("user authentication failed. Error: %s", err.Error())
// 	}

// 	// let's get a single transaction in here to get the test rolling
// 	_, err = GoCreateTransaction(*userToken, map[string]interface{}{
// 		"network_id": ropstenNetworkID,
// 	})
// 	if err != nil {
// 		t.Errorf("error creating transaction")
// 		return
// 	}

// 	transactions, err := GoListNetworkTransactions(*userToken, ropstenNetworkID, map[string]interface{}{})
// 	if err != nil {
// 		t.Errorf("error listing transactions")
// 		return
// 	}

// 	if len(transactions) != 1 {
// 		t.Errorf("incorrect number of transactions returned. expected 1, got %d", len(transactions))
// 		return
// 	}
// 	t.Logf("transactions: %+v", transactions)
// }

// func TestExecuteContract(t *testing.T) {

// }
