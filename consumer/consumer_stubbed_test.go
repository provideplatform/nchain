package consumer

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/stan.go"
	"github.com/vmihailenco/msgpack/v5"
)

//
// Stubbed NetworkInterface
//

type testQueue struct {
	queue     chan []byte
	subject   string
	connected bool
	index     uint64
}

type testStream struct {
	queues       map[string]*testQueue
	acksRecieved uint64
	onSend       func(subject string, message []byte)
}

func (queue *testQueue) waitLoop(handler stan.MsgHandler) {
	for queue.connected {
		data := <-queue.queue

		msg := new(stan.Msg)
		msg.Data = data
		msg.Subject = queue.subject
		msg.Sequence = queue.index

		atomic.AddUint64(&queue.index, 1)

		go handler(msg)
	}
}

func (sub *testStream) Send(subject string, message []byte) (*string, error) {
	if sub.onSend != nil {
		sub.onSend(subject, message)
	}

	if queue, ok := sub.queues[subject]; ok {
		go func() {
			// Add a random delay
			time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
			queue.queue <- message
		}()
	}

	return nil, nil
}

func (sub *testStream) Acknowledge(handler *stan.Msg) {
}

func (sub *testStream) Setup() {
}

func (sub *testStream) GetConsumerConcurrency() uint64 {
	return 4
}

func (sub *testStream) Subscribe(wg *sync.WaitGroup, _ time.Duration, subject string, queuegroup string, handler stan.MsgHandler, _ time.Duration, _ int, _ *string) {
	var queue *testQueue
	if q, ok := sub.queues[subject]; ok {
		queue = q
	} else {
		queue = new(testQueue)
		queue.connected = true
		queue.subject = subject
		queue.queue = make(chan []byte)
		sub.queues[subject] = queue
	}
	go queue.waitLoop(handler)
}

//
// Test Helpers
//

func setupTestNetwork() (*testStream, IDatabase) {
	rand.Seed(time.Now().UnixNano())

	network := new(testStream)
	network.queues = make(map[string]*testQueue)

	db := &Redis{}
	db.Setup()

	consumer := PacketConsumer{
		network,
		db,
		wg,
	}
	consumer.Setup()

	return network, db
}

func checkReassemblyMsg(t *testing.T, db IDatabase, msg *stan.Msg) {
	reassembly := &packetReassembly{}
	err := msgpack.Unmarshal(msg.Data, &reassembly)
	if err != nil {
		t.Errorf("Failed to umarshal packet reassembly message; %s", err.Error())
	} else {
		percentComplete, i, err := reassembly.fragmentIngestProgress()
		if err != nil {
			t.Errorf("Failed to reassemble %d-byte packet consisting of %d fragment(s); failed atomically reading or parsing fragment ingest progress; %s", reassembly.Size, reassembly.Cardinality, err.Error())
			return
		}

		if *percentComplete != 1 {
			t.Errorf("Failed to reassemble %d-byte packet consisting of %d fragment(s); only %d fragments ingested", reassembly.Size, reassembly.Cardinality, *i)
			return
		}

		assembled, err := reassembly.Reassemble(db)
		if !assembled || err != nil {
			t.Errorf("Failed to reassemble data - err: '%s'", err)
		}

		t.Logf("Recieved and reassembled %d length packet", reassembly.Size)
	}

	testDone.Done()
}

//
// Tests
//

func TestStubbedBroadcast(t *testing.T) {
	var testNetwork, db = setupTestNetwork()

	numPayloads := 25
	t.Logf("Generating %d payloads...", numPayloads)
	var payloads [][]byte
	for i := 1; i <= numPayloads; i++ {
		payloads = append(payloads, generateRandomBytes(1024*64*i))
	}
	t.Log("Generated payloads")

	t.Log("Subscribing..")
	var wg sync.WaitGroup
	testNetwork.Subscribe(&wg,
		time.Second*200,
		packetCompleteSubject,
		packetCompleteSubject,
		func(msg *stan.Msg) { checkReassemblyMsg(t, db, msg) },
		time.Second*200,
		1024,
		nil,
	)
	t.Log("Subscribed.")

	testDone.Add(numPayloads)

	t.Log("Sending payloads..")
	for _, payload := range payloads {
		// Add a random delay
		time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
		go BroadcastFragments(testNetwork, payload, true, nil)
	}

	testDone.Wait()
	t.Log("Test complete.")
}
