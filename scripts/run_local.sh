#!/bin/bash

NATS_STREAMING_CONCURRENCY=1 \
NATS_CLUSTER_ID=provide \
NATS_TOKEN=testtoken \
NATS_URL=nats://localhost:4221 \
NATS_STREAMING_URL=nats://localhost:4222 \
GIN_MODE=release \
DATABASE_LOGGING=false \
DATABASE_USER=goldmine \
DATABASE_PASSWORD=goldmine \
DATABASE_NAME=goldmine \
DATABASE_HOST=localhost \
LOG_LEVEL=DEBUG \
./goldmine #dlv debug #./goldmine