.PHONY: build clean ecs_deploy install integration lint migrate mod run_local run_local_api run_local_consumer run_local_statsdaemon run_local_dependencies stop_local_dependencies stop_local test

clean:
	rm -rf ./.bin 2>/dev/null || true
	rm ./goldmine 2>/dev/null || true
	go fix ./...
	go clean -i ./...

build: clean mod
	go fmt ./...
	go build -v -o ./.bin/goldmine_api ./cmd/api
	go build -v -o ./.bin/goldmine_consumer ./cmd/consumer
	go build -v -o ./.bin/goldmine_migrate ./cmd/migrate
	go build -v -o ./.bin/goldmine_statsdaemon ./cmd/statsdaemon

ecs_deploy:
	./ops/ecs_deploy.sh

install: clean
	go install ./...

lint:
	./ops/lint.sh

migrate: mod
	rm -rf ./.bin/goldmine_migrate 2>/dev/null || true
	go build -v -o ./.bin/goldmine_migrate ./cmd/migrate
	./ops/migrate.sh

mod:
	go mod init 2>/dev/null || true
	go mod tidy
	go mod vendor 

run_local: build run_local_dependencies
	./ops/run_local.sh

run_local_api: build run_local_dependencies
	./ops/run_local_api.sh

run_local_consumer: build run_local_dependencies
	./ops/run_local_consumer.sh

run_local_statsdaemon: build run_local_dependencies
	./ops/run_local_statsdaemon.sh

run_local_dependencies:
	./ops/run_local_dependencies.sh

stop_local_dependencies:
	./ops/stop_local_dependencies.sh

stop_local:
	./ops/stop_local.sh

test: build
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_local_dependencies.sh
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_unit_tests.sh

integration: build
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_local_dependencies.sh
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 REDIS_SERVER_PORT=6380 ./ops/run_integration_tests.sh
