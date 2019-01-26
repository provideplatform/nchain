package main

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

// InstallWalletsAPI installs the handlers using the given gin Engine
func InstallWalletsAPI(r *gin.Engine) {
	r.GET("/api/v1/wallets", walletsListHandler)
	r.POST("/api/v1/wallets", createWalletHandler)
	r.GET("/api/v1/wallets/:id", walletDetailsHandler)
	r.GET("/api/v1/wallets/:id/balances/:tokenId", walletBalanceHandler)
}

func createWalletHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	wallet := &Wallet{}
	err = json.Unmarshal(buf, wallet)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	if appID != nil {
		wallet.ApplicationID = appID
	}

	if userID != nil {
		wallet.UserID = userID
	}

	if wallet.Create() {
		render(wallet, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = wallet.Errors
		render(obj, 422, c)
	}
}

func walletsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection()

	if c.Query("network_id") != "" {
		query = query.Where("wallets.network_id = ?", c.Query("network_id"))
	}

	if appID != nil {
		query = query.Where("wallets.application_id = ?", appID)
	} else if userID != nil {
		query = query.Where("wallets.user_id = ?", userID)
	}

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("wallets.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("wallets.created_at DESC")
	}

	var wallets []Wallet
	provide.Paginate(c, query, &Wallet{}).Find(&wallets)
	render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var err error
	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		renderError("wallet not found", 404, c)
		return
	} else if appID != nil && *wallet.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	} else if userID != nil && *wallet.UserID != *userID {
		renderError("forbidden", 403, c)
		return
	}
	network, err := wallet.GetNetwork()
	if err == nil && network.rpcURL() != "" {
		tokenId := c.Param("tokenId")
		if tokenId == "" {
			wallet.Balance, err = wallet.NativeCurrencyBalance()
			if err != nil {
				renderError(err.Error(), 400, c)
				return
			}
		} else {
			wallet.Balance, err = wallet.TokenBalance(c.Param("tokenId"))
			if err != nil {
				renderError(err.Error(), 400, c)
				return
			}
		}
	}

	render(wallet, 200, c)
}

func walletBalanceHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		renderError("wallet not found", 404, c)
		return
	} else if appID != nil && *wallet.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	} else if userID != nil && *wallet.UserID != *userID {
		renderError("forbidden", 403, c)
		return
	}
	balance, err := wallet.TokenBalance(c.Param("tokenId"))
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}
	response := map[string]*big.Int{
		c.Param("tokenId"): balance,
	}
	render(response, 200, c)
}
