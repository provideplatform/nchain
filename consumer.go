package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/go-nats-streaming"

	exchangeConsumer "github.com/kthomas/exchange-consumer"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

const apiUsageDaemonBufferSize = 1024 * 25
const apiUsageDaemonFlushInterval = 10000

const natsDefaultClusterID = "provide"
const natsAPIUsageEventNotificationSubject = "api.usage.event"
const natsAPIUsageEventNotificationMaxInFlight = 32
const natsBlockFinalizedSubject = "goldmine.block.finalized"
const natsBlockFinalizedSubjectMaxInFlight = 64
const natsBlockFinalizedSubjectTimeout = time.Minute * 1
const natsContractCompilerInvocationTimeout = time.Minute * 1
const natsContractCompilerInvocationSubject = "goldmine.contract.compiler-invocation"
const natsContractCompilerInvocationMaxInFlight = 32
const natsStreamingTxFilterExecSubjectPrefix = "ml.filter.exec"
const natsLoadBalancerInvocationTimeout = time.Second * 15
const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"
const natsLoadBalancerDeprovisioningMaxInFlight = 64
const natsLoadBalancerProvisioningSubject = "goldmine.loadbalancer.provision"
const natsLoadBalancerProvisioningMaxInFlight = 64
const natsLoadBalancerBalanceNodeSubject = "goldmine.node.balance"
const natsLoadBalancerBalanceNodeMaxInFlight = 64
const natsLoadBalancerUnbalanceNodeSubject = "goldmine.node.unbalance"
const natsLoadBalancerUnbalanceNodeMaxInFlight = 64
const natsTxSubject = "goldmine.tx"
const natsTxMaxInFlight = 128
const natsTxReceiptSubject = "goldmine.tx.receipt"
const natsTxReceiptMaxInFlight = 64

var (
	waitGroup sync.WaitGroup

	currencyPairs = []string{
		// "BTC-USD",
		// "ETH-USD",
		// "LTC-USD",

		// "PRVD-USD", // FIXME-- pull from tokens database
	}
)

type apiUsageDelegate struct{}

type natsBlockFinalizedMsg struct {
	NetworkID *string `json:"network_id"`
	Block     uint64  `json:"block"`
	BlockHash *string `json:"blockhash"`
	Timestamp uint64  `json:"timestamp"`
}

func (d *apiUsageDelegate) Track(apiCall *provide.APICall) {
	payload, _ := json.Marshal(apiCall)
	natsConnection := getNatsStreamingConnection()
	natsConnection.Publish(natsAPIUsageEventNotificationSubject, payload)
}

func runAPIUsageDaemon() {
	delegate := new(apiUsageDelegate)
	provide.RunAPIUsageDaemon(apiUsageDaemonBufferSize, apiUsageDaemonFlushInterval, delegate)
}

// runConsumers launches a goroutine for each data feed
// that has been configured to consume messages
func runConsumers() {
	go func() {
		waitGroup.Add(1)
		subscribeNatsStreaming()
		for _, currencyPair := range currencyPairs {
			runConsumer(currencyPair)
		}
		waitGroup.Wait()
	}()
}

func runConsumer(currencyPair string) {
	waitGroup.Add(1)
	go func() {
		consumer := exchangeConsumer.GdaxMessageConsumerFactory(Log, priceTick, currencyPair)
		err := consumer.Run()
		if err != nil {
			Log.Warningf("Consumer exited unexpectedly; %s", err)
		} else {
			Log.Infof("Exiting consumer %s", consumer)
		}
	}()
}

func cacheTxFilters() {
	db := DatabaseConnection()
	var filters []Filter
	db.Find(&filters)
	for _, filter := range filters {
		appFilters := txFilters[filter.ApplicationID.String()]
		if appFilters == nil {
			appFilters = make([]*Filter, 0)
			txFilters[filter.ApplicationID.String()] = appFilters
		}
		appFilters = append(appFilters, &filter)
	}
}

