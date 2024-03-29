#!/bin/bash

#
# Copyright 2017-2022 Provide Technologies Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

if hash psql 2>/dev/null
then
    echo 'Using' `psql --version`
else
    echo 'Installing postgresql'
    sudo apt-get update
    sudo apt-get -y install postgresql
fi

if hash psql 2>/dev/null
then
    echo 'Using' `redis-server --version`
else
    echo 'Installing redis-server'
    sudo apt-get update
    sudo apt-get -y install redis-server
fi

if hash gnatsd 2>/dev/null
then
    echo 'Using' `gnatsd --version`
else
    echo 'Installing NATS server'
    go get github.com/nats-io/gnatsd
fi

if hash nats-streaming-server 2>/dev/null
then
    echo 'Using' `nats-streaming-server --version`
else
    echo 'Installing NATS streaming server'
    go get github.com/nats-io/nats-streaming-server
fi

if [[ -z "${NATS_SERVER_PORT}" ]]; then
  NATS_SERVER_PORT=4221
fi

if [[ -z "${NATS_STREAMING_SERVER_PORT}" ]]; then
  NATS_STREAMING_SERVER_PORT=4222
fi

if [[ -z "${REDIS_SERVER_PORT}" ]]; then
  REDIS_SERVER_PORT=6379
fi

gnatsd -auth testtoken -p ${NATS_SERVER_PORT} > /dev/null 2>&1 &
nats-streaming-server -cid provide -auth testtoken -p ${NATS_STREAMING_SERVER_PORT} > /dev/null 2>&1 &
pg_ctl -D /usr/local/var/postgres start > /dev/null 2>&1 &
redis-server --port ${REDIS_SERVER_PORT} > /dev/null 2>&1 &
