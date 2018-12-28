# Contributing to Goldmine

### Architecture

Goldmine is the most core microservice within the Provide architecture. It is a glorified reverse proxy (among other things), similar to [Infura](https://infura.io) but with a few key differences:
* Support beyond the "vanilla EVM" (i.e., not just Parity & Geth EVM clients)
* Support for containerized orchestration of on-premise and cloud-based infrastructure (i.e., deploy global blockchain networks to your favorite datacenters)

In addition to serving as an ABI translation layer as described above, Goldmine has a few other components that run asynchronously to its [REST API](https://docs.provide.services/#goldmine):
* NATS streaming consumer: asynchronous consumer of persistent NATS streaming subscriptions. Upon successful processing of a NATS streaming message, goldmine consumers issue an `ACK`. We need to extend the NATS protocol to support `NACK`ing messages, and we need to implement dead-letter queues based on some business logic per-subject. The following subject subscriptions are created upon startup of each goldmine instance in accordance with the `NATS_CONCURRENCY` environment variable:
    - `api.usage.event`
    - `goldmine.contract.compiler-invocation`
    - `goldmine.tx`
    - `ml.filter.exec`
* Stats daemon: a per-network-per-instance goroutine and buffered channel that monitors (via websocket) or polls (via JSON-RPC) its configured network; calculates and caches stats about the network
* Exchange consumer: a per-instance goroutine that establishes a real-time socket connection to one or more exchanges to receive up-to-the-millsecond pricing details about a set of currency pairs. The data itself flows in via an AMQP connection which acts as a transceiver. *Note: this service is currently disabled as we are not yet utilizing real-time pricing within our infrastructure but will be useful and as such, has not been removed and should not be considered deprecated.*


Goldmine, like all of our microservices, relies heavily on the use of [goroutines](https://gobyexample.com/goroutines) and [buffered channels](https://gobyexample.com/channel-buffering).

---

![Goldmine Architecture](https://github.com/provideapp/goldmine/blob/dev/architecture.svg?raw=true)

---

See installation instructions below for how to get Goldmine running locally on your OS.

### Installation/Pre-Requisites

1. Read and follow [these instructions](https://github.com/provideapp/provide/blob/dev/CONTRIBUTING.md)

2. Create a PostgreSQL database for the goldmine service. The following should work:
    `createuser goldmine -s` ## WARNING: this gives goldmine superuser access to your PostgreSQL installation; this is probably OK for convenience sake on your localhost. If this is not acceptable you will need to manually run the `CREATE EXTENSION IF NOT EXISTS` SQL statements found near the top of `database.go` on your `goldmine_dev` db as the superuser.
    `createdb goldmine_dev -O goldmine`

3. Create a file called `run_local.sh` (`touch run_local.sh && chmod +x run_local.sh`) for convenience inside the `goldmine` path where you cloned the repoistory and populate it with the following script (again, for convenience):
    ```
    #!/bin/bash

    rm goldmine  # remove any previously-compiled binary
    go fmt
    go build .
    NATS_CLUSTER_ID=provide NATS_TOKEN=testtoken NATS_URL=nats://localhost:4221 NATS_STREAMING_URL=nats://localhost:4222 NATS_CONCURRENCY=2 DATABASE_NAME=goldmine_dev DATABASE_USER=goldmine DATABASE_PASSWORD=goldmine DATABASE_HOST=localhost AMQP_URL=amqp://ticker:ticker@localhost AMQP_EXCHANGE=ticker LOG_LEVEL=DEBUG ./goldmine
    ```
    
4. Load the seed networks: `psql goldmine_dev < db/networks.sql`

5. Make sure it works:

    ![Network UI](https://s3.amazonaws.com/provide.services/img/dev/goldmine-setup/001-ui-network-in-sync.png)
    ![Goldmine](https://s3.amazonaws.com/provide.services/img/dev/goldmine-setup/002-goldmine-network-in-sync.png)

### Contributing

#### Pull Request Process

Coming soon.
