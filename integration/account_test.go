// +build integration

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go/api/nchain"
)

func TestCreateAccount(t *testing.T) {

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := createUserAndGetToken(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	status, _, err := provide.CreateAccount(*token, map[string]interface{}{})

	if status != 201 {
		t.Errorf("expected 201 status, got %d", status)
		return
	}
	if err != nil {
		t.Errorf("error creating account. Error: %s", err.Error())
		return
	}
}
