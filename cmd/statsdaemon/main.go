/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

	"github.com/provideplatform/nchain/common"
	_ "github.com/provideplatform/nchain/connector"
	_ "github.com/provideplatform/nchain/contract"
	"github.com/provideplatform/nchain/network"
	_ "github.com/provideplatform/nchain/tx"
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
		//RequireHistoricalBlockStatsDaemon(ntwrk)
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
