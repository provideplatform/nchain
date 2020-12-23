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
	"github.com/joho/godotenv"

	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/connector"
	"github.com/provideapp/nchain/contract"
	"github.com/provideapp/nchain/filter"
	"github.com/provideapp/nchain/network"
	"github.com/provideapp/nchain/oracle"
	"github.com/provideapp/nchain/prices"
	"github.com/provideapp/nchain/token"
	"github.com/provideapp/nchain/tx"
	"github.com/provideapp/nchain/wallet"

	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"
	provide "github.com/provideservices/provide-go/common"
	util "github.com/provideservices/provide-go/common/util"

	identcommon "github.com/provideapp/ident/common"
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
		common.Log.Panicf("dedicated API instance started with CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}
	godotenv.Load()

	util.RequireJWT()
	util.RequireGin()
	pgputil.RequirePGP()
	redisutil.RequireRedis()

	common.RequireInfrastructureSupport()
	common.RequirePayments()
	common.RequireVault()

	identcommon.EnableAPIAccounting()
	filter.CacheTxFilters()
}

func main() {
	common.Log.Debugf("starting nchain API...")
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

	common.Log.Debug("exiting nchain API")
	cancelF()
}

func installSignalHandlers() {
	common.Log.Debug("installing signal handlers for nchain API")
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("shutting down nchain API")
		cancelF()
	}
}

func runAPI() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(provide.CORSMiddleware())
	r.Use(identcommon.AccountingMiddleware())
	r.Use(identcommon.RateLimitingMiddleware())

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
		Addr:    util.ListenAddr,
		Handler: r,
	}

	if util.ServeTLS {
		go srv.ListenAndServeTLS(util.CertificatePath, util.PrivateKeyPath)
	} else {
		go srv.ListenAndServe()
	}

	common.Log.Debugf("listening on %s", util.ListenAddr)
}

func statusHandler(c *gin.Context) {
	provide.Render(nil, 204, c)
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}
