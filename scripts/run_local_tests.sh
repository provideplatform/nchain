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

if [[ -z "${TAGS}" ]]; then
  TAGS=unit
fi

if [[ -z "${RACE}" ]]; then
  RACE=false
fi

PGPASSWORD=${DATABASE_PASSWORD} dropdb -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} createdb -O ${DATABASE_USER} -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} psql -U ${DATABASE_USER} goldmine_test < db/networks_test.sql || true >/dev/null

pkgs=(bridge common connector consumer contract db filter network oracle prices token tx wallet)
for d in "${pkgs[@]}" ; do
  pkg=$(echo $d | sed 's/\/*$//g')
  
  if [ "$RACE" = "true" ]; then
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
    TEST_AWS_ACCESS_KEY_ID=${TEST_AWS_ACCESS_KEY_ID} \
    TEST_AWS_SECRET_ACCESS_KEY=${TEST_AWS_SECRET_ACCESS_KEY} \
    go test "./${pkg}" -v \
                       -race \
                       -timeout 1800s \
                       -cover \
                       -coverpkg="./${pkg}" \
                       -coverprofile=profile.out \
                       -ginkgo.progress \
                       -ginkgo.trace \
                       -tags="$TAGS"
  else
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
    TEST_AWS_ACCESS_KEY_ID=${TEST_AWS_ACCESS_KEY_ID} \
    TEST_AWS_SECRET_ACCESS_KEY=${TEST_AWS_SECRET_ACCESS_KEY} \
    go test "./${pkg}" -v \
                       -timeout 1800s \
                       -cover \
                       -coverpkg="./${pkg}" \
                       -coverprofile=profile.out \
                       -ginkgo.progress \
                       -ginkgo.trace \
                       -tags="$TAGS"
  fi
done
