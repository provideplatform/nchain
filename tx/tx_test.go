// +build unit

package tx_test

import (
	"testing"

	dbconf "github.com/kthomas/go-db-config"
	"github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/tx"
)

func init() {
	pgputil.RequirePGP()
}

func TestTransaction_Create(t *testing.T) {
	accountID, _ := uuid.FromString("1be0f75c-c05d-42d7-85fd-0c406466a95c")
	appID, _ := uuid.FromString("146ab73e-b2eb-4386-8c6f-93663792c741")
	networkID, _ := uuid.FromString("66d44f30-9092-4182-a3c4-bc02736d6ae5")
	txVal := tx.NewTxValue(int64(200))
	tx := &tx.Transaction{
		NetworkID:     networkID,
		ApplicationID: &appID,
		AccountID:     &accountID,
		To:            common.StringOrNil("0x0E6081223ACCE2f7f402edE17ED2B0ABDe4E9D0c"),
		Value:         txVal,
		Data:          common.StringOrNil("0x"),
	}
	if !tx.Create(dbconf.DatabaseConnection()) {
		t.Errorf("tx create() failed; %s", *tx.Errors[0].Message)
	}
}
