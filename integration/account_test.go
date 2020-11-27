// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	uuid "github.com/kthomas/go.uuid"
	"github.com/provideservices/provide-go/api"
	ident "github.com/provideservices/provide-go/api/ident"
	nchain "github.com/provideservices/provide-go/api/nchain"
)

// Account contains the specific account user details
type Account struct {
	api.Model
	NetworkID      *uuid.UUID `json:"network_id,omitempty"`
	WalletID       *uuid.UUID `json:"wallet_id,omitempty"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`

	VaultID *uuid.UUID `json:"vault_id,omitempty"`
	KeyID   *uuid.UUID `json:"key_id,omitempty"`

	Type *string `json:"type,omitempty"`

	HDDerivationPath *string `json:"hd_derivation_path,omitempty"` // i.e. m/44'/60'/0'/0
	PublicKey        *string `json:"public_key,omitempty"`
	PrivateKey       *string `json:"private_key,omitempty"`

	Address    string         `json:"address"`
	Balance    *big.Int       `json:"balance,omitempty"`
	AccessedAt *time.Time     `json:"accessed_at,omitempty"`
	Wallet     *nchain.Wallet `json:"-"`
}

// GoCreateAccount creates a new account
func GoCreateAccount(token string, params map[string]interface{}) (*Account, error) {
	uri := "accounts"
	status, resp, err := nchain.InitNChainService(token).Post(uri, params)

	if err != nil {
		return nil, err
	}

	if status != 201 {
		return nil, fmt.Errorf("failed to create account. status: %v", status)
	}

	account := &Account{}
	accountRaw, _ := json.Marshal(resp)
	err = json.Unmarshal(accountRaw, &account)
	if err != nil {
		return nil, fmt.Errorf("failed to create account. status: %v; %s", status, err.Error())
	}

	return account, nil
}

func TestListAccounts(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	users := []struct {
		firstName string
		lastName  string
		email     string
		password  string
		userID    *uuid.UUID
	}{
		{"joey", "joe joe", "j.j" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey2", "joe joe2", "j.j2" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey3", "joe joe3", "j.j3" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey4", "joe joe4", "j.j4" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey5", "joe joe5", "j.j5" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey6", "joe joe6", "j.j6" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey6", "joe joe7", "j.j7" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey8", "joe joe8", "j.j8" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey9", "joe joe9", "j.j9" + testId.String() + "@email.com", "secrit_password", nil},
		{"joey10", "joe joe10", "j.j10" + testId.String() + "@email.com", "secrit_password", nil},
	}

	setupUser, err := userFactoryByTestId(testId)
	if err != nil {
		t.Errorf("error setting up ident user. Error: %s", err.Error())
		return
	}

	setupUserToken, err := getUserToken(setupUser.Email, "secrit_password") //HACK gen password properly!
	if err != nil {
		t.Errorf("error getting setup user token. Error: %s", err.Error())
		return
	}

	testcaseApp := Application{
		"app" + testId.String(),
		"appdesc " + testId.String(),
	}

	app, err := appFactory(*setupUserToken.Token, testcaseApp.name, testcaseApp.description)
	if err != nil {
		t.Errorf("error setting up application. Error: %s", err.Error())
		return
	}

	appToken, err := appTokenFactory(*setupUserToken.Token, app.ID)
	if err != nil {
		t.Errorf("error getting app token. Error: %s", err.Error())
		return
	}

	for _, user := range users {

		// create the ident user
		identUser, err := userFactory(user.firstName, user.lastName, user.email, user.password)
		if err != nil {
			t.Errorf("error creating user. Error: %s", err.Error())
			return
		}

		// use the app token to add that user to the application
		err = ident.CreateApplicationUser(*appToken.Token, app.ID.String(), map[string]interface{}{
			"user_id": identUser.ID.String(),
		})
		if err != nil {
			t.Errorf("error adding user %s to app %s. Error: %s", identUser.ID, app.ID, err.Error())
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
	}

	accounts, err := nchain.ListAccounts(*appToken.Token, map[string]interface{}{
		"network_id": ropstenNetworkID,
	})
	if err != nil {
		t.Errorf("error listing accounts. Error: %s", err.Error())
		return
	}

	if status != 200 {
		t.Errorf("invalid status returned. Expected 200, got %v", status)
		return
	}

	t.Logf("number of accounts returned: %d", len(accounts))

	if len(accounts) != len(users) {
		t.Errorf("incorrect number of accounts returned. Expected %d, got %d", len(users), len(accounts))
		return
	}
}
