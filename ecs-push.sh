#!/usr/bin/env bash

$(aws ecr get-login --no-include-email --region us-east-1)
docker build --no-cache -t provide/goldmine .
docker tag provide/goldmine:latest 085843810865.dkr.ecr.us-east-1.amazonaws.com/provide/goldmine:latest
docker push 085843810865.dkr.ecr.us-east-1.amazonaws.com/provide/goldmine:latest
