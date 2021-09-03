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
const packetFragmentIngestSubject = "prvd.packet.fragment.ingest"
const packetFragmentIngestMaxInFlight = 1024
const packetFragmentIngestInvocationTimeout = time.Second * 5 // FIXME!!! (see above)
const packetFragmentIngestTimeout = int64(time.Second * 8)    // FIXME!!! (see above)

// TODO: audit arbitrary max in flight & timeouts; investigate dynamic timeout based on reassembled packet size
const packetReassembleSubject = "prvd.packet.reassemble.ingest"
const packetReassembleMaxInFlight = 256
const packetReassembleInvocationTimeout = time.Second * 30 // FIXME!!! (see above)
const packetReassembleTimeout = int64(time.Second * 52)    // FIXME!!! (see above)

const packetCompleteSubject = "prvd.packet.reassemble.finalize"

// PacketConsumer ingests packets that have been broadcasted out, and sends completion messages when a whole message has been ingested
type PacketConsumer struct {
	network   INetwork
	db        IDatabase
	waitGroup sync.WaitGroup
}

// Setup initialises a PacketConsumer and sets up the network as well as subscribing to the fragment and reassembly messages
func (consumer *PacketConsumer) Setup() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Consumer package configured to skip NATS streaming subscription setup")
		return
	}

	consumer.network.Setup()
	consumer.setupFragmentIngestSubscription()
	consumer.setupReassemblyIngestSubscription()
}

func (consumer *PacketConsumer) setupFragmentIngestSubscription() {
	for i := uint64(0); i < consumer.network.GetConsumerConcurrency(); i++ {
		consumer.network.Subscribe(&consumer.waitGroup,
			packetFragmentIngestInvocationTimeout,
			packetFragmentIngestSubject,
			packetFragmentIngestSubject,
			consumer.consumePacketFragmentIngestMsg,
			packetFragmentIngestInvocationTimeout,
			packetFragmentIngestMaxInFlight,
			nil,
		)
	}
}

func (consumer *PacketConsumer) setupReassemblyIngestSubscription() {
	for i := uint64(0); i < consumer.network.GetConsumerConcurrency(); i++ {
		consumer.network.Subscribe(&consumer.waitGroup,
			packetReassembleInvocationTimeout,
			packetReassembleSubject,
			packetReassembleSubject,
			consumer.consumePacketReassembleMsg,
			packetReassembleInvocationTimeout,
			packetReassembleMaxInFlight,
			nil,
		)
	}
}

// Common reassembly code - this is called once all fragments & the header have been consumed
func (consumer *PacketConsumer) handleReassembly(msg *stan.Msg, reassembly *packetReassembly) {
	common.Log.Debugf("All fragments ingested for packet with checksum %s", hex.EncodeToString(*reassembly.Checksum))

	payload, _ := msgpack.Marshal(reassembly)
	_, err := consumer.network.Send(packetCompleteSubject, payload)
	if err != nil {
		common.Log.Warningf("Failed to publish %d-byte packet reassembly message on subject: %s; %s", len(payload), packetReassembleSubject, err.Error())
		natsutil.AttemptNack(msg, packetFragmentIngestTimeout)
	}
}

func (consumer *PacketConsumer) consumePacketFragmentIngestMsg(msg *stan.Msg) {
	fragment := &packetFragment{}
	err := msgpack.Unmarshal(msg.Data, &fragment)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet fragment ingest message; %s", err.Error())
		consumer.network.Acknowledge(msg) // Acknowledge the bad packet so it's not resent
		return
	}

	ingestVerified, ingestCounter, err := fragment.Ingest(consumer.db)
	if err != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d); %s", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality, err.Error())
		// No acknowledgement here - so we can try again
		return
	}

	// Add one to cardinality as we need to account for the reassembly message too
	common.Log.Tracef("Successfully ingested fragment #%d with checksum %s; %d of %d total fragments needed for reassembly have been ingested", fragment.Index, hex.EncodeToString(*fragment.Checksum), *ingestCounter, fragment.Cardinality+1)
	remaining := (fragment.Cardinality + 1) - *ingestCounter
	if remaining == 0 {
		reassembly, err := fragment.FetchReassemblyHeader(consumer.db)
		if err != nil {
			common.Log.Warningf("Unable to publish packet reassembly message on subject: %s; %s", packetReassembleSubject, err.Error())
		} else {
			consumer.handleReassembly(msg, reassembly)
		}
	}

	consumer.network.Acknowledge(msg)
}

func (consumer *PacketConsumer) consumePacketReassembleMsg(msg *stan.Msg) {
	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet reassembly message; %s", err.Error())
		consumer.network.Acknowledge(msg) // Acknowledge the bad packet so it's not resent
		return
	}

	ingestVerified, ingestCounter, err := reassembly.Ingest(consumer.db)
	if err != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet header; %s", len(msg.Data), err.Error())
		// No acknowledgement here - so we can try again
		return
	}

	common.Log.Tracef("Successfully ingested reassembly header. %d of %d total fragments needed for reassembly have been ingested", *ingestCounter, reassembly.Cardinality+1)
	remaining := (reassembly.Cardinality + 1) - *ingestCounter
	if remaining == 0 {
		consumer.handleReassembly(msg, reassembly)
	}

	consumer.network.Acknowledge(msg)
}
