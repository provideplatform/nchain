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

	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/connector"
	"github.com/provideplatform/nchain/contract"
	"github.com/provideplatform/nchain/filter"
	"github.com/provideplatform/nchain/network"
	"github.com/provideplatform/nchain/oracle"
	"github.com/provideplatform/nchain/prices"
	"github.com/provideplatform/nchain/token"
	"github.com/provideplatform/nchain/tx"
	"github.com/provideplatform/nchain/wallet"

	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"
	provide "github.com/provideservices/provide-go/common"
	util "github.com/provideservices/provide-go/common/util"

	identcommon "github.com/provideplatform/ident/common"
	identtoken "github.com/provideplatform/ident/token"
)

const runloopSleepInterval = 250 * time.Millisecond
const runloopTickInterval = 5000 * time.Millisecond
const jwtVerifierRefreshInterval = 60 * time.Second
const jwtVerifierGracePeriod = 60 * time.Second

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

	common.JWTKeypairs = util.RequireJWT() // FIXME-- currently, this is still required by contract handlers call to token.VendNatsBearerAuthorization()
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

	startAt := time.Now()
	gracePeriodEndAt := startAt.Add(jwtVerifierGracePeriod)
	verifiersRefreshedAt := time.Now()

	timer := time.NewTicker(runloopTickInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			now := time.Now()
			if now.Before(gracePeriodEndAt) {
				util.RequireJWTVerifiers()
			} else if now.After(verifiersRefreshedAt.Add(jwtVerifierRefreshInterval)) {
				verifiersRefreshedAt = now
				util.RequireJWTVerifiers()
			}
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

	r.GET("/status", statusHandler)

	r.Use(identtoken.AuthMiddleware())
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
