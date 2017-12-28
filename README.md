## goldmine

Blockchain API, written in golang


The following APIs are exposed:

#### Networks API

	r.GET("/api/v1/networks", networksListHandler)
    r.GET("/api/v1/networks/:id", networkDetailsHandler)
    r.GET("/api/v1/networks/:id/addresses", networkAddressesHandler)
    r.GET("/api/v1/networks/:id/blocks", networkBlocksHandler)
    r.GET("/api/v1/networks/:id/contracts", networkContractsHandler)
    r.GET("/api/v1/networks/:id/transactions", networkTransactionsHandler)

#### Prices API

	r.GET("/api/v1/prices", pricesHandler)

#### Tokens API

	r.GET("/api/v1/tokens", tokensListHandler)
	r.POST("/api/v1/tokens", createTokenHandler)
	r.DELETE("/api/v1/tokens/:id", deleteTokenHandler)

#### Transactions API

	r.GET("/api/v1/transactions", transactionsListHandler)
	r.POST("/api/v1/transactions", createTransactionHandler)
	r.GET("/api/v1/transactions/:id", transactionDetailsHandler)

#### Wallets API

	r.GET("/api/v1/wallets", walletsListHandler)
	r.POST("/api/v1/wallets", createWalletHandler)
	r.GET("/api/v1/wallets/:id", walletDetailsHandler)
	r.DELETE("/api/v1/wallets/:id", deleteWalletHandler)

#### Status API

	r.GET("/status", statusHandler)
