#!/bin/bash

#NATS_ROOT_CA_CERTIFICATES=/Users/kt/selfsigned-ca/ca.pem \
#NATS_TLS_CERTIFICATES='{"/Users/kt/selfsigned-ca/peer.key": "/Users/kt/selfsigned-ca/peer.crt"}' \

CONSUME_NATS_STREAMING_SUBSCRIPTIONS=false \
NATS_CLUSTER_ID=provide \
NATS_TOKEN=testtoken \
NATS_URL=nats://localhost:4221 \
NATS_STREAMING_URL=nats://localhost:4222 \
NATS_FORCE_TLS=false \
GIN_MODE=release \
DATABASE_LOGGING=false \
DATABASE_USER=goldmine \
DATABASE_PASSWORD=goldmine \
DATABASE_NAME=goldmine_sandbox \
DATABASE_HOST=localhost \
LOG_LEVEL=DEBUG \
SOLC_BIN=/usr/local/bin/solc \
./.bin/goldmine_api #dlv debug #./goldmine
