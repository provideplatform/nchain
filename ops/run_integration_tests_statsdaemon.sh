#!/bin/bash

docker volume prune -f
docker build -t nchain/local --no-cache .
docker-compose -f ./ops/docker-compose.yml up -d
TAGS=$LOCAL_TAGS ./ops/run_local_tests_statsdaemon.sh
docker-compose -f ./ops/docker-compose.yml down
docker volume rm ops_provide-db
