package network

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/consumer"
	provide "github.com/provideservices/provide-go"
)

const natsBlockFinalizedSubject = "goldmine.block.finalized"
const natsBlockFinalizedSubjectMaxInFlight = 64
const natsBlockFinalizedInvocationTimeout = time.Minute * 1
const natsBlockFinalizedTimeout = int64(1000 * 60 * 10)

const natsLoadBalancerInvocationTimeout = time.Second * 15

const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"
const natsLoadBalancerDeprovisioningMaxInFlight = 64
const natsLoadBalancerDeprovisioningTimeout = int64(1000 * 60 * 10)

const natsLoadBalancerProvisioningSubject = "goldmine.loadbalancer.provision"
const natsLoadBalancerProvisioningMaxInFlight = 64
const natsLoadBalancerProvisioningTimeout = int64(1000 * 60 * 10)

const natsLoadBalancerBalanceNodeSubject = "goldmine.node.balance"
const natsLoadBalancerBalanceNodeMaxInFlight = 64
const natsLoadBalancerBalanceNodeTimeout = int64(1000 * 60 * 10)

const natsLoadBalancerUnbalanceNodeSubject = "goldmine.node.unbalance"
const natsLoadBalancerUnbalanceNodeMaxInFlight = 64
const natsLoadBalancerUnbalanceNodeTimeout = int64(1000 * 60 * 10)

const natsDeployNetworkNodeSubject = "goldmine.node.deploy"
const natsDeployNetworkNodeMaxInFlight = 64
const natsDeployNetworkNodeInvocationTimeout = time.Minute * 1
const natsDeployNetworkNodeTimeout = int64(1000 * 60 * 10)

const natsDeleteTerminatedNetworkNodeSubject = "goldmine.node.delete"
const natsDeleteTerminatedNetworkNodeMaxInFlight = 64
const natsDeleteTerminatedNetworkNodeInvocationTimeout = time.Minute * 1
const natsDeleteTerminatedNetworkNodeTimeout = int64(1000 * 60 * 10)

const natsResolveNetworkNodeHostSubject = "goldmine.node.resolve-host"
const natsResolveNetworkNodeHostMaxInFlight = 64
const natsResolveNetworkNodeHostInvocationTimeout = time.Second * 10
const natsResolveNetworkNodeHostTimeout = int64(1000 * 60 * 10)

const natsResolveNetworkNodePeerURLSubject = "goldmine.node.resolve-peer"
const natsResolveNetworkNodePeerURLMaxInFlight = 64
const natsResolveNetworkNodePeerURLInvocationTimeout = time.Second * 10
const natsResolveNetworkNodePeerURLTimeout = int64(1000 * 60 * 10)

const natsTxFinalizeSubject = "goldmine.tx.finalize"

type natsBlockFinalizedMsg struct {
	NetworkID *string `json:"network_id"`
	Block     uint64  `json:"block"`
	BlockHash *string `json:"blockhash"`
	Timestamp uint64  `json:"timestamp"`
}

var waitGroup sync.WaitGroup

func init() {
	natsConnection := consumer.GetNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsBlockFinalizedSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerProvisioningSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerDeprovisioningSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerBalanceNodeSubscriptions(natsConnection, &waitGroup)
	createNatsLoadBalancerUnbalanceNodeSubscriptions(natsConnection, &waitGroup)
	createNatsDeployNetworkNodeSubscriptions(natsConnection, &waitGroup)
	createNatsDeleteTerminatedNetworkNodeSubscriptions(natsConnection, &waitGroup)
	createNatsResolveNetworkNodeHostSubscriptions(natsConnection, &waitGroup)
	createNatsResolveNetworkNodePeerURLSubscriptions(natsConnection, &waitGroup)
}

func createNatsBlockFinalizedSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			blockFinalizedSubscription, err := natsConnection.QueueSubscribe(natsBlockFinalizedSubject, natsBlockFinalizedSubject, consumeBlockFinalizedMsg, stan.SetManualAckMode(), stan.AckWait(natsBlockFinalizedInvocationTimeout), stan.MaxInflight(natsBlockFinalizedSubjectMaxInFlight), stan.DurableName(natsBlockFinalizedSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsBlockFinalizedSubject)
				wg.Done()
				return
			}
			defer blockFinalizedSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsBlockFinalizedSubject)

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
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerProvisioningSubject)
				wg.Done()
				return
			}
			defer loadBalancerProvisioningSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerProvisioningSubject)

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
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)
				wg.Done()
				return
			}
			defer loadBalancerDeprovisioningSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerDeprovisioningSubject)

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
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)
				wg.Done()
				return
			}
			defer loadBalancerBalanceNodeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerBalanceNodeSubject)

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
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)
				wg.Done()
				return
			}
			defer loadBalancerUnbalanceNodeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsLoadBalancerUnbalanceNodeSubject)

			wg.Wait()
		}()
	}
}

func createNatsDeployNetworkNodeSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			deployNetworkNodeSubscription, err := natsConnection.QueueSubscribe(natsDeployNetworkNodeSubject, natsDeployNetworkNodeSubject, consumeDeployNetworkNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsDeployNetworkNodeInvocationTimeout), stan.MaxInflight(natsDeployNetworkNodeMaxInFlight), stan.DurableName(natsDeployNetworkNodeSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsDeployNetworkNodeSubject)
				wg.Done()
				return
			}
			defer deployNetworkNodeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsDeployNetworkNodeSubject)

			wg.Wait()
		}()
	}
}

func createNatsResolveNetworkNodeHostSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			resolveNetworkNodeHostSubscription, err := natsConnection.QueueSubscribe(natsResolveNetworkNodeHostSubject, natsResolveNetworkNodeHostSubject, consumeResolveNetworkNodeHostMsg, stan.SetManualAckMode(), stan.AckWait(natsResolveNetworkNodeHostInvocationTimeout), stan.MaxInflight(natsResolveNetworkNodeHostMaxInFlight), stan.DurableName(natsResolveNetworkNodeHostSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsResolveNetworkNodeHostSubject)
				wg.Done()
				return
			}
			defer resolveNetworkNodeHostSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsResolveNetworkNodeHostSubject)

			wg.Wait()
		}()
	}
}

func createNatsResolveNetworkNodePeerURLSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			resolveNetworkNodePeerURLSubscription, err := natsConnection.QueueSubscribe(natsResolveNetworkNodePeerURLSubject, natsResolveNetworkNodePeerURLSubject, consumeResolveNetworkNodePeerURLMsg, stan.SetManualAckMode(), stan.AckWait(natsResolveNetworkNodePeerURLInvocationTimeout), stan.MaxInflight(natsResolveNetworkNodePeerURLMaxInFlight), stan.DurableName(natsResolveNetworkNodePeerURLSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsResolveNetworkNodePeerURLSubject)
				wg.Done()
				return
			}
			defer resolveNetworkNodePeerURLSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsResolveNetworkNodePeerURLSubject)

			wg.Wait()
		}()
	}
}

func createNatsDeleteTerminatedNetworkNodeSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			deleteTerminatedNetworkNodeSubscription, err := natsConnection.QueueSubscribe(natsDeleteTerminatedNetworkNodeSubject, natsDeleteTerminatedNetworkNodeSubject, consumeDeleteTerminatedNetworkNodeMsg, stan.SetManualAckMode(), stan.AckWait(natsDeleteTerminatedNetworkNodeInvocationTimeout), stan.MaxInflight(natsDeleteTerminatedNetworkNodeMaxInFlight), stan.DurableName(natsDeleteTerminatedNetworkNodeSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsDeleteTerminatedNetworkNodeSubject)
				wg.Done()
				return
			}
			defer deleteTerminatedNetworkNodeSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsDeleteTerminatedNetworkNodeSubject)

			wg.Wait()
		}()
	}
}

func consumeBlockFinalizedMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS block finalized message: %s", msg)
	var err error

	blockFinalizedMsg := &natsBlockFinalizedMsg{}
	err = json.Unmarshal(msg.Data, &blockFinalizedMsg)
	if err != nil {
		common.Log.Warningf("Failed to unmarshal block finalized message; %s", err.Error())
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
			if network.IsEthereumNetwork() {
				if err == nil {
					block, err := provide.EVMGetBlockByNumber(network.ID.String(), network.RpcURL(), blockFinalizedMsg.Block)
					if err != nil {
						err = fmt.Errorf("Failed to fetch block; %s", err.Error())
					} else if result, resultOk := block.Result.(map[string]interface{}); resultOk {
						blockTimestamp := time.Unix(int64(blockFinalizedMsg.Timestamp/1000), 0)
						finalizedAt := time.Now()

						if txs, txsOk := result["transactions"].([]interface{}); txsOk {
							for _, _tx := range txs {
								txHash := _tx.(map[string]interface{})["hash"].(string)
								common.Log.Debugf("Setting tx block and finalized_at timestamp %s on tx: %s", finalizedAt, txHash)

								params := map[string]interface{}{
									"block":           blockFinalizedMsg.Block,
									"block_timestamp": blockTimestamp,
									"finalized_at":    finalizedAt,
									"hash":            txHash,
								}

								msgPayload, _ := json.Marshal(params)
								natsConnection := common.GetDefaultNatsStreamingConnection()
								natsConnection.Publish(natsTxFinalizeSubject, msgPayload)
							}
						}
					}
				} else {
					err = fmt.Errorf("Failed to decode EVM block header; %s", err.Error())
				}
			} else {
				common.Log.Debugf("Received unhandled finalized block header type: %s", blockFinalizedMsg.Block)
			}
		}
	}

	if err != nil {
		common.Log.Warningf("Failed to handle block finalized message; %s", err.Error())
		consumer.AttemptNack(msg, natsBlockFinalizedTimeout)
	} else {
		msg.Ack()
	}
}

func consumeLoadBalancerProvisioningMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS load balancer provisioning message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal load balancer provisioning message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	if !balancerIDOk {
		common.Log.Warningf("Failed to provision load balancer; no load balancer id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		common.Log.Warningf("Failed to provision load balancer; no load balancer resolved for id: %s", balancerID)
		consumer.Nack(msg)
		return
	}

	err = balancer.provision(dbconf.DatabaseConnection())
	if err != nil {
		common.Log.Warningf("Failed to provision load balancer; %s", err.Error())
		consumer.AttemptNack(msg, natsLoadBalancerProvisioningTimeout)
	} else {
		common.Log.Debugf("Load balancer provisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerDeprovisioningMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS load balancer deprovisioning message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal load balancer deprovisioning message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	if !balancerIDOk {
		common.Log.Warningf("Failed to deprovision load balancer; no load balancer id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		common.Log.Warningf("Failed to deprovision load balancer; no load balancer resolved for id: %s", balancerID)
		consumer.Nack(msg)
		return
	}

	err = balancer.deprovision(dbconf.DatabaseConnection())
	if err != nil {
		common.Log.Warningf("Failed to deprovision load balancer; %s", err.Error())
		consumer.AttemptNack(msg, natsLoadBalancerDeprovisioningTimeout)
	} else {
		common.Log.Debugf("Load balancer deprovisioning succeeded; ACKing NATS message for balancer: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerBalanceNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS load balancer balance node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal load balancer balance node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		common.Log.Warningf("Failed to load balance network node; no load balancer id provided")
		consumer.Nack(msg)
		return
	}
	if !networkNodeIDOk {
		common.Log.Warningf("Failed to load balance network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		common.Log.Warningf("Failed to load balance network node; no load balancer resolved for id: %s", balancerID)
		consumer.Nack(msg)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to load balance network node; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
	}

	err = balancer.balanceNode(db, node)
	if err != nil {
		common.Log.Warningf("Failed to balance node on load balancer; %s", err.Error())
		consumer.AttemptNack(msg, natsLoadBalancerBalanceNodeTimeout)
	} else {
		common.Log.Debugf("Load balancer node balancing succeeded; ACKing NATS message: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeLoadBalancerUnbalanceNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS load balancer unbalance node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal load balancer unbalance node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	balancerID, balancerIDOk := params["load_balancer_id"].(string)
	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		common.Log.Warningf("Failed to unbalance network node; no load balancer id provided")
		consumer.Nack(msg)
		return
	}
	if !networkNodeIDOk {
		common.Log.Warningf("Failed to load balance network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	balancer := &LoadBalancer{}
	db.Where("id = ?", balancerID).Find(&balancer)
	if balancer == nil || balancer.ID == uuid.Nil {
		common.Log.Warningf("Failed to remove network node from load balancer; no load balancer resolved for id: %s", balancerID)
		consumer.Nack(msg)
		return
	}

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to remove network node from load balancer; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
		return
	}

	err = balancer.unbalanceNode(db, node)
	if err != nil {
		common.Log.Warningf("Failed to remove node from load balancer; %s", err.Error())
		consumer.AttemptNack(msg, natsLoadBalancerUnbalanceNodeTimeout)
	} else {
		common.Log.Debugf("Load balancer node removal succeeded; ACKing NATS message: %s", balancer.ID)
		msg.Ack()
	}
}

func consumeDeployNetworkNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !networkNodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to deploy network node; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
		return
	}

	err = node.deploy(db)
	if err != nil {
		common.Log.Warningf("Failed to deploy network node; %s", err.Error())
		consumer.AttemptNack(msg, natsDeployNetworkNodeTimeout)
		return
	}

	msg.Ack()
}

func consumeDeleteTerminatedNetworkNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS terminated network node deletion message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal terminated network node deletion message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !networkNodeIDOk {
		common.Log.Warningf("Failed to delete terminated network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to delete terminated network node; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
		return
	}

	loadBalancerAssociations := db.Model(&node).Association("LoadBalancers").Count()
	if loadBalancerAssociations > 0 {
		common.Log.Debugf("Unable to delete terminated network node: %s; %d load balancer association(s) pending", node.ID, loadBalancerAssociations)
		return
	}

	result := db.Delete(&node)
	errors := result.GetErrors()
	if len(errors) > 0 {
		err := errors[0]
		common.Log.Warningf("Failed to delete terminated network node; %s", common.StringOrNil(err.Error()))
		consumer.AttemptNack(msg, natsDeleteTerminatedNetworkNodeTimeout)
		return
	}
	msg.Ack()
}

func consumeResolveNetworkNodeHostMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !networkNodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to deploy network node; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
		return
	}

	err = node.resolveHost(db)
	if err != nil {
		common.Log.Warningf("Failed to resolve network node host; %s", err.Error())
		consumer.AttemptNack(msg, natsResolveNetworkNodeHostTimeout)
		return
	}

	msg.Ack()
}

func consumeResolveNetworkNodePeerURLMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	networkNodeID, networkNodeIDOk := params["network_node_id"].(string)

	if !networkNodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &NetworkNode{}
	db.Where("id = ?", networkNodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to resolve network node; no network node resolved for id: %s", networkNodeID)
		consumer.Nack(msg)
		return
	}

	err = node.resolvePeerURL(db)
	if err != nil {
		common.Log.Warningf("Failed to resolve network node peer url; %s", err.Error())
		consumer.AttemptNack(msg, natsResolveNetworkNodePeerURLTimeout)
		return
	}

	msg.Ack()
}
