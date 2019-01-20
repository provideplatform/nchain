package amqputil

import (
	"errors"
	"fmt"

	"github.com/kthomas/go-logger"
	"github.com/streadway/amqp"
)

type Consumer struct {
	config *AmqpConfig
	delegate ConsumerDelegate
	tag string

	log *logger.Logger

	rmqConn *amqp.Connection
	rmqChan *amqp.Channel
}

type ConsumerDelegate interface {
	Deliver(*amqp.Delivery) error
}

func (c *Consumer) bindQueue() error {
	err := c.declareQueue()
	if err == nil {
		c.log.Debugf("Attempting to bind %s queue to routing key %s in %s AMQP exchange", c.config.AmqpQueue, c.config.AmqpBindingKey, c.config.AmqpExchange)
		err = c.rmqChan.QueueBind(
			c.config.AmqpQueue,		// queue
			c.config.AmqpBindingKey, 	// binding key
			c.config.AmqpExchange,		// exchange
			false, 				// no-wait
			nil,				// arguments
		)
		if err == nil {
			c.log.Debugf("Bound %s queue to %s routing key in %s AMQP exchange", c.config.AmqpQueue, c.config.AmqpBindingKey, c.config.AmqpExchange)
		}
	}
	return err
}

func (c *Consumer) declareQueue() error {
	c.log.Debugf("Attempting to declare %s queue in %s AMQP exchange", c.config.AmqpQueue, c.config.AmqpBindingKey)
	_, err := c.rmqChan.QueueDeclare(
		c.config.AmqpQueue,	// queue
		true, 			// durable
		false,			// auto-delete
		false,			// exclusive
		false,			// no-wait
		nil,			// arguments
	)
	if err != nil {
		c.log.Warningf("Failed to declare %s queue in %s AMQP exchange", c.config.AmqpQueue, c.config.AmqpExchange)
	}
	return err
}

func (c *Consumer) Run() error {
	q, err := c.rmqChan.Consume(
		c.config.AmqpQueue,	// queue
		c.tag,			// consumer tag
		false,			// auto-ack
		false,			// exclusive
		false,			// no-local
		false,			// no-wait
		nil,			// arguments
	)
	if err != nil {
		c.log.Errorf("Failed to consume messages from AMQP queue %s", c.config.AmqpQueue)
		return err
	}

	chanClosed := c.rmqChan.NotifyClose(make(chan *amqp.Error, 1))
	chanCanceled := c.rmqChan.NotifyCancel(make(chan string, 1))
	chanReturnedMsg := c.rmqChan.NotifyReturn(make(chan amqp.Return, 1))

	// TODO: add shutdown channel

	c.log.Debugf("Consuming messages from AMQP queue %s", c.config.AmqpQueue)
	for {
		select {
		case msg, ok := <-q:
			if ok {
				c.log.Debugf("Attempting to deliver %v-byte AMQP message received from queue %s: %s", len(msg.Body), c.config.AmqpQueue, msg)
				if err := c.delegate.Deliver(&msg); err != nil {
					c.log.Warningf("Delivery of AMQP message likely resulted in failure; %s", err)
					requeue := (err == AmqpDeliveryErrRequireRequeue) || !msg.Redelivered
					err := msg.Nack(false, requeue)
					if err != nil {
						c.log.Errorf("Failed to NACK AMQP message with delivery tag %s; %s", msg.DeliveryTag, err)
					}
				} else {
					c.log.Debugf("AMQP message delivered successfuly; delivery tag: %s", msg.DeliveryTag)
				}
			} else {
				c.log.Debugf("Closed AMQP consumer channel queue %s", c.config.AmqpQueue)
			}

		case tag := <-chanCanceled:
			msg := fmt.Sprintf("RabbitMQ channel canceled subscription with tag: %s", tag)
			c.log.Errorf(msg)
			return errors.New(msg)

		case err := <-chanClosed:
			c.log.Errorf("RabbitMQ channel closed; %s", err)
			return err

		case msg := <-chanReturnedMsg:
			c.log.Warningf("RabbitMQ message returned: %s", msg)
		}
	}
}

func NewConsumer(lg *logger.Logger, config *AmqpConfig, tag string, delegate ConsumerDelegate) (*Consumer, error) {
	c := new(Consumer)
	c.log = lg.Clone()

	c.config = config

	c.tag = tag
	if c.tag == "" {
		c.tag = "amqputil"
	}

	if delegate == nil {
		return nil, errors.New("AMQP consumer initialized without delegate")
	}
	c.delegate = delegate

	rmqConn, rmqChan, err := DialAmqpConfig(config)
	if err == nil {
		c.rmqConn = rmqConn
		c.rmqChan = rmqChan
		c.bindQueue()
	} else {
		c.log.Errorf("Failed to dial AMQP; %s", err)
		return nil, err
	}

	return c, nil
}
