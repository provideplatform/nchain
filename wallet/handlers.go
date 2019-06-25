package wallet

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
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
	appID := common.AuthorizedSubjectId(c, "application")
	userID := common.AuthorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}

	wallet := &Wallet{}
	err = json.Unmarshal(buf, wallet)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}

	if appID != nil {
		wallet.ApplicationID = appID
	}

	if userID != nil {
		wallet.UserID = userID
	}

	if wallet.Create() {
		common.Render(wallet, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = wallet.Errors
		common.Render(obj, 422, c)
	}
}

func walletsListHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	userID := common.AuthorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection()
	query = query.Joins("JOIN networks ON networks.id=wallets.network_id")

	if c.Query("network_id") != "" {
		query = query.Where("wallets.network_id = ?", c.Query("network_id"))
	}

	if appID != nil {
		query = query.Where("wallets.application_id = ?", appID)
	} else if userID != nil {
		query = query.Where("wallets.user_id = ?", userID)
	}

	query = query.Where("networks.enabled = ?", true)

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("wallets.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("wallets.created_at DESC")
	}

	var wallets []Wallet
	provide.Paginate(c, query, &Wallet{}).Find(&wallets)
	common.Render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	userID := common.AuthorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var err error
	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		common.RenderError("wallet not found", 404, c)
		return
	} else if appID != nil && *wallet.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && *wallet.UserID != *userID {
		common.RenderError("forbidden", 403, c)
		return
	}
	network, err := wallet.GetNetwork()
	if err == nil && network.RpcURL() != "" {
		tokenId := c.Param("tokenId")
		if tokenId == "" {
			wallet.Balance, err = wallet.NativeCurrencyBalance()
			if err != nil {
				common.RenderError(err.Error(), 400, c)
				return
			}
		} else {
			wallet.Balance, err = wallet.TokenBalance(c.Param("tokenId"))
			if err != nil {
				common.RenderError(err.Error(), 400, c)
				return
			}
		}
	}

	common.Render(wallet, 200, c)
}

func walletBalanceHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	userID := common.AuthorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
	if wallet == nil || wallet.ID == uuid.Nil {
		common.RenderError("wallet not found", 404, c)
		return
	} else if appID != nil && *wallet.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && *wallet.UserID != *userID {
		common.RenderError("forbidden", 403, c)
		return
	}
	balance, err := wallet.TokenBalance(c.Param("tokenId"))
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}
	response := map[string]*big.Int{
		c.Param("tokenId"): balance,
	}
	common.Render(response, 200, c)
}
