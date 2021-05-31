#!/bin/bash

docker-compose -f ./ops/docker-compose.yml build --no-cache nchain
docker-compose -f ./ops/docker-compose.yml up -d
TAGS=$LOCAL_TAGS ./ops/run_local_tests_debug.sh
# docker-compose -f ./ops/docker-compose.yml logs
#docker-compose -f ./ops/docker-compose-debug.yml down
#docker volume rm ops_provide-db
