package db

import (
	"io/ioutil"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // PostgreSQL dialect
	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideapp/goldmine/bridge"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/connector"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/filter"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/oracle"
	"github.com/provideapp/goldmine/token"
	"github.com/provideapp/goldmine/tx"
	"github.com/provideapp/goldmine/wallet"
)

var (
	migrateOnce sync.Once
)

// MigrateSchema migrates the database schema from scratch
func MigrateSchema() {
	migrateOnce.Do(func() {
		db := dbconf.DatabaseConnection()

		db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
		db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

		db.AutoMigrate(&network.Network{})
		db.Model(&network.Network{}).AddIndex("idx_networks_application_id", "application_id")
		db.Model(&network.Network{}).AddIndex("idx_networks_network_id", "network_id")
		db.Model(&network.Network{}).AddIndex("idx_networks_user_id", "user_id")
		db.Model(&network.Network{}).AddIndex("idx_networks_cloneable", "cloneable")
		db.Model(&network.Network{}).AddIndex("idx_networks_enabled", "enabled")
		db.Model(&network.Network{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&network.Network{}).AddForeignKey("sidechain_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&network.Network{}).AddUniqueIndex("idx_chain_id", "chain_id")

		db.AutoMigrate(&network.Node{})
		db.Model(&network.Node{}).AddIndex("idx_nodes_network_id", "network_id")
		db.Model(&network.Node{}).AddIndex("idx_nodes_user_id", "user_id")
		db.Model(&network.Node{}).AddIndex("idx_nodes_application_id", "application_id")
		db.Model(&network.Node{}).AddIndex("idx_nodes_role", "role")
		db.Model(&network.Node{}).AddIndex("idx_nodes_status", "status")
		db.Model(&network.Node{}).AddIndex("idx_nodes_bootnode", "bootnode")
		db.Model(&network.Node{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&network.LoadBalancer{})
		db.Model(&network.LoadBalancer{}).AddIndex("idx_load_balancers_network_id", "network_id")
		db.Model(&network.LoadBalancer{}).AddIndex("idx_load_balancers_region", "region")
		db.Model(&network.LoadBalancer{}).AddIndex("idx_load_balancers_status", "status")
		db.Model(&network.LoadBalancer{}).AddIndex("idx_load_balancers_type", "type")
		db.Model(&network.LoadBalancer{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&wallet.Wallet{})
		db.Model(&wallet.Wallet{}).AddIndex("idx_wallets_application_id", "application_id")
		db.Model(&wallet.Wallet{}).AddIndex("idx_wallets_user_id", "user_id")
		db.Model(&wallet.Wallet{}).AddIndex("idx_wallets_accessed_at", "accessed_at")
		db.Model(&wallet.Wallet{}).AddIndex("idx_wallets_network_id", "network_id")
		db.Model(&wallet.Wallet{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&tx.Transaction{})
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_application_id", "application_id")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_created_at", "created_at")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_status", "status")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_network_id", "network_id")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_user_id", "user_id")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_wallet_id", "wallet_id")
		db.Model(&tx.Transaction{}).AddIndex("idx_transactions_ref", "ref")
		db.Model(&tx.Transaction{}).AddUniqueIndex("idx_transactions_hash", "hash")
		db.Model(&tx.Transaction{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&tx.Transaction{}).AddForeignKey("wallet_id", "wallets(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&contract.Contract{})
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_application_id", "application_id")
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_accessed_at", "accessed_at")
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_address", "address")
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_contract_id", "contract_id")
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_network_id", "network_id")
		db.Model(&contract.Contract{}).AddIndex("idx_contracts_transaction_id", "transaction_id")
		db.Model(&contract.Contract{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
		db.Model(&contract.Contract{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&contract.Contract{}).AddForeignKey("transaction_id", "transactions(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&bridge.Bridge{})
		db.Model(&bridge.Bridge{}).AddIndex("idx_bridges_application_id", "application_id")
		db.Model(&bridge.Bridge{}).AddIndex("idx_bridges_network_id", "network_id")
		db.Model(&bridge.Bridge{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&connector.Connector{})
		db.Model(&connector.Connector{}).AddIndex("idx_connectors_application_id", "application_id")
		db.Model(&connector.Connector{}).AddIndex("idx_connectors_network_id", "network_id")
		db.Model(&connector.Connector{}).AddIndex("idx_connectors_type", "type")
		db.Model(&connector.Connector{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&filter.Filter{})
		db.Model(&filter.Filter{}).AddIndex("idx_filters_application_id", "application_id")
		db.Model(&filter.Filter{}).AddIndex("idx_filters_network_id", "network_id")
		db.Model(&filter.Filter{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&oracle.Oracle{})
		db.Model(&oracle.Oracle{}).AddIndex("idx_oracles_application_id", "application_id")
		db.Model(&oracle.Oracle{}).AddIndex("idx_oracles_network_id", "network_id")
		db.Model(&oracle.Oracle{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&oracle.Oracle{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")

		db.AutoMigrate(&token.Token{})
		db.Model(&token.Token{}).AddIndex("idx_tokens_application_id", "application_id")
		db.Model(&token.Token{}).AddIndex("idx_tokens_accessed_at", "accessed_at")
		db.Model(&token.Token{}).AddIndex("idx_tokens_network_id", "network_id")
		db.Model(&token.Token{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
		db.Model(&token.Token{}).AddForeignKey("contract_id", "contracts(id)", "SET NULL", "CASCADE")
		db.Model(&token.Token{}).AddForeignKey("sale_contract_id", "contracts(id)", "SET NULL", "CASCADE")

		db.Exec("ALTER TABLE load_balancers_nodes ADD CONSTRAINT load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;")
		db.Exec("ALTER TABLE load_balancers_nodes ADD CONSTRAINT load_balancers_node_id_nodes_id_foreign FOREIGN KEY (node_id) REFERENCES nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;")

		db.Exec("ALTER TABLE connectors_load_balancers ADD CONSTRAINT connectors_load_balancers_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;")
		db.Exec("ALTER TABLE connectors_load_balancers ADD CONSTRAINT connectors_load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;")

		db.Exec("ALTER TABLE connectors_nodes ADD CONSTRAINT connectors_nodes_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;")
		db.Exec("ALTER TABLE connectors_nodes ADD CONSTRAINT connectors_nodes_node_id_nodes_id_foreign FOREIGN KEY (node_id) REFERENCES nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;")
	})
}

// SeedNetworks inserts the network seeds from networks.sql
func SeedNetworks() {
	rawsql, err := ioutil.ReadFile("./db/networks.sql")
	common.Log.PanicOnError(err, "Failed to seed networks")

	db := dbconf.DatabaseConnection()
	lines := strings.Split(string(rawsql), "\n")
	for _, sqlcmd := range lines {
		db.Exec(sqlcmd)
	}

	common.Log.Debugf("Migrated seed networks")
}

// DatabaseConnection returns a pooled DB connection
func DatabaseConnection() *gorm.DB {
	return dbconf.DatabaseConnection()
}
