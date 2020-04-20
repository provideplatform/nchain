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

// Account represents a single address associated with a specific network and application or user
type Account struct {
	provide.Model
	NetworkID      *uuid.UUID `sql:"type:uuid" json:"network_id,omitempty"`
	WalletID       *uuid.UUID `sql:"type:uuid" json:"wallet_id,omitempty"`
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	UserID         *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`
	OrganizationID *uuid.UUID `sql:"type:uuid" json:"organization_id,omitempty"`
	Type           *string    `json:"type,omitempty"`

	HDDerivationPath *string `json:"hd_derivation_path,omitempty"` // i.e. m/44'/60'/0'/0
	PublicKey        *string `sql:"type:bytea" json:"public_key,omitempty"`
	PrivateKey       *string `sql:"not null;type:bytea" json:"-"`

	Address    string     `sql:"not null" json:"address"`
	Balance    *big.Int   `sql:"-" json:"balance,omitempty"`
	AccessedAt *time.Time `json:"accessed_at,omitempty"`
	Wallet     *Wallet    `sql:"-" json:"-"`
}

// SetID sets the account id in-memory
func (a *Account) SetID(accountID uuid.UUID) {
	if a.ID != uuid.Nil {
		common.Log.Warningf("Attempted to change a account id in memory; account id not changed: %s", a.ID)
		return
	}
	a.ID = accountID
}

func (a *Account) generate(db *gorm.DB) error {
	network, _ := a.GetNetwork()

	if a.NetworkID != nil && *a.NetworkID == uuid.Nil {
		err := errors.New("Unable to generate private key for account without an associated network")
		common.Log.Warning(err.Error())
		return err
	}

	var encodedPrivateKey *string

	if network.IsEthereumNetwork() {
		addr, privateKey, err := provide.EVMGenerateKeyPair()
		if err != nil {
			err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
			common.Log.Warning(err.Error())
			return err
		}

		a.Address = *addr
		encodedPrivateKey = common.StringOrNil(hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
	} else if network.IsBcoinNetwork() {
		var version byte = 0x00
		networkCfg := network.ParseConfig()
		if networkVersion, networkVersionOk := networkCfg["version"].(string); networkVersionOk {
			versionBytes, err := hex.DecodeString(networkVersion)
			if err != nil {
				err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
				common.Log.Warning(err.Error())
				return err
			} else if len(versionBytes) != 1 {
				err := fmt.Errorf("Unable to generate private key for unsupported bitcoin network version %s for network: %s", networkVersion, a.NetworkID.String())
				common.Log.Warning(err.Error())
				return err
			} else {
				version = versionBytes[0]
			}
		}

		addr, privateKey, err := provide.BcoinGenerateKeyPair(version)
		if err != nil {
			err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
			common.Log.Warning(err.Error())
			return err
		}

		a.Address = *addr
		encodedPrivateKey = common.StringOrNil(fmt.Sprintf("%X", privateKey.D))
	} else {
		err := fmt.Errorf("Unable to generate private key for account using unsupported network: %s", a.NetworkID.String())
		common.Log.Warning(err.Error())
		return err
	}

	if encodedPrivateKey != nil {
		encryptedPrivateKey, err := pgputil.PGPPubEncrypt([]byte(*encodedPrivateKey))
		if err != nil {
			common.Log.Warningf("Failed to encrypt private key; %s", err.Error())
			return err
		}
		a.PrivateKey = common.StringOrNil(string(encryptedPrivateKey))
		common.Log.Debugf("Generated account signing address: %s", a.Address)
	} else {
		err := errors.New("Unable to generate private key for account due to an unhandled error")
		common.Log.Warning(err.Error())
		return err
	}

	return nil
}

// GetNetwork - retrieve the associated network
func (a *Account) GetNetwork() (*network.Network, error) {
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(a).Related(&network)
	if network == nil {
		return nil, fmt.Errorf("Failed to retrieve associated network for account: %s", a.ID)
	}
	return network, nil
}

// Create and persist a network-specific account used for storing crpyotcurrency or digital tokens native to a specific network
func (a *Account) Create() bool {
	db := dbconf.DatabaseConnection()

	a.generate(db)
	if !a.Validate() {
		return false
	}

	if db.NewRecord(a) {
		result := db.Create(&a)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				a.Errors = append(a.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(a) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate an account for persistence
func (a *Account) Validate() bool {
	a.Errors = make([]*provide.Error, 0)
	var network = &network.Network{}
	dbconf.DatabaseConnection().Model(a).Related(&network)
	if a.NetworkID != nil && *a.NetworkID == uuid.Nil {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil(fmt.Sprintf("invalid network association attempted with network id: %s", a.NetworkID.String())),
		})
	}
	if a.ApplicationID == nil && a.UserID == nil && a.OrganizationID == nil {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil("no application, user or organization identifier provided"),
		})
	} else if a.ApplicationID != nil && (a.UserID != nil || a.OrganizationID != nil) {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil("only an application OR user or organization identifier should be provided"),
		})
	}
	var err error
	if a.PrivateKey != nil {
		if network.IsEthereumNetwork() {
			_, err = common.DecryptECDSAPrivateKey(*a.PrivateKey)
		}
	} else {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil("private key generation failed"),
		})
	}

	if err != nil {
		msg := err.Error()
		a.Errors = append(a.Errors, &provide.Error{
			Message: &msg,
		})
	}
	return len(a.Errors) == 0
}

// NativeCurrencyBalance retrieves a account's native currency/token balance
func (a *Account) NativeCurrencyBalance() (*big.Int, error) {
	var balance *big.Int
	var err error
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	db.Model(a).Related(&network)
	if network.IsEthereumNetwork() {
		balance, err = provide.EVMGetNativeBalance(network.ID.String(), network.RPCURL(), a.Address)
		if err != nil {
			return nil, err
		}
	} else {
		common.Log.Warningf("unable to read native currency balance for network: %s", a.NetworkID)
	}
	return balance, nil
}

// TokenBalance retrieves a account's token balance for a given token id
func (a *Account) TokenBalance(tokenID string) (*big.Int, error) {
	var balance *big.Int
	db := dbconf.DatabaseConnection()
	var network = &network.Network{}
	var token = &token.Token{}
	db.Model(a).Related(&network)
	db.Where("id = ?", tokenID).Find(&token)
	if token == nil {
		return nil, fmt.Errorf("Unable to read token balance for invalid token: %s", tokenID)
	}
	if network.IsEthereumNetwork() {
		contractAbi, err := token.ReadEthereumContractAbi()
		balance, err = provide.EVMGetTokenBalance(network.ID.String(), network.RPCURL(), *token.Address, a.Address, contractAbi)
		if err != nil {
			return nil, err
		}
	} else {
		common.Log.Warningf("Unable to read native currency balance for network: %s", a.NetworkID)
	}
	return balance, nil
}

func (a *Account) ephemeralResponse() map[string]interface{} {
	ephemeralResponse := map[string]interface{}{
		"address":            a.Address,
		"hd_derivation_path": a.HDDerivationPath,
		"public_key":         a.PublicKey,
		"private_key":        a.PrivateKey,
	}

	if a.Balance != nil {
		ephemeralResponse["balance"] = a.Balance
	}

	return ephemeralResponse
}
