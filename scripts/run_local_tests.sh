#!/bin/bash

DB_NAME=goldmine_test
PGPASSWORD=goldmine dropdb -U goldmine goldmine_test
PGPASSWORD=goldmine createdb -O goldmine -U goldmine goldmine_test 
PGPASSWORD=goldmine psql -Ugoldmine goldmine_test < db/networks_test.sql

rm goldmine > /dev/null
go build .
NATS_TOKEN=testtoken NATS_URL=nats://localhost:4221 NATS_STREAMING_URL=nats://localhost:4222 NATS_CLUSTER_ID=provide NATS_STREAMING_CONCURRENCY=1 GIN_MODE=release DATABASE_NAME=${DB_NAME} DATABASE_USER=goldmine DATABASE_PASSWORD=goldmine DATABASE_HOST=localhost AMQP_URL=amqp://ticker:ticker@10.0.0.126 AMQP_EXCHANGE=ticker LOG_LEVEL=DEBUG /usr/local/bin/go test -v -race -cover -timeout 30s -ginkgo.randomizeAllSpecs -ginkgo.progress -ginkgo.trace
