package db

import (
	"io/ioutil"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // PostgreSQL dialect
	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideapp/goldmine/common"
)

var (
	migrateOnce sync.Once
)

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
