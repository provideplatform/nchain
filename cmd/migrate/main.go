package main

import "github.com/provideapp/goldmine/db"

func main() {
	db.MigrateSchema()
	db.SeedNetworks()
}
