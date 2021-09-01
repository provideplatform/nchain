// +build integration nchain ropsten rinkeby kovan goerli nobookie basic

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	ident "github.com/provideplatform/provide-go/api/ident"
	nchain "github.com/provideplatform/provide-go/api/nchain"
)

func TestListAccounts(t *testing.T) {
	t.Parallel()

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

	app, err := appFactory(*setupUserToken.AccessToken, testcaseApp.name, testcaseApp.description)
	if err != nil {
		t.Errorf("error setting up application. Error: %s", err.Error())
		return
	}

	if app == nil {
		t.Errorf("error creating application")
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
		account, err := nchain.CreateAccount(*appToken.Token, map[string]interface{}{
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

	t.Logf("number of accounts returned: %d", len(accounts))

	if len(accounts) != len(users) {
		t.Errorf("incorrect number of accounts returned. Expected %d, got %d", len(users), len(accounts))
		return
	}
}
