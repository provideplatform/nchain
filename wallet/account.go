package wallet

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
	"github.com/provideapp/nchain/token"
	provide "github.com/provideservices/provide-go/api"
	vault "github.com/provideservices/provide-go/api/vault"
	util "github.com/provideservices/provide-go/common/util"
	providecrypto "github.com/provideservices/provide-go/crypto"
)

// Account represents a single address associated with a specific network and application or user
type Account struct {
	provide.Model
	NetworkID      *uuid.UUID `sql:"type:uuid" json:"network_id,omitempty"`
	WalletID       *uuid.UUID `sql:"type:uuid" json:"wallet_id,omitempty"`
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id,omitempty"`
	UserID         *uuid.UUID `sql:"type:uuid" json:"user_id,omitempty"`
	OrganizationID *uuid.UUID `sql:"type:uuid" json:"organization_id,omitempty"`

	VaultID *uuid.UUID `sql:"type:uuid" json:"vault_id,omitempty"`
	KeyID   *uuid.UUID `sql:"type:uuid" json:"key_id,omitempty"`

	Type *string `json:"type,omitempty"`

	HDDerivationPath *string `json:"hd_derivation_path,omitempty"` // i.e. m/44'/60'/0'/0
	PublicKey        *string `sql:"type:bytea" json:"public_key,omitempty"`
	PrivateKey       *string `sql:"-" json:"private_key,omitempty"`

	Address    string     `sql:"not null" json:"address"`
	Balance    *big.Int   `sql:"-" json:"balance,omitempty"`
	AccessedAt *time.Time `json:"accessed_at,omitempty"`
	Wallet     *Wallet    `sql:"-" json:"-"`
	// Key     *vault.Key    `sql:"-" json:"-"`
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
		err := errors.New("unable to generate key material for account without an associated network")
		common.Log.Warning(err.Error())
		return err
	}

	key, err := vault.CreateKey(util.DefaultVaultAccessJWT, common.DefaultVault.ID.String(), map[string]interface{}{
		"type":  "asymmetric",
		"usage": "sign/verify",
		"spec":  "secp256k1",
		"name":  fmt.Sprintf("nchain account for network: %s", network.ID.String()),
	})

	if err != nil {
		err := fmt.Errorf("unable to generate key material for account; %s", err.Error())
		common.Log.Warning(err.Error())
		return err
	}

	if key.Address != nil {
		a.Address = *key.Address
	}

	a.PublicKey = key.PublicKey
	a.VaultID = key.VaultID
	a.KeyID = &key.ID

	// if network.IsEthereumNetwork() {
	// 	// addr, privateKey, err := providecrypto.EVMGenerateKeyPair()
	// 	// if err != nil {
	// 	// 	err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
	// 	// 	common.Log.Warning(err.Error())
	// 	// 	return err
	// 	// }

	// 	// encodedPrivateKey = common.StringOrNil(hex.EncodeToString(ethcrypto.FromECDSA(privateKey)))
	// } else if network.IsBcoinNetwork() {
	// 	// var version byte = 0x00
	// 	// networkCfg := network.ParseConfig()
	// 	// if networkVersion, networkVersionOk := networkCfg["version"].(string); networkVersionOk {
	// 	// 	versionBytes, err := hex.DecodeString(networkVersion)
	// 	// 	if err != nil {
	// 	// 		err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
	// 	// 		common.Log.Warning(err.Error())
	// 	// 		return err
	// 	// 	} else if len(versionBytes) != 1 {
	// 	// 		err := fmt.Errorf("Unable to generate private key for unsupported bitcoin network version %s for network: %s", networkVersion, a.NetworkID.String())
	// 	// 		common.Log.Warning(err.Error())
	// 	// 		return err
	// 	// 	} else {
	// 	// 		version = versionBytes[0]
	// 	// 	}
	// 	// }

	// 	// addr, privateKey, err := providecrypto.BcoinGenerateKeyPair(version)
	// 	// if err != nil {
	// 	// 	err := fmt.Errorf("Unable to generate private key for bitcoin account for network: %s; %s", a.NetworkID.String(), err.Error())
	// 	// 	common.Log.Warning(err.Error())
	// 	// 	return err
	// 	// }

	// 	// a.Address = *addr
	// 	// encodedPrivateKey = common.StringOrNil(fmt.Sprintf("%X", privateKey.D))
	// } else {
	// 	err := fmt.Errorf("Unable to generate key material for account using unsupported network: %s", a.NetworkID.String())
	// 	common.Log.Warning(err.Error())
	// 	return err
	// }

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
	if a.VaultID == nil || *a.VaultID == uuid.Nil {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil("vault id required"),
		})
	}
	if a.KeyID == nil || *a.KeyID == uuid.Nil {
		a.Errors = append(a.Errors, &provide.Error{
			Message: common.StringOrNil("vault key id required"),
		})
	}
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
		balance, err = providecrypto.EVMGetNativeBalance(network.ID.String(), network.RPCURL(), a.Address)
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
		balance, err = providecrypto.EVMGetTokenBalance(network.ID.String(), network.RPCURL(), *token.Address, a.Address, contractAbi)
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
