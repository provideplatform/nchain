#!/bin/bash


pkill nats-streaming-server
pkill gnatsd 
pg_ctl -D /usr/local/var/postgres stop -s -m fast
