#!/bin/bash

./ops/run_local_api.sh &
./ops/run_local_consumer.sh &
./ops/run_local_statsdaemon.sh &
