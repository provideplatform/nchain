// +build integration

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
)

func TestListWalletAccounts(t *testing.T) {
	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	_, err = UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

	t.Logf("need wallet handlers in provide-go")

}
