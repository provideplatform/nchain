package wallet

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

// InstallAccountsAPI installs the handlers using the given gin Engine
func InstallAccountsAPI(r *gin.Engine) {
	r.GET("/api/v1/accounts", accountsListHandler)
	r.POST("/api/v1/accounts", createAccountHandler)
	r.GET("/api/v1/accounts/:id", accountDetailsHandler)
	r.GET("/api/v1/accounts/:id/balances/:tokenId", accountBalanceHandler)
}

// InstallWalletsAPI installs the handlers using the given gin Engine
func InstallWalletsAPI(r *gin.Engine) {
	r.GET("/api/v1/wallets", walletsListHandler)
	r.POST("/api/v1/wallets", createWalletHandler)
	r.GET("/api/v1/wallets/:id", walletDetailsHandler)
}

func createAccountHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	account := &Account{}
	err = json.Unmarshal(buf, account)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	if appID != nil {
		account.ApplicationID = appID
	}

	if userID != nil {
		account.UserID = userID
	}

	if account.Create() {
		provide.Render(account, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = account.Errors
		provide.Render(obj, 422, c)
	}
}

func accountsListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection()
	query = query.Joins("JOIN networks ON networks.id=accounts.network_id")

	if c.Query("network_id") != "" {
		query = query.Where("accounts.network_id = ?", c.Query("network_id"))
	}

	if c.Query("wallet_id") != "" {
		query = query.Where("accounts.wallet_id = ?", c.Query("wallet_id"))
	}

	if appID != nil {
		query = query.Where("accounts.application_id = ?", appID)
	} else if userID != nil {
		query = query.Where("accounts.user_id = ?", userID)
	}

	query = query.Where("networks.enabled = ?", true)

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("accounts.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("accounts.created_at DESC")
	}

	var accounts []Account
	provide.Paginate(c, query, &Account{}).Find(&accounts)
	provide.Render(accounts, 200, c)
}

func accountDetailsHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var err error
	var account = &Account{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&account)
	if account == nil || account.ID == uuid.Nil {
		provide.RenderError("account not found", 404, c)
		return
	} else if appID != nil && *account.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && *account.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	network, err := account.GetNetwork()
	if err == nil && network.RPCURL() != "" {
		tokenID := c.Param("tokenId")
		if tokenID == "" {
			account.Balance, err = account.NativeCurrencyBalance()
			if err != nil {
				provide.RenderError(err.Error(), 400, c)
				return
			}
		} else {
			account.Balance, err = account.TokenBalance(c.Param("tokenId"))
			if err != nil {
				provide.RenderError(err.Error(), 400, c)
				return
			}
		}
	}

	provide.Render(account, 200, c)
}

func accountBalanceHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var account = &Account{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&account)
	if account == nil || account.ID == uuid.Nil {
		provide.RenderError("account not found", 404, c)
		return
	} else if appID != nil && *account.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && *account.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	balance, err := account.TokenBalance(c.Param("tokenId"))
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}
	response := map[string]*big.Int{
		c.Param("tokenId"): balance,
	}
	provide.Render(response, 200, c)
}

func createWalletHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	wallet := &Wallet{}
	err = json.Unmarshal(buf, wallet)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	if appID != nil {
		wallet.ApplicationID = appID
	}

	if userID != nil {
		wallet.UserID = userID
	}

	if wallet.Create() {
		wallet.decrypt()
		provide.Render(wallet, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = wallet.Errors
		provide.Render(obj, 422, c)
	}
}

func walletsListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection()

	if c.Query("wallet_id") != "" {
		query = query.Where("wallets.wallet_id = ?", c.Query("account_id"))
	}

	if appID != nil {
		query = query.Where("wallets.application_id = ?", appID)
	} else if userID != nil {
		query = query.Where("wallets.user_id = ?", userID)
	}
	query = query.Order("wallets.created_at DESC")

	var wallets []Wallet
	provide.Paginate(c, query, &Wallet{}).Find(&wallets)
	provide.Render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)

	if wallet == nil || wallet.ID == uuid.Nil {
		provide.RenderError("wallet not found", 404, c)
		return
	} else if appID != nil && *wallet.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && *wallet.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	provide.Render(wallet, 200, c)
}