func priceTick(msg *exchangeConsumer.GdaxMessage) error {
	if msg.Type == "match" && msg.Price != "" {
		price, err := strconv.ParseFloat(msg.Price, 64)
		if err == nil {
			SetPrice(msg.ProductId, msg.Sequence, price)
		}
	} else {
		Log.Debugf("Dropping GDAX message; %s", msg)
	}
	return nil
}

func getNatsStreamingConnection() stan.Conn {
	conn := natsutil.GetNatsStreamingConnection(func(_ stan.Conn, reason error) {
		subscribeNatsStreaming()
	})
	if conn == nil {
		return nil
	}
	return *conn
}

func subscribeNatsStreaming() {
	natsConnection := getNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsTxSubscriptions(natsConnection)
	createNatsTxReceiptSubscriptions(natsConnection)
	createNatsBlockFinalizedSubscriptions(natsConnection)
	createNatsContractCompilerInvocationSubscriptions(natsConnection)
	createNatsLoadBalancerProvisioningSubscriptions(natsConnection)
	createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection)
	createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection)
	createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection)
}

func createNatsTxSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txSubscription, err := natsConnection.QueueSubscribe(natsTxSubject, natsTxSubject, consumeTxMsg, stan.SetManualAckMode(), stan.AckWait(time.Millisecond*10000), stan.MaxInflight(natsTxMaxInFlight), stan.DurableName(natsTxSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxSubject)
				waitGroup.Done()
				return
			}
			defer txSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsTxSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsTxReceiptSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			txReceiptSubscription, err := natsConnection.QueueSubscribe(natsTxReceiptSubject, natsTxReceiptSubject, consumeTxReceiptMsg, stan.SetManualAckMode(), stan.AckWait(receiptTickerTimeout), stan.MaxInflight(natsTxReceiptMaxInFlight), stan.DurableName(natsTxReceiptSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsTxReceiptSubject)
				waitGroup.Done()
				return
			}
			defer txReceiptSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsTxReceiptSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsBlockFinalizedSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			blockFinalizedSubscription, err := natsConnection.QueueSubscribe(natsBlockFinalizedSubject, natsBlockFinalizedSubject, consumeBlockFinalizedMsg, stan.SetManualAckMode(), stan.AckWait(natsBlockFinalizedSubjectTimeout), stan.MaxInflight(natsBlockFinalizedSubjectMaxInFlight), stan.DurableName(natsBlockFinalizedSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsContractCompilerInvocationSubject)
				waitGroup.Done()
				return
			}
			defer blockFinalizedSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsContractCompilerInvocationSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsContractCompilerInvocationSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			contractCompilerInvocationSubscription, err := natsConnection.QueueSubscribe(natsContractCompilerInvocationSubject, natsContractCompilerInvocationSubject, consumeContractCompilerInvocationMsg, stan.SetManualAckMode(), stan.AckWait(natsContractCompilerInvocationTimeout), stan.MaxInflight(natsContractCompilerInvocationMaxInFlight), stan.DurableName(natsContractCompilerInvocationSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsContractCompilerInvocationSubject)
				waitGroup.Done()
				return
			}
			defer contractCompilerInvocationSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsContractCompilerInvocationSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsLoadBalancerProvisioningSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerProvisioningSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerProvisioningSubject, natsLoadBalancerProvisioningSubject, consumeLoadBalancerProvisioningMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerProvisioningMaxInFlight), stan.DurableName(natsLoadBalancerProvisioningSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerProvisioningSubject)
				waitGroup.Done()
				return
			}
			defer loadBalancerProvisioningSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerProvisioningSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerDeprovisioningSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerDeprovisioningSubject, natsLoadBalancerDeprovisioningSubject, consumeLoadBalancerDeprovisioningMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerDeprovisioningMaxInFlight), stan.DurableName(natsLoadBalancerDeprovisioningSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)
				waitGroup.Done()
				return
			}
			defer loadBalancerDeprovisioningSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerBalanceNodeSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerBalanceNodeSubject, natsLoadBalancerBalanceNodeSubject, consumeLoadBalancerBalanceNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerBalanceNodeMaxInFlight), stan.DurableName(natsLoadBalancerBalanceNodeSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)
				waitGroup.Done()
				return
			}
			defer loadBalancerBalanceNodeSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)

			waitGroup.Wait()
		}()
	}
}

func createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerUnbalanceNodeSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerUnbalanceNodeSubject, natsLoadBalancerUnbalanceNodeSubject, consumeLoadBalancerUnbalanceNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerUnbalanceNodeMaxInFlight), stan.DurableName(natsLoadBalancerUnbalanceNodeSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)
				waitGroup.Done()
				return
			}
			defer loadBalancerUnbalanceNodeSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)

			waitGroup.Wait()
		}()
	}
}

func consumeBlockFinalizedMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS block finalized message: %s", msg)
	var err error

	blockFinalizedMsg := &natsBlockFinalizedMsg{}
	err = json.Unmarshal(msg.Data, &blockFinalizedMsg)
	if err != nil {
		Log.Warningf("Failed to unmarshal block finalized message; %s", err.Error())
		return
	}

	if blockFinalizedMsg.NetworkID == nil {
		err = fmt.Errorf("Parsed NATS block finalized message did not contain network id: %s", msg)
	}

	if err == nil {
		db := DatabaseConnection()

		network := &Network{}
		db.Where("id = ?", blockFinalizedMsg.NetworkID).Find(&network)

		if network == nil || network.ID == uuid.Nil {
			err = fmt.Errorf("Failed to retrieve network by id: %s", *blockFinalizedMsg.NetworkID)
		}

		if err == nil {
			if network.isEthereumNetwork() {
				if err == nil {
					block, err := provide.EVMGetBlockByNumber(network.ID.String(), network.rpcURL(), blockFinalizedMsg.Block)
					if err != nil {
						err = fmt.Errorf("Failed to fetch block; %s", err.Error())
					} else if result, resultOk := block.Result.(map[string]interface{}); resultOk {
						finalizedAt := time.Unix(int64(blockFinalizedMsg.Timestamp/1000), 0)

						if txs, txsOk := result["transactions"].([]interface{}); txsOk {
							for _, _tx := range txs {
								txHash := _tx.(map[string]interface{})["hash"].(string)
								Log.Debugf("Setting tx finalized at timestamp %s on tx: %s", finalizedAt, txHash)

								tx := &Transaction{}
								db.Where("hash = ?", txHash).Find(&tx)
								if tx == nil || tx.ID == uuid.Nil {
									Log.Warningf("Failed to set finalized_at timestamp on tx: %s", txHash)
									continue
								}
								tx.FinalizedAt = &finalizedAt
								result := db.Save(&tx)
								errors := result.GetErrors()
								if len(errors) > 0 {
									for _, err := range errors {
										tx.Errors = append(tx.Errors, &provide.Error{
											Message: stringOrNil(err.Error()),
										})
									}
								}
								if len(tx.Errors) > 0 {
									Log.Warningf("Failed to set finalized_at timestamp on tx: %s; error: %s", txHash, tx.Errors[0].Message)
								}
							}
						}
					}
				} else {
					err = fmt.Errorf("Failed to decode EVM block header; %s", err.Error())
				}
			} else {
				Log.Debugf("Received unhandled finalized block header type: %s", blockFinalizedMsg.Block)
			}
		}
	}

	if err != nil {
		Log.Warningf("Failed to handle block finalized message; %s", err.Error())
	} else {
		msg.Ack()
	}
}

func consumeTxMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS tx message: %s", msg)

	execution := &ContractExecution{}
	err := json.Unmarshal(msg.Data, execution)
	if err != nil {
		Log.Warningf("Failed to unmarshal contract execution during NATS tx message handling")
		return
	}

	if execution.ContractID == nil {
		Log.Errorf("Invalid tx message; missing contract_id")
		return
	}

	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil && execution.Wallet.ID != execution.Wallet.ID {
			Log.Errorf("Invalid tx message specifying a wallet_id and wallet")
			return
		}
		wallet := &Wallet{}
		wallet.setID(*execution.WalletID)
		execution.Wallet = wallet
	}

	contract := &Contract{}
	DatabaseConnection().Where("id = ?", *execution.ContractID).Find(&contract)
	if contract == nil || contract.ID == uuid.Nil {
		Log.Errorf("Unable to execute contract; contract not found: %s", contract.ID)
		return
	}

	gas := execution.Gas
	if gas == nil {
		gas64 := float64(0)
		gas = &gas64
	}
	_gas, _ := big.NewFloat(*gas).Uint64()

	executionResponse, err := contract.Execute(execution.Ref, execution.Wallet, execution.Value, execution.Method, execution.Params, _gas)
	if err != nil {
		Log.Warningf("Failed to execute contract; %s", err.Error())
		Log.Warningf("NATS message dropped: %s", msg)
		// FIXME-- Augment NATS support and Nack?
	} else {
		Log.Debugf("Executed contract; tx: %s", executionResponse)
	}

	msg.Ack()
}

func consumeLoadBalancerDeprovisioningMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer deprovisioning message: %s", msg)
	var balancer *LoadBalancer

	err := json.Unmarshal(msg.Data, &balancer)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer deprovisioning message; %s", err.Error())
		return
	}

	err = balancer.deprovision(DatabaseConnection())
	if err != nil {
		Log.Warningf("Failed to deprovision load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer deprovisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerProvisioningMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer provisioning message: %s", msg)
	var balancer *LoadBalancer

	err := json.Unmarshal(msg.Data, &balancer)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer provisioning message; %s", err.Error())
		return
	}

	err = balancer.provision(DatabaseConnection())
	if err != nil {
		Log.Warningf("Failed to provision load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer provisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerBalanceNodeMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer balance node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer balance node message; %s", err.Error())
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		Log.Warningf("Failed to load balance network node; no load balancer id provided")
		return
	}
	if !networkNodeIDOk {
		Log.Warningf("Failed to load balance network node; no network node id provided")
		return
	}

	db := DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		Log.Warningf("Failed to load balance network node; no load balancer resolved for id: %s", balancerID)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		Log.Warningf("Failed to load balance network node; no network node resolved for id: %s", networkNodeID)
		msg.Ack() // FIXME: Nack to deadletter
	}

	err = balancer.balanceNode(db, node)
	if err != nil {
		Log.Warningf("Failed to balance node on load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer node balancing succeeded; ACKing NATS message: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerUnbalanceNodeMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer unbalance node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer unbalance node message; %s", err.Error())
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		Log.Warningf("Failed to unbalance network node; no load balancer id provided")
		return
	}
	if !networkNodeIDOk {
		Log.Warningf("Failed to load balance network node; no network node id provided")
		return
	}

	db := DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		Log.Warningf("Failed to remove network node from load balancer; no load balancer resolved for id: %s", balancerID)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		Log.Warningf("Failed to removal network node from load balancer; no network node resolved for id: %s", networkNodeID)
		return
	}

	err = balancer.unbalanceNode(db, node)
	if err != nil {
		Log.Warningf("Failed to remove node from load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer node removal succeeded; ACKing NATS message: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeTxReceiptMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS tx receipt message: %s", msg)

	db := DatabaseConnection()

	var tx *Transaction

	err := json.Unmarshal(msg.Data, &tx)
	if err != nil {
		Log.Warningf("Failed to umarshal tx receipt message; %s", err.Error())
		return
	}

	network, err := tx.GetNetwork()
	if err != nil {
		Log.Warningf("Failed to resolve tx network; %s", err.Error())
	}

	wallet, err := tx.GetWallet()
	if err != nil {
		Log.Warningf("Failed to resolve tx wallet")
	}

	tx.fetchReceipt(db, network, wallet)
	msg.Ack()
}

func consumeContractCompilerInvocationMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS contract compiler invocation message: %s", msg)
	var contract *Contract

	err := json.Unmarshal(msg.Data, &contract)
	if err != nil {
		Log.Warningf("Failed to umarshal contract compiler invocation message; %s", err.Error())
		return
	}

	_, err = contract.Compile()
	if err != nil {
		Log.Warningf("Failed to compile contract; %s", err.Error())
	} else {
		Log.Debugf("Contract compiler invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	}
}
