// +build integration

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	ident "github.com/provideservices/provide-go/api/ident"
	provide "github.com/provideservices/provide-go/api/nchain"
)

func TestCreateAccount(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	account, err := provide.CreateAccount(*token, map[string]interface{}{
		"network_id": ropstenNetworkID,
	})

	if err != nil {
		t.Errorf("error creating account. Error: %s", err.Error())
		return
	}
	t.Logf("account created: %+v", account)
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

	userToken := &ident.Token{}

	for _, user := range users {

		// create the ident user
		_, err = userFactory(user.firstName, user.lastName, user.email, user.password)
		if err != nil {
			t.Errorf("error creating user. Error: %s", err.Error())
		}

		// get the ident user token
		token, err := getUserToken(user.email, user.password)
		if err != nil {
			t.Errorf("error getting user token. Error: %s", err.Error())
		}

		if userToken.Token == nil {
			userToken = token
		}

		// create the account for that user, for the Ropsten network
		account, err := provide.CreateAccount(*token.Token, map[string]interface{}{
			"network_id": ropstenNetworkID,
		})

		if err != nil {
			t.Errorf("error creating user account. Error: %s", err.Error())
		}
		t.Logf("account created: %+v", account)
	}

	t.Logf("userToken: %+v", *userToken)

	// damn, need to enable ropsten before this works...
	status, resp, err := provide.ListAccounts(*userToken.Token, map[string]interface{}{
		"network_id": ropstenNetworkID,
	})

	if err != nil {
		t.Errorf("error listing accounts. Error: %s", err.Error())
		return
	}
	t.Logf("status: %v", status)
	t.Logf("response: %+v", resp)
}
