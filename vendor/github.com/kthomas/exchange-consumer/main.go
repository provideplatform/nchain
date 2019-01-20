package exchangeconsumer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kthomas/go-amqputil"
	"github.com/kthomas/go-logger"
	"github.com/streadway/amqp"
)

type GdaxTickerMessageConsumer struct {
	log  *logger.Logger
	tick func(*GdaxMessage) error
}

func (c *GdaxTickerMessageConsumer) Deliver(msg *amqp.Delivery) error {
	var gdaxMessage GdaxMessage
	err := json.Unmarshal(msg.Body, &gdaxMessage)
	if err == nil {
		c.log.Debugf("Unmarshaled AMQP message body to GDAX message: %s", gdaxMessage)
		err = c.tick(&gdaxMessage)
		if err == nil {
			msg.Ack(false)
		} else {
			c.log.Warningf("Failed to handle GDAX message: %s; error: %s", gdaxMessage, err)
			if !msg.Redelivered {
				return amqputil.AmqpDeliveryErrRequireRequeue
			}
			c.log.Errorf("GDAX message has already failed redelivery attempt, dropping message: %s", err)
		}
	} else {
		c.log.Debugf("Failed to parse GDAX message: %s; %s", msg.Body, err)
		return amqputil.AmqpDeliveryErrRequireRequeue
	}
	return nil
}

func GdaxMessageConsumerFactory(lg *logger.Logger, tickFn func(*GdaxMessage) error, symbol string) *amqputil.Consumer {
	consumer, err := newGdaxTickerMessageConsumer(lg, tickFn, symbol)
	if err != nil {
		lg.Errorf("Failed to initialize GDAX message consumer for symbol %s; %s", symbol, err)
	}
	return consumer
}

func newGdaxTickerMessageConsumer(lg *logger.Logger, tickFn func(*GdaxMessage) error, queue string) (*amqputil.Consumer, error) {
	delegate := new(GdaxTickerMessageConsumer)
	delegate.log = lg.Clone()
	delegate.tick = tickFn

	config := amqputil.AmqpConfigFactory(queue)

	tag := fmt.Sprintf("exchange-consumer-gdax-%s-%d", queue, time.Now().UnixNano())
	c, err := amqputil.NewConsumer(lg, config, tag, delegate)
	if err != nil {
		lg.Errorf("Failed to initialize AMQP consumer instance with config %s; %s", config, err)
		return nil, err
	}

	return c, nil
}

type OandaTickerMessageConsumer struct {
	log  *logger.Logger
	tick func(*OandaMessage) error
}

func (c *OandaTickerMessageConsumer) Deliver(msg *amqp.Delivery) error {
	var oandaMessage OandaMessage
	err := json.Unmarshal(msg.Body, &oandaMessage)
	if err == nil {
		c.log.Debugf("Unmarshaled AMQP message body to OANDA message: %s", oandaMessage)
		err = c.tick(&oandaMessage)
		if err == nil {
			msg.Ack(false)
		} else {
			c.log.Warningf("Failed to handle OANDA message: %s; error: %s", oandaMessage, err)
			if !msg.Redelivered {
				return amqputil.AmqpDeliveryErrRequireRequeue
			}
			c.log.Errorf("GDAX message has already failed redelivery attempt, dropping message: %s", err)
		}
	} else {
		c.log.Debugf("Failed to parse GDAX message: %s; %s", msg.Body, err)
		return amqputil.AmqpDeliveryErrRequireRequeue
	}
	return nil
}

func OandaMessageConsumerFactory(lg *logger.Logger, tickFn func(*OandaMessage) error, symbol string) *amqputil.Consumer {
	consumer, err := newOandaTickerMessageConsumer(lg, tickFn, symbol)
	if err != nil {
		lg.Errorf("Failed to initialize OANDA message consumer for symbol %s; %s", symbol, err)
	}
	return consumer
}

func newOandaTickerMessageConsumer(lg *logger.Logger, tickFn func(*OandaMessage) error, queue string) (*amqputil.Consumer, error) {
	delegate := new(OandaTickerMessageConsumer)
	delegate.log = lg.Clone()
	delegate.tick = tickFn

	config := amqputil.AmqpConfigFactory(queue)

	tag := fmt.Sprintf("exchange-consumer-oanda-%s-%d", queue, time.Now().UnixNano())
	c, err := amqputil.NewConsumer(lg, config, tag, delegate)
	if err != nil {
		lg.Errorf("Failed to initialize AMQP consumer instance with config %s; %s", config, err)
		return nil, err
	}

	return c, nil
}
