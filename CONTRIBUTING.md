# Contributing to NChain

### Architecture

NChain is the most core microservice within the Provide architecture. It is a glorified reverse proxy (among other things), similar to [Infura](https://infura.io) but with a few key differences:
* Support beyond the "vanilla EVM" (i.e., not just Parity & Geth EVM clients)
* Support for containerized orchestration of on-premise and cloud-based infrastructure (i.e., deploy global blockchain networks to your favorite datacenters)

In addition to serving as an ABI translation layer as described above, NChain has a few other components that run asynchronously to its [REST API](https://docs.provide.services/#nchain):
* NATS streaming consumer: asynchronous consumer of persistent NATS streaming subscriptions. Upon successful processing of a NATS streaming message, nchain consumers issue an `ACK`. We need to extend the NATS protocol to support `NACK`ing messages, and we need to implement dead-letter queues based on some business logic per-subject. The following subject subscriptions are created upon startup of each nchain instance in accordance with the `NATS_CONCURRENCY` environment variable:
    - `api.usage.event`
    - `nchain.contract.compiler-invocation`
    - `nchain.tx`
    - `ml.filter.exec`
* Stats daemon: a per-network-per-instance goroutine and buffered channel that monitors (via websocket) or polls (via JSON-RPC) its configured network; calculates and caches stats about the network
* Exchange consumer: a per-instance goroutine that establishes a real-time socket connection to one or more exchanges to receive up-to-the-millsecond pricing details about a set of currency pairs. The data itself flows in via an AMQP connection which acts as a transceiver. *Note: this service is currently disabled as we are not yet utilizing real-time pricing within our infrastructure but will be useful and as such, has not been removed and should not be considered deprecated.*


NChain, like all of our microservices, relies heavily on the use of [goroutines](https://gobyexample.com/goroutines) and [buffered channels](https://gobyexample.com/channel-buffering).

---

![NChain Architecture](https://github.com/provideplatform/nchain/blob/dev/architecture.svg)

---

See installation instructions below for how to get NChain running locally on your OS.

### Installation/Pre-Requisites

1. Read and follow [these instructions](https://github.com/provideplatform/provide/blob/dev/CONTRIBUTING.md)

2. Create a PostgreSQL database for the nchain service. The following should work:
    `createuser nchain -s` ## WARNING: this gives nchain superuser access to your PostgreSQL installation; this is probably OK for convenience sake on your localhost. If this is not acceptable you will need to manually run the `CREATE EXTENSION IF NOT EXISTS` SQL statements found near the top of `database.go` on your `nchain_dev` db as the superuser.
    `createdb nchain_dev -O nchain`

3. Create a file called `run_local.sh` (`touch run_local.sh && chmod +x run_local.sh`) for convenience inside the `nchain` path where you cloned the repoistory and populate it with the following script (again, for convenience):
    ```
    #!/bin/bash

    rm nchain  # remove any previously-compiled binary
    go fmt
    go build .
    NATS_CLUSTER_ID=provide NATS_TOKEN=testtoken NATS_URL=nats://localhost:4221 NATS_STREAMING_URL=nats://localhost:4222 NATS_CONCURRENCY=2 DATABASE_NAME=nchain_dev DATABASE_USER=nchain DATABASE_PASSWORD=nchain DATABASE_HOST=localhost AMQP_URL=amqp://ticker:ticker@localhost AMQP_EXCHANGE=ticker LOG_LEVEL=DEBUG ./nchain
    ```
    
4. Load the seed networks: `psql nchain_dev < db/networks.sql`

5. Make sure it works:

    ![Network UI](https://s3.amazonaws.com/provide.services/img/dev/nchain-setup/001-ui-network-in-sync.png)
    ![NChain](https://s3.amazonaws.com/provide.services/img/dev/nchain-setup/002-nchain-network-in-sync.png)

### Contributing

#### Pull Request Process

Coming soon.
