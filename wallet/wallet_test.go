package wallet_test

import (
	"testing"

	bip32 "github.com/FactomProject/go-bip32"
	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/wallet"
)

var defaultPurpose = 44

func init() {
	pgputil.RequirePGP()
}

func TestWalletCreate(t *testing.T) {
	appID, _ := uuid.NewV4()
	wallet := &wallet.Wallet{
		ApplicationID: &appID,
		Purpose:       &defaultPurpose,
	}
	if !wallet.Create() {
		t.Errorf("failed to create wallet; %s", *wallet.Errors[0].Message)
	}

	masterKey, err := bip32.NewMasterKey([]byte(*wallet.Seed))
	if err != nil {
		t.Errorf("failed to init master key from seed; %s", err.Error())
	}

	masterKey2, err := bip32.NewMasterKey([]byte(*wallet.Seed))
	if err != nil {
		t.Errorf("failed to init master key from seed; %s", err.Error())
	}

	if masterKey.String() != masterKey2.String() {
		t.Errorf("failed to deterministically generate master key for seed: %s", string(*wallet.Seed))
	}

	w0, err := wallet.DeriveHardened(dbconf.DatabaseConnection(), uint32(60), uint32(0))
	if err != nil {
		t.Errorf("failed to derive hardened account; %s", err.Error())
	}
	w1, err := wallet.DeriveHardened(dbconf.DatabaseConnection(), uint32(60), uint32(0))
	if err != nil {
		t.Errorf("failed to derive hardened account; %s", err.Error())
	}

	if w0.PublicKey == nil || w1.PublicKey == nil || *w0.PublicKey != *w1.PublicKey {
		t.Errorf("failed to deterministically generate master key for seed: %s", string(*wallet.Seed))
	}

	a0, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), nil)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}
	a1, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), nil)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}

	if a0.Address == "" || a1.Address == "" || a0.Address != a1.Address {
		t.Errorf("failed to deterministically generate master key for seed: %s", string(*wallet.Seed))
	}
}
