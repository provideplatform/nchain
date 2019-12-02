package wallet_test

import (
	"testing"

	bip32 "github.com/FactomProject/go-bip32"
	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/wallet"
)

var defaultPurpose = 44

func init() {
	pgputil.RequirePGP()
}

func decrypt(w *wallet.Wallet) error {
	if w.Mnemonic != nil {
		mnemonic, err := pgputil.PGPPubDecrypt([]byte(*w.Mnemonic))
		if err != nil {
			common.Log.Warningf("Failed to decrypt mnemonic; %s", err.Error())
			return err
		}
		w.Mnemonic = common.StringOrNil(string(mnemonic))
	}

	if w.Seed != nil {
		seed, err := pgputil.PGPPubDecrypt([]byte(*w.Seed))
		if err != nil {
			common.Log.Warningf("Failed to decrypt seed; %s", err.Error())
			return err
		}
		w.Seed = common.StringOrNil(string(seed))
	}

	if w.PublicKey != nil {
		publicKey, err := pgputil.PGPPubDecrypt([]byte(*w.PublicKey))
		if err != nil {
			common.Log.Warningf("Failed to decrypt public key; %s", err.Error())
			return err
		}
		w.PublicKey = common.StringOrNil(string(publicKey))
	}

	if w.PrivateKey != nil {
		privateKey, err := pgputil.PGPPubDecrypt([]byte(*w.PrivateKey))
		if err != nil {
			common.Log.Warningf("Failed to decrypt private key; %s", err.Error())
			return err
		}
		w.PrivateKey = common.StringOrNil(string(privateKey))
	}

	return nil
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

	decrypt(wallet)

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

	decrypt(w0)
	decrypt(w1)

	if w0.PublicKey == nil || w1.PublicKey == nil || *w0.PublicKey != *w1.PublicKey {
		t.Errorf("failed to deterministically generate master key for seed: %s", string(*wallet.Seed))
	}

	chain := uint32(0)

	a0, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), &chain)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}
	a1, err := w1.DeriveAddress(dbconf.DatabaseConnection(), uint32(0), &chain)
	if err != nil {
		t.Errorf("failed to derive address; %s", err.Error())
	}

	if a0.Address == "" || a1.Address == "" || a0.Address != a1.Address {
		t.Errorf("failed to deterministically generate master key for seed: %s", string(*wallet.Seed))
	}
}
