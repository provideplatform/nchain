package consumer

import (
	"sync"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/stan.go"
)

type NetworkInterface interface {
	Send(subject string, msg []byte) (*string, error)
	Subscribe(wg *sync.WaitGroup, drainTimeout time.Duration, subject string, qgroup string, cb stan.MsgHandler, ackWait time.Duration, maxInFlight int, jwt *string)
	Setup()
	Acknowledge(msg *stan.Msg)
	GetConsumerConcurrency() uint64
}

type NatsStreaming struct {
}

func (n *NatsStreaming) Subscribe(wg *sync.WaitGroup, drainTimeout time.Duration, subject string, qgroup string, cb stan.MsgHandler, ackWait time.Duration, maxInFlight int, jwt *string) {
	natsutil.RequireNatsStreamingSubscription(wg, drainTimeout, subject, qgroup, cb, ackWait, maxInFlight, jwt)
}

func (n *NatsStreaming) Setup() {
	natsutil.EstablishSharedNatsStreamingConnection(nil)
}

func (n *NatsStreaming) Acknowledge(msg *stan.Msg) {
	msg.Ack()
}

func (n *NatsStreaming) GetConsumerConcurrency() uint64 {
	return natsutil.GetNatsConsumerConcurrency()
}

func (n *NatsStreaming) Send(subject string, msg []byte) (*string, error) {
	return natsutil.NatsStreamingPublishAsync(subject, msg)
}
