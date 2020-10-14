package wallet

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	bip32 "github.com/FactomProject/go-bip32"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go/api"
	vault "github.com/provideservices/provide-go/api/vault"
	util "github.com/provideservices/provide-go/common/util"

	"github.com/jinzhu/gorm"
	"github.com/provideapp/nchain/common"
)

// Wallet instances are logical collections of accounts; wallet instances are HD wallets
// conforming to BIP44, (i.e., m / purpose' / coin_type' / account' / change / address_index);
// supports derivation of up to 2,147,483,648 associated addresses per hardened account.
type Wallet struct {
	provide.Model
	WalletID       *uuid.UUID `sql:"-" json:"wallet_id,omitempty"`
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	UserID         *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`
	OrganizationID *uuid.UUID `sql:"type:uuid" json:"organization_id,omitempty"`

	VaultID *uuid.UUID `sql:"type:uuid" json:"vault_id,omitempty"`
	KeyID   *uuid.UUID `sql:"type:uuid" json:"key_id,omitempty"`

	Path        *string    `sql:"-" json:"path,omitempty"`
	Purpose     *int       `sql:"not null;default:44" json:"purpose,omitempty"`
	Mnemonic    *string    `sql:"-" json:"mnemonic,omitempty"`
	ExtendedKey *bip32.Key `sql:"-" json:"-"`

	PublicKey  *string `sql:"not null;type:bytea" json:"public_key,omitempty"`
	PrivateKey *string `sql:"-" json:"private_key,omitempty"`

	Wallet   *Wallet   `sql:"-" json:"-"`
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

// SetID sets the wallet id in-memory
func (w *Wallet) SetID(walletID uuid.UUID) {
	if w.ID != uuid.Nil {
		common.Log.Warningf("Attempted to change a wallet id in memory; wallet id not changed: %s", w.ID)
		return
	}
	w.ID = walletID
}

// Validate a wallet for persistence
func (w *Wallet) Validate() bool {
	w.Errors = make([]*provide.Error, 0)

	if w.ApplicationID == nil && w.UserID == nil && w.OrganizationID == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("no application, user or organization identifier provided"),
		})
	} else if w.ApplicationID != nil && w.UserID != nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR user or organization identifier should be provided"),
		})
	} else if w.UserID != nil && w.OrganizationID != nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("only a user OR organization identifier should be provided"),
		})
	}
	if w.VaultID == nil || *w.VaultID == uuid.Nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("vault id required"),
		})
	}
	if w.KeyID == nil || *w.KeyID == uuid.Nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("vault key id required"),
		})
	}
	if w.Purpose == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("purpose required; see https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki"),
		})
	}
	if w.PublicKey == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("public key required"),
		})
	}

	return len(w.Errors) == 0
}

// DeriveHardened derives the hardened child account from the parent wallet (i.e., per bip32);
// the derived wallet is initialized for the given purpose and coin such that the new account
// exists at `m/purpose'/coin_type'/account'`; this method will fail if the next level in
// the HD hierarchy must be non-hardened.
func (w *Wallet) DeriveHardened(db *gorm.DB, coin, account uint32) (*Wallet, error) {
	pathstr := fmt.Sprintf("m/%d'/%d'/%d'", *w.Purpose, coin, account)

	key, err := vault.CreateKey(util.DefaultVaultAccessJWT, common.DefaultVault.ID.String(), map[string]interface{}{
		"type":               "asymmetric",
		"usage":              "sign/verify",
		"spec":               "BIP39",
		"name":               "nchain hd wallet",
		"hd_derivation_path": pathstr,
	})

	if err != nil {
		err := fmt.Errorf("unable to generate key material for HD wallet; %s", err.Error())
		common.Log.Warning(err.Error())
		return nil, err
	}

	w.Path = key.HDDerivationPath
	w.PublicKey = key.PublicKey
	w.VaultID = key.VaultID
	w.KeyID = &key.ID

	return nil, fmt.Errorf("failed to derive hardened child with path: %s", pathstr)

	// masterKey := w.ExtendedKey
	// if masterKey == nil {
	// 	masterKey, _ = bip32.NewMasterKey([]byte(*w.Seed))
	// }
	// if masterKey == nil {
	// 	return nil, fmt.Errorf("failed to reinitialize master key to attempt account derivation at path: %s", pathstr)
	// }
	// common.Log.Debugf("reinitialized master key to attempt account derivation at path: %s", pathstr)

	// childKey, err := masterKey.NewChildKey(0x80000000 + coin)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to initialize child key at derivation path: m/%d'/%d'; %s", *w.Purpose, coin, err.Error())
	// }
	// common.Log.Debugf("derived child key at derivation path: m/%d'/%d'", *w.Purpose, coin)

	// w0 := &Wallet{
	// 	Path:    &pathstr,
	// 	Purpose: w.Purpose,
	// 	Wallet:  w,
	// }
	// w0.populate(childKey)

	// childKey, err = childKey.NewChildKey(0x80000000 + account)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to initialize child key at derivation path: m/%d'/%d'/%d'; %s", *w.Purpose, coin, account, err.Error())
	// }
	// common.Log.Debugf("derived child key at derivation path: m/%d'/%d'/%d'", *w.Purpose, coin, account)

	// w1 := &Wallet{
	// 	Path:    &pathstr,
	// 	Purpose: w.Purpose,
	// 	Wallet:  w0,
	// }
	// w1.populate(childKey)

	// return w1, nil
}

// DeriveAddress derives a child address from the parent wallet which should be a hardened
// child account (i.e., per bip32); the derived account is initialized for the purpose,
// coin type and hardened account per the rules of the active subtree such that the new signer
// exists at `m/purpose'/coin_type'/account'/<index>` (or `m/purpose'/coin_type'/account'/<chain>/<index>`
// if a chain index is provided). Returns an `Account` instance.
func (w *Wallet) DeriveAddress(db *gorm.DB, index uint32, chain *uint32) (*Account, error) {
	pathstr := fmt.Sprintf("%s/%d", *w.Path, index)
	if chain != nil {
		pathstr = fmt.Sprintf("%s/%d/%d", *w.Path, *chain, index)
	}

	components := strings.Split(*w.Path, "/")
	_, err := strconv.Atoi(strings.Trim(components[len(components)-2], "'"))
	if err != nil {
		common.Log.Warningf("failed to derive signing account at derivation path: %s; %s", pathstr, err.Error())
		return nil, err
	}

	if index >= bip32.FirstHardenedChild {
		return nil, fmt.Errorf("unable to derive signing address above 0x80000000; index: %d", index)
	}

	if w.Path == nil {
		return nil, errors.New("failed to derive signing address without hardened HD path")
	}

	key, err := vault.CreateKey(util.DefaultVaultAccessJWT, common.DefaultVault.ID.String(), map[string]interface{}{
		"type":               "asymmetric",
		"usage":              "sign/verify",
		"spec":               "BIP39",
		"name":               "nchain hd wallet",
		"hd_derivation_path": pathstr,
	})

	if err != nil {
		err := fmt.Errorf("unable to generate key material for HD wallet; %s", err.Error())
		common.Log.Warning(err.Error())
		return nil, err
	}

	w.Path = key.HDDerivationPath
	w.PublicKey = key.PublicKey
	w.VaultID = key.VaultID
	w.KeyID = &key.ID

	return nil, fmt.Errorf("failed to derive address with path: %s", pathstr)

	// var mnemonic *string
	// parent := w
	// for {
	// 	parent = parent.Wallet
	// 	if parent != nil {
	// 		mnemonic = parent.Mnemonic
	// 	} else {
	// 		break
	// 	}
	// }

	// pathstr := fmt.Sprintf("%s/%d", *w.Path, index)
	// if chain != nil {
	// 	pathstr = fmt.Sprintf("%s/%d/%d", *w.Path, *chain, index)
	// }

	// components := strings.Split(*w.Path, "/")
	// coin, err := strconv.Atoi(strings.Trim(components[len(components)-2], "'"))
	// if err != nil {
	// 	common.Log.Warningf("failed to derive signing account at derivation path: %s; %s", pathstr, err.Error())
	// 	return nil, err
	// }

	// var account *Account
	// purpose := 0x80000000 + uint32(*w.Purpose)

	// switch purpose {
	// case bip44.Purpose:
	// 	switch 0x80000000 + uint32(coin) {
	// 	case bip44.TypeEther:
	// 		hdw, err := hdwallet.NewFromMnemonic(*mnemonic)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to derive signing account at derivation path: %s; %s", pathstr, err.Error())
	// 		}

	// 		path, err := hdwallet.ParseDerivationPath(pathstr)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to derive signing account at derivation path: %s; %s", pathstr, err.Error())
	// 		}

	// 		derived, err := hdw.Derive(path, false)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to derive signing account at derivation path: %s; %s", pathstr, err.Error())
	// 		}

	// 		privateKey, err := hdw.PrivateKeyBytes(derived)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to derive private key for signing account at derivation path: %s; %s", pathstr, err.Error())
	// 		}

	// 		publicKey, err := hdw.PublicKeyBytes(derived)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to derive private key for signing account at derivation path: %s; %s", pathstr, err.Error())
	// 		}

	// 		account = &Account{
	// 			ApplicationID:    w.ApplicationID,
	// 			UserID:           w.UserID,
	// 			WalletID:         &w.ID,
	// 			Wallet:           w,
	// 			HDDerivationPath: &pathstr,
	// 			Address:          derived.Address.Hex(),
	// 			PublicKey:        common.StringOrNil(hexutil.Encode(publicKey)[4:]),
	// 			PrivateKey:       common.StringOrNil(hexutil.Encode(privateKey)[2:]),
	// 		}

	// 		common.Log.Debugf("derived address for signing account at derivation path: %s; address: %s", pathstr, account.Address)
	// 	default:
	// 		return nil, fmt.Errorf("failed to derive signing account at derivation path: %s; unsupported coin: %d", pathstr, coin)
	// 	}
	// default:
	// 	return nil, fmt.Errorf("failed to derive signing account at derivation path: %s; unsupported purpose: %d", pathstr, *w.Purpose)
	// }

	// return account, nil
}

func (w *Wallet) generate(db *gorm.DB) error {
	key, err := vault.CreateKey(util.DefaultVaultAccessJWT, common.DefaultVault.ID.String(), map[string]interface{}{
		"type":  "asymmetric",
		"usage": "sign/verify",
		"spec":  "BIP39",
		"name":  "nchain hd wallet",
	})

	if err != nil {
		err := fmt.Errorf("unable to generate key material for HD wallet; %s", err.Error())
		common.Log.Warning(err.Error())
		return err
	}

	w.Path = key.HDDerivationPath
	w.PublicKey = key.PublicKey
	w.VaultID = key.VaultID
	w.KeyID = &key.ID

	// FIXME!!!! w.populate(masterKey)

	common.Log.Debugf("generated HD wallet using vault: %s; key id: %s; public key: %s", w.VaultID.String(), key.ID.String(), *w.PublicKey)
	// common.Log.Debugf("generated HD wallet using vault; key id: %s%d-byte master seed with mnemonic; xpub: %s", len(seed), *w.PublicKey)
	return nil
}

// func (w *Wallet) populate(key *bip32.Key) {
// 	xpub := key.PublicKey().String()
// 	xprv := key.String()

// 	w.ExtendedKey = key
// 	w.PublicKey = &xpub
// 	// w.PrivateKey = &xprv
// }
