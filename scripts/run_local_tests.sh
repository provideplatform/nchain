#!/bin/bash

set -e
echo "" > coverage.txt

if [[ -z "${NATS_SERVER_PORT}" ]]; then
  NATS_SERVER_PORT=4221
fi

if [[ -z "${NATS_STREAMING_SERVER_PORT}" ]]; then
  NATS_STREAMING_SERVER_PORT=4222
fi

if [[ -z "${DATABASE_USER}" ]]; then
  DATABASE_USER=goldmine
fi

if [[ -z "${DATABASE_PASSWORD}" ]]; then
  DATABASE_PASSWORD=goldmine
fi

PGPASSWORD=${DATABASE_PASSWORD} dropdb -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} createdb -O ${DATABASE_USER} -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} psql -U ${DATABASE_USER} goldmine_test < db/networks_test.sql || true >/dev/null


# for d in $(go list ./... | grep -v vendor); do
echo $d
NATS_TOKEN=testtoken \
NATS_URL=nats://localhost:${NATS_SERVER_PORT} \
NATS_STREAMING_URL=nats://localhost:${NATS_STREAMING_SERVER_PORT} \
NATS_CLUSTER_ID=provide \
NATS_STREAMING_CONCURRENCY=1 \
GIN_MODE=release \
DATABASE_HOST=localhost \
DATABASE_NAME=goldmine_test \
DATABASE_USER=${DATABASE_USER} \
DATABASE_PASSWORD=${DATABASE_PASSWORD} \
LOG_LEVEL=DEBUG \
go test github.com/provideapp/goldmine/network -v -race -cover -timeout 30s -ginkgo.randomizeAllSpecs -ginkgo.progress -ginkgo.trace \
-coverprofile=profile.out -coverpkg=github.com/provideapp/goldmine/network
    # if [ -f profile.out ]; then
    #     cat profile.out >> coverage.txt
    #     rm profile.out
    # fi
# done

