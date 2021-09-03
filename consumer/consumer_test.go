package consumer

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/stan.go"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	testDone sync.WaitGroup
	wg       sync.WaitGroup
)

func setupRealNatsNetwork() (INetwork, IDatabase) {
	var db = &Redis{}
	db.Setup()

	var nats = &NatsStreaming{}
	consumer := PacketConsumer{
		nats,
		db,
		wg,
	}

	consumer.Setup()

	return nats, db
}

func consumePacketCompleteMsg(t *testing.T, network INetwork, db IDatabase, msg *stan.Msg, expectedPayload *[]byte) {
	network.Acknowledge(msg)

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

	if *percentComplete < 1 {
		t.Errorf("Failed to reassemble %d-byte packet consisting of %d fragment(s); only %d fragments ingested", reassembly.Size, reassembly.Cardinality, *i)
		return
	}

	assembled, err := reassembly.Reassemble(db)
	if !assembled || err != nil {
		t.Errorf("Failed to reassemble data - err: '%s'", err)
	}

	// Reassemble does a checksum comparison but just as an extra test we compare payload to expected
	if bytes.Compare(*expectedPayload, *reassembly.Payload) != 0 {
		t.Error("Reassembled payload did not match expected")
	}

	testDone.Done()
}

// TestConsumerBroadcastAndReassemble uses a real NATS-Streaming connection to test broadcast and reassembly of data
func TestConsumerBroadcastAndReassemble(t *testing.T) {
	nats, db := setupRealNatsNetwork()
	payload := generateRandomBytes(1024 * 16 * 64)

	nats.Subscribe(&wg,
		time.Second*5,
		packetCompleteSubject,
		packetCompleteSubject,
		func(msg *stan.Msg) {
			consumePacketCompleteMsg(t, nats, db, msg, &payload)
		},
		time.Second*5,
		1024,
		nil,
	)

	// TODO: this is crap
	time.Sleep(time.Second * 6)

	testDone.Add(1)

	err := BroadcastFragments(nats, payload, true, nil)
	if err != nil {
		t.Errorf("BroadcastFragments() error; %s", err.Error())
	}

	testDone.Wait()
}
