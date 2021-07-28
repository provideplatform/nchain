package consumer

import (
	"encoding/hex"
	"sync"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/stan.go"
	"github.com/provideplatform/nchain/common"
)

// TODO: audit arbitrary max in flight & timeouts
const natsPacketFragmentIngestSubject = "prvd.packet.fragment.ingest"
const natsPacketFragmentIngestMaxInFlight = 1024
const natsPacketFragmentIngestInvocationTimeout = time.Second * 5 // FIXME!!! (see above)
const natsPacketFragmentIngestTimeout = int64(time.Second * 8)    // FIXME!!! (see above)

// TODO: audit arbitrary max in flight & timeouts; investigate dynamic timeout based on reassembled packet size
const natsPacketReassembleSubject = "prvd.packet.reassemble.ingest"
const natsPacketReassembleMaxInFlight = 256
const natsPacketReassembleInvocationTimeout = time.Second * 30 // FIXME!!! (see above)
const natsPacketReassembleTimeout = int64(time.Second * 52)    // FIXME!!! (see above)

const natsPacketCompleteSubject = "prvd.packet.reassemble.finalize"

var (
	waitGroup     sync.WaitGroup
	currencyPairs = []string{}
)

type subscribeFn = func(*sync.WaitGroup, time.Duration, string, string, stan.MsgHandler, time.Duration, int, *string)

var subscribe subscribeFn = natsutil.RequireNatsStreamingSubscription

// setSubscribeFn sets the implementation that will be used to listen for packets (default is natsutil.RequireNatsStreamingSubscription)
func setSubscribeFunction(function subscribeFn) {
	subscribe = function
}

func natsAck(msg *stan.Msg) {
	msg.Ack()
}

type ackFn = func(*stan.Msg)

var ack ackFn = natsAck

func setAckFunction(function ackFn) {
	ack = function
}

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Consumer package configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsStreamingConnection(nil)

	createNatsPacketReassemblySubscriptions(&waitGroup)
	createNatsPacketFragmentIngestSubscriptions(&waitGroup)
}

func createNatsPacketFragmentIngestSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		subscribe(wg,
			natsPacketFragmentIngestInvocationTimeout,
			natsPacketFragmentIngestSubject,
			natsPacketFragmentIngestSubject,
			consumePacketFragmentIngestMsg,
			natsPacketFragmentIngestInvocationTimeout,
			natsPacketFragmentIngestMaxInFlight,
			nil,
		)
	}
}

func createNatsPacketReassemblySubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		subscribe(wg,
			natsPacketReassembleInvocationTimeout,
			natsPacketReassembleSubject,
			natsPacketReassembleSubject,
			consumePacketReassembleMsg,
			natsPacketReassembleInvocationTimeout,
			natsPacketReassembleMaxInFlight,
			nil,
		)
	}
}

// Common reassembly code - this is called once all fragments & the header have been consumed
func handleReassembly(msg *stan.Msg, reassembly *packetReassembly) {
	common.Log.Debugf("All fragments ingested for packet with checksum %s", hex.EncodeToString(*reassembly.Checksum))

	payload, _ := msgpack.Marshal(reassembly)
	_, err := streamingPublish(natsPacketCompleteSubject, payload)
	if err != nil {
		common.Log.Warningf("Failed to publish %d-byte packet reassembly message on subject: %s; %s", len(payload), natsPacketReassembleSubject, err.Error())
		natsutil.AttemptNack(msg, natsPacketFragmentIngestTimeout)
	}
}

func consumePacketFragmentIngestMsg(msg *stan.Msg) {
	fragment := &packetFragment{}
	err := msgpack.Unmarshal(msg.Data, &fragment)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet fragment ingest message; %s", err.Error())
		ack(msg) // Acknowledge the bad packet so it's not resent
		return
	}

	ingestVerified, ingestCounter, err := fragment.Ingest()
	if err != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d); %s", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality, err.Error())
		// No acknowledgement here - so we can try again
		return
	}

	// Add one to cardinality as we need to account for the reassembly message too
	common.Log.Debugf("Successfully ingested fragment #%d with checksum %s; %d of %d total fragments needed for reassembly have been ingested", fragment.Index, hex.EncodeToString(*fragment.Checksum), *ingestCounter, fragment.Cardinality+1)
	remaining := (fragment.Cardinality + 1) - *ingestCounter
	if remaining == 0 {
		reassembly, err := fragment.FetchReassemblyHeader()
		if err != nil {
			common.Log.Warningf("Unable to publish packet reassembly message on subject: %s; %s", natsPacketReassembleSubject, err.Error())
		} else {
			handleReassembly(msg, reassembly)
		}
	}

	ack(msg)
}

func consumePacketReassembleMsg(msg *stan.Msg) {
	common.Log.Debugf("Recieved assembly message: Seq:%d, RedeliveryCount: %d", msg.Sequence, msg.RedeliveryCount)

	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet reassembly message; %s", err.Error())
		ack(msg) // Acknowledge the bad packet so it's not resent
		return
	}

	ingestVerified, ingestCounter, err := reassembly.Ingest()
	if err != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet header; %s", len(msg.Data), err.Error())
		// No acknowledgement here - so we can try again
		return
	}

	common.Log.Debugf("Successfully ingested reassembly header. %d of %d total fragments needed for reassembly have been ingested", *ingestCounter, reassembly.Cardinality+1)
	remaining := (reassembly.Cardinality + 1) - *ingestCounter
	if remaining == 0 {
		handleReassembly(msg, reassembly)
	}

	ack(msg)
}
