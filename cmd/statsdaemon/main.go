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
	redisutil "github.com/kthomas/go-redisutil"

	"github.com/provideapp/goldmine/common"
	_ "github.com/provideapp/goldmine/connector"
	_ "github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	_ "github.com/provideapp/goldmine/tx"
)

const statsdaemonTickerInterval = 5 * time.Second
const statsdaemonSleepInterval = 250 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	mutex    sync.Mutex
	networks []*network.Network
)

func init() {
	redisutil.RequireRedis()
}

func main() {
	common.Log.Debug("Installing signal handlers for statsdaemon")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

	requireNetworkStatsDaemonInstances()

	common.Log.Debugf("Running statsdaemon main()")
	timer := time.NewTicker(statsdaemonTickerInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// TODO: check statsdaemon statuses
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			time.Sleep(statsdaemonSleepInterval)
		}
	}

	common.Log.Debug("Exiting statsdaemon main()")
	cancelF()
}

func requireNetworkStatsDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	networks = make([]*network.Network, 0)
	dbconf.DatabaseConnection().Where("user_id IS NULL AND enabled IS TRUE").Find(&networks)

	for _, ntwrk := range networks {
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
