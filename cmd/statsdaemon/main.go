package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"

	"github.com/provideapp/nchain/common"
	_ "github.com/provideapp/nchain/connector"
	_ "github.com/provideapp/nchain/contract"
	"github.com/provideapp/nchain/network"
	_ "github.com/provideapp/nchain/tx"
)

const runloopTickerInterval = 5 * time.Second
const runloopSleepInterval = 250 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	mutex sync.Mutex

	networks []*network.Network
)

func init() {
	if common.ConsumeNATSStreamingSubscriptions {
		common.Log.Panicf("statsdaemon instance started with CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}

	pgputil.RequirePGP()
	redisutil.RequireRedis()

	common.RequireInfrastructureSupport()
}

func main() {
	common.Log.Debug("Installing signal handlers for statsdaemon")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

	requireNetworkDaemonInstances()

	common.Log.Debugf("Running statsdaemon main()")
	timer := time.NewTicker(runloopTickerInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// TODO: check statsdaemon statuses
			// TODO: check logsdaemon statuses
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			time.Sleep(runloopSleepInterval)
		}
	}

	common.Log.Debug("Exiting statsdaemon main()")
	cancelF()
}

func requireNetworkDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	networks = make([]*network.Network, 0)
	dbconf.DatabaseConnection().Where("user_id IS NULL AND enabled IS TRUE").Find(&networks)

	for _, ntwrk := range networks {
		RequireNetworkLogTransceiver(ntwrk)
		RequireNetworkStatsDaemon(ntwrk)
	}
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("Shutting down statsdaemon")
		cancelF()
	}
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}
