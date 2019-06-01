#!/bin/bash

TAGS=unit RACE=true ./scripts/run_local_tests.sh
TAGS=integration RACE=true ./scripts/run_local_tests.sh
