package token

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/contract"
	provide "github.com/provideservices/provide-go"
)

// Token instances must be associated with an application identifier.
type Token struct {
	provide.Model
	ApplicationID  *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID      uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
	ContractID     *uuid.UUID `sql:"type:uuid" json:"contract_id"`
	SaleContractID *uuid.UUID `sql:"type:uuid" json:"sale_contract_id"`
	Name           *string    `sql:"not null" json:"name"`
	Symbol         *string    `sql:"not null" json:"symbol"`
	Decimals       uint64     `sql:"not null" json:"decimals"`
	Address        *string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress    *string    `json:"sale_address"`           // non-null if token sale contract is specified
	AccessedAt     *time.Time `json:"accessed_at"`
}

// Create and persist a token
func (t *Token) Create() bool {
	db := dbconf.DatabaseConnection()

	if !t.Validate() {
		return false
	}

	if db.NewRecord(t) {
		result := db.Create(&t)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(t) {
			return rowsAffected > 0
		}
	}
	return false
}

// Validate a token for persistence
func (t *Token) Validate() bool {
	db := dbconf.DatabaseConnection()
	var contract = &contract.Contract{}
	if t.NetworkID != uuid.Nil {
		db.Model(t).Related(&contract)
	}
	t.Errors = make([]*provide.Error, 0)
	if t.NetworkID == uuid.Nil {
		t.Errors = append(t.Errors, &provide.Error{
			Message: common.StringOrNil("Unable to deploy token contract using unspecified network"),
		})
	} else {
		if contract != nil {
			if t.NetworkID != contract.NetworkID {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil("Token network did not match token contract network"),
				})
			}
			if t.Address == nil {
				t.Address = contract.Address
			} else if t.Address != nil && *t.Address != *contract.Address {
				t.Errors = append(t.Errors, &provide.Error{
					Message: common.StringOrNil("Token contract address did not match referenced contract address"),
				})
			}
		}
	}
	return len(t.Errors) == 0
}

// GetContract - retreieve the associated token contract
func (t *Token) GetContract() (*contract.Contract, error) {
	db := dbconf.DatabaseConnection()
	var contract = &contract.Contract{}
	db.Model(t).Related(&contract)
	if contract == nil {
		return nil, fmt.Errorf("Failed to retrieve token contract for token: %s", t.ID)
	}
	return contract, nil
}

// ReadEthereumContractAbi is called from wallet
func (t *Token) ReadEthereumContractAbi() (*abi.ABI, error) {
	contract, err := t.GetContract()
	if err != nil {
		return nil, err
	}
	return contract.ReadEthereumContractAbi()
}
