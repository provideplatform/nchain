package consumer

import (
	"sync"
	"time"

	"github.com/kthomas/go-natsutil"
	"github.com/nats-io/stan.go"
)

// INetwork abstracts all of the network requests
type INetwork interface {
	Send(subject string, msg []byte) (*string, error)
	Subscribe(wg *sync.WaitGroup, drainTimeout time.Duration, subject string, qgroup string, cb stan.MsgHandler, ackWait time.Duration, maxInFlight int, jwt *string)
	Setup()
	Acknowledge(msg *stan.Msg)
	GetConsumerConcurrency() uint64
}

// NatsStreaming is a NATS wrapper for INetwork
type NatsStreaming struct {
}

// Subscribe will subscribe to the given Nats Streaming subject
func (n *NatsStreaming) Subscribe(wg *sync.WaitGroup, drainTimeout time.Duration, subject string, qgroup string, cb stan.MsgHandler, ackWait time.Duration, maxInFlight int, jwt *string) {
	natsutil.RequireNatsStreamingSubscription(wg, drainTimeout, subject, qgroup, cb, ackWait, maxInFlight, jwt)
}

// Setup will establish the Nats Streaming connection
func (n *NatsStreaming) Setup() {
	natsutil.EstablishSharedNatsStreamingConnection(nil)
}

// Acknowledge acks the given message
func (n *NatsStreaming) Acknowledge(msg *stan.Msg) {
	msg.Ack()
}

// GetConsumerConcurrency controls how many concurrent subscribers should be setup when listening for messages
func (n *NatsStreaming) GetConsumerConcurrency() uint64 {
	return natsutil.GetNatsConsumerConcurrency()
}

// Send sends an async Nats Streaming message
func (n *NatsStreaming) Send(subject string, msg []byte) (*string, error) {
	return natsutil.NatsStreamingPublishAsync(subject, msg)
}
