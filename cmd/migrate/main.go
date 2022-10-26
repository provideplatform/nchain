/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/provideplatform/nchain/common"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"

	dbconf "github.com/kthomas/go-db-config"
)

const initIfNotExistsRetryInterval = time.Millisecond * 2500
const initIfNotExistsTimeout = time.Second * 10

func main() {
	cfg := dbconf.GetDBConfig()

	err := initIfNotExists(
		cfg,
		os.Getenv("DATABASE_SUPERUSER"),
		os.Getenv("DATABASE_SUPERUSER_PASSWORD"),
	)
	if err != nil && !strings.Contains(err.Error(), "exists") { // HACK -- could be replaced with query
		common.Log.Warningf("migration failed; %s", err.Error())
		panic(err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s/%s?user=%s&password=%s&sslmode=%s",
		cfg.DatabaseHost,
		cfg.DatabaseName,
		url.QueryEscape(cfg.DatabaseUser),
		url.QueryEscape(cfg.DatabasePassword),
		cfg.DatabaseSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		common.Log.Warningf("migration failed; %s", err.Error())
		panic(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		common.Log.Warningf("migration failed; %s", err.Error())
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://./ops/migrations", cfg.DatabaseName, driver)
	if err != nil {
		common.Log.Warningf("migration failed; %s", err.Error())
		panic(err)
	}

	initialMigration := false
	_, _, versionErr := m.Version()
	if versionErr != nil {
		initialMigration = versionErr == migrate.ErrNilVersion
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		common.Log.Warningf("migration failed; %s", err.Error())
	}

	gormDB, _ := dbconf.DatabaseConnectionFactory(cfg)
	if initialMigration {
		defaultL1NetworksErr := initDefaultL1Networks(gormDB)
		if defaultL1NetworksErr != nil {
			common.Log.Warningf("default l1 networks not upserted in database %s; %s", cfg.DatabaseName, defaultL1NetworksErr.Error())
		}

		defaultL2NetworksErr := initDefaultL2Networks(gormDB)
		if defaultL2NetworksErr != nil {
			common.Log.Warningf("default l2 networks not upserted in database %s; %s", cfg.DatabaseName, defaultL2NetworksErr.Error())
		}

		defaultL3NetworksErr := initDefaultL3Networks(gormDB)
		if defaultL3NetworksErr != nil {
			common.Log.Warningf("default l3 networks not upserted in database %s; %s", cfg.DatabaseName, defaultL3NetworksErr.Error())
		}
	}

	enabledL1NetworksErr := setEnabledL1Networks(gormDB)
	if enabledL1NetworksErr != nil {
		common.Log.Warningf("failed to set enabled L1 networks in database %s; %s", cfg.DatabaseName, enabledL1NetworksErr.Error())
	}

	enabledL2NetworksErr := setEnabledL2Networks(gormDB)
	if enabledL2NetworksErr != nil {
		common.Log.Warningf("failed to set enabled L2 networks in database %s; %s", cfg.DatabaseName, enabledL2NetworksErr.Error())
	}

	enabledL3NetworksErr := setEnabledL3Networks(gormDB)
	if enabledL2NetworksErr != nil {
		common.Log.Warningf("failed to set enabled L3 networks in database %s; %s", cfg.DatabaseName, enabledL3NetworksErr.Error())
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
				common.Log.Debugf("migration failed; db connection not established; %s", err.Error())
			}

			if time.Now().Sub(startedAt) >= initIfNotExistsTimeout {
				ticker.Stop()
				common.Log.Panicf("migration failed; initIfNotExists timed out connecting to %s:%d", superuserCfg.DatabaseHost, superuserCfg.DatabasePort)
			}
		}

		if client != nil {
			defer client.Close()
			break
		}
	}

	if err != nil {
		common.Log.Warningf("migration failed on host: %s:%d; %s", superuserCfg.DatabaseHost, superuserCfg.DatabasePort, err.Error())
		return err
	}

	result := client.Exec(fmt.Sprintf("CREATE USER \"%s\" WITH SUPERUSER PASSWORD '%s'", cfg.DatabaseUser, cfg.DatabasePassword))
	err = result.Error
	if err != nil {
		common.Log.Debugf("failed to create db superuser during attempted migration: %s; %s; attempting without superuser privileges", cfg.DatabaseUser, err.Error())

		result = client.Exec(fmt.Sprintf("CREATE USER \"%s\" PASSWORD '%s'", cfg.DatabaseUser, cfg.DatabasePassword))
		err = result.Error
		if err != nil {
			common.Log.Warningf("migration failed; failed to create user: %s; %s", cfg.DatabaseUser, err.Error())
			return err
		}
	}

	if err == nil {
		result = client.Exec(fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\"", cfg.DatabaseName, cfg.DatabaseUser))
		err = result.Error
		if err != nil {
			common.Log.Warningf("migration failed; failed to create database %s using user %s; %s", cfg.DatabaseName, cfg.DatabaseUser, err.Error())
			return err
		}
	}

	return nil
}

