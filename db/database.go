package db

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // postgres requirement
	"github.com/kthomas/go-db-config"
)

func init() {
	db := dbconf.DatabaseConnection()

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";")

	db.Exec("ALTER TABLE load_balancers_network_nodes ADD CONSTRAINT load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;")
	db.Exec("ALTER TABLE load_balancers_network_nodes ADD CONSTRAINT load_balancers_network_node_id_network_nodes_id_foreign FOREIGN KEY (network_node_id) REFERENCES network_nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;")
}

// DatabaseConnection returns a configured, pooled connection
func DatabaseConnection() *gorm.DB {
	return dbconf.DatabaseConnection()
}
