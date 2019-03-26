.PHONY: build clean flat test

clean:
	rm -rf .tmp/

build: clean flat
	pushd .tmp/
	go build .
	popd

flat: clean
	mkdir .tmp/

test:
	PGPASSWORD=goldmine dropdb -U goldmine goldmine_test || true >/dev/null
	PGPASSWORD=goldmine createdb -O goldmine -U goldmine goldmine_test || true >/dev/null
	PGPASSWORD=goldmine psql -Ugoldmine goldmine_test < db/networks_test.sql || true >/dev/null

	NATS_TOKEN=testtoken \
	NATS_URL=nats://localhost:4221 \
	NATS_STREAMING_URL=nats://localhost:4222 \
	NATS_CLUSTER_ID=provide \
	NATS_STREAMING_CONCURRENCY=1 \
	GIN_MODE=release \
	DATABASE_HOST=localhost \
	DATABASE_NAME=goldmine_test \
	DATABASE_USER=goldmine \
	DATABASE_PASSWORD=goldmine \
	LOG_LEVEL=DEBUG \
	go test -v -race -cover -timeout 30s -ginkgo.randomizeAllSpecs -ginkgo.progress -ginkgo.trace
