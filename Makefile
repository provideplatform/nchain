.PHONY: build clean run_dependencies run_local stop_dependencies test

clean:
	rm ./goldmine || true > /dev/null

build: clean
	go fmt
	go build .

run_local: build run_dependencies
	./scripts/run_local.sh

run_dependencies:
	./scripts/run_dependencies.sh

stop_dependencies:
	./scripts/stop_dependencies.sh

test: build
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 ./scripts/run_dependencies.sh
	NATS_SERVER_PORT=4223 NATS_STREAMING_SERVER_PORT=4224 ./scripts/run_local_tests.sh
