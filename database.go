package main

import (
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/kthomas/go-db-config"
)

var (
	migrateOnce sync.Once
)

func migrateSchema() {
	migrateOnce.Do(func() {
		db := dbconf.DatabaseConnection()

		db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
		db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

		initial := !db.HasTable(&Network{})

		db.AutoMigrate(&Network{})
		db.Model(&Network{}).AddForeignKey("sidechain_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Token{})
		db.Model(&Token{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Wallet{})
		db.Model(&Wallet{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Transaction{})
		db.Model(&Transaction{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Transaction{}).AddForeignKey("wallet_id", "wallets(id)", "SET NULL", "CASCADE")

		if initial {
			populateInitialNetworks()
		}
	})
}

func populateInitialNetworks() {
	db := dbconf.DatabaseConnection()

	var btcMainnet = &Network{}
	db.Raw("INSERT INTO networks (name, description, is_production) values ('Bitcoin', 'Bitcoin Mainnet', true) RETURNING id").Scan(&btcMainnet)

	var btcTestnet = &Network{}
	db.Raw("INSERT INTO networks (name, description, is_production) values ('Bitcoin Testnet', 'Bitcoin Testnet', false) RETURNING id").Scan(&btcTestnet)

	var ltcMainnet = &Network{}
	db.Raw("INSERT INTO networks (name, description, is_production) values ('Lightning', 'Lightning Network mainnet', true) RETURNING id").Scan(&ltcMainnet)

	var ltcTestnet = &Network{}
	db.Raw("INSERT INTO networks (name, description, is_production) values ('Lightning Testnet', 'Lightning Network testnet', false) RETURNING id").Scan(&ltcTestnet)

	db.Exec("UPDATE networks SET sidechain_id = ? WHERE id = ?", ltcMainnet.Id, btcMainnet.Id)
	db.Exec("UPDATE networks SET sidechain_id = ? WHERE id = ?", ltcTestnet.Id, btcTestnet.Id)

	db.Exec("INSERT INTO networks (name, description, is_production) values ('Ethereum', 'Ethereum mainnet', true)")
	db.Exec("INSERT INTO networks (name, description, is_production) values ('Ethereum Testnet', 'ROPSTEN (Revival) TESTNET', false)")
}

func DatabaseConnection() *gorm.DB {
	return dbconf.DatabaseConnection()
}
