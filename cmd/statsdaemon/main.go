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
const enableDaemonsTickerInterval = 10 * time.Second
const enableDaemonsSleepInterval = 5 * time.Second

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

	monitorNetworkDaemonInstances()

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

func monitorNetworkDaemonInstances() {
	go func() {
		timer := time.NewTicker(enableDaemonsTickerInterval)
		defer timer.Stop()

		for !shuttingDown() {
			select {
			case <-timer.C:
				networks := requireNetworkDaemonInstances()

				for networkID := range currentNetworkStats {
					evict := true
					for _, netwrk := range networks {
						if netwrk.ID.String() == networkID {
							evict = false
							break
						}
					}

					if evict {
						common.Log.Debugf("evicting network statsdaemon and log transceiver: %s", networkID)
						EvictNetworkLogTransceiver(currentLogTransceivers[networkID].Network)
						EvictNetworkStatsDaemon(currentNetworkStats[networkID].dataSource.Network)
					}
				}
			default:
				time.Sleep(enableDaemonsSleepInterval)
			}
		}
	}()
}

func requireNetworkDaemonInstances() []*network.Network {
	mutex.Lock()
	defer mutex.Unlock()

	networks = make([]*network.Network, 0)
	dbconf.DatabaseConnection().Where("user_id IS NULL AND enabled IS TRUE").Find(&networks)

	for _, ntwrk := range networks {
		RequireNetworkLogTransceiver(ntwrk)
		RequireNetworkStatsDaemon(ntwrk)
		//TODO RequireHistoricalStatsDaemon(ntwrk)
		//TODO copy the stats_daemon into another file
		//TODO create a simple blocknumber/tx/logs table(s)
		//TODO add an origin to this table (historical/websocket)
		//TODO update networks with the start_block (for historical)
		//TODO update networks with the latest_block (??)
		// this will use JSON RPC to call blocks that are higher than the start_block
		// if there is no start_block, it will ignore historicals and just run stats_daemon
		// from scratch:
		// - it gets start_block,
		//     - if no start_block,
		//       -  nop
		//     - if start_block,
		//       -  check table for lowest missing value
		//          - higher than start block in the table
		//          - and lower than the MAX origin:websocket block (populated every time the ws finalizes a block)
		//       -  check if this is possible in single sql query
		//       - pull the block details using jsonrpc
		//       - populate the block details in the table
		//       - run query again and rinse repeat
		//       - query might be a pain/slow. maybe there's a naive implementation
		//         we can do first (increment by 1, and only rerun query if there's a record there)
		//       -
		// TODO Blockchain table created, check the queries on it.

	}

	return networks
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
