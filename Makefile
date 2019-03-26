.PHONY: build clean run_dependencies test

clean:
	rm ./goldmine > /dev/null

build: clean
	go fmt
	go build .

run_local: build run_dependencies
	./scripts/run_local.sh

run_dependencies:
	./scripts/run_dependencies.sh

stop_dependencies:
	./scripts/stop_dependencies.sh

test: build run_dependencies
	./scripts/run_local_tests.sh
