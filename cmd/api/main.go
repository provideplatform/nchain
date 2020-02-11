package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/connector"
	"github.com/provideapp/goldmine/consumer"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/filter"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/oracle"
	"github.com/provideapp/goldmine/prices"
	"github.com/provideapp/goldmine/token"
	"github.com/provideapp/goldmine/tx"
	"github.com/provideapp/goldmine/wallet"

	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"
	provide "github.com/provideservices/provide-go"

	identcommon "github.com/provideapp/ident"
)

const runloopSleepInterval = 250 * time.Millisecond
const runloopTickInterval = 5000 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context
	sigs        chan os.Signal

	srv *http.Server
)

func init() {
	if common.ConsumeNATSStreamingSubscriptions {
		common.Log.Panicf("Dedicated API instance started with CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}

	identcommon.RequireJWT()
	pgputil.RequirePGP()
	redisutil.RequireRedis()

	consumer.RunAPIUsageDaemon()
	filter.CacheTxFilters()
}

func main() {
	common.Log.Debugf("starting goldmine API...")
	installSignalHandlers()

	runAPI()

	timer := time.NewTicker(runloopTickInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// tick... no-op
		case sig := <-sigs:
			common.Log.Debugf("received signal: %s", sig)
			srv.Shutdown(shutdownCtx)
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			time.Sleep(runloopSleepInterval)
		}
	}

	common.Log.Debug("exiting goldmine API")
	cancelF()
}

func installSignalHandlers() {
	common.Log.Debug("installing signal handlers for goldmine API")
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("shutting down goldmine API")
		cancelF()
	}
}

func runAPI() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(provide.CORSMiddleware())
	r.Use(provide.TrackAPICalls())

	network.InstallNetworksAPI(r)
	prices.InstallPricesAPI(r)
	connector.InstallConnectorsAPI(r)
	contract.InstallContractsAPI(r)
	oracle.InstallOraclesAPI(r)
	token.InstallTokensAPI(r)
	tx.InstallTransactionsAPI(r)
	wallet.InstallAccountsAPI(r)
	wallet.InstallWalletsAPI(r)

	r.GET("/status", statusHandler)

	srv = &http.Server{
		Addr:    common.ListenAddr,
		Handler: r,
	}

	if common.ShouldServeTLS() {
		go srv.ListenAndServeTLS(common.CertificatePath, common.PrivateKeyPath)
	} else {
		go srv.ListenAndServe()
	}

	common.Log.Debugf("Listening on %s", common.ListenAddr)
}

func statusHandler(c *gin.Context) {
	provide.Render(nil, 204, c)
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}
