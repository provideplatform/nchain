# goldmine

API for building best-of-breed applications which leverage a public blockchain.

## Authentication

Consumers of this API will present a `bearer` authentication header (i.e., using a `JWT` token) for all requests. The mechanism to require this authentication has not yet been included in the codebase to simplify development and integration, but it will be included in the coming weeks; this section will be updated with specifics when authentication is required.

## Authorization

The `bearer` authorization header will be scoped to an authorized application. The `bearer` authorization header may contain a `sub` (see [RFC-7519 §§ 4.1.2](https://tools.ietf.org/html/rfc7519#section-4.1.2)) to further limit its authorized scope to a specific token or smart contract, wallet or other entity.

Certain APIs will be metered similarly to how AWS meters some of its webservices. Production applications will need a sufficient PRVD token balance to consume metered APIs (based on market conditions at the time of consumption, some quantity of PRVD tokens will be burned as a result of such metered API usage. *The PRVD token model and economics are in the process of being peer-reviewed and finalized; the documentation will be updated accordingly with specifics.*

---

The following APIs are exposed:

## Networks API

### `GET /api/v1/networks`

Enumerate available blockchain networks and related configuration details.

```console
$ curl -v https://goldmine.provide.services/api/v1/networks

> GET /api/v1/networks HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 28 Dec 2017 16:54:12 GMT
< Content-Type: application/json; charset=UTF-8
< Transfer-Encoding: chunked
< Connection: keep-alive
<
[
    {
        "id": "b696c1f0-8c5b-4c0e-8400-da9d2046be15",
        "created_at": "2018-01-13T22:00:47.871841Z",
        "application_id": null,
        "name": "Bitcoin",
        "description": "Bitcoin mainnet",
        "is_production": true,
        "sidechain_id": "2d86e27b-36ae-4d19-b647-0301f275846a",
        "config": null
    },
    ...
    {
        "id": "9f7a08cb-4d8d-469d-a53a-f39fde5ece41",
        "created_at": "2018-02-07T22:42:13.897144Z",
        "application_id": null,
        "name": "POA Network Sokol",
        "description": "Oracles Proof-of-Authority Sokol (testnet)",
        "is_production": false,
        "sidechain_id": null,
        "config": {
            "json_rpc_url": "https://sokol.poa.network",
            "parity_json_rpc_url": "https://sokol.poa.network",
            "network_id": 77
        }
    }
]
```

### `GET /api/v1/networks/:id`

### `GET /api/v1/networks/:id/addresses`

### `GET /api/v1/networks/:id/blocks`

### `GET /api/v1/networks/:id/contracts`

### `GET /api/v1/networks/:id/transactions`

## Prices API

### `GET /api/v1/prices`

Fetch real-time pricing data for major currency pairs and supported tokens.

```console
$ curl -v https://goldmine.provide.services/api/v1/prices

> GET /api/v1/prices HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 28 Dec 2017 17:03:42 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 88
< Connection: keep-alive
<
{
    "btcusd": 14105.2,
    "ethusd": 707,
    "ltcusd": 244.48,
    "prvdusd": 0.22
}
```

## Contracts API

### `GET /api/v1/contracts`

Enumerate managed smart contracts. Each `Contract` contains a `params` object which includes `Network`-specific descriptors. `Token` contracts can be filtered from the response by passing a query string with the `filter_tokens` parameter set to `true`.

```console
$ curl -v https://goldmine.provide.services/api/v1/contracts

> GET /api/v1/contracts HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Mon, 01 Jan 2018 20:27:18 GMT
< Content-Type: application/json; charset=UTF-8
< Transfer-Encoding: chunked
< Connection: keep-alive
<
[
    {
        "id": "76e6f407-735e-46e4-8281-390f770b2717",
        "created_at": "2018-01-01T19:29:38.845343Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "transaction_id": "e4a09a42-c584-4b7f-981e-c72f337b672b",
        "name": "0x0079F773651C8B7bAEAf4461C81A1494639F830F",
        "address": "0x0079F773651C8B7bAEAf4461C81A1494639F830F",
        "params": null
    },
    {
        "id": "98e1958a-aab6-45c4-ac06-45c8eaa66b57",
        "created_at": "2018-01-01T19:35:44.313535Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "transaction_id": "e050ba50-6cb2-4107-a4df-23be0782f864",
        "name": "0x7B3e4D37F4d38ec772ec0593294c269425807e8E",
        "address": "0x7B3e4D37F4d38ec772ec0593294c269425807e8E",
        "params": null
    },
    {
        "id": "fcb1bd1e-2640-46ff-a48a-e45328ccdc4f",
        "created_at": "2018-01-01T20:20:07.252249Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "transaction_id": "a1e55081-52d3-452b-bc24-fd4030317ac5",
        "name": "ProvideToken",
        "address": "0x4D4734f4bc0A602bBC9833A26E680ccC87B012F6",
        "params": {
            "name": "ProvideToken",
            "abi": [
                {
                    "constant": true,
                    "inputs": [
                        {
                            "name": "_holder",
                            "type": "address"
                        }
                    ],
                    "name": "tokenGrantsCount",
                    "outputs": [
                        {
                            "name": "index",
                            "type": "uint256"
                        }
                    ],
                    "payable": false,
                    "type": "function"
                },
                ...
                {
                    "anonymous": false,
                    "inputs": [
                        {
                            "indexed": true,
                            "name": "from",
                            "type": "address"
                        },
                        {
                            "indexed": true,
                            "name": "to",
                            "type": "address"
                        },
                        {
                            "indexed": false,
                            "name": "value",
                            "type": "uint256"
                        }
                    ],
                    "name": "Transfer",
                    "type": "event"
                }
            ]
        }
    }
```

### `POST /api/v1/contracts/:id/execute`

Execute specific functionality encapsulated within a given `Contract`.

*This API and documentation is still being developed.*

## Oracles API

### `GET /api/v1/oracles`

Enumerate managed oracle contracts.

*This API and documentation is still being developed.*

### `POST /api/v1/oracles`

Create a managed `Oracle` smart contract and deploy it to a given `Network`. Upon successful deployment of the `Oracle` contract, the configured data feed will be consumed on the configured schedule and written onto the ledger associated with the given `Network`.

*This API and documentation is still being developed.*

### `GET /api/v1/oracles/:id`

This method is not yet implemented; it will details for the requested `Oracle`.

*This API and documentation is still being developed.*

## Tokens API

### `GET /api/v1/tokens`

Enumerate managed token contracts.

*This API and documentation is still being developed.*

## Transactions API

### `GET /api/v1/transactions`

Enumerate transactions.

*The response returned by this API will soon include network-specific metadata.*

```console
$ curl https://goldmine.provide.services/api/v1/transactions
[
    {
        "id": "b2569500-c0d2-42bf-8992-120e7ada875d",
        "created_at": "2017-12-28T16:56:42.965056Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "wallet_id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b",
        "to": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d",
        "value": 1000,
        "data": "",
        "hash": "a20441a6bf1f40cfc3de3238189a44af102f02aa2c97b91ae1484f7cbd9ab393"
    },
    {
        "id": "ca1e83b1-fc25-4471-8130-c53eb4e29623",
        "created_at": "2017-12-28T17:25:50.591828Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "wallet_id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b",
        "to": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d",
        "value": 1000,
        "data": "",
        "hash": "c7b0b39276fa65801561f49adab795361c1e99e93f6f1cf727328137f7343944"
    }
]
```

### `POST /api/v1/transactions`

Prepare and sign a protocol transaction using a managed signing `Wallet` on behalf of a specific application user and broadcast the transaction to the public blockchain `Network`. Under certain conditions, calling this API will result in a `Transaction` being created which requires lifecylce management (i.e., in the case when a managed `Sidechain` has been configured to scale micropayments channels and/or coalesce an application's transactions for on-chain settlement.

```console
$ curl -v -XPOST -H 'content-type: application/json' https://goldmine.provide.services/api/v1/transactions \
-d '{"network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc", "wallet_id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b", "to": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d", "value": 1000}'

> POST /api/v1/transactions HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
> Content-Length: 174
> Content-Type: application/json
>
* upload completely sent off: 174 out of 174 bytes
< HTTP/1.1 201 Created
< Date: Thu, 28 Dec 2017 16:56:43 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 393
< Connection: keep-alive
<
{
    "id": "b2569500-c0d2-42bf-8992-120e7ada875d",
    "created_at": "2017-12-28T16:56:42.965055765Z",
    "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
    "wallet_id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b",
    "to": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d",
    "value": 1000,
    "data": null,
    "hash": "a20441a6bf1f40cfc3de3238189a44af102f02aa2c97b91ae1484f7cbd9ab393"
}
```

The signed transaction is broadcast to the `Network` targeted by the given `network_id`:
![Tx broadcast to Ropsten testnet](https://s3.amazonaws.com/provide-github/ropsten-tx-example.png)

*If the broadcast transaction represents a contract deployment, a `Contract` will be created implicitly after the deployment has been confirmed with the `Network`. The following example represents a `Contract` creation with provided `params` specific to the Ethereum network:*

```console
$ curl -v -XPOST -H 'content-type: application/json' https://goldmine.provide.services/api/v1/transactions -d '{"network_id":"ba02ff92-f5bb-4d44-9187-7e1cc214b9fc","wallet_id":"ce1fa3b8-049e-467b-90d8-53b9a5098b7b","data":"60606040526003805460a060020a60ff021916905560006...", "params": {"name": "ProvideToken", "abi": [{"constant":true,"inputs":[{"name":"_holder","type":"address"}],"name":"tokenGrantsCount","outputs":[{"name":"index","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"controller","type":"address"}],"name":"setUpgradeController","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"uint256"}],"name":"grants","outputs":[{"name":"granter","type":"address"},{"name":"value","type":"uint256"},{"name":"cliff","type":"uint64"},{"name":"vesting","type":"uint64"},{"name":"start","type":"uint64"},{"name":"revokable","type":"bool"},{"name":"burnsOnRevoke","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_controller","type":"address"}],"name":"changeController","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"burnAmount","type":"uint256"}],"name":"burn","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"value","type":"uint256"}],"name":"upgrade","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"upgradeAgent","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_holder","type":"address"},{"name":"_grantId","type":"uint256"}],"name":"tokenGrant","outputs":[{"name":"granter","type":"address"},{"name":"value","type":"uint256"},{"name":"vested","type":"uint256"},{"name":"start","type":"uint64"},{"name":"cliff","type":"uint64"},{"name":"vesting","type":"uint64"},{"name":"revokable","type":"bool"},{"name":"burnsOnRevoke","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"holder","type":"address"}],"name":"lastTokenIsTransferableDate","outputs":[{"name":"date","type":"uint64"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"getUpgradeState","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"upgradeController","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"canUpgrade","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"},{"name":"_start","type":"uint64"},{"name":"_cliff","type":"uint64"},{"name":"_vesting","type":"uint64"},{"name":"_revokable","type":"bool"},{"name":"_burnsOnRevoke","type":"bool"}],"name":"grantVestedTokens","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalUpgraded","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"holder","type":"address"},{"name":"time","type":"uint64"}],"name":"transferableTokens","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"agent","type":"address"}],"name":"setUpgradeAgent","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokens","type":"uint256"},{"name":"time","type":"uint256"},{"name":"start","type":"uint256"},{"name":"cliff","type":"uint256"},{"name":"vesting","type":"uint256"}],"name":"calculateVestedTokens","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_holder","type":"address"},{"name":"_grantId","type":"uint256"}],"name":"revokeTokenGrant","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"controller","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"BURN_ADDRESS","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Upgrade","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"agent","type":"address"}],"name":"UpgradeAgentSet","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"},{"indexed":false,"name":"grantId","type":"uint256"}],"name":"NewTokenGrant","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"burner","type":"address"},{"indexed":false,"name":"burnedAmount","type":"uint256"}],"name":"Burned","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]}}'

> POST /api/v1/transactions HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
> Content-Type: application/json
> Content-Length: 24054
> Expect: 100-continue
>
< HTTP/1.1 100 Continue
* We are completely uploaded and fine
< HTTP/1.1 201 Created
< Date: Mon, 01 Jan 2018 20:20:07 GMT
< Content-Type: application/json; charset=UTF-8
< Transfer-Encoding: chunked
< Connection: keep-alive
<
{
    "id": "a1e55081-52d3-452b-bc24-fd4030317ac5",
    "created_at": "2018-01-01T20:19:39.527211009Z",
    "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
    "wallet_id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b",
    "to": null,
    "value": 0,
    "data": "60606040526003805460a060020a60ff02191690556000600455601460055534156100...",
    "hash": "6409dd7c75a3f291c266eb45e54534c68153102c4d275c9b0d37bf43994c9d2b",
    "params": {
        "name": "ProvideToken",
        "abi": [
            {
                "constant": true,
                "inputs": [
                    {
                        "name": "_holder",
                        "type": "address"
                    }
                ],
                "name": "tokenGrantsCount",
                "outputs": [
                    {
                        "name": "index",
                        "type": "uint256"
                    }
                ],
                "payable": false,
                "type": "function"
            },
            ...
            {
                "anonymous": false,
                "inputs": [
                    {
                        "indexed": true,
                        "name": "from",
                        "type": "address"
                    },
                    {
                        "indexed": true,
                        "name": "to",
                        "type": "address"
                    },
                    {
                        "indexed": false,
                        "name": "value",
                        "type": "uint256"
                    }
                ],
                "name": "Transfer",
                "type": "event"
            }
        ]
    }
}
```

### `GET /api/v1/transactions/:id`

## Wallets API

### `GET /api/v1/wallets`

Enumerate wallets used for storing cryptocurrency or tokens on behalf of users for which Provide is managing cryptographic material (i.e., for signing transactions).

```console
$ curl -v https://goldmine.provide.services/api/v1/wallets

> GET /api/v1/wallets HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 28 Dec 2017 16:54:18 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 740
< Connection: keep-alive
<
[
    {
        "id": "1e3b1b0f-c756-46dd-969d-2a5da3f1d24e",
        "created_at": "2017-12-25T12:10:04.72013Z",
        "network_id": "5bc7d17f-653f-4599-a6dd-618ae3a1ecb2",
        "address": "0xEA38C255b33FB4A8aE25998842cedF865398D286"
    },
    {
        "id": "ce1fa3b8-049e-467b-90d8-53b9a5098b7b",
        "created_at": "2017-12-28T09:57:04.365995Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "address": "0x605557a2dF436B8D4f9300450A3baD7fcc3FEBf8"
    },
    {
        "id": "e61a3a6b-3873-4edc-b3f9-fa7e45b92452",
        "created_at": "2017-12-28T10:21:41.607995Z",
        "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "address": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d"
    }
]
```

### `POST /api/v1/wallets`

Create a managed wallet capable of storing cryptocurrencies native to a specified `Network`.

```console
$ curl -v -XPOST https://goldmine.provide.services/api/v1/wallets -d '{"network_id":"ba02ff92-f5bb-4d44-9187-7e1cc214b9fc"}'

> POST /api/v1/wallets HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
> Content-Length: 53
> Content-Type: application/json
>
* upload completely sent off: 53 out of 53 bytes
< HTTP/1.1 201 Created
< Date: Thu, 28 Dec 2017 17:36:20 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 224
< Connection: keep-alive
<
{
    "id": "d24bc784-32b9-4c18-9f89-110986d6a0c4",
    "created_at": "2017-12-28T17:36:20.298961785Z",
    "network_id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
    "address": "0x6282e042BE5b437Bb04E800509494186c04db882"
}
```

### `GET /api/v1/wallets/:id`

This method is not yet implemented; it will return `Network`-specific details for the requested `Wallet`.

## Status API

### `GET /status`

The status API is used by loadbalancers to determine if the `goldmine` instance if healthy. It returns `204 No Content` when the running microservice instance is capable of handling API requests.
