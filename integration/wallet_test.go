// +build integration nchain

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

func TestListWallets(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallets, err := nchain.ListWallets(*userToken, map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing wallets")
		return
	}
	t.Logf("wallets: %+v", wallets)
}

func TestCreateWallet(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := nchain.CreateWallet(*userToken, map[string]interface{}{
		"purpose": 44,
	})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}
	t.Logf("wallet: %+v", wallet)

}

func TestWalletDetails(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := nchain.CreateWallet(*userToken, map[string]interface{}{
		"purpose": 44,
	})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}

	t.Logf("*** wallet returned %+v", *wallet)
	deets, err := nchain.GetWalletDetails(*userToken, wallet.ID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error getting wallet details")
		return
	}

	if *wallet.UserID != *deets.UserID {
		t.Errorf("UserID details do not match. Expected: %s. Got %s", *wallet.UserID, *deets.UserID)
		return
	}
	if wallet.ApplicationID != nil {
		if *wallet.ApplicationID != *deets.ApplicationID {
			t.Errorf("ApplicationID details do not match. Expected: %s. Got %s", *wallet.ApplicationID, *deets.ApplicationID)
			return
		}
	}
	if *wallet.UserID != *deets.UserID {
		t.Errorf("user id details do not match. Expected: %s. Got %s", *wallet.UserID, *deets.UserID)
		return
	}
	if *wallet.KeyID != *deets.KeyID {
		t.Errorf("KeyID details do not match. Expected: %s. Got %s", *wallet.KeyID, *deets.KeyID)
		return
	}
	if wallet.Path != nil {
		if *wallet.Path != *deets.Path {
			t.Errorf("Path details do not match. Expected: %s. Got %s", *wallet.Path, *deets.Path)
			return
		}
	}
	if *wallet.Purpose != *deets.Purpose {
		t.Errorf("Purpose details do not match. Expected: %d. Got %d", *wallet.Purpose, *deets.Purpose)
		return
	}

	// walletExtendedKeyJSON, _ := json.Marshal(*wallet.ExtendedKey)
	// deetsExtendedKeyJson, _ := json.Marshal(*deets.ExtendedKey)

	// if bytes.Compare(walletExtendedKeyJSON, deetsExtendedKeyJson) != 0 {
	// 	t.Errorf("ExtendedKey details do not match. Expected: %+v. Got %+v", *wallet.ExtendedKey, *deets.ExtendedKey)
	// 	return
	// }

	t.Logf("wallet details %+v", wallet)
}

func TestListWalletAccounts(t *testing.T) {
	t.Parallel()
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	userToken, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	wallet, err := nchain.CreateWallet(*userToken, map[string]interface{}{
		"purpose": 44,
	})
	if err != nil {
		t.Errorf("error creating wallet")
		return
	}
	accounts, err := nchain.ListWalletAccounts(*userToken, wallet.ID.String(), map[string]interface{}{})
	if err != nil {
		t.Errorf("error listing wallet accounts. Error: %s", err.Error())
		return
	}
	t.Logf("accounts: %+v", accounts)
}
