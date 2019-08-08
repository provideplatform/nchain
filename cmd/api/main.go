package main

import (
	"github.com/gin-gonic/gin"
	provide "github.com/provideservices/provide-go"

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
)

func main() {
	if common.ConsumeNATSStreamingSubscriptions {
		common.Log.Panicf("Dedicated API instance started with CONSUME_NATS_STREAMING_SUBSCRIPTIONS=true")
		return
	}

	common.Log.Debugf("Running dedicated API instance main()")

	consumer.RunAPIUsageDaemon()
	filter.CacheTxFilters()

	r := gin.Default()
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
	wallet.InstallWalletsAPI(r)

	r.GET("/status", statusHandler)

	network.MigrateEncryptedLBConfigs()
	network.MigrateEncryptedNodeConfigs()

	if common.ShouldServeTLS() {
		r.RunTLS(common.ListenAddr, common.CertificatePath, common.PrivateKeyPath)
	} else {
		r.Run(common.ListenAddr)
	}
}

func statusHandler(c *gin.Context) {
	provide.Render(nil, 204, c)
}
