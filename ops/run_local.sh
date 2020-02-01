#!/bin/bash

./ops/run_api.sh &
./ops/run_consumer.sh &
./ops/run_reachabilitydaemon.sh &
./ops/run_statsdaemon.sh &
