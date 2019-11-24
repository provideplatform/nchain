package wallet_test

import (
	"testing"

	bip32 "github.com/FactomProject/go-bip32"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/wallet"
)

var defaultPurpose = 44

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

	common.Log.Debugf("master key: %s", masterKey)
	common.Log.Debugf("master key 2: %s", masterKey2)

	w0, err := wallet.DeriveHardened(dbconf.DatabaseConnection(), uint32(60), uint32(0))
	if err != nil {
		t.Errorf("failed to derive hardened account; %s", err.Error())
	}
	w1, err := wallet.DeriveHardened(dbconf.DatabaseConnection(), uint32(60), uint32(0))
	if err != nil {
		t.Errorf("failed to derive hardened account; %s", err.Error())
	}

	common.Log.Debugf("hardened ephemeral HD wallet account (attempt 1); xpub: %s; xprv: %s", *w0.PublicKey, *w0.PrivateKey)
	common.Log.Debugf("hardened ephemeral HD wallet account (attempt 2); xpub: %s; xprv: %s", *w1.PublicKey, *w1.PrivateKey)

	a0, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), nil)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}
	a1, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), nil)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}

	common.Log.Debugf("HD wallet address (attempt 1); xpub: %s; xprv: %s", *a0.PublicKey, *a0.PrivateKey)
	common.Log.Debugf("HD wallet address (attempt 2); xpub: %s; xprv: %s", *a1.PublicKey, *a1.PrivateKey)
}
