package consumer

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/nats.go"
	"github.com/provideplatform/nchain/common"
)

const defaultNatsStream = "nchain"

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

var (
	waitGroup     sync.WaitGroup
	currencyPairs = []string{}
)

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Consumer package configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsConnection(nil)
	natsutil.NatsCreateStream(defaultNatsStream, []string{
		fmt.Sprintf("%s.>", defaultNatsStream),
	})

	createNatsPacketReassemblySubscriptions(&waitGroup)
	createNatsPacketFragmentIngestSubscriptions(&waitGroup)
}

func createNatsPacketFragmentIngestSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
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
		natsutil.RequireNatsJetstreamSubscription(wg,
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

func consumePacketFragmentIngestMsg(msg *nats.Msg) {
	common.Log.Debugf("Consuming NATS packet fragment ingest message: %s", msg)

	fragment := &packetFragment{}
	err := json.Unmarshal(msg.Data, &fragment)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet fragment ingest message; %s", err.Error())
		msg.Nak()
		return
	}

	if fragment.Checksum == nil {
		common.Log.Warning("Failed to ingest packet fragment; nil checksum")
		msg.Nak()
		return
	}

	if fragment.Payload == nil {
		common.Log.Warning("Failed to ingest packet fragment; nil payload")
		msg.Nak()
		return
	}

	if fragment.Cardinality == 0 {
		common.Log.Warning("Failed to ingest packet fragment; cardinality must be greater than zero")
		msg.Nak()
		return
	}

	if fragment.Index >= fragment.Cardinality {
		common.Log.Warning("Failed to ingest packet fragment; fragment index must be less than the packet cardinality")
		msg.Nak()
		return
	}

	if fragment.Index == 0 {
		if fragment.Reassembly == nil {
			common.Log.Warning("Failed to ingest packet fragment; reassembly 'header' required within 'fragment 0' encapsulation")
			msg.Nak()
			return
		}
	}

	common.Log.Debugf("Attempting to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d)", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality)
	ingestVerified, i, ingestErr := fragment.Ingest()
	if ingestErr != nil || !ingestVerified {
		common.Log.Warningf("Failed to ingest %d-byte packet fragment containing %d-byte fragment payload (%d of %d); %s", len(msg.Data), len(*fragment.Payload), fragment.Index+1, fragment.Cardinality, ingestErr.Error())
		msg.Nak()
		return
	}

	progress := float64(*i) / float64(fragment.Cardinality)
	if progress == 1 {
		common.Log.Debugf("All fragments ingested for packet with checksum %s; dispatching reassembly message on subject: %s", *fragment.Checksum, *fragment.Reassembly.Next)

		if fragment.Reassembly == nil {
			_, err := fragment.FetchReassemblyHeader()
			if err != nil {
				common.Log.Warningf("Unable to publish packet reassembly message on subject: %s; %s", natsPacketReassembleSubject, err.Error())
				msg.Nak()
				return
			}
		}

		payload, _ := json.Marshal(fragment.Reassembly)
		err = natsutil.NatsPublish(natsPacketReassembleSubject, payload)
		if err != nil {
			common.Log.Warningf("Failed to publish %d-byte packet reassembly message on subject: %s; %s", len(payload), natsPacketReassembleSubject, err.Error())
			msg.Nak()
			return
		}
	}

	common.Log.Debugf("Successfully ingested fragment #%d with checksum %s; %d of %d total fragments needed for reassembly have been ingested (%f%%)", fragment.Index+1, *fragment.Checksum, i, fragment.Cardinality, progress)
	msg.Ack()
}

func consumePacketReassembleMsg(msg *nats.Msg) {
	common.Log.Debugf("Consuming NATS packet reassembly message: %s", msg)

	reassembly := &packetReassembly{}
	err := json.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		common.Log.Warningf("Failed to umarshal packet reassembly message; %s", err.Error())
		msg.Nak()
		return
	}

	if reassembly.Checksum == nil {
		common.Log.Warning("Failed to reassemble packet; nil checksum")
		msg.Nak()
		return
	}

	if reassembly.Next == nil {
		common.Log.Warning("Failed to reassemble packet; next hop not specified") // TODO-- relax this to support pure p2p file transfer
		msg.Nak()
		return
	}

	if reassembly.Cardinality == 0 {
		common.Log.Warning("Failed to reassemble packet; cardinality must be greater than zero")
		msg.Nak()
		return
	}

	fragSize := reassembly.Size / reassembly.Cardinality
	progress, i, err := reassembly.fragmentIngestProgress()
	if err != nil {
		common.Log.Warningf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); failed atomically reading or parsing fragment ingest progress; %s", reassembly.Size, reassembly.Cardinality, fragSize, err.Error())
		msg.Nak()
		return
	}

	if *progress == 1 {
		common.Log.Debugf("All fragments ingested for packet with checksum %s; dispatching next hop with pointer to reconstituted packet as message on subject: %s", *reassembly.Checksum, *reassembly.Next)

		reassemblyVerified, reassemblyErr := reassembly.Reassemble()
		if reassemblyErr != nil || !reassemblyVerified {
			if reassemblyErr != nil {
				common.Log.Warningf(reassemblyErr.Error())
			} else {
				common.Log.Warningf("Failed to reassemble packet with checksum %s; verification failed", *reassembly.Checksum)
			}

			msg.Nak()
			return
		}

		payload, _ := json.Marshal(reassembly)
		err = natsutil.NatsPublish(*reassembly.Next, payload)
		if err != nil {
			common.Log.Warningf("Failed to publish %d-byte next hop message on subject: %s; %s", len(payload), *reassembly.Next, err.Error())
			msg.Nak()
			return
		}

		common.Log.Debugf("Published %d-byte next hop message after successful reassembly of packet with checksum %s on subject: %s", len(payload), *reassembly.Checksum, *reassembly.Next)
		msg.Ack()
	} else {
		common.Log.Warningf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); only %d fragments ingested (%f%%)", reassembly.Size, reassembly.Cardinality, fragSize, *i, *progress)
		msg.Nak()
		return
	}
}
