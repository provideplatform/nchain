## nchain ##
a low-code multi-network blockchain interface

### networks ###
 - ropsten
 - rinkeby
 - goerli
 - kovan
 - mainnet

### associated microservices ###
 - **nchain**
   - REST API endpoints
   - mostly asynchronous actions (with an exception)
   - pushes messages onto *nats-streaming* for later consumption by...
 - **nchain-consumer**
   - msg bus interface (consumer.go file in each package)
   - has methods to consume nats messages and process
   - output is additional nats messages, but no direct user interaction
 - **statsdaemon**
   - pulls logs and block data from each of the active networks
 - **redis**
   - used primarily for fast nonce management
 - **nats-streaming**
   - can be clustered for resilience (similar to redis)
   - at least once delivery (unless it times out)
   - messages are always delivered in order (unless concurrent processing)
   - messages are binary data with a text subject (e.g. "nchain.tx.create")
   - messages are delivered until they either time out, get acked or nacked

### primary interaction endpoints 
 - create a contract
 - execute a contract method (change state)
 - execute a contract method (read only)
 - perform an ETH transfer (more later)
 - pass a transaction to *bookie* for execution (more later)


**create a contract**

```api/v1/contract/create```

(for example post data, check out the integration tests e.g. ```integration_basic_kovan_test.go```)


```		
contract, err := nchain.CreateContract(*appToken.Token, map[string]interface{}{
			"network_id":     tc.network,
			"application_id": app.ID.String(),
			"wallet_id":      tc.walletID,
			"name":           tc.name,
			"address":        "0x",
			"params": map[string]interface{}{
				"wallet_id":          tc.walletID,
				"hd_derivation_path": tc.derivationPath,
				"compiled_artifact":  tc.artifact,
				// "gas_price":          6000000000, //6 GWei
				//"ref":                contractRef.String(),
			}
```

**execute a contract method**

```api/v1/contract/execute```

```
	msg := common.RandomString(118)

	params = map[string]interface{}{}
	parameter = fmt.Sprintf(`{"method":"broadcast", "params": ["%s"], "value":0, "account_id":"%s"}`, msg, account.ID.String())

	json.Unmarshal([]byte(parameter), &params)

	execResponse, err := nchain.ExecuteContract(*appToken.Token, contract.ID.String(), params)
  ```

*note contract execute is inside the tx package, not the contract package*

*note that contract method executions that are readonly return data synchronously (200) while method executions that change state require a transaction, and return only the tx ref that will be broadcast to the required network(202)*

**executing a public contract method**
this is useful for running contract code that has been deployed as a public contract, e.g. an ERC20 contract

to access a public contract, add the contract to the db
```
	erc20Artifact, err := ioutil.ReadFile("artifacts/erc20.json")
	if err != nil {
		t.Errorf("error loading erc20 artifact. Error: %s", err.Error())
	}

	erc20CompiledArtifact := nchain.CompiledArtifact{}
	err = json.Unmarshal(erc20Artifact, &erc20CompiledArtifact)
	if err != nil {
		t.Errorf("error converting readwritetester compiled artifact. Error: %s", err.Error())
	}

	//t.Logf("compiled artifact for erc20: %+v", erc20Artifact)
	contractName := "MONEH - erc20 contract"
	contractAddress := "0x45a67Fd75765721D0275d3925a768E86E7a2599c"
	// MONEH contract deployed to Rinekby - 0x45a67Fd75765721D0275d3925a768E86E7a2599c
	contract, err := nchain.CreatePublicContract(*appToken.Token, map[string]interface{}{
		"network_id":     rinkebyNetworkID,
		"application_id": app.ID.String(),
		"name":           contractName,
		"address":        contractAddress,
		"params": map[string]interface{}{
			"compiled_artifact": erc20CompiledArtifact,
		},
	})
  ```
this gives you a contract id, and then you can run contract/execution methods against that contract id, e.g. for transferring ERC20 tokens

see ```integration_basic_readonly_test.go``` for examples of this.


## transaction processing ##
core assumption is that system being available, broadcast messages will succeed
i.e. mined in a block

currently (branch: idempotent angel) transactions succeed even with a 970s second delay between broadcast and being mined (longest delay experienced on ropsten so far)

