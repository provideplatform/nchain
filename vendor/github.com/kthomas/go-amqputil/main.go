package amqputil

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kthomas/go-logger"
	"github.com/streadway/amqp"
)

var (
	AmqpDeliveryErrRequireRequeue = errors.New("AMQP delivery rejected with explicit requeue")

	Log = logger.NewLogger("AMQP", "INFO", true)  // TODO: allow log level to be configured
)

type AmqpConfig struct {
	AmqpBindingKey string
	AmqpExchange string
	AmqpExchangeType string
	AmqpExchangeDurable bool
	AmqpHeartbeat time.Duration
	AmqpQueue string
	AmqpUrl string
}

func AmqpConfigFactory(queue string) *AmqpConfig {
	url := os.Getenv("AMQP_URL")
	if url == "" {
		err := errors.New("Consumer environment not configured with AMQP_URL")
		Log.PanicOnError(err, "")
	}

	exchange := os.Getenv("AMQP_EXCHANGE")
	if url == "" {
		err := errors.New("Consumer environment not configured with AMQP_EXCHANGE")
		Log.PanicOnError(err, "")
	}

	exchangeType := os.Getenv("AMQP_EXCHANGE_TYPE")
	if exchangeType == "" {
		exchangeType = "topic"
	}

	exchangeDurable := true
	if os.Getenv("AMQP_EXCHANGE_DURABLE") != "" {
		exchangeDurable = strings.ToLower(os.Getenv("AMQP_EXCHANGE_DURABLE")) == "true"
	}

	heartbeat := time.Duration(60) * time.Millisecond
	if os.Getenv("AMQP_HEARTBEAT") != "" {
		heartbeatInt, err := strconv.ParseInt(os.Getenv("AMQP_HEARTBEAT"), 10, 8)
		if err != nil {
			heartbeat = time.Duration(heartbeatInt) * time.Millisecond
		}
	}

	if queue == "" {
		queue = os.Getenv("AMQP_QUEUE")
	}

	return &AmqpConfig{
		AmqpUrl: url,
		AmqpExchange: exchange,
		AmqpExchangeType: exchangeType,
		AmqpExchangeDurable: exchangeDurable,
		AmqpHeartbeat: heartbeat,
		AmqpQueue: queue,
	}
}

func DialAmqpConfig(config *AmqpConfig) (*amqp.Connection, *amqp.Channel, error) {
	return DialAmqp(
		config.AmqpUrl,
		config.AmqpHeartbeat,
		config.AmqpExchange,
		config.AmqpExchangeType,
		config.AmqpExchangeDurable,
	)
}

func DialAmqp(url string, heartbeat time.Duration, exchange string, exchangeType string, durable bool) (*amqp.Connection, *amqp.Channel, error) {
	Log.Debugf("Dialing AMQP: %s; heartbeat interval: %s", url, heartbeat)

	cfg := amqp.Config{
		Heartbeat: heartbeat,
	}

	var err error
	conn, err := amqp.DialConfig(url, cfg)
	if err != nil {
		return nil, nil, err
	}

	Log.Debugf("AMQP connection successful: %s <====> %s", url, conn.LocalAddr())

	ch, err := conn.Channel()
	if err != nil {
		return conn, nil, err
	}

	Log.Debugf("Declaring AMQP %s %s exchange: %s", durable, exchangeType, exchange)
	err = ch.ExchangeDeclare(
		exchange,	// name
		exchangeType,	// type
		durable,	// durable
		false,		// auto-deleted
		false,		// internal
		false,		// no-wait
		nil,		// arguments
	)
	if err != nil {
		Log.Warningf("Failed to declare AMQP %s %s exchange: %s; %s", durable, exchangeType, exchange, err)
		return conn, ch, err
	}

	return conn, ch, nil
}
