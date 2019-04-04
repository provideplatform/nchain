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

PGPASSWORD=${DATABASE_PASSWORD} dropdb -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} createdb -O ${DATABASE_USER} -U ${DATABASE_USER} goldmine_test || true >/dev/null
PGPASSWORD=${DATABASE_PASSWORD} psql -U ${DATABASE_USER} goldmine_test < db/networks_test.sql || true >/dev/null

REGEX_TEMPLATE=github.com/provideapp/goldmine

#for d in $(go list ./... | grep -v vendor); do
# for  c in $(find . -type d -name '*' | grep -v vendor | grep -v .git | grep -v scripts | grep -v bin | grep -v src | grep -v reports); do
# echo $d
# c=$(echo "$d" | sed 's+.*/goldmine+\.+g')
c="./network"
echo $c
# for f in $c/*_test.go
# do

# array=(unit integration)
# for t in "${array[@]}"
# do


# f="$c/oracle_test.go"


# AWS_RUN=0
# if [ "$f" = "./network/node_test.go" ]; then
#   AWS_RUN=1
# fi
# echo $AWS_RUN

# TAGS="unit"
# if [ "$f" = "./oracle/oracle_test.go" ]; then
#   TAGS="integration"
# fi

# echo $f
# if [[ $f =~ /integration/ ]]; then
#   TAGS="integration"
# else
#   TAGS="unit"
# fi

echo $TAGS

# AWS_RUN=${AWS_RUN} \
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
go test "$c" -v -race -timeout 1800s -cover  -ginkgo.progress -ginkgo.trace -coverprofile=profile.out -coverpkg="$c" -tags="$TAGS"