**contract creation process flow**

1. prep steps - generate a compiled artifact of your contract (using truffle)
2. get a token for your application or organisaction from ident
3. get your account (has a 256k1 key in vault behind it) id
4. OR get your wallet ID (has a bip39 wallet behind it) + optional path e.g. ````m/44'/60'/2'/0/0````
5. POST to contract/create including this compiled artifact

**steps - contract creation**
1. processes POST data and creates contract record in DB
2. creates nchain.tx.create message and publishes to nats
3. swirls around in nats for a while...
4. *nchain-consumer* pulls msg from nats and processes
5. creates tx object in db (default status pending)
6. creates ethereum tx object and signs (using vault)
7. attempts to broadcast tx to chain
8. if broadcast ok, posts receipt message to nats which polls the appropriate chain for the tx hash

**steps - transaction execution -on chain**
1. processes POST data and ...
2. creates nchain.tx.create message and publishes to nats
3. swirls around in nats for a while...
4. *nchain-consumer* pulls msg from nats and processes
5. creates tx object in db (default status pending)
6. creates ethereum tx object and signs (using vault) (status: ready)
7. attempts to broadcast tx to chain (status:broadcast)
8. if broadcast ok, posts receipt message to nats which polls the appropriate chain for the tx hash
9. receipt message pulled off nats until it either times out or receipt found on chain

**steps - transaction execution -readonly**
1. processes POST data and ...
2. determines method is readonly, executes an RPC call and returns the result

## nonce and tx management ##
it's a little more complex than the above

transactions are executed in parallel for each unique address, and serial within each address

for each transaction, nchain-consumer attempts to get the right nonce

1. from redis (short term cache - 5 seconds only, used for bursting transactions)
2. from the db (if there's a previous successful tx for this address)
3. from the chain (slowest option)

the transaction then is passed to a goroutine, which signs and waits for a "go for broadcast" signal

if it's the first tx for that address, the go signal is broadcast straight away
if successful, that first tx then sends the go signal to the next serial address tx (over a channel)
and so on

within a broadcast attempt, nchain makes every effort to successfully broadcast
 - if there's a nonce-too-low error, it corrects off chain and attempts broadcast again using updated nonce
 - if there's an underpriced error, it increases the gas (still needs EIP1559) and attempts broadcast again
 - if there's a nonce-too-high scenario, it says nothing (that's the danger zone)

if a broadcast fails, the next queued tx for that address waits (or else we'd have a missing tx)
after its ack timeout, nats then publishes the message again to nchain-consumer, which processes it **idempotently** and tries again to successfully broadcast

broadcasts usually fail for 
- rpc nil response errors (v. common with provide-go interface before timeout increased)
- vault signing error (not authorised because of token failure)

but...
as nats msg consumption is idempotent, so it can reprocess the same message multiple times without creating additional transactions, and can attempt borked transactions again

every time nchain-consumer successfully broadcasts a transaction, it publishes a nchain.tx.receipt message to nats-streaming so nchain-consumer can check if it's in a mined block

## more later ##
ETH transfers are useful for testing (can set up multiple wallets and transfer enough ETH to them to run bulk tests to simulate prod environment under load)

bookie path is currently borked. previous processing rules were any signing error sent the tx down bookie route. as one of the signing failures is not enough ETH in the address balance to pay for gas, that made sense, but signing errors are common, and bookie txs post as 0x308...., not your required address, so it could cause issues if there's any ACL in the smart contract. This is being changed to an explicit request to send the tx via bookie (likely reuse something like the "subsidize" param)

**note that bookie should not be able to process ETH transfers!!  v. imp!**

## chaos monkeys ##
part of the testing was checking what happens under various conditions, nonce too low specified, tx failing to broadcast, but signing errors not happening often enough, so I introduced hacky chaos monkey code to fail some signing attempts

```	chaosMonkeyCounter++
	// if chaosMonkeyCounter%5 == 0 {
	// 	chaosErr := fmt.Errorf("chaos monkey error for tx ref %s", *t.Ref)
	// 	return chaosErr
	// }
  ```

*above fails signing every 5th attempt, but easy to configure with a quick code change*






