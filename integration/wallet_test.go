// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func GoGetWalletDetails(token string, walletID string, params map[string]interface{}) (*nchain.Wallet, error) {
	uri := fmt.Sprintf("wallets/%s", walletID)
	status, resp, err := nchain.InitNChainService(token).Get(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to get wallet details. status: %v", status)
	}

	wallet := &nchain.Wallet{}
	walletRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(walletRaw, &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet details. status: %v; %s", status, err.Error())
	}

	return wallet, nil
}

func GoCreateWallet(token string, params map[string]interface{}) (*nchain.Wallet, error) {
	uri := "wallets"
	status, resp, err := nchain.InitNChainService(token).Post(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 201 {
		return nil, fmt.Errorf("failed to create wallet. status: %v", status)
	}

	wallet := &nchain.Wallet{}
	walletRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(walletRaw, &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet. status: %v; %s", status, err.Error())
	}

	return wallet, nil
}

func GoListWallets(token string, params map[string]interface{}) ([]*nchain.Wallet, error) {
	uri := "wallets"

	status, resp, err := nchain.InitNChainService(token).Get(uri, params)
	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to list wallets. status: %v", status)
	}

	wallets := make([]*nchain.Wallet, 0)
	for _, item := range resp.([]interface{}) {
		wallet := &nchain.Wallet{}
		walletRaw, _ := json.Marshal(item)
		json.Unmarshal(walletRaw, &wallet)
		wallets = append(wallets, wallet)
	}
	return wallets, nil
}

func GoListWalletAccounts(token string, walletID string, params map[string]interface{}) ([]*nchain.Account, error) {
	uri := fmt.Sprintf("wallets/%s/accounts", walletID)

	status, resp, err := nchain.InitNChainService(token).Get(uri, params)
	if err != nil {
		return nil, err
	}

	if status != 200 {
		return nil, fmt.Errorf("failed to list wallet accounts. status: %v", status)
	}

	accounts := make([]*nchain.Account, 0)
	for _, item := range resp.([]interface{}) {
		account := &nchain.Account{}
		accountRaw, _ := json.Marshal(item)
		json.Unmarshal(accountRaw, &account)
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func TestListWallets(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallets, err := GoListWallets(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing wallets")
		return
	}
	t.Logf("wallets: %+v", wallets)
}

func TestCreateWallet(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := GoCreateWallet(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}
	t.Logf("wallet: %+v", wallet)

}

func TestWalletDetails(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := GoCreateWallet(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}

	deets, err := GoGetWalletDetails(*userToken, wallet.WalletID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error getting wallet details")
		return
	}

	if *wallet.UserID != *deets.UserID {
		t.Errorf("UserID details do not match. Expected: %s. Got %s", *wallet.UserID, *deets.UserID)
		return
	}
	if *wallet.ApplicationID != *deets.ApplicationID {
		t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *wallet.ApplicationID, *deets.ApplicationID)
		return
	}
	if *wallet.UserID != *deets.UserID {
		t.Errorf("user id details do not match. Expected: %s. Got %s", *wallet.UserID, *deets.UserID)
		return
	}
	if *wallet.KeyID != *deets.KeyID {
		t.Errorf("KeyID details do not match. Expected: %s. Got %s", *wallet.KeyID, *deets.KeyID)
		return
	}
	if *wallet.Path != *deets.Path {
		t.Errorf("Path details do not match. Expected: %s. Got %s", *wallet.Path, *deets.Path)
		return
	}
	if *wallet.Purpose != *deets.Purpose {
		t.Errorf("Purpose details do not match. Expected: %d. Got %d", *wallet.Purpose, *deets.Purpose)
		return
	}
	if *wallet.Mnemonic != *deets.Mnemonic {
		t.Errorf("Mnemonic details do not match. Expected: %s. Got %s", *wallet.Mnemonic, *deets.Mnemonic)
		return
	}

	walletExtendedKeyJSON, _ := json.Marshal(*wallet.ExtendedKey)
	deetsExtendedKeyJson, _ := json.Marshal(*deets.ExtendedKey)

	if bytes.Compare(walletExtendedKeyJSON, deetsExtendedKeyJson) != 0 {
		t.Errorf("ExtendedKey details do not match. Expected: %+v. Got %+v", *wallet.ExtendedKey, *deets.ExtendedKey)
		return
	}

	t.Logf("wallet details %+v", wallet)
}

func TestListWalletAccounts(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := GoCreateWallet(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}
	accounts, err := GoListWalletAccounts(*userToken, wallet.WalletID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing wallet accounts. Error: %s", err.Error())
		return
	}
	t.Logf("accounts: %+v", accounts)
}
