package main

import (
	"database/sql"
	"fmt"

	"github.com/provideapp/goldmine/common"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"

	dbconf "github.com/kthomas/go-db-config"
)

func main() {
	cfg := dbconf.GetDBConfig()
	dsn := fmt.Sprintf("postgres://%s/%s?user=%s&password=%s&sslmode=%s",
		cfg.DatabaseHost,
		cfg.DatabaseName,
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		common.Log.Warningf("migrations failed: %s", err.Error())
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		common.Log.Warningf("migrations failed: %s", err.Error())
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://./scripts/migrations", cfg.DatabaseName, driver)
	if err != nil {
		common.Log.Warningf("migrations failed: %s", err.Error())
		panic(err)
	}

	_, _, err = m.Version()
	initial := err == migrate.ErrNilVersion

	err = m.Up()
	if err != nil && initial && err.Error() == "pq: relation \"schema_migrations\" does not exist in line 0: TRUNCATE \"schema_migrations\"" {
		// HACK initial migration issue
		_, err = db.Exec("UPDATE schema_migrations SET dirty = false WHERE version = 1")
	}

	if err != nil && err != migrate.ErrNoChange {
		common.Log.Warningf("migrations failed: %s", err.Error())
	}
}
