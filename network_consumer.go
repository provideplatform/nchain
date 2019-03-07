package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/nats-io/go-nats-streaming"
	provide "github.com/provideservices/provide-go"
)

const natsBlockFinalizedSubject = "goldmine.block.finalized"
const natsBlockFinalizedSubjectMaxInFlight = 64
const natsBlockFinalizedSubjectTimeout = time.Minute * 1

const natsLoadBalancerInvocationTimeout = time.Second * 15
const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"
const natsLoadBalancerDeprovisioningMaxInFlight = 64
const natsLoadBalancerProvisioningSubject = "goldmine.loadbalancer.provision"
const natsLoadBalancerProvisioningMaxInFlight = 64
const natsLoadBalancerBalanceNodeSubject = "goldmine.node.balance"
const natsLoadBalancerBalanceNodeMaxInFlight = 64
const natsLoadBalancerUnbalanceNodeSubject = "goldmine.node.unbalance"
const natsLoadBalancerUnbalanceNodeMaxInFlight = 64

type natsBlockFinalizedMsg struct {
	NetworkID *string `json:"network_id"`
	Block     uint64  `json:"block"`
	BlockHash *string `json:"blockhash"`
	Timestamp uint64  `json:"timestamp"`
}

func createNatsBlockFinalizedSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			blockFinalizedSubscription, err := natsConnection.QueueSubscribe(natsBlockFinalizedSubject, natsBlockFinalizedSubject, consumeBlockFinalizedMsg, stan.SetManualAckMode(), stan.AckWait(natsBlockFinalizedSubjectTimeout), stan.MaxInflight(natsBlockFinalizedSubjectMaxInFlight), stan.DurableName(natsBlockFinalizedSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsBlockFinalizedSubject)
				wg.Done()
				return
			}
			defer blockFinalizedSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsBlockFinalizedSubject)

			wg.Wait()
		}()
	}
}

func createNatsLoadBalancerProvisioningSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerProvisioningSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerProvisioningSubject, natsLoadBalancerProvisioningSubject, consumeLoadBalancerProvisioningMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerProvisioningMaxInFlight), stan.DurableName(natsLoadBalancerProvisioningSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerProvisioningSubject)
				wg.Done()
				return
			}
			defer loadBalancerProvisioningSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerProvisioningSubject)

			wg.Wait()
		}()
	}
}

func createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerDeprovisioningSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerDeprovisioningSubject, natsLoadBalancerDeprovisioningSubject, consumeLoadBalancerDeprovisioningMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerDeprovisioningMaxInFlight), stan.DurableName(natsLoadBalancerDeprovisioningSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)
				wg.Done()
				return
			}
			defer loadBalancerDeprovisioningSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)

			wg.Wait()
		}()
	}
}

func createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerBalanceNodeSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerBalanceNodeSubject, natsLoadBalancerBalanceNodeSubject, consumeLoadBalancerBalanceNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerBalanceNodeMaxInFlight), stan.DurableName(natsLoadBalancerBalanceNodeSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)
				wg.Done()
				return
			}
			defer loadBalancerBalanceNodeSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)

			wg.Wait()
		}()
	}
}

func createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			loadBalancerUnbalanceNodeSubscription, err := natsConnection.QueueSubscribe(natsLoadBalancerUnbalanceNodeSubject, natsLoadBalancerUnbalanceNodeSubject, consumeLoadBalancerUnbalanceNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsLoadBalancerInvocationTimeout), stan.MaxInflight(natsLoadBalancerUnbalanceNodeMaxInFlight), stan.DurableName(natsLoadBalancerUnbalanceNodeSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)
				wg.Done()
				return
			}
			defer loadBalancerUnbalanceNodeSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)

			wg.Wait()
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
		db := dbconf.DatabaseConnection()

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
								Log.Debugf("Setting tx block and finalized_at timestamp %s on tx: %s", finalizedAt, txHash)

								tx := &Transaction{}
								db.Where("hash = ?", txHash).Find(&tx)
								if tx == nil || tx.ID == uuid.Nil {
									Log.Warningf("Failed to set block and finalized_at timestamp on tx: %s", txHash)
									continue
								}
								tx.Block = &blockFinalizedMsg.Block
								tx.FinalizedAt = &finalizedAt
								if tx.BroadcastAt != nil {
									if tx.PublishedAt != nil {
										publishLatency := uint64(tx.BroadcastAt.Sub(*tx.PublishedAt)*time.Millisecond) / 1000000
										tx.PublishLatency = &publishLatency

										e2eLatency := uint64(tx.FinalizedAt.Sub(*tx.PublishedAt)*time.Millisecond) / 1000000
										tx.E2ELatency = &e2eLatency
									}

									broadcastLatency := uint64(tx.FinalizedAt.Sub(*tx.BroadcastAt)*time.Millisecond) / 1000000
									tx.BroadcastLatency = &broadcastLatency
								}
								tx.Status = StringOrNil("success")
								result := db.Save(&tx)
								errors := result.GetErrors()
								if len(errors) > 0 {
									for _, err := range errors {
										tx.Errors = append(tx.Errors, &provide.Error{
											Message: StringOrNil(err.Error()),
										})
									}
								}
								if len(tx.Errors) > 0 {
									Log.Warningf("Failed to set block and finalized_at timestamp on tx: %s; error: %s", txHash, tx.Errors[0].Message)
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
		nack(msg)
	} else {
		msg.Ack()
	}
}

func consumeLoadBalancerProvisioningMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer provisioning message: %s", msg)
	var balancer *LoadBalancer

	err := json.Unmarshal(msg.Data, &balancer)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer provisioning message; %s", err.Error())
		nack(msg)
		return
	}

	err = balancer.provision(dbconf.DatabaseConnection())
	if err != nil {
		Log.Warningf("Failed to provision load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer provisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerDeprovisioningMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer deprovisioning message: %s", msg)
	var balancer *LoadBalancer

	err := json.Unmarshal(msg.Data, &balancer)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer deprovisioning message; %s", err.Error())
		nack(msg)
		return
	}

	err = balancer.deprovision(dbconf.DatabaseConnection())
	if err != nil {
		Log.Warningf("Failed to deprovision load balancer; %s", err.Error())
	} else {
		Log.Debugf("Load balancer deprovisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerBalanceNodeMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS load balancer balance node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		Log.Warningf("Failed to umarshal load balancer balance node message; %s", err.Error())
		nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		Log.Warningf("Failed to load balance network node; no load balancer id provided")
		nack(msg)
		return
	}
	if !networkNodeIDOk {
		Log.Warningf("Failed to load balance network node; no network node id provided")
		nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		Log.Warningf("Failed to load balance network node; no load balancer resolved for id: %s", balancerID)
		nack(msg)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		Log.Warningf("Failed to load balance network node; no network node resolved for id: %s", networkNodeID)
		nack(msg)
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
		nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		Log.Warningf("Failed to unbalance network node; no load balancer id provided")
		nack(msg)
		return
	}
	if !networkNodeIDOk {
		Log.Warningf("Failed to load balance network node; no network node id provided")
		nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		Log.Warningf("Failed to remove network node from load balancer; no load balancer resolved for id: %s", balancerID)
		nack(msg)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		Log.Warningf("Failed to remove network node from load balancer; no network node resolved for id: %s", networkNodeID)
		nack(msg)
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
