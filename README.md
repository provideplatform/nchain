## goldmine

API for building best-of-breed applications which leverage a public blockchain.

### Authentication

Consumers of this API will present a `bearer` authentication header (i.e., using a `JWT` token) for all requests. The mechanism to require this authentication has not yet been included in the codebase to simplify development and integration, but it will be included in the coming weeks; this section will be updated with specifics when authentication is required.

### Authorization

The `bearer` authorization header will be scoped to an authorized application. The `bearer` authorization header may contain a `sub` (see [RFC-7519 §§ 4.1.2](https://tools.ietf.org/html/rfc7519#section-4.1.2)) to further limit its authorized scope to a specific token or smart contract, wallet or other entity.

Certain APIs will be metered similarly to how AWS meters some of its webservices. Production applications will need a sufficient PRVD token balance to consume metered APIs (based on market conditions at the time of consumption, some quantity of PRVD tokens will be burned as a result of such metered API usage. *The PRVD token model and economics are in the process of being peer-reviewed and finalized; the documentation will be updated accordingly with specifics.*

---
The following APIs are exposed:


### Networks API

##### `GET /api/v1/networks`

Enumerate available blockchain networks and related configuration details.

```
[prvd@vpc ~]# curl -v http://goldmine.provide.services/api/v1/networks

> GET /api/v1/networks HTTP/1.1
> Host: goldmine.provide.services
> User-Agent: curl/7.51.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 28 Dec 2017 16:54:12 GMT
< Content-Type: application/json; charset=UTF-8
< Content-Length: 1899
< Connection: keep-alive
<
[
    {
        "id": "07c85f35-aa6d-4ec2-8a92-2240a85e91e9",
        "created_at": "2017-12-25T11:54:57.62033Z",
        "name": "Lightning",
        "description": "Lightning Network mainnet",
        "is_production": true,
        "sidechain_id": null,
        "config": null
    },
    {
        "id": "017428e4-41ac-41ab-bd08-8fc234e8169f",
        "created_at": "2017-12-25T11:54:57.622145Z",
        "name": "Lightning Testnet",
        "description": "Lightning Network testnet",
        "is_production": false,
        "sidechain_id": null,
        "config": null
    },
    {
        "id": "6eafb694-95a9-407f-a6d2-f541659c49b9",
        "created_at": "2017-12-25T11:54:57.615578Z",
        "name": "Bitcoin",
        "description": "Bitcoin mainnet",
        "is_production": true,
        "sidechain_id": "07c85f35-aa6d-4ec2-8a92-2240a85e91e9",
        "config": null
    },
    {
        "id": "b018af93-7c7f-4b76-a0d3-2f4282250e82",
        "created_at": "2017-12-25T11:54:57.61854Z",
        "name": "Bitcoin Testnet",
        "description": "Bitcoin testnet",
        "is_production": false,
        "sidechain_id": "017428e4-41ac-41ab-bd08-8fc234e8169f",
        "config": null
    },
    {
        "id": "5bc7d17f-653f-4599-a6dd-618ae3a1ecb2",
        "created_at": "2017-12-25T11:54:57.629505Z",
        "name": "Ethereum",
        "description": "Ethereum mainnet",
        "is_production": true,
        "sidechain_id": null,
        "config": null
    },
    {
        "id": "ba02ff92-f5bb-4d44-9187-7e1cc214b9fc",
        "created_at": "2017-12-25T11:54:57.63379Z",
        "name": "Ethereum testnet",
        "description": "ROPSTEN (Revival) TESTNET",
        "is_production": false,
        "sidechain_id": null,
        "config": {
            "json_rpc_url": "http://ethereum-ropsten-testnet-json-rpc.provide.services",
            "testnet": "ropsten"
        }
    }
]
```

##### `GET /api/v1/networks/:id`
##### `GET /api/v1/networks/:id/addresses`
##### `GET /api/v1/networks/:id/blocks`
##### `GET /api/v1/networks/:id/contracts`
##### `GET /api/v1/networks/:id/transactions`


### Prices API

##### `GET /api/v1/prices`

Fetch real-time pricing data for major currency pairs and supported tokens.

```
[prvd@vpc ~]# curl -v http://goldmine.provide.services/api/v1/prices

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


### Tokens API

##### `GET /api/v1/tokens`

Enumerate managed token contracts.

*This API and documentation is still being developed.*

##### `POST /api/v1/tokens`

Create a new token smart contract in accordance with the given parameters and deploy it to the specified `Network`.

*This API and documentation is still being developed.*


### Transactions API

##### `GET /api/v1/transactions`

Enumerate transactions.

*The response returned by this API will soon include network-specific metadata.*

```
[prvd@vpc ~]# curl http://goldmine.provide.services/api/v1/transactions
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

##### `POST /api/v1/transactions`

Prepare, sign and broadcast a transaction using a specific `Wallet` signer and broadcast the transaction to the public blockchain supported by the `Network`. Under certain conditions, calling this API will result in a `Transaction` being created which requires lifecylce management (i.e., when using managed state channels).

```
[prvd@vpc ~]# curl -v -XPOST -H 'content-type: application/json' http://goldmine.provide.services/api/v1/transactions \
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

##### `GET /api/v1/transactions/:id`


### Wallets API

##### `GET /api/v1/wallets`

Enumerate wallets used for storing cryptocurrency or tokens on behalf of users for which Provide is managing cryptographic material (i.e., for signing transactions).

```
[prvd@vpc ~]# curl -v http://goldmine.provide.services/api/v1/wallets

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
        "address": "0xB73eAB7130a673618Ccfc18daa6509C198a92aDc"
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

##### `POST /api/v1/wallets`

Create a managed wallet capable of storing cryptocurrencies native to a specified `Network`.

```
[prvd@vpc ~]# curl -v -XPOST http://goldmine.provide.services/api/v1/wallets -d '{"network_id":"ba02ff92-f5bb-4d44-9187-7e1cc214b9fc"}'

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

##### `GET /api/v1/wallets/:id`

This method is not yet implemented; it will return `Network`-specific details for the requested `Wallet`.


### Status API

##### `GET /status`

The status API is used by loadbalancers to determine if the `goldmine` instance if healthy. It returns `204 No Content` when the running microservice instance is capable of handling API requests.
