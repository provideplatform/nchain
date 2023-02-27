# nchain

[![Go Report Card](https://goreportcard.com/badge/github.com/provideplatform/nchain)](https://goreportcard.com/report/github.com/provideplatform/nchain)

Microservice providing a low-code, chain-agnostic web3 API for building enterprise applications.

## Usage

See the [nchain API Reference](https://docs.provide.services/nchain).

## Run your own nchain with Docker

Requires [Docker](https://www.docker.com/get-started)

```shell
/ops/docker-compose up
```

## Build and run your own nchain from source

Requires [GNU Make](https://www.gnu.org/software/make), [Go](https://go.dev/doc/install), [Postgres](https://www.postgresql.org/download), [Redis](https://redis.io/docs/getting-started/installation)

```shell
make run_local
```

## Executables

The project comes with several wrappers/executables found in the `cmd`
directory.

|       Command        | Description                   |
|:--------------------:|-------------------------------|
|      **`api`**       | Runs the API server.          |
|      `consumer`      | Runs a consumer.              |
|      `migrate`       | Runs migrations.              |
| `reachabilitydaemon` | Runs the reachability daemon. |
|    `statsdaemon`     | Runs the stats daemon.        |
