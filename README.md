# goldmine

API for building best-of-breed applications which leverage a public blockchain.

## Authentication

Consumers of this API will present a `bearer` authentication header (i.e., using a `JWT` token) for all requests. The mechanism to require this authentication has not yet been included in the codebase to simplify development and integration, but it will be included in the coming weeks; this section will be updated with specifics when authentication is required.

## Authorization

The `bearer` authorization header will be scoped to an authorized application. The `bearer` authorization header may contain a `sub` (see [RFC-7519 §§ 4.1.2](https://tools.ietf.org/html/rfc7519#section-4.1.2)) to further limit its authorized scope to a specific token or smart contract, wallet or other entity.

Certain APIs will be metered similarly to how AWS meters some of its webservices. Production applications will need a sufficient PRVD token balance to consume metered APIs (based on market conditions at the time of consumption, some quantity of PRVD tokens will be burned as a result of such metered API usage. *The PRVD token model and economics are in the process of being peer-reviewed and finalized; the documentation will be updated accordingly with specifics.*

---

**TODO: replace the request tokens with the 'fake' ones from Ident (do this last).**
**TODO: revisit the "not yet implemented" to verify if that's really true (and complete if not) - should only be 501s...**

The following APIs are exposed:

## Networks API

### `GET /api/v1/networks`

Enumerate available blockchain `Network`s and related configuration details.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks
HTTP/2 200
date: Wed, 10 Oct 2018 14:56:22 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 1

[
    {
        "id": "e5e0a051-6af7-4d1e-88cd-0ea1f67abd50",
        "created_at": "2018-10-10T14:56:11.812732Z",
        "application_id": null,
        "user_id": "3d9d62e8-0acf-47cd-b74f-52c1f96f8397",
        "name": "provide.network \"unicorn\" testnet CLONE",
        "description": null,
        "is_production": false,
        "cloneable": false,
        "enabled": true,
        "chain_id": "0x5bbe130b",
        "sidechain_id": null,
        "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
        "config": {
            "block_explorer_url": null,
            "chain": null,
            "chainspec": {
                "accounts": {
                    "0x0000000000000000000000000000000000000001": {
                        "balance": "1",
                        "builtin": {
                            "name": "ecrecover",
                            "pricing": {
                                "linear": {
                                    "base": 3000,
                                    "word": 0
                                }
                            }
                        }
                    },
                    ...
                    "0x0000000000000000000000000000000000000008": {
                        "balance": "1",
                        "builtin": {
                            "activate_at": "0x0",
                            "name": "alt_bn128_pairing",
                            "pricing": {
                                "alt_bn128_pairing": {
                                    "base": 100000,
                                    "pair": 80000
                                }
                            }
                        }
                    },
                    "0x0000000000000000000000000000000000000009": {
                        "balance": "1",
                        "constructor": "0x608060405234801561001057600080fd5b50611bfe806100206000..."
                    },
                    "0x0000000000000000000000000000000000000010": {
                        "balance": "1",
                        "constructor": "0x612988610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000011": {
                        "balance": "1",
                        "constructor": "0x610876610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000012": {
                        "balance": "1",
                        "constructor": "0x610f03610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000013": {
                        "balance": "1",
                        "constructor": "0x6110cc610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000014": {
                        "balance": "1",
                        "constructor": "0x610d6b610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000015": {
                        "balance": "1",
                        "constructor": "0x61032a610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000016": {
                        "balance": "1",
                        "constructor": "0x6101db610030600b82828239805160001a60731460008114610020..."
                    },
                    "0x0000000000000000000000000000000000000017": {
                        "balance": "1",
                        "constructor": "0x60806040523480156200001157600080fd5b506040516101208062..."
                    },
                    "0x9077F27fDD606c41822f711231eEDA88317aa67a": {
                        "balance": "1"
                    }
                },
                "engine": {
                    "authorityRound": {
                        "params": {
                            "blockReward": "0xDE0B6B3A7640000",
                            "maximumUncleCount": 0,
                            "maximumUncleCountTransition": 0,
                            "stepDuration": 5,
                            "validators": {
                                "multi": {
                                    "0": {
                                        "contract": "0x0000000000000000000000000000000000000017"
                                    }
                                }
                            }
                        }
                    }
                },
                "genesis": {
                    "difficulty": "0x20000",
                    "gasLimit": "0x7A1200",
                    "seal": {
                        "authorityRound": {
                            "signature": "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
                            "step": "0x0"
                        }
                    }
                },
                "name": "unicorn",
                "params": {
                    "chainID": "0x5bbe130b",
                    "eip140Transition": "0x0",
                    "eip211Transition": "0x0",
                    "eip214Transition": "0x0",
                    "eip658Transition": "0x0",
                    "gasLimitBoundDivisor": "0x400",
                    "maximumExtraDataSize": "0x20",
                    "minGasLimit": "0x1388",
                    "networkID": "0x5bbe130b"
                },
                "subprotocolName": "prvd"
            },
            "chainspec_abi": {
                "0x0000000000000000000000000000000000000017": [
                    {
                        "constant": true,
                        "inputs": [
                            {
                                "name": "_validator",
                                "type": "address"
                            }
                        ],
                        "name": "getValidatorSupportCount",
                        "outputs": [
                            {
                                "name": "",
                                "type": "uint256"
                            }
                        ],
                        "payable": false,
                        "stateMutability": "view",
                        "type": "function"
                    },
                    ...
                    {
                        "constant": false,
                        "inputs": [
                            {
                                "name": "_validator",
                                "type": "address"
                            }
                        ],
                        "name": "removeValidator",
                        "outputs": null,
                        "payable": false,
                        "stateMutability": "nonpayable",
                        "type": "function"
                    },
                    {
                        "constant": false,
                        "inputs": [
                            {
                                "name": "_app_name",
                                "type": "bytes32"
                            },
                            {
                                "name": "_version_name",
                                "type": "bytes32"
                            },
                            {
                                "name": "_init",
                                "type": "address"
                            },
                            {
                                "name": "_init_sel",
                                "type": "bytes4"
                            },
                            {
                                "name": "_init_calldata",
                                "type": "bytes"
                            },
                            {
                                "name": "_fn_sels",
                                "type": "bytes4[]"
                            },
                            {
                                "name": "_fn_addrs",
                                "type": "address[]"
                            },
                            {
                                "name": "_version_desc",
                                "type": "bytes"
                            },
                            {
                                "name": "_init_desc",
                                "type": "bytes"
                            }
                        ],
                        "name": "deployConsensus",
                        "outputs": null,
                        "payable": false,
                        "stateMutability": "nonpayable",
                        "type": "function"
                    },
                    ...
                    {
                        "anonymous": false,
                        "inputs": [
                            {
                                "indexed": true,
                                "name": "supporter",
                                "type": "address"
                            },
                            {
                                "indexed": true,
                                "name": "supported",
                                "type": "address"
                            },
                            {
                                "indexed": true,
                                "name": "added",
                                "type": "bool"
                            }
                        ],
                        "name": "Support",
                        "type": "event"
                    }
                ]
            },
            "chainspec_abi_url": null,
            "chainspec_url": null,
            "cloneable_cfg": {
                "_security": {
                    "egress": "*",
                    "ingress": {
                        "0.0.0.0/0": {
                            "tcp": [
                                8050,
                                8051,
                                30300
                            ],
                            "udp": [
                                30300
                            ]
                        }
                    }
                },
                "aws": {
                    "docker": {
                        "regions": {
                            "ap-northeast-1": {
                                "peer": "providenetwork-node",
                                "validator": "providenetwork-node"
                            },
                            ...
                            "us-west-2": {
                                "peer": "providenetwork-node",
                                "validator": "providenetwork-node"
                            }
                        }
                    },
                    "ubuntu-vm": {
                        "regions": {
                            "ap-northeast-1": {
                                "peer": {
                                    "0.0.9": "ami-1e22e061"
                                },
                                "validator": {
                                    "0.0.9": "ami-1e22e061"
                                }
                            },
                            ...
                            "us-west-2": {
                                "peer": {
                                    "0.0.9": "ami-813777f9"
                                },
                                "validator": {
                                    "0.0.9": "ami-813777f9"
                                }
                            }
                        }
                    }
                }
            },
            "engine_id": "authorityRound",
            "is_ethereum_network": true,
            "is_load_balanced": false,
            "json_rpc_url": null,
            "native_currency": "PRVDDOC",
            "network_id": 1539183371,
            "parity_json_rpc_url": null,
            "protocol_id": "poa",
            "websocket_url": null
        },
        "stats": null
    }
]
```

### `GET /api/v1/networks/:id`

Fetch details for the given `Network`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50
HTTP/2 200
date: Wed, 10 Oct 2018 15:03:53 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "e5e0a051-6af7-4d1e-88cd-0ea1f67abd50",
    "created_at": "2018-10-10T14:56:11.812732Z",
    "application_id": null,
    "user_id": "3d9d62e8-0acf-47cd-b74f-52c1f96f8397",
    "name": "provide.network \"unicorn\" testnet CLONE",
    "description": null,
    "is_production": false,
    "cloneable": false,
    "enabled": true,
    "chain_id": "0x5bbe130b",
    "sidechain_id": null,
    "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
    "config": {
        "block_explorer_url": null,
        "chain": null,
        "chainspec": {
            "accounts": {
                "0x0000000000000000000000000000000000000001": {
                    "balance": "1",
                    "builtin": {
                        "name": "ecrecover",
                        "pricing": {
                            "linear": {
                                "base": 3000,
                                "word": 0
                            }
                        }
                    }
                },
                ...
                "0x0000000000000000000000000000000000000008": {
                    "balance": "1",
                    "builtin": {
                        "activate_at": "0x0",
                        "name": "alt_bn128_pairing",
                        "pricing": {
                            "alt_bn128_pairing": {
                                "base": 100000,
                                "pair": 80000
                            }
                        }
                    }
                },
                "0x0000000000000000000000000000000000000009": {
                    "balance": "1",
                    "constructor": "0x608060405234801561001057600080fd5b50611bfe806100206000396..."
                },
                ...
                "0x0000000000000000000000000000000000000017": {
                    "balance": "1",
                    "constructor": "0x60806040523480156200001157600080fd5b50604051610120806200a..."
                },
                "0x9077F27fDD606c41822f711231eEDA88317aa67a": {
                    "balance": "1"
                }
            },
            "engine": {
                "authorityRound": {
                    "params": {
                        "blockReward": "0xDE0B6B3A7640000",
                        "maximumUncleCount": 0,
                        "maximumUncleCountTransition": 0,
                        "stepDuration": 5,
                        "validators": {
                            "multi": {
                                "0": {
                                    "contract": "0x0000000000000000000000000000000000000017"
                                }
                            }
                        }
                    }
                }
            },
            "genesis": {
                "difficulty": "0x20000",
                "gasLimit": "0x7A1200",
                "seal": {
                    "authorityRound": {
                        "signature": "0x0000000000000000000000000000000000000000000000000000000....",
                        "step": "0x0"
                    }
                }
            },
            "name": "unicorn",
            "params": {
                "chainID": "0x5bbe130b",
                "eip140Transition": "0x0",
                "eip211Transition": "0x0",
                "eip214Transition": "0x0",
                "eip658Transition": "0x0",
                "gasLimitBoundDivisor": "0x400",
                "maximumExtraDataSize": "0x20",
                "minGasLimit": "0x1388",
                "networkID": "0x5bbe130b"
            },
            "subprotocolName": "prvd"
        },
        "chainspec_abi": {
            "0x0000000000000000000000000000000000000017": [
                {
                    "constant": true,
                    "inputs": [
                        {
                            "name": "_validator",
                            "type": "address"
                        }
                    ],
                    "name": "getValidatorSupportCount",
                    "outputs": [
                        {
                            "name": "",
                            "type": "uint256"
                        }
                    ],
                    "payable": false,
                    "stateMutability": "view",
                    "type": "function"
                },
                ...
                {
                    "anonymous": false,
                    "inputs": [
                        {
                            "indexed": true,
                            "name": "supporter",
                            "type": "address"
                        },
                        {
                            "indexed": true,
                            "name": "supported",
                            "type": "address"
                        },
                        {
                            "indexed": true,
                            "name": "added",
                            "type": "bool"
                        }
                    ],
                    "name": "Support",
                    "type": "event"
                }
            ]
        },
        "chainspec_abi_url": null,
        "chainspec_url": null,
        "cloneable_cfg": {
            "_security": {
                "egress": "*",
                "ingress": {
                    "0.0.0.0/0": {
                        "tcp": [
                            8050,
                            8051,
                            30300
                        ],
                        "udp": [
                            30300
                        ]
                    }
                }
            },
            "aws": {
                "docker": {
                    "regions": {
                        "ap-northeast-1": {
                            "peer": "providenetwork-node",
                            "validator": "providenetwork-node"
                        },
                        ...
                        "us-west-2": {
                            "peer": "providenetwork-node",
                            "validator": "providenetwork-node"
                        }
                    }
                },
                "ubuntu-vm": {
                    "regions": {
                        "ap-northeast-1": {
                            "peer": {
                                "0.0.9": "ami-1e22e061"
                            },
                            "validator": {
                                "0.0.9": "ami-1e22e061"
                            }
                        },
                        ...
                        "us-west-2": {
                            "peer": {
                                "0.0.9": "ami-813777f9"
                            },
                            "validator": {
                                "0.0.9": "ami-813777f9"
                            }
                        }
                    }
                }
            }
        },
        "engine_id": "authorityRound",
        "is_ethereum_network": true,
        "is_load_balanced": false,
        "json_rpc_url": null,
        "native_currency": "PRVDDOC",
        "network_id": 1539183371,
        "parity_json_rpc_url": null,
        "protocol_id": "poa",
        "websocket_url": null
    },
    "stats": null
}
```

### `GET /api/v1/networks/:id/addresses`

Enumerate the addresses of the given `Network`.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/addresses
HTTP/2 501
date: Wed, 10 Oct 2018 15:07:42 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

### `GET /api/v1/networks/:id/blocks`

Enumerate the blocks of the given `Network`.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/blocks
HTTP/2 501
date: Wed, 10 Oct 2018 15:09:07 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

### `GET /api/v1/networks/:id/contracts`

Enumerate the contracts of the given `Network`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/contracts
HTTP/2 200
date: Wed, 10 Oct 2018 15:09:44 GMT
content-type: application/json; charset=UTF-8
content-length: 467
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 1

[
    {
        "id": "752eadfa-f138-4c73-a9b9-822a5bfb4dc6",
        "created_at": "0001-01-01T00:00:00Z",
        "application_id": null,
        "network_id": "e5e0a051-6af7-4d1e-88cd-0ea1f67abd50",
        "contract_id": null,
        "transaction_id": null,
        "name": "Network Contract 0x0000000000000000000000000000000000000017",
        "address": "0x0000000000000000000000000000000000000017",
        "params": null,
        "accessed_at": null
    }
]
```

### `GET /api/v1/networks/:id/transactions`

Enumerate the transactions of the given `Network`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/024ff1ef-7369-4dee-969c-1918c6edb5d4/transactions
HTTP/2 200
date: Wed, 10 Oct 2018 15:11:38 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 1061221

[
    {
        "id": "f1125746-fa73-4392-a935-7a57faf3f7b0",
        "created_at": "2018-10-01T11:28:26.24501Z",
        "application_id": null,
        "user_id": "7062d7f8-d536-4c53-bba5-7486a8724ac3",
        "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
        "wallet_id": "f924d014-9d76-444a-b8e1-810bd3a75319",
        "to": null,
        "value": 0,
        "data": "0x6080604052604051602080612faa833981016040818152915160008054600160a060020a0383...",
        "hash": "0x79b9bef674435c69b25f7564c2cf216b2a3640343bce2a36067016f6a3811894",
        "status": "success",
        "params": null,
        "traces": null,
        "ref": null,
        "description": null
    },
    ...
    {
        "id": "44ce12a0-3779-4e0d-b095-ca15fec0a872",
        "created_at": "2018-09-01T05:55:22.7759Z",
        "application_id": null,
        "user_id": null,
        "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
        "wallet_id": "dfc12196-44d8-4ca2-8e6e-46f3ccf4c3a8",
        "to": "0x139c328F1fF7910757E63F5045469CbDE18152b2",
        "value": 0,
        "data": "0x676f7fec31313233343500000000000000000000000000000000000000000000000000003031...",
        "hash": "0x8584a8a2167868c3dbd948e3eade46ae0c98ca43bdf7d7e17b0693fabf004690",
        "status": "success",
        "params": null,
        "traces": null,
        "ref": "bd90e2e8-60a4-4522-89ce-53fe0611b268",
        "description": null
    }
]
```

### `GET /api/v1/networks/:id/transactions/:transactionId`

Fetch the details of the `Network`'s specified `Transaction`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/024ff1ef-7369-4dee-969c-1918c6edb5d4/transactions/f1125746-fa73-4392-a935-7a57faf3f7b0
HTTP/2 200
date: Sat, 13 Oct 2018 16:59:32 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "f1125746-fa73-4392-a935-7a57faf3f7b0",
    "created_at": "2018-10-01T11:28:26.24501Z",
    "application_id": null,
    "user_id": "7062d7f8-d536-4c53-bba5-7486a8724ac3",
    "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
    "wallet_id": "f924d014-9d76-444a-b8e1-810bd3a75319",
    "to": null,
    "value": 0,
    "data": "0x6080604052604051602080612faa833981016040818152915160008054600160a060020a03831660...",
    "hash": "0x79b9bef674435c69b25f7564c2cf216b2a3640343bce2a36067016f6a3811894",
    "status": "success",
    "params": null,
    "traces": {
        "result": [
            {
                "action": {
                    "callType": null,
                    "from": "0x8cd7e456d1276e7074b10ae94dcd25f62d1ea147",
                    "gas": "0x2516fd",
                    "init": "0x6080604052604051602080612faa833981016040818152915160008054600160...",
                    "input": null,
                    "to": null,
                    "value": "0x0"
                },
                "blockHash": "0x519dd9d06775eb36d0795c469ed6892f12c5dadc881916f29cca3de9c4fd238d",
                "blockNumber": 2348545,
                "result": {
                    "address": "0x2e49f7d86daf4c3590646c4c2619b2956e8e027b",
                    "code": "0x608060405260043610620000955763ffffffff60e060020a6000350416632e9d...",
                    "gasUsed": "0x2516fd",
                    "output": null
                },
                "error": null,
                "subtraces": 0,
                "traceAddress": [],
                "transactionHash": "0x79b9bef674435c69b25f7564c2cf216b2a3640343bce2a36067016f6a3811894",
                "transactionPosition": 0,
                "type": "create"
            }
        ]
    },
    "ref": null,
    "description": null
}
```

### `PUT /api/v1/networks/:id`

Update the given `Network`.

```console
$ curl -i -XPUT \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50 \
    -d '{"name": "provide.network \"unicorn\" testnet CLONE now with more update"}'
HTTP/2 204
date: Sat, 13 Oct 2018 17:04:43 GMT
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
```

### `GET /api/v1/networks/:id/contracts/:contractId`

Fetch the details of the `Network`s specified `Contract`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/contracts/752eadfa-f138-4c73-a9b9-822a5bfb4dc6
HTTP/2 200
date: Sat, 13 Oct 2018 17:39:52 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "752eadfa-f138-4c73-a9b9-822a5bfb4dc6",
    "created_at": "2018-10-10T14:56:11.84238Z",
    "application_id": null,
    "network_id": "e5e0a051-6af7-4d1e-88cd-0ea1f67abd50",
    "contract_id": null,
    "transaction_id": null,
    "name": "Network Contract 0x0000000000000000000000000000000000000017",
    "address": "0x0000000000000000000000000000000000000017",
    "params": {
        "abi": [
            {
                "constant": true,
                "inputs": [
                    {
                        "name": "_validator",
                        "type": "address"
                    }
                ],
                "name": "getValidatorSupportCount",
                "outputs": [
                    {
                        "name": "",
                        "type": "uint256"
                    }
                ],
                "payable": false,
                "stateMutability": "view",
                "type": "function"
            },
            ...
            {
                "anonymous": false,
                "inputs": [
                    {
                        "indexed": true,
                        "name": "supporter",
                        "type": "address"
                    },
                    {
                        "indexed": true,
                        "name": "supported",
                        "type": "address"
                    },
                    {
                        "indexed": true,
                        "name": "added",
                        "type": "bool"
                    }
                ],
                "name": "Support",
                "type": "event"
            }
        ],
        "name": "Network Contract 0x0000000000000000000000000000000000000017"
    },
    "accessed_at": null
}
```

### `POST /api/v1/networks`

Create a new `Network`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/networks/:id/bridges`

Enumerate the specified `Network`'s `Bridge`s.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/bridges
HTTP/2 501
date: Sat, 13 Oct 2018 17:41:42 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

### `GET /api/v1/networks/:id/connectors`

Enumerate the specified `Network`'s `Connector`s.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/connectors
HTTP/2 501
date: Sat, 13 Oct 2018 17:43:05 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

### `GET /api/v1/networks/:id/nodes`

Enumerate the specified `Network`'s `Node`s.

**TODO: try with different network(s) to get actual data.**

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/nodes
HTTP/2 200
date: Sat, 13 Oct 2018 17:43:40 GMT
content-type: application/json; charset=UTF-8
content-length: 3
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 0

[]
```

### `POST /api/v1/networks/:id/nodes`

Create a new `Node` on the given `Network`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/networks/:id/nodes/:nodeId`

Fetch the details of the `Network`'s specified `Node`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/networks/:id/nodes/:nodeId/logs`

Get the logs for the specified `Node`.

_Not yet implemented._

```console
**TODO: complete**
```

### `DELETE /api/v1/networks/:id/nodes/:nodeId`

Remove the `Network`'s specified `Node`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/networks/:id/oracles`

Enumerate the specified `Network`'s `Oracle`s.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/e5e0a051-6af7-4d1e-88cd-0ea1f67abd50/oracles
HTTP/2 501
date: Sat, 13 Oct 2018 17:58:52 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

### `GET /api/v1/networks/:id/tokens`

Enumerate the specified `Network`'s `Tokens`s.

_Not yet implemented._

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/024ff1ef-7369-4dee-969c-1918c6edb5d4/tokens
HTTP/2 200
date: Sat, 13 Oct 2018 17:06:52 GMT
content-type: application/json; charset=UTF-8
content-length: 3
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 0

[]
```

### `GET /api/v1/networks/:id/status`

Check the status of the given `Network`.

```console
curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/networks/024ff1ef-7369-4dee-969c-1918c6edb5d4/transactions
HTTP/2 200
date: Fri, 12 Oct 2018 22:44:01 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "block": 2546646,
    "chain_id": "22",
    "height": null,
    "last_block_at": 1539384240093,
    "peer_count": 0,
    "protocol_version": null,
    "state": null,
    "syncing": false,
    "meta": {
        "average_blocktime": 5.164322580645161,
        "blocktimes": [
            5.032,
            ...
            5.085
        ],
        "last_block_hash": "0xb6836eed97fd2961832f450b0b3aca0ea0df956f9c28cd5a6e281f2de483938d",
        "last_block_header": {
            "parentHash": "0x48a19fb89487ad54990840a19c06467ab0e06b01e407f13f68bc6eb70b25e83f",
            "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
            "miner": "0x9077f27fdd606c41822f711231eeda88317aa67a",
            "stateRoot": "0x38b2dba574c7519e18af5eff4f545d7361ee497677449b4701a0d21200c5643f",
            "transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
            "logsBloom": "0x0000000000000000000000000000000000000000000000000000000000000000000...",
            "difficulty": "0xfffffffffffffffffffffffffffffffe",
            "number": "0x26dbd6",
            "gasLimit": "0x47b760",
            "gasUsed": "0x0",
            "timestamp": "0x5bc123b0",
            "extraData": "0xd583010b058650617269747986312e32372e30826c69",
            "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "0x0000000000000000",
            "hash": "0xb6836eed97fd2961832f450b0b3aca0ea0df956f9c28cd5a6e281f2de483938d"
        }
    }
}
```

## Prices API

### `GET /api/v1/prices`

Fetch real-time pricing data for major currency pairs and supported tokens.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/prices
HTTP/2 200
date: Wed, 10 Oct 2018 15:19:12 GMT
content-type: application/json; charset=UTF-8
content-length: 72
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "btcusd": 14105.2,
    "ethusd": 707,
    "ltcusd": 244.48,
    "prvdusd": 0.42
}
```

## Connectors API

### `GET /api/v1/connectors`

Enumerate configured `Connector`s.

_Not yet implemented._

```console
curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/connectors
HTTP/2 200
date: Fri, 12 Oct 2018 22:14:36 GMT
content-type: application/json; charset=UTF-8
content-length: 3
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 0

[]
```

### `GET /api/v1/connectors`

Enumerate configured `Connector`s.

_Not yet implemented._

```console
**TODO: complete**
```

### `POST /api/v1/connectors`

Configure a neew `Connector`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/connectors/:id`

fetch the details for the specified `Connector`.

_Not yet implemented._

```console
**TODO: complete**
```

### `DELETE /api/v1/connectors/:id`

Remove the specified `Connector`.

_Not yet implemented._

```console
**TODO: complete**
```

## Contracts API

### `GET /api/v1/contracts`

Enumerate managed smart contracts. Each `Contract` contains a `params` object which includes `Network`-specific descriptors. `Token` contracts can be filtered from the response by passing a query string with the `filter_tokens` parameter set to `true`.

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/contracts
HTTP/2 200
date: Wed, 10 Oct 2018 15:21:44 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 289

[
    {
        "id": "7ede6b96-874a-424d-ae85-669c1c90dfb8",
        "created_at": "0001-01-01T00:00:00Z",
        "application_id": null,
        "network_id": "4e9e29de-fbd2-4d6a-962e-93bc55c1a306",
        "contract_id": null,
        "transaction_id": null,
        "name": "Network Contract 0x0000000000000000000000000000000000000009",
        "address": "0x0000000000000000000000000000000000000009",
        "params": null,
        "accessed_at": null
    },
    ...
    {
        "id": "652befe6-6cc7-42b2-8994-eeaa98adbf0e",
        "created_at": "0001-01-01T00:00:00Z",
        "application_id": null,
        "network_id": "73c82453-2a19-4f2a-86c6-37f66ae78189",
        "contract_id": null,
        "transaction_id": null,
        "name": "Network Contract 0x0000000000000000000000000000000000000009",
        "address": "0x0000000000000000000000000000000000000009",
        "params": null,
        "accessed_at": null
    }
]
```

### `POST /api/v1/contracts`

Deploy a new `Contract`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/contracts/:id`

Fetch details for the specified `Contract`.

**TODO: broken?/can't find any damn contract ID... Have Kyle look.**

```console
$ curl -i \
    -H 'Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1MzkwNDI5NjgsImp0aSI6IjUyZDljYWYzLTBhNTYtNDY3MC04ODZlLTgxMzZiNjMzZDUyYiIsInN1YiI6InVzZXI6M2Q5ZDYyZTgtMGFjZi00N2NkLWI3NGYtNTJjMWY5NmY4Mzk3In0.BVMNmeGt2IcOjOzjF0s5q4KVbqbVA0t9S9K_POFewCM' \
    https://goldmine.provide.services/api/v1/contracts/
    3b9fe62e-5da7-43dc-838f-3cfa1421ed0f
    652befe6-6cc7-42b2-8994-eeaa98adbf0e
    7ede6b96-874a-424d-ae85-669c1c90dfb8
    752eadfa-f138-4c73-a9b9-822a5bfb4dc6

HTTP/2 404
date: Wed, 10 Oct 2018 15:43:11 GMT
content-type: application/json; charset=UTF-8
content-length: 40
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "contract not found"
}
```

### `POST /api/v1/contracts/:id/execute`

Execute specific functionality encapsulated within a given `Contract`. 

- Returns a `200 OK` if the invoked `Contract` method does not modify state. 
- Returns a`202 Accepted` if the method will modify state, signifying that the request has been accepted for asynchronous processing. 
- Error responses vary with the nature of the error. 

Explanation of request body:
- `wallet_id` is the signing identity to use for the request
- `method` is the function name from the contract to invoke 
- `params` is an array of values to pass as arguments to the `method`
- `value` is the amount of token transfer

_Not yet implemented._

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/contracts/3b9fe62e-5da7-43dc-838f-3cfa1421ed0f/execute \
     -d '{"wallet_id": "a-signing-identity-identifier", "method": "a_method_from_the_contract", "params": ["arguments", "for", "the", "method"], "value": 0.5}'
HTTP/2 202
date: Wed, 10 Oct 2018 15:34:35 GMT
content-type: application/json; charset=UTF-8
content-length: 54
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "ref": "specifics-depend-on-the-nature-of-the-contract-and-method"
}
```

## Oracles API

### `GET /api/v1/oracles`

Enumerate managed oracle contracts.

_Not yet implemented._

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/oracles
HTTP/2 200
date: Fri, 12 Oct 2018 21:29:35 GMT
content-type: application/json; charset=UTF-8
content-length: 3
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 0

[]
````

### `POST /api/v1/oracles`

Create a managed `Oracle` smart contract and deploy it to a given `Network`. Upon successful deployment of the `Oracle` contract, the configured data feed will be consumed on the configured schedule and written onto the ledger associated with the given `Network`.

_Not yet implemented._

```console
**TODO: complete**
```

### `GET /api/v1/oracles/:id`

This method is not yet implemented; it will details for the requested `Oracle`.

_Not yet implemented._

```console
**TODO: complete**
```

## Tokens API

### `GET /api/v1/tokens`

Enumerate managed `Token` contracts.

_Not yet implemented._

```console
curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/tokens
HTTP/2 200
date: Fri, 12 Oct 2018 21:31:31 GMT
content-type: application/json; charset=UTF-8
content-length: 3
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 0

[]
```

### `POST /api/v1/tokens`

Create a new `Token` for the given... **TODO: complete description**

_Not yet implemented._

```console
curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/tokens \
     -d '{"TODO": "need params"}'
```

### `GET /api/v1/tokens/:id`

Fetch details for the given `Token`.

_Not yet implemented._

```console
curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/tokens/52d9caf3-0a56-4670-886e-8136b633d52b
HTTP/2 501
date: Sat, 13 Oct 2018 17:34:49 GMT
content-type: application/json; charset=UTF-8
content-length: 37
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "message": "not implemented"
}
```

## Transactions API

### `GET /api/v1/transactions`

Enumerate `Transaction`s.

*The response returned by this API will soon include network-specific metadata.* **TODO: yet? is that metadata there, Kyle?**

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/transactions
HTTP/2 200
date: Fri, 12 Oct 2018 21:37:43 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 1

[
    {
        "id": "fb4c5912-628f-4d8e-9c67-76d379010a71",
        "created_at": "2018-10-03T20:48:37.292823Z",
        "application_id": "e49302c5-e485-4e14-9b0f-db5643b6a15c",
        "user_id": null,
        "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
        "wallet_id": "efef1044-4958-43bc-903b-28f2bb938037",
        "to": null,
        "value": 0,
        "data": "0x608060405234801561001057600080fd5b50610704806100206000396000f300608060405260...",
        "hash": "0xf769d96671abfd8c7dfe9b747db1380d1b974d6282833e573900bad7b11e51e5",
        "status": "success",
        "params": null,
        "traces": null,
        "ref": null,
        "description": null
    }
]
```

### `POST /api/v1/transactions`

Prepare and sign a protocol `Transaction` using a managed signing `Wallet` on behalf of a specific application `User` and broadcast the transaction to the public blockchain `Network`. Under certain conditions, calling this API will result in a transaction being created which requires lifecylce management (i.e., in the case when a managed `Sidechain` has been configured to scale micropayments channels and/or coalesce an application's transactions for on-chain settlement).

**TODO: apparently needs to be requested with funds -- have Kyle get updated response. Same with continuation below.**

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/transactions \
     -d '{"network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4", "wallet_id": "efef1044-4958-43bc-903b-28f2bb938037", "to": "0xfb17cB7bb99128AAb60B1DD103271d99C8237c0d", "value": 0}'
HTTP/2 422
date: Fri, 12 Oct 2018 21:54:01 GMT
content-type: application/json; charset=UTF-8
content-length: 277
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "b2569500-c0d2-42bf-8992-120e7ada875d",
    "created_at": "2018-10-12T121:54:01.965055765Z",
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
$ curl -i \
    -H 'content-type: application/json' https://goldmine.provide.services/api/v1/transactions 
    -d '{"network_id":"ba02ff92-f5bb-4d44-9187-7e1cc214b9fc","wallet_id":"ce1fa3b8-049e-467b-90d8-53b9a5098b7b","data":"60606040526003805460a060020a60ff021916905560006...", "params": {"name": "ProvideToken", "abi": [{"constant":true,"inputs":[{"name":"_holder","type":"address"}],"name":"tokenGrantsCount","outputs":[{"name":"index","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"controller","type":"address"}],"name":"setUpgradeController","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"uint256"}],"name":"grants","outputs":[{"name":"granter","type":"address"},{"name":"value","type":"uint256"},{"name":"cliff","type":"uint64"},{"name":"vesting","type":"uint64"},{"name":"start","type":"uint64"},{"name":"revokable","type":"bool"},{"name":"burnsOnRevoke","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_controller","type":"address"}],"name":"changeController","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"burnAmount","type":"uint256"}],"name":"burn","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"value","type":"uint256"}],"name":"upgrade","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"upgradeAgent","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_holder","type":"address"},{"name":"_grantId","type":"uint256"}],"name":"tokenGrant","outputs":[{"name":"granter","type":"address"},{"name":"value","type":"uint256"},{"name":"vested","type":"uint256"},{"name":"start","type":"uint64"},{"name":"cliff","type":"uint64"},{"name":"vesting","type":"uint64"},{"name":"revokable","type":"bool"},{"name":"burnsOnRevoke","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"holder","type":"address"}],"name":"lastTokenIsTransferableDate","outputs":[{"name":"date","type":"uint64"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"getUpgradeState","outputs":[{"name":"","type":"uint8"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"upgradeController","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"canUpgrade","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"},{"name":"_start","type":"uint64"},{"name":"_cliff","type":"uint64"},{"name":"_vesting","type":"uint64"},{"name":"_revokable","type":"bool"},{"name":"_burnsOnRevoke","type":"bool"}],"name":"grantVestedTokens","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalUpgraded","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"holder","type":"address"},{"name":"time","type":"uint64"}],"name":"transferableTokens","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"agent","type":"address"}],"name":"setUpgradeAgent","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokens","type":"uint256"},{"name":"time","type":"uint256"},{"name":"start","type":"uint256"},{"name":"cliff","type":"uint256"},{"name":"vesting","type":"uint256"}],"name":"calculateVestedTokens","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_holder","type":"address"},{"name":"_grantId","type":"uint256"}],"name":"revokeTokenGrant","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"controller","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"BURN_ADDRESS","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"inputs":[],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Upgrade","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"agent","type":"address"}],"name":"UpgradeAgentSet","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"},{"indexed":false,"name":"grantId","type":"uint256"}],"name":"NewTokenGrant","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"burner","type":"address"},{"indexed":false,"name":"burnedAmount","type":"uint256"}],"name":"Burned","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]}}'

HTTP/2 422
date: Fri, 12 Oct 2018 21:57:11 GMT
content-type: application/json; charset=UTF-8
content-length: 277
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: * Connection: keep-alive
<
{
    "id": "a1e55081-52d3-452b-bc24-fd4030317ac5",
    "created_at": "2018-10-12T21:21:57.527211009Z",
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

Fetch the details of the specified `Transaction`. 

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/transactions/fb4c5912-628f-4d8e-9c67-76d379010a71
HTTP/2 200
date: Fri, 12 Oct 2018 21:41:43 GMT
content-type: application/json; charset=UTF-8
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "fb4c5912-628f-4d8e-9c67-76d379010a71",
    "created_at": "2018-10-03T20:48:37.292823Z",
    "application_id": "e49302c5-e485-4e14-9b0f-db5643b6a15c",
    "user_id": null,
    "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
    "wallet_id": "efef1044-4958-43bc-903b-28f2bb938037",
    "to": null,
    "value": 0,
    "data": "0x608060405234801561001057600080fd5b50610704806100206000396000f3006080604052600436...",
    "hash": "0xf769d96671abfd8c7dfe9b747db1380d1b974d6282833e573900bad7b11e51e5",
    "status": "success",
    "params": null,
    "traces": {
        "result": [
            {
                "action": {
                    "callType": null,
                    "from": "0xac805f1c2bf9a19b448bc207075b992be29bc91a",
                    "gas": "0x57caf",
                    "init": "0x608060405234801561001057600080fd5b50610704806100206000396000f300...",
                    "input": null,
                    "to": null,
                    "value": "0x0"
                },
                "blockHash": "0x4a5e8e1bf594dc4df5c54c4b9c91f34749c8a35801132790693f9e74a2fecb86",
                "blockNumber": 2389804,
                "result": {
                    "address": "0x2a3d64307a551a1f2902867f5b2437bc52f0c5c4",
                    "code": "0x6080604052600436106100565763ffffffff7c01000000000000000000000000...",
                    "gasUsed": "0x57caf",
                    "output": null
                },
                "error": null,
                "subtraces": 0,
                "traceAddress": [],
                "transactionHash": "0xf769d96671abfd8c7dfe9b747db1380d1b974d6282833e573900bad7b11e51e5",
                "transactionPosition": 0,
                "type": "create"
            }
        ]
    },
    "ref": null,
    "description": null
}
```

## Wallets API

### `GET /api/v1/wallets`

Enumerate wallets used for storing cryptocurrency or tokens on behalf of users for which Provide is managing cryptographic material (i.e., for signing transactions).

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/wallets
HTTP/2 200
date: Fri, 12 Oct 2018 21:45:14 GMT
content-type: application/json; charset=UTF-8
content-length: 418
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
x-total-results-count: 1

[
    {
        "id": "efef1044-4958-43bc-903b-28f2bb938037",
        "created_at": "2018-10-03T20:48:03.24878Z",
        "application_id": "e49302c5-e485-4e14-9b0f-db5643b6a15c",
        "user_id": null,
        "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
        "address": "0xAC805F1c2Bf9a19b448bc207075B992Be29bC91a",
        "balance": null,
        "accessed_at": "2018-10-03T20:48:37.291739Z"
    }
]
```

### `POST /api/v1/wallets`

Create a managed `Wallet` (signing identity) capable of storing cryptocurrencies native to a specified `Network`.

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/wallets \
     -d '{"network_id":"024ff1ef-7369-4dee-969c-1918c6edb5d4"}'
HTTP/2 201
date: Fri, 12 Oct 2018 21:47:13 GMT
content-type: application/json; charset=UTF-8
content-length: 353
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "4059f749-55ad-4c1c-975d-6c5040801079",
    "created_at": "2018-10-12T21:47:13.698524641Z",
    "application_id": "e49302c5-e485-4e14-9b0f-db5643b6a15c",
    "user_id": null,
    "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
    "address": "0xa4f8874C971EB257C0Fd8e33401b274e2a27133d",
    "balance": null,
    "accessed_at": null
}
```

### `GET /api/v1/wallets/:id`

Return `Network`-specific details for the requested `Wallet`.

**TODO: shouldn't this find the one just created above?!**

```console
$ curl -i \
     -H 'content-type: application/json' \
     -H 'authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjp7fSwiZXhwIjpudWxsLCJpYXQiOjE1Mzg1OTk2ODMsImp0aSI6IjUwMTM1YzlhLTYxNDEtNDhmNC05ODJmLTkxNmYwYjFlOTgyNCIsInN1YiI6ImFwcGxpY2F0aW9uOmU0OTMwMmM1LWU0ODUtNGUxNC05YjBmLWRiNTY0M2I2YTE1YyJ9.rsmn_Mq_mMcon47b2nU4ziqIRhW64rO4pqrAWVRmcgc' \
     https://goldmine.provide.services/api/v1/wallets/efef1044-4958-43bc-903b-28f2bb938037
HTTP/2 200
date: Fri, 12 Oct 2018 21:49:34 GMT
content-type: application/json; charset=UTF-8
content-length: 371
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *

{
    "id": "efef1044-4958-43bc-903b-28f2bb938037",
    "created_at": "2018-10-03T20:48:03.24878Z",
    "application_id": "e49302c5-e485-4e14-9b0f-db5643b6a15c",
    "user_id": null,
    "network_id": "024ff1ef-7369-4dee-969c-1918c6edb5d4",
    "address": "0xAC805F1c2Bf9a19b448bc207075B992Be29bC91a",
    "balance": 0,
    "accessed_at": "2018-10-03T20:48:37.291739Z"
}
```

### `GET /api/v1/wallets/:id/balances/:tokenId`

**TODO: is description accurate? Why no wallets/:id/balances by itself?*

Fetch the details of the given `Wallet`'s balance. 

_Not yet implemented._

```console
**TODO: complete**
```

## Status API

### `GET /status`

The status API is used by loadbalancers to determine if the `goldmine` instance if healthy. It returns `204 No Content` when the running microservice instance is capable of handling API requests.

```console
$ curl -i https://goldmine.provide.services/status
HTTP/2 204
date: Wed, 10 Oct 2018 15:17:09 GMT
access-control-allow-credentials: true
access-control-allow-headers: Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With
access-control-allow-methods: GET, POST, PUT, DELETE, OPTIONS
access-control-allow-origin: *
```
