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

package db

import (
	"io/ioutil"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" // PostgreSQL dialect
	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideplatform/nchain/common"
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
