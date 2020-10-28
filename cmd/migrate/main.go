package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/provideapp/nchain/common"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"

	dbconf "github.com/kthomas/go-db-config"
)

const initIfNotExistsRetryInterval = time.Second * 5
const initIfNotExistsTimeout = time.Second * 30

func main() {
	cfg := dbconf.GetDBConfig()

	err := initIfNotExists(
		cfg,
		os.Getenv("DATABASE_SUPERUSER"),
		os.Getenv("DATABASE_SUPERUSER_PASSWORD"),
	)
	if err != nil && !strings.Contains(err.Error(), "exists") { // HACK -- could be replaced with query
		common.Log.Warningf("migrations failed; %s", err.Error())
		// panic(err) // HACK!
	}

	dsn := fmt.Sprintf(
		"postgres://%s/%s?user=%s&password=%s&sslmode=%s",
		cfg.DatabaseHost,
		cfg.DatabaseName,
		cfg.DatabaseUser,
		url.QueryEscape(cfg.DatabasePassword),
		cfg.DatabaseSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		common.Log.Warningf("migrations failed 1: %s", err.Error())
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		common.Log.Warningf("migrations failed 2; %s", err.Error())
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://./ops/migrations", cfg.DatabaseName, driver)
	if err != nil {
		common.Log.Warningf("migrations failed 3: %s", err.Error())
		panic(err)
	}

	initialMigration := false
	_, _, versionErr := m.Version()
	if versionErr != nil {
		initialMigration = versionErr == migrate.ErrNilVersion
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		common.Log.Warningf("migrations failed 4: %s", err.Error())
	}

	if initialMigration {
		db, _ := dbconf.DatabaseConnectionFactory(cfg)
		defaultNetworksErr := initDefaultNetworks(db)
		if defaultNetworksErr != nil {
			common.Log.Warningf("default networks not upserted in database %s; %s", cfg.DatabaseName, defaultNetworksErr.Error())
		}
	}
}

func initIfNotExists(cfg *dbconf.DBConfig, superuser, password string) error {
	if superuser == "" || password == "" {
		return nil
	}

	superuserCfg := &dbconf.DBConfig{
		DatabaseName:     superuser,
		DatabaseHost:     cfg.DatabaseHost,
		DatabasePort:     cfg.DatabasePort,
		DatabaseUser:     superuser,
		DatabasePassword: password,
		DatabaseSSLMode:  cfg.DatabaseSSLMode,
	}

	var client *gorm.DB
	var err error

	ticker := time.NewTicker(initIfNotExistsRetryInterval)
	startedAt := time.Now()
	for {
		select {
		case <-ticker.C:
			client, err = dbconf.DatabaseConnectionFactory(superuserCfg)
			if err == nil {
				ticker.Stop()
				break
			} else {
				common.Log.Debugf("migrations db connection not established; %s", err.Error())
			}

			if time.Now().Sub(startedAt) >= initIfNotExistsTimeout {
				ticker.Stop()
				panic(fmt.Sprintf("migrations failed; initIfNotExists timed out connecting to %s:%d", superuserCfg.DatabaseHost, superuserCfg.DatabasePort))
			}
		}

		if client != nil {
			defer client.Close()
			break
		}
	}

	if err != nil {
		common.Log.Warningf("migrations failed 5: %s :debug: host name %s, port name %d", err.Error(), superuserCfg.DatabaseHost, superuserCfg.DatabasePort)
		return err
	}

	result := client.Exec(fmt.Sprintf("CREATE USER %s WITH SUPERUSER PASSWORD '%s'", cfg.DatabaseUser, cfg.DatabasePassword))
	if err != nil {
		common.Log.Warningf("migrations failed; failed to create user: %s; %s", err.Error(), cfg.DatabaseUser)
		return err
	}

	if err == nil {
		result = client.Exec(fmt.Sprintf("CREATE DATABASE %s OWNER %s", cfg.DatabaseName, cfg.DatabaseUser))
		err = result.Error
		if err != nil {
			common.Log.Warningf("migrations failed; failed to create database %s using user %s; %s", cfg.DatabaseName, cfg.DatabaseUser, err.Error())
			return err
		}
	}

	return nil
}

func initDefaultNetworks(db *gorm.DB) error {
	networkUpsertQueries := []string{
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('deca2436-21ba-4ff5-b225-ad1b0b2f5c59', now(), 'Ethereum mainnet', 'Ethereum mainnet', '1', true, false, false, '{\"block_explorer_url\":\"https://etherscan.io\",\"chainspec_url\":\"https://gist.githubusercontent.com/kthomas/3ac2e29ee1b2fb22d501ae7b52884c24/raw/161c6a9de91db7044fb93852aed7b0fa0e78e55f/mainnet.chainspec.json\",\"is_ethereum_network\":true,\"json_rpc_url\":\"https://mainnet.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"native_currency\":\"ETH\",\"network_id\":1,\"websocket_url\":\"wss://mainnet.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"platform\":\"evm\",\"protocol_id\":\"pow\",\"engine_id\":\"ethash\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8050,8051,30300],\"udp\":[30300]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('07102258-5e49-480e-86af-6d0c3260827d', now(), 'Ethereum Rinkeby testnet', 'Ethereum Rinkeby testnet', '4', true, false, false, '{\"block_explorer_url\":\"https://rinkeby.etherscan.io\",\"is_ethereum_network\":true,\"json_rpc_url\":\"https://rinkeby.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"native_currency\":\"ETH\",\"network_id\":4,\"websocket_url\":\"wss://rinkeby.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"platform\":\"evm\",\"protocol_id\":\"pow\",\"engine_id\":\"ethash\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8050,8051,30300],\"udp\":[30300]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('66d44f30-9092-4182-a3c4-bc02736d6ae5', now(), 'Ethereum Ropsten testnet', 'Ethereum Ropsten testnet', '3', true, false, false, '{\"block_explorer_url\":\"https://ropsten.etherscan.io\",\"client\":\"geth\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"ETH\",\"network_id\":3,\"platform\":\"evm\",\"protocol_id\":\"pow\",\"websocket_url\":\"wss://ropsten.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"json_rpc_url\":\"https://ropsten.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('8d31bf48-df6b-4a71-9d7c-3cb291111e27', now(), 'Ethereum Kovan testnet', 'Ethereum Kovan testnet', '42', true, false, false, '{\"block_explorer_url\":\"https://kovan.etherscan.io\",\"is_ethereum_network\":true,\"json_rpc_url\":\"https://kovan.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"native_currency\":\"KETH\",\"network_id\":42,\"websocket_url\":\"wss://kovan.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"platform\":\"evm\",\"client\":\"parity\",\"protocol_id\":\"poa\",\"engine_id\":\"aura\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8050,8051,30300],\"udp\":[30300]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('1b16996e-3595-4985-816c-043345d22f8c', now(), 'Ethereum Görli Testnet', 'Ethereum Görli Testnet', '5', true, false, false, '{\"block_explorer_url\":\"https://goerli.etherscan.io\",\"engine_id\":\"clique\",\"is_ethereum_network\":true,\"native_currency\":\"ETH\",\"network_id\":5,\"platform\":\"evm\",\"protocol_id\":\"poa\",\"websocket_url\":\"wss://goerli.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"json_rpc_url\":\"https://goerli.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}');",
	}
	for _, raw := range networkUpsertQueries {
		result := db.Exec(raw)
		err := result.Error
		if err != nil {
			return err
		}
	}

	return nil
}
