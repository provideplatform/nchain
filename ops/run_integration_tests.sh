#!/bin/bash

docker-compose -f ./ops/docker-compose-integration.yml build --no-cache nchain
docker-compose -f ./ops/docker-compose-integration.yml up -d
TAGS=$LOCAL_TAGS ./ops/run_local_tests_short.sh
docker-compose -f ./ops/docker-compose-integration.yml logs
docker-compose -f ./ops/docker-compose-integration.yml down
docker volume rm ops_provide-db
