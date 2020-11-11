// +build integration

package integration

import (
	"os"
	"testing"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go/api/nchain"
)

func TestCreateAccount(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	identUser := User{
		"nchain" + testId.String(),
		"user " + testId.String(),
		"nchain.user" + testId.String() + "@email.com",
		"secrit_password",
	}

	auth, err := getUserToken(identUser.firstName, identUser.lastName, identUser.email, identUser.password)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	ident_api := os.Getenv("IDENT_API_HOST")
	t.Logf("env var IDENT_API_HOST: %s", ident_api)
	status, resp, err := provide.CreateAccount(*auth.Token, map[string]interface{}{})

	t.Logf("response is %+v", resp)
	if status != 201 {
		t.Errorf("expected 201 status, got %d", status)
		return
	}
	if err != nil {
		t.Errorf("error creating account. Error: %s", err.Error())
		return
	}
}
