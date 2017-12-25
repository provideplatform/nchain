package main

import (
	"time"

	"github.com/satori/go.uuid"
)

type Model struct {
	Id        uuid.UUID `sql:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `sql:"not null" json:"created_at"`
	Errors    []*Error  `gorm:"-" json:"-"`
}

type Error struct {
	Message *string `json:"message"`
	Status  *int    `json:"status"`
}

type Network struct {
	Model
	Name         string    `sql:"not null" json:"name"`
	IsProduction bool      `sql:"not null" json:"is_production"`
	SidechainId  uuid.UUID `sql:"type:uuid" json:"sidechain_id"` // network id used as the transactional sidechain (or null)
}

type Token struct {
	Model
	NetworkId   uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Address     string    `sql:"not null" json:"address"` // network-specific token contract address
	SaleAddress string    `json:"sale_address"`           // non-null if token sale contract is specified
}

type Wallet struct {
	Model
	NetworkId  uuid.UUID `sql:"not null;type:uuid" json:"network_id"`
	Address    string    `sql:"not null" json:"address"`
	PrivateKey string    `sql:"not null" json:"-"`
}
