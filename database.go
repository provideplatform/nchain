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

		db.AutoMigrate(&Network{})
		db.Model(&Network{}).AddIndex("idx_networks_application_id", "application_id")
		db.Model(&Network{}).AddIndex("idx_networks_network_id", "network_id")
		db.Model(&Network{}).AddIndex("idx_networks_user_id", "user_id")
		db.Model(&Network{}).AddIndex("idx_networks_cloneable", "cloneable")
		db.Model(&Network{}).AddIndex("idx_networks_enabled", "enabled")
		db.Model(&Network{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Network{}).AddForeignKey("sidechain_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Network{}).AddUniqueIndex("idx_chain_id", "chain_id")

		db.AutoMigrate(&NetworkNode{})
		db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_network_id", "network_id")
		db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_region", "region")
		db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_role", "role")
		db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_status", "status")
		db.Model(&NetworkNode{}).AddIndex("idx_network_nodes_bootnode", "bootnode")
		db.Model(&NetworkNode{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&LoadBalancer{})
		db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_network_id", "network_id")
		db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_region", "region")
		db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_status", "status")
		db.Model(&LoadBalancer{}).AddIndex("idx_load_balancers_type", "type")
		db.Model(&LoadBalancer{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Wallet{})
		db.Model(&Wallet{}).AddIndex("idx_wallets_application_id", "application_id")
		db.Model(&Wallet{}).AddIndex("idx_wallets_user_id", "user_id")
		db.Model(&Wallet{}).AddIndex("idx_wallets_accessed_at", "accessed_at")
		db.Model(&Wallet{}).AddIndex("idx_wallets_network_id", "network_id")
		db.Model(&Wallet{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Transaction{})
		db.Model(&Transaction{}).AddIndex("idx_transactions_application_id", "application_id")
		db.Model(&Transaction{}).AddIndex("idx_transactions_created_at", "created_at")
		db.Model(&Transaction{}).AddIndex("idx_transactions_status", "status")
		db.Model(&Transaction{}).AddIndex("idx_transactions_network_id", "network_id")
		db.Model(&Transaction{}).AddIndex("idx_transactions_user_id", "user_id")
		db.Model(&Transaction{}).AddIndex("idx_transactions_wallet_id", "wallet_id")
		db.Model(&Transaction{}).AddIndex("idx_transactions_ref", "ref")
		db.Model(&Transaction{}).AddUniqueIndex("idx_transactions_hash", "hash")
		db.Model(&Transaction{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Transaction{}).AddForeignKey("wallet_id", "wallets(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Contract{})
		db.Model(&Contract{}).AddIndex("idx_contracts_application_id", "application_id")
		db.Model(&Contract{}).AddIndex("idx_contracts_accessed_at", "accessed_at")
		db.Model(&Contract{}).AddIndex("idx_contracts_address", "address")
		db.Model(&Contract{}).AddIndex("idx_contracts_contract_id", "contract_id")
		db.Model(&Contract{}).AddIndex("idx_contracts_network_id", "network_id")
		db.Model(&Contract{}).AddUniqueIndex("idx_contracts_transaction_id", "transaction_id")
		db.Model(&Contract{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
		db.Model(&Contract{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Contract{}).AddForeignKey("transaction_id", "transactions(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Bridge{})
		db.Model(&Bridge{}).AddIndex("idx_bridges_application_id", "application_id")
		db.Model(&Bridge{}).AddIndex("idx_bridges_network_id", "network_id")
		db.Model(&Bridge{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Connector{})
		db.Model(&Connector{}).AddIndex("idx_connectors_application_id", "application_id")
		db.Model(&Connector{}).AddIndex("idx_connectors_network_id", "network_id")
		db.Model(&Connector{}).AddIndex("idx_connectors_type", "type")
		db.Model(&Connector{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Filter{})
		db.Model(&Filter{}).AddIndex("idx_filters_application_id", "application_id")
		db.Model(&Filter{}).AddIndex("idx_filters_network_id", "network_id")
		db.Model(&Filter{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Oracle{})
		db.Model(&Oracle{}).AddIndex("idx_oracles_application_id", "application_id")
		db.Model(&Oracle{}).AddIndex("idx_oracles_network_id", "network_id")
		db.Model(&Oracle{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Oracle{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&Token{})
		db.Model(&Token{}).AddIndex("idx_tokens_application_id", "application_id")
		db.Model(&Token{}).AddIndex("idx_tokens_accessed_at", "accessed_at")
		db.Model(&Token{}).AddIndex("idx_tokens_network_id", "network_id")
		db.Model(&Token{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&Token{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
		db.Model(&Token{}).AddForeignKey("sale_contract_id", "contracts(id)", "SET NULL", "CASCADE")

		db.Exec("ALTER TABLE load_balancers_network_nodes ADD CONSTRAINT load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;")
		db.Exec("ALTER TABLE load_balancers_network_nodes ADD CONSTRAINT load_balancers_network_node_id_network_nodes_id_foreign FOREIGN KEY (network_node_id) REFERENCES network_nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;")
	})
}

func DatabaseConnection() *gorm.DB {
	return dbconf.DatabaseConnection()
}
