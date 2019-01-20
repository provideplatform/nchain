package dbconf

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"fmt"
	"sync"
)

var dbInstance *gorm.DB
var dbOnce sync.Once

func DatabaseConnection() (*gorm.DB) {
	dbOnce.Do(func() {
		args := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s",
			dbConf.DatabaseHost,
			dbConf.DatabaseUser,
			dbConf.DatabasePassword,
			dbConf.DatabaseName,
			dbConf.DatabaseSslMode)

		db, err := gorm.Open("postgres", args)

		if err != nil {
			panic("Database connection failed")
		} else {
			db.LogMode(dbConf.DatabaseEnableLogging)

			db.DB().SetMaxOpenConns(dbConf.DatabasePoolMaxOpenConnections)

			if dbConf.DatabasePoolMaxIdleConnections >= 0 {
				db.DB().SetMaxIdleConns(dbConf.DatabasePoolMaxIdleConnections)
			}

			dbInstance = db
		}
	})
	return dbInstance
}
