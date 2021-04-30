#!/bin/bash

docker-compose -f ./ops/docker-compose.yml build --no-cache nchain-consumer
docker-compose -f ./ops/docker-compose.yml up -d
TAGS=$LOCAL_TAGS ./ops/run_local_tests_short.sh
docker-compose -f ./ops/docker-compose.yml down
docker volume rm ops_provide-db
