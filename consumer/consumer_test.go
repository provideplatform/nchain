package consumer

import (
	"sync"
	"testing"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/stan.go"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	testDone sync.WaitGroup
)

func consumePacketCompleteMsg(t *testing.T, msg *stan.Msg) {
	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		t.Errorf("Failed to umarshal packet reassembly message; %s", err.Error())
		natsutil.Nack(msg)
		return
	}

	fragSize := reassembly.Size / reassembly.Cardinality
	percentComplete, i, err := reassembly.fragmentIngestProgress()
	if err != nil {
		t.Errorf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); failed atomically reading or parsing fragment ingest progress; %s", reassembly.Size, reassembly.Cardinality, fragSize, err.Error())
		natsutil.AttemptNack(msg, natsPacketReassembleTimeout)
		return
	}

	if *percentComplete != 1 {
		t.Errorf("Failed to reassemble %d-byte packet consisting of %d %d-byte fragment(s); only %d fragments ingested", reassembly.Size, reassembly.Cardinality, fragSize, *i)
		natsutil.AttemptNack(msg, natsPacketReassembleTimeout)
		return
	}

	testDone.Done()
	msg.Ack()
}

func TestConsumerBroadcastAndReassemble(t *testing.T) {
	payload := generateRandomBytes(1024 * 16)
	setupRedis()

	var wg sync.WaitGroup
	natsutil.RequireNatsStreamingSubscription(&wg,
		time.Second*200,
		natsPacketCompleteSubject,
		natsPacketCompleteSubject,
		func(msg *stan.Msg) { consumePacketCompleteMsg(t, msg) },
		time.Second*200,
		1024,
		nil,
	)

	// TODO: this is crap
	time.Sleep(time.Second * 6)

	testDone.Add(1)

	err := BroadcastFragments(payload, true)
	if err != nil {
		t.Errorf("BroadcastFragments() error; %s", err.Error())
	}

	testDone.Wait()
}
