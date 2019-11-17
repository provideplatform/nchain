#!/bin/bash

./scripts/run_local_api.sh &
./scripts/run_local_consumer.sh &
./scripts/run_local_statsdaemon.sh &