func initDefaultL1Networks(db *gorm.DB) error {
	networkUpsertQueries := []string{
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('deca2436-21ba-4ff5-b225-ad1b0b2f5c59', now(), 'Ethereum mainnet', 'Ethereum mainnet', '1', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://etherscan.io\",\"chainspec_url\":\"https://gist.githubusercontent.com/kthomas/3ac2e29ee1b2fb22d501ae7b52884c24/raw/161c6a9de91db7044fb93852aed7b0fa0e78e55f/mainnet.chainspec.json\",\"is_ethereum_network\":true,\"json_rpc_url\":\"https://mainnet.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"native_currency\":\"ETH\",\"network_id\":1,\"websocket_url\":\"wss://mainnet.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"platform\":\"evm\",\"protocol_id\":\"pos\",\"engine_id\":\"ethash\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8050,8051,30300],\"udp\":[30300]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('1b16996e-3595-4985-816c-043345d22f8c', now(), 'Ethereum Görli Testnet', 'Ethereum Görli Testnet', '5', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://goerli.etherscan.io\",\"engine_id\":\"clique\",\"is_ethereum_network\":true,\"native_currency\":\"ETH\",\"network_id\":5,\"platform\":\"evm\",\"protocol_id\":\"poa\",\"websocket_url\":\"wss://goerli.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"json_rpc_url\":\"https://goerli.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}');",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config) VALUES ('ffc9a168-1327-4ddc-8af9-2301da82cccb', now(), 'Ethereum Sepolia Testnet', 'Ethereum Sepolia Testnet', '11155111', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://sepolia.etherscan.io\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"ETH\",\"network_id\":11155111,\"platform\":\"evm\",\"protocol_id\":\"pos\",\"websocket_url\":\"wss://sepolia.infura.io/ws/v3/fde5e81d5d3141a093def423db3eeb33\",\"json_rpc_url\":\"https://sepolia.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}');",

		// TODO... add the following networks
		// 1e00f6c7-71db-4942-b570-0894be6c5e9c - Starknet
		// 273f62fa-b7ee-4c4a-87b9-4a4c3769b52a - Starknet Görli
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

func initDefaultL2Networks(db *gorm.DB) error {
	networkUpsertQueries := []string{
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config, layer2) VALUES ('2fd61fde-5031-41f1-86b8-8a72e2945ead', now(), 'Polygon Mainnet', 'Polygon Mainnet', '137', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://polygonscan.com\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"MATIC\",\"network_id\":137,\"platform\":\"evm\",\"protocol_id\":\"pos\",\"websocket_url\":null,\"json_rpc_url\":\"https://polygon.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}', true);",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config, layer2) VALUES ('4251b6fd-c98d-4017-87a3-d691a77a52a7', now(), 'Polygon Mumbai Testnet', 'Polygon Mumbia Testnet', '80001', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://mumbai.polygonscan.com\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"TMATIC\",\"network_id\":80001,\"platform\":\"evm\",\"protocol_id\":\"pos\",\"websocket_url\":null,\"json_rpc_url\":\"https://polygon-mumbai.infura.io/v3/fde5e81d5d3141a093def423db3eeb33\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}', true);",

		// TODO... add the following L2 networks
		// b3d928b9-a1f4-4502-a9af-28d417e80435 Arbitrum
		// d4e050b6-3402-4707-8b07-e94dbd9851f8 Arbitrum Görli
		// 4ac304ad-a5ea-4063-be3c-6723cadf5f5c Optimism
		// ebf8af38-2270-4b54-b98c-206f2a63334e Optimism Görli
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

func initDefaultL3Networks(db *gorm.DB) error {
	networkUpsertQueries := []string{
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config, layer2) VALUES ('27795197-2a45-4a84-aed2-218a737d77f2', now(), 'PRVD Mainnet', 'PRVD Mainnet', '1337', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://explorer.provide.network\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"PRVG\",\"network_id\":137,\"platform\":\"evm\",\"protocol_id\":\"pos\",\"websocket_url\":null,\"json_rpc_url\":\"https://rpc.provide.network\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}', true);",
		"INSERT INTO networks (id, created_at, name, description, chain_id, is_production, cloneable, enabled, config, layer2) VALUES ('f6d2383b-8e0b-48d8-b539-cfdc13c7b970', now(), 'PRVD Peachtree Testnet', 'PRVD Peachtree Testnet', '1338', true, false, false, '{\"client\":\"geth\",\"block_explorer_url\":\"https://explorer.provide.network\",\"engine_id\":\"ethash\",\"is_ethereum_network\":true,\"native_currency\":\"PRVG\",\"network_id\":80001,\"platform\":\"evm\",\"protocol_id\":\"pos\",\"websocket_url\":null,\"json_rpc_url\":\"https://rpc.peachtree.provide.network\",\"security\":{\"egress\":\"*\",\"ingress\":{\"0.0.0.0/0\":{\"tcp\":[8545,8546,8547,30303],\"udp\":[30303]}}}}', true);",
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

func setEnabledL1Networks(db *gorm.DB) error {
	if os.Getenv("ENABLED_L1_NETWORK_IDS") != "" {
		network_ids := strings.Split(os.Getenv("ENABLED_L1_NETWORK_IDS"), ",")

		for _, id := range network_ids {
			result := db.Exec("UPDATE networks SET enabled = true where id = ? AND layer2 = false", id)
			err := result.Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setEnabledL2Networks(db *gorm.DB) error {
	if os.Getenv("ENABLED_L2_NETWORK_IDS") != "" {
		l2_network_ids := strings.Split(os.Getenv("ENABLED_L2_NETWORK_IDS"), ",")

		for _, id := range l2_network_ids {
			result := db.Exec("UPDATE networks SET enabled = true where id = ? AND layer2 = true", id)
			err := result.Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setEnabledL3Networks(db *gorm.DB) error {
	if os.Getenv("ENABLED_L3_NETWORK_IDS") != "" {
		l3_network_ids := strings.Split(os.Getenv("ENABLED_L3_NETWORK_IDS"), ",")

		for _, id := range l3_network_ids {
			result := db.Exec("UPDATE networks SET enabled = true where id = ? AND layer3 = true", id)
			err := result.Error
			if err != nil {
				return err
			}
		}
	}

	return nil
}
