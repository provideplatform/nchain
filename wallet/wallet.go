package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/token"
	provide "github.com/provideservices/provide-go"
)

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Wallet{})
	db.Model(&Wallet{}).AddIndex("idx_wallets_application_id", "application_id")
	db.Model(&Wallet{}).AddIndex("idx_wallets_user_id", "user_id")
	db.Model(&Wallet{}).AddIndex("idx_wallets_accessed_at", "accessed_at")
	db.Model(&Wallet{}).AddIndex("idx_wallets_network_id", "network_id")
	db.Model(&Wallet{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// Wallet instances must be associated with exactly one instance of either an a) application identifier or b) user identifier.
type Wallet struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	UserID        *uuid.UUID `sql:"type:uuid" json:"user_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	Address       string     `sql:"not null" json:"address"`
	PrivateKey    *string    `sql:"not null;type:bytea" json:"-"`
	Balance       *big.Int   `sql:"-" json:"balance"`
	AccessedAt    *time.Time `json:"accessed_at"`
}

func MigrateEncryptedPrivateKeys() {
	db := dbconf.DatabaseConnection()
	var wallets []Wallet
	db.Find(&wallets)
	for _, w := range wallets {
		err := w.migrateEncryptedPrivateKey(db)
		if err != nil {
			common.Log.Warningf("Failed to migrate encrypted wallet private key: %s", err.Error())
		}
	}
	common.Log.Debugf("Migrated encrypted private key for %d wallets...", len(wallets))
}

func (w *Wallet) migrateEncryptedPrivateKey(db *gorm.DB) error {
	if w.PrivateKey != nil {
		decryptedPrivateKey, err := common.PSQLPGPPubDecrypt(*w.PrivateKey, common.GpgPrivateKey, common.GpgPassword)
		if err != nil {
			common.Log.Warningf("Failed to decrypt private key; %s", err.Error())
			return err
		}
		encryptedPrivateKey, err := pgputil.PGPPubEncrypt(decryptedPrivateKey)
		if err != nil {
			common.Log.Warningf("Failed to encrypt private key; %s", err.Error())
			return err
		}
		w.PrivateKey = common.StringOrNil(string(encryptedPrivateKey))

		result := db.Save(&w)
		errors := result.GetErrors()
		if len(errors) > 0 {
			return errors[0]
		}
	}
	return nil
}

// SetID sets the wallet id in-memory
func (w *Wallet) SetID(walletID uuid.UUID) {
	if w.ID != uuid.Nil {
		common.Log.Warningf("Attempted to change a wallet id in memory; wallet id not changed: %s", w.ID)
		return
	}
	w.ID = walletID
}

func (w *Wallet) generate(db *gorm.DB) error {
	network, _ := w.GetNetwork()

	if w.NetworkID == uuid.Nil {
		err := errors.New("Unable to generate private key for wallet without an associated network")
		common.Log.Warning(err.Error())
		return err
	}

	var encodedPrivateKey *string

	if network.IsEthereumNetwork() {
		addr, privateKey, err := provide.EVMGenerateKeyPair()
		if err != nil {
			err := fmt.Errorf("Unable to generate private key for bitcoin wallet for network: %s; %s", w.NetworkID.String(), err.Error())
			common.Log.Warning(err.Error())
			return err
		}

		w.Address = *addr
		encodedPrivateKey = common.StringOrNil(hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
	} else if network.IsBcoinNetwork() {
		var version byte = 0x00
		networkCfg := network.ParseConfig()
		if networkVersion, networkVersionOk := networkCfg["version"].(string); networkVersionOk {
			versionBytes, err := hex.DecodeString(networkVersion)
			if err != nil {
				err := fmt.Errorf("Unable to generate private key for bitcoin wallet for network: %s; %s", w.NetworkID.String(), err.Error())
				common.Log.Warning(err.Error())
				return err
			} else if len(versionBytes) != 1 {
				err := fmt.Errorf("Unable to generate private key for unsupported bitcoin network version %s for network: %s", networkVersion, w.NetworkID.String())
				common.Log.Warning(err.Error())
				return err
			} else {
				version = versionBytes[0]
			}
		}

		addr, privateKey, err := provide.BcoinGenerateKeyPair(version)
		if err != nil {
			err := fmt.Errorf("Unable to generate private key for bitcoin wallet for network: %s; %s", w.NetworkID.String(), err.Error())
			common.Log.Warning(err.Error())
			return err
		}

		w.Address = *addr
		encodedPrivateKey = common.StringOrNil(fmt.Sprintf("%X", privateKey.D))
	} else {
		err := fmt.Errorf("Unable to generate private key for wallet using unsupported network: %s", w.NetworkID.String())
		common.Log.Warning(err.Error())
		return err
	}

	if encodedPrivateKey != nil {
		encryptedPrivateKey, err := pgputil.PGPPubEncrypt([]byte(*encodedPrivateKey))
		if err != nil {
			common.Log.Warningf("Failed to encrypt private key; %s", err.Error())
			return err
		}
		w.PrivateKey = common.StringOrNil(string(encryptedPrivateKey))
		common.Log.Debugf("Generated wallet signing address: %s", w.Address)
	} else {
		err := errors.New("Unable to generate private key for wallet due to an unhandled error")
		common.Log.Warning(err.Error())
		return err
	}

	return nil
}

// GetNetwork - retrieve the associated transaction network
func (w *Wallet) GetNetwork() (*network.Network, error) {
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(w).Related(&network)
	if network == nil {
		return nil, fmt.Errorf("Failed to retrieve associated network for wallet: %s", w.ID)
	}
	return network, nil
}

// Create and persist a network-specific wallet used for storing crpyotcurrency or digital tokens native to a specific network
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
	var network = &network.Network{}
	dbconf.DatabaseConnection().Model(w).Related(&network)
	if w.NetworkID == uuid.Nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil(fmt.Sprintf("invalid network association attempted with network id: %s", w.NetworkID.String())),
		})
	}
	if w.ApplicationID == nil && w.UserID == nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("no application or user identifier provided"),
		})
	} else if w.ApplicationID != nil && w.UserID != nil {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR user identifier should be provided"),
		})
	}
	var err error
	if w.PrivateKey != nil {
		if network.IsEthereumNetwork() {
			_, err = common.DecryptECDSAPrivateKey(*w.PrivateKey)
		}
	} else {
		w.Errors = append(w.Errors, &provide.Error{
			Message: common.StringOrNil("private key generation failed"),
		})
	}

	if err != nil {
		msg := err.Error()
		w.Errors = append(w.Errors, &provide.Error{
			Message: &msg,
		})
	}
	return len(w.Errors) == 0
}

// NativeCurrencyBalance retrieves a wallet's native currency/token balance
func (w *Wallet) NativeCurrencyBalance() (*big.Int, error) {
	var balance *big.Int
	var err error
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(w).Related(&network)
	if network.IsEthereumNetwork() {
		balance, err = provide.EVMGetNativeBalance(network.ID.String(), network.RPCURL(), w.Address)
		if err != nil {
			return nil, err
		}
	} else {
		common.Log.Warningf("Unable to read native currency balance for network: %s", w.NetworkID)
	}
	return balance, nil
}

// TokenBalance retrieves a wallet's token balance for a given token id
func (w *Wallet) TokenBalance(tokenID string) (*big.Int, error) {
	var balance *big.Int
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	var token = &token.Token{}
	db.Model(w).Related(&network)
	db.Where("id = ?", tokenID).Find(&token)
	if token == nil {
		return nil, fmt.Errorf("Unable to read token balance for invalid token: %s", tokenID)
	}
	if network.IsEthereumNetwork() {
		contractAbi, err := token.ReadEthereumContractAbi()
		balance, err = provide.EVMGetTokenBalance(network.ID.String(), network.RPCURL(), *token.Address, w.Address, contractAbi)
		if err != nil {
			return nil, err
		}
	} else {
		common.Log.Warningf("Unable to read native currency balance for network: %s", w.NetworkID)
	}
	return balance, nil
}

// not used
// TxCount retrieves a count of transactions signed by the wallet
// func (w *Wallet) TxCount() (count *uint64) {
// 	db := dbconf.DatabaseConnection()
// 	db.Model(&Transaction{}).Where("wallet_id = ?", w.ID).Count(&count)
// 	return count
// }
