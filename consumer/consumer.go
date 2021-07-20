package consumer

import (
	"encoding/hex"
	"sync"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/stan.go"
	"github.com/provideplatform/nchain/common"
	"github.com/vmihailenco/msgpack/v5"
)

// TODO: audit arbitrary max in flight & timeouts
const natsPacketFragmentIngestSubject = "prvd.packet.fragment.ingest"
const natsPacketFragmentIngestMaxInFlight = 1024
const natsPacketFragmentIngestInvocationTimeout = time.Second * 5 // FIXME!!! (see above)
const natsPacketFragmentIngestTimeout = int64(time.Second * 8)    // FIXME!!! (see above)

// TODO: audit arbitrary max in flight & timeouts; investigate dynamic timeout based on reassembled packet size
const natsPacketReassembleSubject = "prvd.packet.reassemble"
const natsPacketReassembleMaxInFlight = 256
const natsPacketReassembleInvocationTimeout = time.Second * 30 // FIXME!!! (see above)
const natsPacketReassembleTimeout = int64(time.Second * 52)    // FIXME!!! (see above)

const natsPacketCompleteSubject = "prvd.packet.complete"

var (
	waitGroup     sync.WaitGroup
	currencyPairs = []string{}
)

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
		natsutil.RequireNatsStreamingSubscription(wg,
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
		natsutil.RequireNatsStreamingSubscription(wg,
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
// TODO: Figure this out
func handleReassembly(msg *stan.Msg, reassembly *packetReassembly) {
	common.Log.Debugf("All fragments ingested for packet with checksum %s", hex.EncodeToString(*reassembly.Checksum))

	payload, _ := msgpack.Marshal(reassembly)
	err := natsutil.NatsPublish(natsPacketCompleteSubject, payload)
	if err != nil {
		common.Log.Warningf("Failed to publish %d-byte packet reassembly message on subject: %s; %s", len(payload), natsPacketReassembleSubject, err.Error())
		natsutil.AttemptNack(msg, natsPacketFragmentIngestTimeout)
	}
}

func consumePacketFragmentIngestMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS packet fragment ingest message: %s", msg)

	fragment := &packetFragment{}
	err := msgpack.Unmarshal(msg.Data, &fragment)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet fragment ingest message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	common.Log.Debugf("Attempting to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d)", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality)
	ingestVerified, i, ingestErr := fragment.Ingest()
	if ingestErr != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d); %s", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality, ingestErr.Error())
		natsutil.AttemptNack(msg, natsPacketFragmentIngestTimeout)
		return
	}

	remaining := fragment.Cardinality - *i
	if remaining == 0 {
		reassembly, err := fragment.FetchReassemblyHeader()
		if err != nil {
			common.Log.Warningf("Unable to publish packet reassembly message on subject: %s; %s", natsPacketReassembleSubject, err.Error())
			natsutil.AttemptNack(msg, natsPacketFragmentIngestTimeout)
			return
		}

		handleReassembly(msg, reassembly)
	}

	common.Log.Debugf("Successfully ingested fragment #%d with checksum %s; %d of %d total fragments needed for reassembly have been ingested", fragment.Index+1, hex.EncodeToString(*fragment.Checksum), i, fragment.Cardinality)
	msg.Ack()
}

func consumePacketReassembleMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS packet reassembly message: %s", msg)

	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet reassembly message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	fragSize := reassembly.Size / reassembly.Cardinality
	remaining, i, err := reassembly.fragmentsRemaining()
	if err != nil {
		common.Log.Warningf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); failed atomically reading or parsing fragment ingest progress; %s", reassembly.Size, reassembly.Cardinality, fragSize, err.Error())
		natsutil.AttemptNack(msg, natsPacketReassembleTimeout)
		return
	}

	if *remaining == 0 {
		handleReassembly(msg, reassembly)
	} else {
		common.Log.Warningf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); only %d fragments ingested", reassembly.Size, reassembly.Cardinality, fragSize, *i)
		natsutil.AttemptNack(msg, natsPacketReassembleTimeout)
		return
	}
}
