package network

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/stan.go"
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

const natsDeployNodeSubject = "goldmine.node.deploy"
const natsDeployNodeMaxInFlight = 64
const natsDeployNodeInvocationTimeout = time.Minute * 1
const natsDeployNodeTimeout = int64(1000 * 60 * 10)

const natsDeleteTerminatedNodeSubject = "goldmine.node.delete"
const natsDeleteTerminatedNodeMaxInFlight = 64
const natsDeleteTerminatedNodeInvocationTimeout = time.Minute * 1
const natsDeleteTerminatedNodeTimeout = int64(1000 * 60 * 10)

const natsResolveNodeHostSubject = "goldmine.node.host.resolve"
const natsResolveNodeHostMaxInFlight = 64
const natsResolveNodeHostInvocationTimeout = time.Second * 10
const natsResolveNodeHostTimeout = int64(1000 * 60 * 10)

const natsResolveNodePeerURLSubject = "goldmine.node.peer.resolve"
const natsResolveNodePeerURLMaxInFlight = 64
const natsResolveNodePeerURLInvocationTimeout = time.Second * 10
const natsResolveNodePeerURLTimeout = int64(1000 * 60 * 10)

const natsAddNodePeerSubject = "goldmine.node.peer.add"
const natsAddNodePeerMaxInFlight = 64
const natsAddNodePeerInvocationTimeout = time.Second * 10
const natsAddNodePeerTimeout = int64(1000 * 60 * 10)

const natsRemoveNodePeerSubject = "goldmine.node.peer.remove"
const natsRemoveNodePeerMaxInFlight = 64
const natsRemoveNodePeerInvocationTimeout = time.Second * 10
const natsRemoveNodePeerTimeout = int64(1000 * 60 * 10)

const natsTxFinalizeSubject = "goldmine.tx.finalize"

type natsBlockFinalizedMsg struct {
	NetworkID *string `json:"network_id"`
	Block     uint64  `json:"block"`
	BlockHash *string `json:"blockhash"`
	Timestamp uint64  `json:"timestamp"`
}

var waitGroup sync.WaitGroup

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Network package consumer configured to skip NATS streaming subscription setup")
		return
	}

	createNatsBlockFinalizedSubscriptions(&waitGroup)
	createNatsLoadBalancerProvisioningSubscriptions(&waitGroup)
	createNatsLoadBalancerDeprovisioningSubscriptions(&waitGroup)
	createNatsLoadBalancerBalanceNodeSubscriptions(&waitGroup)
	createNatsLoadBalancerUnbalanceNodeSubscriptions(&waitGroup)
	createNatsDeployNodeSubscriptions(&waitGroup)
	createNatsDeleteTerminatedNodeSubscriptions(&waitGroup)
	createNatsResolveNodeHostSubscriptions(&waitGroup)
	createNatsResolveNodePeerURLSubscriptions(&waitGroup)
	createNatsAddNodePeerSubscriptions(&waitGroup)
	createNatsRemoveNodePeerSubscriptions(&waitGroup)
}

func createNatsBlockFinalizedSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsBlockFinalizedInvocationTimeout,
			natsBlockFinalizedSubject,
			natsBlockFinalizedSubject,
			consumeBlockFinalizedMsg,
			natsBlockFinalizedInvocationTimeout,
			natsBlockFinalizedSubjectMaxInFlight,
		)
	}
}

func createNatsLoadBalancerProvisioningSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerProvisioningSubject,
			natsLoadBalancerProvisioningSubject,
			consumeLoadBalancerProvisioningMsg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerProvisioningMaxInFlight,
		)
	}
}

func createNatsLoadBalancerDeprovisioningSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerDeprovisioningSubject,
			natsLoadBalancerDeprovisioningSubject,
			consumeLoadBalancerDeprovisioningMsg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerDeprovisioningMaxInFlight,
		)
	}
}

func createNatsLoadBalancerBalanceNodeSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerBalanceNodeSubject,
			natsLoadBalancerBalanceNodeSubject,
			consumeLoadBalancerBalanceNodeMsg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerBalanceNodeMaxInFlight,
		)
	}
}

func createNatsLoadBalancerUnbalanceNodeSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerUnbalanceNodeSubject,
			natsLoadBalancerUnbalanceNodeSubject,
			consumeLoadBalancerUnbalanceNodeMsg,
			natsLoadBalancerInvocationTimeout,
			natsLoadBalancerUnbalanceNodeMaxInFlight,
		)
	}
}

func createNatsDeployNodeSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsDeployNodeInvocationTimeout,
			natsDeployNodeSubject,
			natsDeployNodeSubject,
			consumeDeployNodeMsg,
			natsDeployNodeInvocationTimeout,
			natsDeployNodeMaxInFlight,
		)
	}
}

func createNatsResolveNodeHostSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsResolveNodeHostInvocationTimeout,
			natsResolveNodeHostSubject,
			natsResolveNodeHostSubject,
			consumeResolveNodeHostMsg,
			natsResolveNodeHostInvocationTimeout,
			natsResolveNodeHostMaxInFlight,
		)
	}
}

func createNatsResolveNodePeerURLSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsResolveNodePeerURLInvocationTimeout,
			natsResolveNodePeerURLSubject,
			natsResolveNodePeerURLSubject,
			consumeResolveNodePeerURLMsg,
			natsResolveNodePeerURLInvocationTimeout,
			natsResolveNodePeerURLMaxInFlight,
		)
	}
}

func createNatsDeleteTerminatedNodeSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsDeleteTerminatedNodeInvocationTimeout,
			natsDeleteTerminatedNodeSubject,
			natsDeleteTerminatedNodeSubject,
			consumeDeleteTerminatedNodeMsg,
			natsDeleteTerminatedNodeInvocationTimeout,
			natsDeleteTerminatedNodeMaxInFlight,
		)
	}
}

func createNatsAddNodePeerSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsAddNodePeerInvocationTimeout,
			natsAddNodePeerSubject,
			natsAddNodePeerSubject,
			consumeAddNodePeerMsg,
			natsAddNodePeerInvocationTimeout,
			natsAddNodePeerMaxInFlight,
		)
	}
}

func createNatsRemoveNodePeerSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsRemoveNodePeerInvocationTimeout,
			natsRemoveNodePeerSubject,
			natsRemoveNodePeerSubject,
			consumeRemoveNodePeerMsg,
			natsRemoveNodePeerInvocationTimeout,
			natsRemoveNodePeerMaxInFlight,
		)
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
					block, err := provide.EVMGetBlockByNumber(network.ID.String(), network.RPCURL(), blockFinalizedMsg.Block)
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
								common.NATSPublish(natsTxFinalizeSubject, msgPayload)
							}
						}
					}
				} else {
					err = fmt.Errorf("Failed to decode EVM block header; %s", err.Error())
				}
			} else {
				common.Log.Warningf("Received unhandled finalized block header; network id: %s", *blockFinalizedMsg.NetworkID)
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
	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		common.Log.Warningf("Failed to load balance network node; no load balancer id provided")
		consumer.Nack(msg)
		return
	}
	if !nodeIDOk {
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

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to load balance network node; no network node resolved for id: %s", nodeID)
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
	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !balancerIDOk {
		common.Log.Warningf("Failed to unbalance network node; no load balancer id provided")
		consumer.Nack(msg)
		return
	}
	if !nodeIDOk {
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

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to remove network node from load balancer; no network node resolved for id: %s", nodeID)
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

func consumeDeployNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to deploy network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	err = node.deploy(db)
	if err != nil {
		common.Log.Warningf("Failed to deploy network node; %s", err.Error())
		consumer.AttemptNack(msg, natsDeployNodeTimeout)
		return
	}

	msg.Ack()
}

func consumeDeleteTerminatedNodeMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS terminated network node deletion message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal terminated network node deletion message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to delete terminated network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to delete terminated network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	loadBalancerAssociations := db.Model(&node).Association("LoadBalancers").Count()
	if loadBalancerAssociations > 0 {
		common.Log.Debugf("Unable to delete terminated network node: %s; %d load balancer association pending", node.ID, loadBalancerAssociations)
		node.unbalance(db)
		return
	}

	peerURL := node.peerURL()
	if peerURL != nil {
		network := node.relatedNetwork(db)
		go network.removePeer(*peerURL)
	}

	result := db.Delete(&node)
	errors := result.GetErrors()
	if len(errors) > 0 {
		err := errors[0]
		common.Log.Warningf("Failed to delete terminated network node; %s", err.Error())
		consumer.AttemptNack(msg, natsDeleteTerminatedNodeTimeout)
		return
	}

	msg.Ack()
}

func consumeResolveNodeHostMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to deploy network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	err = node.resolveHost(db)
	if err != nil {
		common.Log.Warningf("Failed to resolve network node host; %s", err.Error())
		consumer.AttemptNack(msg, natsResolveNodeHostTimeout)
		return
	}

	msg.Ack()
}

func consumeResolveNodePeerURLMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS deploy network node message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal deploy network node message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to deploy network node; no network node id provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to resolve network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	err = node.resolvePeerURL(db)
	if err != nil {
		common.Log.Warningf("Failed to resolve network node peer url; %s", err.Error())
		consumer.AttemptNack(msg, natsResolveNodePeerURLTimeout)
		return
	}

	peerURL := node.peerURL()
	if peerURL != nil {
		network := node.relatedNetwork(db)
		go network.addPeer(*peerURL)
	}

	msg.Ack()
}

func consumeAddNodePeerMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS add peer message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal add peer message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)
	peerURL, peerURLOk := params["peer_url"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to add network peer; no network node id provided")
		consumer.Nack(msg)
		return
	}

	if !peerURLOk {
		common.Log.Warningf("Failed to add network peer; no peer url provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to resolve network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	err = node.addPeer(peerURL)
	if err != nil {
		common.Log.Warningf("Failed to add network peer; %s", err.Error())
		consumer.AttemptNack(msg, natsAddNodePeerTimeout)
		return
	}

	msg.Ack()
}

func consumeRemoveNodePeerMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS remove peer message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal remove peer message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	nodeID, nodeIDOk := params["network_node_id"].(string)
	peerURL, peerURLOk := params["peer_url"].(string)

	if !nodeIDOk {
		common.Log.Warningf("Failed to remove network peer; no network node id provided")
		consumer.Nack(msg)
		return
	}

	if !peerURLOk {
		common.Log.Warningf("Failed to remove network peer; no peer url provided")
		consumer.Nack(msg)
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("Failed to resolve network node; no network node resolved for id: %s", nodeID)
		consumer.Nack(msg)
		return
	}

	err = node.removePeer(peerURL)
	if err != nil {
		common.Log.Warningf("Failed to remove network peer; %s", err.Error())
		consumer.AttemptNack(msg, natsRemoveNodePeerTimeout)
		return
	}

	msg.Ack()
}
