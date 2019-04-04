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

REGEX_TEMPLATE=github.com/provideapp/goldmine

for c in $(find . -type d -name '*' | grep -v vendor | grep -v .git | grep -v scripts | grep -v bin | grep -v src | grep -v reports); do
echo $c

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
go test "$c" -v -timeout 30s -race
done
