package consumer

import (
	"bytes"
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

func consumePacketCompleteMsg(t *testing.T, msg *stan.Msg, expectedPayload *[]byte) {
	ack(msg)

	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		t.Errorf("Failed to umarshal packet reassembly message; %s", err.Error())
		return
	}

	percentComplete, i, err := reassembly.fragmentIngestProgress()
	if err != nil {
		t.Errorf("Failed to reassemble %d-byte packet consisting of %d fragment(s); failed atomically reading or parsing fragment ingest progress; %s", reassembly.Size, reassembly.Cardinality, err.Error())
		return
	}

	if *percentComplete != 1 {
		t.Errorf("Failed to reassemble %d-byte packet consisting of %d fragment(s); only %d fragments ingested", reassembly.Size, reassembly.Cardinality, *i)
		return
	}

	assembled, err := reassembly.Reassemble()
	if !assembled || err != nil {
		t.Errorf("Failed to reassemble data - err: '%s'", err)
	}

	// In theory Reassemble does a checksum comparison but just as an extra test we compare payload to expected
	if bytes.Compare(*expectedPayload, *reassembly.Payload) != 0 {
		t.Error("Reassembled payload did not match expected")
	}

	testDone.Done()
}

// TestConsumerBroadcastAndReassemble uses a real NATS-Streaming connection to test broadcast and reassembly of data
func TestConsumerBroadcastAndReassemble(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	payload := generateRandomBytes(1024 * 16)
	setupRedis()

	var wg sync.WaitGroup
	natsutil.RequireNatsStreamingSubscription(&wg,
		time.Second*200,
		natsPacketCompleteSubject,
		natsPacketCompleteSubject,
		func(msg *stan.Msg) { consumePacketCompleteMsg(t, msg, &payload) },
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
