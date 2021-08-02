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

var dataStream = setupStub()

type testQueue struct {
	queue     chan []byte
	subject   string
	connected bool
	index     uint64
}

type testStream struct {
	queues       map[string]*testQueue
	acksRecieved uint64
}

func (queue *testQueue) enqueue(data []byte) {
	// Add a random delay
	time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)

	queue.queue <- data
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

func (sub *testStream) send(subject string, message []byte) (*string, error) {
	if queue, ok := sub.queues[subject]; ok {
		go queue.enqueue(message)
	}

	return nil, nil
}

func (sub *testStream) ack(handler *stan.Msg) {
	atomic.AddUint64(&sub.acksRecieved, 1)
}

func (sub *testStream) subscribe(wg *sync.WaitGroup, _ time.Duration, subject string, queuegroup string, handler stan.MsgHandler, _ time.Duration, _ int, _ *string) {
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

func setupStub() *testStream {
	rand.Seed(time.Now().UnixNano())
	sub := new(testStream)
	sub.queues = make(map[string]*testQueue)
	setSubscribeFunction(sub.subscribe)
	setBroadcastPublishFunction(sub.send)
	setAckFunction(sub.ack)
	return sub
}

func checkReassemblyMsg(t *testing.T, msg *stan.Msg) {
	t.Log("Recieved packet complete message")

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

		assembled, err := reassembly.Reassemble()
		if !assembled || err != nil {
			t.Errorf("Failed to reassemble data - err: '%s'", err)
		}
	}

	testDone.Done()
}

func TestStubbedBroadcast(t *testing.T) {
	payload1 := generateRandomBytes(1024 * 128)
	payload2 := generateRandomBytes(1024 * 64)
	payload3 := generateRandomBytes(1024 * 256)
	setupRedis()

	var wg sync.WaitGroup
	dataStream.subscribe(&wg,
		time.Second*200,
		natsPacketCompleteSubject,
		natsPacketCompleteSubject,
		func(msg *stan.Msg) { checkReassemblyMsg(t, msg) },
		time.Second*200,
		1024,
		nil,
	)

	testDone.Add(3)

	go BroadcastFragments(payload1, true)
	go BroadcastFragments(payload2, true)
	go BroadcastFragments(payload3, true)

	testDone.Wait()
}
