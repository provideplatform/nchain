// +build integration_m

package integration

import (
	"testing"

	uuid "github.com/kthomas/go.uuid"
)

func TestCreateNetwork(t *testing.T) {
	// let's try it from the docs!

	testId, err := uuid.NewV4()
	if err != nil {
		t.Logf("error creating new UUID")
	}

	token, err := UserAndTokenFactory(testId)
	if err != nil {
		t.Errorf("user authentication failed. Error: %s", err.Error())
	}

}
