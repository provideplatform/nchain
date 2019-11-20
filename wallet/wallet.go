package wallet

import (
	bip32 "github.com/FactomProject/go-bip32"
	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/jinzhu/gorm"
	"github.com/provideapp/goldmine/common"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Wallet{})
	db.Model(&Wallet{}).AddIndex("idx_wallets_wallet_id", "wallet_id")
	db.Model(&Wallet{}).AddIndex("idx_wallets_application_id", "application_id")
	db.Model(&Wallet{}).AddIndex("idx_wallets_user_id", "user_id")
	db.Model(&Wallet{}).AddIndex("idx_wallets_accessed_at", "accessed_at")
	db.Model(&Wallet{}).AddIndex("idx_wallets_network_id", "network_id")
	db.Model(&Wallet{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
	db.Model(&Wallet{}).AddForeignKey("wallet_id", "wallets(id)", "SET NULL", "CASCADE")
}

// Wallet instances are logical collections of accounts; wallet instances are HD wallets
// conforming to BIP44, (i.e., m / purpose' / coin_type' / account' / change / address_index);
// ephemeral wallet instances can be derived from the top-level wallet. (WIP)
type Wallet struct {
	provide.Model
	WalletID      *uuid.UUID `sql:"type:uuid" json:"wallet_id,omitempty"`
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`

	Purpose    *int    `sql:"not null;default:44" json:"purpose,omitempty"`
	Mnemonic   *string `sql:"not null;type:bytea" json:"mnemonic,omitempty"`
	Seed       *string `sql:"not null;type:bytea" json:"-"`
	PublicKey  *string `sql:"not null;type:bytea" json:"public_key,omitempty"`  // i.e., the master public key
	PrivateKey *string `sql:"not null;type:bytea" json:"private_key,omitempty"` // i.e., the master private key

	Accounts []Account `gorm:"many2many:wallets_accounts" json:"-"`
}

// Create and persist an HD wallet
func (w *Wallet) Create() bool {
	db := dbconf.DatabaseConnection()

	w.generate(db)
	if !w.Validate() {
		return false
	}

	if db.NewRecord(w) {
		result := db.Create(&w)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				w.Errors = append(w.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(w) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate a wallet for persistence
func (w *Wallet) Validate() bool {
	w.Errors = make([]*provide.Error, 0)

	if w.ApplicationID == nil && w.UserID == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("no application or user identifier provided"),
		})
	} else if w.ApplicationID != nil && w.UserID != nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR user identifier should be provided"),
		})
	}
	if w.Purpose == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("purpose required; see https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki"),
		})
	}
	if w.Mnemonic == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("mnemonic required"),
		})
	}
	if w.Seed == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("seed required"),
		})
	}
	if w.PublicKey == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("public key required"),
		})
	}
	if w.PrivateKey == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("private key required"),
		})
	}

	return len(w.Errors) == 0
}

func (w *Wallet) decrypt() error {
	mnemonic, err := pgputil.PGPPubDecrypt([]byte(*w.Mnemonic))
	if err != nil {
		common.Log.Warningf("Failed to decrypt mnemonic; %s", err.Error())
		return err
	}
	w.Mnemonic = common.StringOrNil(string(mnemonic))

	seed, err := pgputil.PGPPubDecrypt([]byte(*w.Seed))
	if err != nil {
		common.Log.Warningf("Failed to decrypt seed; %s", err.Error())
		return err
	}
	w.Seed = common.StringOrNil(string(seed))

	publicKey, err := pgputil.PGPPubDecrypt([]byte(*w.PublicKey))
	if err != nil {
		common.Log.Warningf("Failed to decrypt public key; %s", err.Error())
		return err
	}
	w.PublicKey = common.StringOrNil(string(publicKey))

	privateKey, err := pgputil.PGPPubDecrypt([]byte(*w.PrivateKey))
	if err != nil {
		common.Log.Warningf("Failed to decrypt private key; %s", err.Error())
		return err
	}
	w.PrivateKey = common.StringOrNil(string(privateKey))

	return nil
}

func (w *Wallet) encrypt() error {
	encryptedMnemonic, err := pgputil.PGPPubEncrypt([]byte(*w.Mnemonic))
	if err != nil {
		common.Log.Warningf("Failed to encrypt mnemonic; %s", err.Error())
		return err
	}
	w.Mnemonic = common.StringOrNil(string(encryptedMnemonic))

	encryptedSeed, err := pgputil.PGPPubEncrypt([]byte(*w.Seed))
	if err != nil {
		common.Log.Warningf("Failed to encrypt seed; %s", err.Error())
		return err
	}
	w.Seed = common.StringOrNil(string(encryptedSeed))

	encryptedPublicKey, err := pgputil.PGPPubEncrypt([]byte(*w.PublicKey))
	if err != nil {
		common.Log.Warningf("Failed to encrypt public key; %s", err.Error())
		return err
	}
	w.PublicKey = common.StringOrNil(string(encryptedPublicKey))

	encryptedPrivateKey, err := pgputil.PGPPubEncrypt([]byte(*w.PrivateKey))
	if err != nil {
		common.Log.Warningf("Failed to encrypt private key; %s", err.Error())
		return err
	}
	w.PrivateKey = common.StringOrNil(string(encryptedPrivateKey))

	return nil
}

func (w *Wallet) generate(db *gorm.DB) error {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		common.Log.Warningf("failed to create entropy for HD wallet mnemonic; %s", err.Error())
		return err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		common.Log.Warningf("failed to generate HD wallet mnemonic from %d-bit entropy; %s", len(entropy)*8, err.Error())
		return err
	}

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		common.Log.Warningf("failed to generate seed from mnemonic: %s; %s", mnemonic, err.Error())
		return err
	}

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		common.Log.Warningf("failed to generate master key from mnemonic: %s; %s", mnemonic, err.Error())
		return err
	}

	seedstr := string(seed)
	xpub := masterKey.PublicKey().String()
	xprv := masterKey.String()

	w.Mnemonic = &mnemonic
	w.Seed = &seedstr
	w.PublicKey = &xpub
	w.PrivateKey = &xprv

	common.Log.Debugf("generated HD wallet master seed; mnemonic: %s; seed: %s; xpub: %s; xprv: %s", mnemonic, string(seed), xpub, xprv)

	return nil
}
