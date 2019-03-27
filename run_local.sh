#!/bin/bash

rm goldmine
#go fmt
go build .
#NATS_TOKEN=kfp0oOFp3k2W4m 
AWS_DEFAULT_VPC_ID=vpc-44fe6321 NATS_TOKEN=testtoken NATS_URL=nats://localhost:4221 NATS_STREAMING_URL=nats://localhost:4222 NATS_CLUSTER_ID=provide NATS_STREAMING_CONCURRENCY=2 GIN_MODE=release DATABASE_NAME=goldmine_sandbox DATABASE_USER=goldmine DATABASE_PASSWORD=goldmine DATABASE_HOST=localhost AMQP_URL=amqp://ticker:ticker@10.0.0.126 AMQP_EXCHANGE=ticker LOG_LEVEL=DEBUG ./goldmine
