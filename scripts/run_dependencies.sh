#!/bin/bash

if hash psql 2>/dev/null
then
    echo 'Using' `psql --version`
else
    echo 'Installing postgresql'
    sudo apt-get update
    sudo apt-get -y install postgresql
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

gnatsd -auth testtoken -p 4221 &
nats-streaming-server -cid provide -auth testtoken -p 4222 &
pg_ctl -D /usr/local/var/postgres start &
