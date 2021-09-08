package network

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/nats-io/nats.go"
	"github.com/provideplatform/nchain/common"
	providego "github.com/provideplatform/provide-go/api"
	provide "github.com/provideplatform/provide-go/crypto"
)

const defaultNatsStream = "nchain"

const natsBlockFinalizedSubject = "nchain.block.finalized"
const natsBlockFinalizedSubjectMaxInFlight = 1024 * 30
const natsBlockFinalizedInvocationTimeout = time.Second * 60
const natsBlockFinalizedMaxDeliveries = 10

const natsResolveNodePeerURLSubject = "nchain.node.peer.resolve"
const natsResolveNodePeerURLMaxInFlight = 32
const natsResolveNodePeerURLInvocationTimeout = time.Second * 10
const natsResolveNodePeerURLMaxDeliveries = 100

const natsAddNodePeerSubject = "nchain.node.peer.add"
const natsAddNodePeerMaxInFlight = 32
const natsAddNodePeerInvocationTimeout = time.Second * 10
const natsAddNodePeerMaxDeliveries = 10

const natsRemoveNodePeerSubject = "nchain.node.peer.remove"
const natsRemoveNodePeerMaxInFlight = 32
const natsRemoveNodePeerInvocationTimeout = time.Second * 10
const natsRemoveNodePeerMaxDeliveries = 10

const natsTxFinalizeSubject = "nchain.tx.finalize"

type Block struct {
	providego.Model

	NetworkID uuid.UUID `sql:"type:uuid" json:"network_id"`
	Block     int       `json:"block"`
	Hash      string    `json:"hash"` // FIXME: should be blockhash
}

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

	natsutil.EstablishSharedNatsConnection(nil)
	natsutil.NatsCreateStream(defaultNatsStream, []string{
		fmt.Sprintf("%s.>", defaultNatsStream),
	})

	createNatsBlockFinalizedSubscriptions(&waitGroup)
	createNatsResolveNodePeerURLSubscriptions(&waitGroup)
	createNatsAddNodePeerSubscriptions(&waitGroup)
	createNatsRemoveNodePeerSubscriptions(&waitGroup)
}

func createNatsBlockFinalizedSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsBlockFinalizedInvocationTimeout,
			natsBlockFinalizedSubject,
			natsBlockFinalizedSubject,
			natsBlockFinalizedSubject,
			consumeBlockFinalizedMsg,
			natsBlockFinalizedInvocationTimeout,
			natsBlockFinalizedSubjectMaxInFlight,
			natsBlockFinalizedMaxDeliveries,
			nil,
		)
	}
}

func createNatsResolveNodePeerURLSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsResolveNodePeerURLInvocationTimeout,
			natsResolveNodePeerURLSubject,
			natsResolveNodePeerURLSubject,
			natsResolveNodePeerURLSubject,
			consumeResolveNodePeerURLMsg,
			natsResolveNodePeerURLInvocationTimeout,
			natsResolveNodePeerURLMaxInFlight,
			natsResolveNodePeerURLMaxDeliveries,
			nil,
		)
	}
}

func createNatsAddNodePeerSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsAddNodePeerInvocationTimeout,
			natsAddNodePeerSubject,
			natsAddNodePeerSubject,
			natsAddNodePeerSubject,
			consumeAddNodePeerMsg,
			natsAddNodePeerInvocationTimeout,
			natsAddNodePeerMaxInFlight,
			natsAddNodePeerMaxDeliveries,
			nil,
		)
	}
}

func createNatsRemoveNodePeerSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsRemoveNodePeerInvocationTimeout,
			natsRemoveNodePeerSubject,
			natsRemoveNodePeerSubject,
			natsRemoveNodePeerSubject,
			consumeRemoveNodePeerMsg,
			natsRemoveNodePeerInvocationTimeout,
			natsRemoveNodePeerMaxInFlight,
			natsRemoveNodePeerMaxDeliveries,
			nil,
		)
	}
}

func consumeBlockFinalizedMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from panic during NATS block finalized message handling; %s", r)
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS block finalized message: %s", msg)
	var err error

	blockFinalizedMsg := &natsBlockFinalizedMsg{}
	err = json.Unmarshal(msg.Data, &blockFinalizedMsg)
	if err != nil {
		common.Log.Warningf("failed to unmarshal block finalized message; %s", err.Error())
		msg.Nak()
		return
	}

	if blockFinalizedMsg.NetworkID == nil {
		err = fmt.Errorf("parsed %d-byte NATS block finalized message did not contain network id", len(msg.Data))
	}

	if err == nil {
		db := dbconf.DatabaseConnection()

		network := &Network{}
		db.Where("id = ?", blockFinalizedMsg.NetworkID).Find(&network)

		if network == nil || network.ID == uuid.Nil {
			err = fmt.Errorf("failed to retrieve network by id: %s", *blockFinalizedMsg.NetworkID)
		}

		if err == nil {
			if network.IsEthereumNetwork() {
				if err == nil {
					block, err := provide.EVMGetBlockByNumber(network.ID.String(), network.RPCURL(), blockFinalizedMsg.Block)
					if err != nil {
						common.Log.Warningf("failed to handle block finalized message; failed to fetch block for network id: %s; %s", network.ID.String(), err.Error())
						msg.Nak()
						return
					} else if result, resultOk := block.Result.(map[string]interface{}); resultOk {
						blockTimestamp := time.Unix(int64(blockFinalizedMsg.Timestamp/1000), 0)
						finalizedAt := time.Now()

						// save the finalized block to the db
						var minedBlock Block
						minedBlock.NetworkID = network.ID
						minedBlock.Block = int(blockFinalizedMsg.Block)
						minedBlock.Hash = *blockFinalizedMsg.BlockHash
						dbResult := db.Create(&minedBlock)
						if dbResult.RowsAffected == 0 {
							common.Log.Warningf("error saving block to db; error: %s", dbResult.Error.Error())
						}

						if txs, txsOk := result["transactions"].([]interface{}); txsOk {
							for _, _tx := range txs {
								txHash := _tx.(map[string]interface{})["hash"].(string)
								common.Log.Tracef("setting tx block (%v) and finalized_at timestamp %s on tx: %s", blockFinalizedMsg.Block, finalizedAt, txHash)

								params := map[string]interface{}{
									"block":           blockFinalizedMsg.Block,
									"block_timestamp": blockTimestamp,
									"finalized_at":    finalizedAt,
									"hash":            txHash,
								}

								msgPayload, _ := json.Marshal(params)
								_, err = natsutil.NatsJetstreamPublish(natsTxFinalizeSubject, msgPayload)
								if err != nil {
									common.Log.Warningf("failed to handle block finalized message; failed publish tx finalized event on subject %s; network: %s; %s", natsTxFinalizeSubject, network.ID.String(), err.Error())
									msg.Nak()
									return
								}
							}
						}
					}
				}
			} else {
				common.Log.Warningf("received unhandled finalized block header; network id: %s", *blockFinalizedMsg.NetworkID)
				msg.Term()
				return
			}
		}
	}

	msg.Ack()
}

func consumeResolveNodePeerURLMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS resolve node peer url message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal resolve node peer url message; %s", err.Error())
		msg.Nak()
		return
	}

	nodeID, nodeIDOk := params["node_id"].(string)

	if !nodeIDOk {
		common.Log.Warningf("failed to resolve peer url for node; no node id provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("failed to resolve node; no node resolved for id: %s", nodeID)
		msg.Term()
		return
	}

	err = node.resolvePeerURL(db)
	if err != nil {
		common.Log.Debugf("attempt to resolve node peer url did not succeed; %s", err.Error())
		msg.Term()
		return
	}

	peerURL := node.peerURL()
	if peerURL != nil {
		network := node.relatedNetwork(db)
		go network.addPeer(*peerURL)
	}

	msg.Ack()
}

func consumeAddNodePeerMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS add peer message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal add peer message; %s", err.Error())
		msg.Nak()
		return
	}

	nodeID, nodeIDOk := params["node_id"].(string)
	peerURL, peerURLOk := params["peer_url"].(string)

	if !nodeIDOk {
		common.Log.Warningf("failed to add network peer; no node id provided")
		msg.Term()
		return
	}

	if !peerURLOk {
		common.Log.Warningf("failed to add network peer; no peer url provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("failed to resolve node; no node resolved for id: %s", nodeID)
		msg.Term()
		return
	}

	err = node.addPeer(peerURL)
	if err != nil {
		common.Log.Debugf("attempt to to add network peer failed; %s", err.Error())
		msg.Nak()
		return
	}

	msg.Ack()
}

func consumeRemoveNodePeerMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS remove peer message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal remove peer message; %s", err.Error())
		msg.Nak()
		return
	}

	nodeID, nodeIDOk := params["node_id"].(string)
	peerURL, peerURLOk := params["peer_url"].(string)

	if !nodeIDOk {
		common.Log.Warningf("failed to remove network peer; no node id provided")
		msg.Term()
		return
	}

	if !peerURLOk {
		common.Log.Warningf("failed to remove network peer; no peer url provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	node := &Node{}
	db.Where("id = ?", nodeID).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.Log.Warningf("failed to resolve node; no node resolved for id: %s", nodeID)
		msg.Term()
		return
	}

	err = node.removePeer(peerURL)
	if err != nil {
		common.Log.Debugf("attempt to remove network peer failed; %s", err.Error())
		msg.Nak()
		return
	}

	msg.Ack()
}
