#!/bin/bash


nats-streaming-server -cid provide -auth testtoken -p 4222 &
gnatsd -auth testtoken -p 4221 &
pg_ctl -D /usr/local/var/postgres start &
