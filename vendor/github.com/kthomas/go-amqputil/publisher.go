package amqputil

import (
	"github.com/kthomas/go-logger"
	"github.com/streadway/amqp"
)

type Publisher struct {
	config *AmqpConfig

	log *logger.Logger

	rmqConn *amqp.Connection
	rmqChan *amqp.Channel
}

func (p *Publisher) bindQueue() error {
	err := p.declareQueue()
	if err == nil {
		p.log.Debugf("Attempting to bind %s queue to routing key %s in %s AMQP exchange", p.config.AmqpQueue, p.config.AmqpBindingKey, p.config.AmqpExchange)
		err = p.rmqChan.QueueBind(
			p.config.AmqpQueue,		// queue
			p.config.AmqpBindingKey, 	// binding key
			p.config.AmqpExchange,		// exchange
			false, 				// no-wait
			nil,				// arguments
		)
		if err == nil {
			p.log.Debugf("Bound %s queue to %s routing key in %s AMQP exchange", p.config.AmqpQueue, p.config.AmqpBindingKey, p.config.AmqpExchange)
		}
	}
	return err
}

func (p *Publisher) declareQueue() error {
	p.log.Debugf("Attempting to declare %s queue in %s AMQP exchange", p.config.AmqpQueue, p.config.AmqpBindingKey)
	_, err := p.rmqChan.QueueDeclare(
		p.config.AmqpQueue,	// queue
		true, 			// durable
		false,			// auto-delete
		false,			// exclusive
		false,			// no-wait
		nil,			// arguments
	)
	if err != nil {
		p.log.Warningf("Failed to declare %s queue in %s AMQP exchange", p.config.AmqpQueue, p.config.AmqpExchange)
	}
	return err
}

func (p *Publisher) Publish(msg *[]byte) error {
	p.log.Debugf("Publishing %v-byte message to AMQP binding key %s; %s", len(*msg), p.config.AmqpBindingKey, msg)

	payload := amqp.Publishing{
		Body: *msg,
		ContentType: "application/json",
	}

	err := p.rmqChan.Publish(
		p.config.AmqpExchange,		// exchange
		p.config.AmqpBindingKey,	// binding key
		true,				// mandatory
		false,				// immediate
		payload,			// payload
	)

	if err != nil {
		p.log.Errorf("Error attempting to publish %v-byte message: %s", len(*msg), msg)
	} else {
		p.log.Debugf("Published %v-byte message to binding key %s: %s", len(*msg), p.config.AmqpBindingKey, msg)
	}

	return err
}

func NewPublisher(lg *logger.Logger, config *AmqpConfig) (*Publisher, error) {
	p := new(Publisher)
	p.log = lg.Clone()

	p.config = config

	rmqConn, rmqChan, err := DialAmqpConfig(config)
	if err == nil {
		p.rmqConn = rmqConn
		p.rmqChan = rmqChan
		p.bindQueue()
	} else {
		p.log.Errorf("Failed to dial AMQP; %s", err)
		return nil, err
	}

	return p, nil
}
