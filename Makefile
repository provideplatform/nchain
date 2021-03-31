.PHONY: build clean ecs_deploy install integration lint migrate mod run_api run_consumer run_reachabilitydaemon run_statsdaemon run_local run_local_dependencies stop_local_dependencies stop_local test

clean:
	rm -rf ./.bin 2>/dev/null || true
	rm ./nchain 2>/dev/null || true
	go fix ./...
	go clean -i ./...

build: clean mod
	go fmt ./...
	go build -v -o ./.bin/nchain_api ./cmd/api
	go build -v -o ./.bin/nchain_consumer ./cmd/consumer
	go build -v -o ./.bin/nchain_migrate ./cmd/migrate
	go build -v -o ./.bin/nchain_reachabilitydaemon ./cmd/reachabilitydaemon
	go build -v -o ./.bin/nchain_statsdaemon ./cmd/statsdaemon

ecs_deploy:
	./ops/ecs_deploy.sh

install: clean
	go install ./...

lint:
	./ops/lint.sh

migrate: mod
	rm -rf ./.bin/nchain_migrate 2>/dev/null || true
	go build -v -o ./.bin/nchain_migrate ./cmd/migrate
	./ops/migrate.sh

mod:
	go mod init 2>/dev/null || true
	go mod tidy
	go mod vendor 

run_api: build run_local_dependencies
	./ops/run_api.sh

run_consumer: build run_local_dependencies
	./ops/run_consumer.sh

run_reachabilitydaemon: build run_local_dependencies
	./ops/run_reachabilitydaemon.sh

run_statsdaemon: build run_local_dependencies
	./ops/run_statsdaemon.sh

run_local: build run_local_dependencies
	./ops/run_local.sh

run_local_dependencies:
	./ops/run_local_dependencies.sh

stop_local_dependencies:
	./ops/stop_local_dependencies.sh

stop_local:
	./ops/stop_local.sh

test: build
	#NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_local_dependencies.sh
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_unit_tests.sh

# integration_ropsten:
# 	LOCAL_TAGS=ropsten NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests_long.sh

# integration_rinkeby:
# 	LOCAL_TAGS=rinkeby NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests_long.sh

# integration_kovan:
# 	LOCAL_TAGS=kovan NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests_long.sh

# integration_gorli:
# 	LOCAL_TAGS=gorli NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests_long.sh

integration_nchain_short:
	LOCAL_TAGS=nchain NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

integration_ropsten_short:
	LOCAL_TAGS=ropsten NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

integration_rinkeby_short:
	LOCAL_TAGS=rinkeby NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

integration_kovan_short:
	LOCAL_TAGS=kovan NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

integration_gorli_short:
	LOCAL_TAGS=gorli NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

integration_nobookie:
	LOCAL_TAGS=nobookie NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh

debug:
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=3224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests_debug.sh

nobookie_up:
	docker-compose -f ./ops/docker-compose-integration.yml up -d
	docker kill nchain
	docker kill nchain-consumer

nobookie_down:
	docker-compose -f ./ops/docker-compose-integration.yml down
	docker volume rm ops_provide-db

nobookie_bounce:
	docker-compose -f ./ops/docker-compose-integration.yml down
	docker volume rm ops_provide-db
	docker-compose -f ./ops/docker-compose-integration.yml up -d
	docker kill nchain
	docker kill nchain-consumer	

# statsdaemon_up:
# 	docker-compose -f ./ops/docker-compose-integration.yml up -d
# 	docker kill statsdaemon

statsdaemon_down:	
	docker-compose -f ./ops/docker-compose-integration.yml down
	docker volume rm ops_provide-db

statsdaemon_bounce:
	docker-compose -f ./ops/docker-compose-integration.yml down
	docker volume rm ops_provide-db
	docker-compose -f ./ops/docker-compose-integration.yml up -d
	docker kill statsdaemon
