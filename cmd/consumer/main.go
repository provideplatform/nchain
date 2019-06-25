package main

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/provideapp/goldmine/common"
	_ "github.com/provideapp/goldmine/contract"
	_ "github.com/provideapp/goldmine/network"
	_ "github.com/provideapp/goldmine/tx"
)

const natsStreamingSubscriptionStatusTickerInterval = 5 * time.Second

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context
)

func main() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Panicf("Dedicated NATS streaming subscription consumer started without CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}

	common.Log.Debug("Installing signal handlers for dedicated NATS streaming subscription consumer")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	common.Log.Debugf("Running dedicated NATS streaming subscription consumer main()")
	timer := time.NewTicker(natsStreamingSubscriptionStatusTickerInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// TODO: check NATS subscription statuses
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			common.Log.Warningf("NATS streaming connection subscriptions are not yet being drained...")
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			// no-op
		}
	}

	common.Log.Debug("Exiting dedicated NATS streaming subscription consumer main()")
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("Shutting down dedicated NATS streaming subscription consumer")
		cancelF()
	}
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}