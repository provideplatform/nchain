package dbconf

import (
	"sync"
	"os"
	"strconv"
)

type DbConfig struct {
	DatabaseName			string
	DatabaseHost			string
	DatabasePort			uint
	DatabaseUser			string
	DatabasePassword		string
	DatabaseSslMode			string
	DatabasePoolMaxIdleConnections	int
	DatabasePoolMaxOpenConnections	int
	DatabaseEnableLogging		bool
}

var configInstance *DbConfig
var configOnce sync.Once

func GetDbConfig() (*DbConfig) {
	configOnce.Do(func() {
		databaseName := os.Getenv("DATABASE_NAME")
		if databaseName == "" {
			databaseName = "dbconf_test"
		}

		databaseHost := os.Getenv("DATABASE_HOST")
		if databaseHost == "" {
			databaseHost = "localhost"
		}

		databasePort, _ := strconv.ParseUint(os.Getenv("DATABASE_PORT"), 10, 8)
		if databasePort == 0 {
			databasePort = 5432
		}

		databaseUser := os.Getenv("DATABASE_USER")
		if databaseUser == "" {
			databaseUser = "root"
		}

		databasePassword := os.Getenv("DATABASE_PASSWORD")
		if databasePassword == "" {
			databasePassword = "password"
		}

		databaseSslMode := os.Getenv("DATABASE_SSL_MODE")
		if databaseSslMode == "" {
			databaseSslMode = "disable"
		}

		databasePoolMaxIdleConnections, _ := strconv.ParseInt(os.Getenv("DATABASE_POOL_MAX_IDLE_CONNECTIONS"), 10, 8)
		if databasePoolMaxIdleConnections == 0 {
			databasePoolMaxIdleConnections = -1
		}

		databasePoolMaxOpenConnections, _ := strconv.ParseInt(os.Getenv("DATABASE_POOL_MAX_OPEN_CONNECTIONS"), 10, 8)

		configInstance = &DbConfig{
			DatabaseName: databaseName,
			DatabaseHost: databaseHost,
			DatabasePort: uint(databasePort),
			DatabaseUser: databaseUser,
			DatabasePassword: databasePassword,
			DatabaseSslMode: databaseSslMode,
			DatabasePoolMaxIdleConnections: int(databasePoolMaxIdleConnections),
			DatabasePoolMaxOpenConnections: int(databasePoolMaxOpenConnections),
			DatabaseEnableLogging: os.Getenv("DATABASE_LOGGING") == "true",
		}
	})
	return configInstance
}

var dbConf = GetDbConfig()
