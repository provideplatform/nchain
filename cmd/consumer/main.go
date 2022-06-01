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
	"sync/atomic"
	"syscall"
	"time"

	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"

	"github.com/provideplatform/nchain/common"
	_ "github.com/provideplatform/nchain/connector"
	_ "github.com/provideplatform/nchain/consumer"
	_ "github.com/provideplatform/nchain/contract"
	_ "github.com/provideplatform/nchain/network"
	_ "github.com/provideplatform/nchain/tx"
)

const natsStreamingSubscriptionStatusTickerInterval = 5 * time.Second
const natsStreamingSubscriptionStatusSleepInterval = 250 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context
)

func init() {
	pgputil.RequirePGP()
	redisutil.RequireRedis()

	common.RequireInfrastructureSupport()
	common.RequirePayments()
	common.RequireVault()
}

func main() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Panicf("Dedicated NATS streaming subscription consumer started without CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}

	common.Log.Debug("Installing signal handlers for dedicated NATS streaming subscription consumer")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

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
			time.Sleep(natsStreamingSubscriptionStatusSleepInterval)
		}
	}

	common.Log.Debug("Exiting dedicated NATS streaming subscription consumer main()")
	cancelF()
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
