package wallet

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	provide "github.com/provideplatform/provide-go/common"
	util "github.com/provideplatform/provide-go/common/util"
)

const defaultDerivedAccountsPerPage = uint32(10)
const defaultDerivedCoinType = uint32(60)
const defaultDerivedChainPath = uint32(0) // i.e., the external or internal chain (also known as change addresses if internal chain)
const firstHardenedChildIndex = uint32(0x80000000)

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
	r.GET("/api/v1/wallets/:id/accounts", walletAccountsListHandler)
}

func createAccountHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
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

	if organizationID != nil {
		account.OrganizationID = organizationID
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
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
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
	} else if organizationID != nil {
		query = query.Where("accounts.organization_id = ?", organizationID)
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
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var err error
	var account = &Account{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&account)
	if account == nil || account.ID == uuid.Nil {
		provide.RenderError("account not found", 404, c)
		return
	} else if appID != nil && account.ApplicationID != nil && *account.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && account.UserID != nil && *account.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if organizationID != nil && account.OrganizationID != nil && *account.OrganizationID != *organizationID {
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
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var account = &Account{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&account)
	if account == nil || account.ID == uuid.Nil {
		provide.RenderError("account not found", 404, c)
		return
	} else if appID != nil && (account.ApplicationID == nil || *account.ApplicationID != *appID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && (account.UserID == nil || *account.UserID != *userID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if organizationID != nil && (account.OrganizationID == nil || *account.OrganizationID != *organizationID) {
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
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
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

	if organizationID != nil {
		wallet.OrganizationID = organizationID
	}

	if wallet.Create() {
		provide.Render(wallet, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = wallet.Errors
		provide.Render(obj, 422, c)
	}
}

func walletsListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
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
	} else if organizationID != nil {
		query = query.Where("wallets.organization_id = ?", organizationID)
	}
	query = query.Order("wallets.created_at DESC")

	var wallets []*Wallet
	provide.Paginate(c, query, &Wallet{}).Find(&wallets)
	// for _, wallet := range wallets {
	// }
	provide.Render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var wallet = &Wallet{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)

	if wallet == nil || wallet.ID == uuid.Nil {
		provide.RenderError("wallet not found", 404, c)
		return
	} else if appID != nil && (wallet.ApplicationID == nil && *wallet.ApplicationID != *appID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && (wallet.UserID == nil || *wallet.UserID != *userID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if organizationID != nil && (wallet.OrganizationID == nil || *wallet.OrganizationID != *organizationID) {
		provide.RenderError("forbidden", 403, c)
		return
	}

	// wallet.decrypt()
	provide.Render(wallet, 200, c)
}

func walletAccountsListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	organizationID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && organizationID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()

	var wallet = &Wallet{}
	db.Where("id = ?", c.Param("id")).Find(&wallet)

	if wallet == nil || wallet.ID == uuid.Nil {
		provide.RenderError("wallet not found", 404, c)
		return
	} else if appID != nil && (wallet.ApplicationID == nil || *wallet.ApplicationID != *appID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if userID != nil && (wallet.UserID == nil || *wallet.UserID != *userID) {
		provide.RenderError("forbidden", 403, c)
		return
	} else if organizationID != nil && (wallet.OrganizationID == nil || *wallet.OrganizationID != *organizationID) {
		provide.RenderError("forbidden", 403, c)
		return
	}

	page := uint32(1)
	rpp := defaultDerivedAccountsPerPage
	if c.Query("page") != "" {
		if _page, err := strconv.ParseInt(c.Query("page"), 10, 32); err == nil {
			page = uint32(_page)
		}
	}
	if c.Query("rpp") != "" {
		if _rpp, err := strconv.ParseInt(c.Query("rpp"), 10, 32); err == nil {
			rpp = uint32(_rpp)
		}
	}
	c.Header("x-total-results-count", fmt.Sprintf("%d", firstHardenedChildIndex))

	coin := defaultDerivedCoinType
	if c.Query("coin_type") != "" {
		cointype, err := strconv.ParseInt(c.Query("coin_type"), 10, 32)
		if err != nil {
			msg := fmt.Sprintf("Failed to derive address for HD wallet: %s; invalid coin type: %s", wallet.ID, c.Query("coin_type"))
			common.Log.Warningf(msg)
			provide.RenderError(msg, 400, c)
			return
		}
		coin = uint32(cointype)
	}

	chainPath := defaultDerivedChainPath
	if c.Query("chain_path") != "" {
		path, err := strconv.ParseInt(c.Query("chain_path"), 10, 32) // FIXME-- documentation for chain_path parameter is missing
		if err != nil {
			msg := fmt.Sprintf("Failed to derive address for HD wallet: %s; invalid chain path index: %s", wallet.ID, c.Query("chain_path"))
			common.Log.Warningf(msg)
			provide.RenderError(msg, 400, c)
			return
		}
		chainPath = uint32(path)
	}

	hardenedChildIndex := uint32(0)
	if c.Query("index") != "" {
		childIndex, err := strconv.ParseInt(c.Query("index"), 10, 32)
		if err != nil {
			msg := fmt.Sprintf("Failed to derive address for HD wallet: %s; invalid child account index: %s", wallet.ID, c.Query("index"))
			common.Log.Warningf(msg)
			provide.RenderError(msg, 400, c)
			return
		}
		hardenedChildIndex = uint32(childIndex)
	}

	hardenedChild, err := wallet.DeriveHardened(db, coin, hardenedChildIndex)
	if err != nil {
		msg := fmt.Sprintf("Failed to derive address for HD wallet: %s; %s", wallet.ID, err.Error())
		common.Log.Warningf(msg)
		provide.RenderError(msg, 500, c)
		return
	}

	var accounts []map[string]interface{}

	i := uint32(0)
	for {
		idx := ((page - 1) * rpp) + i
		derivedAccount, err := hardenedChild.DeriveAddress(db, idx, &chainPath)
		if err != nil {
			msg := fmt.Sprintf("Failed to derive address for HD wallet: %s; %s", wallet.ID, err.Error())
			common.Log.Warningf(msg)
			provide.RenderError(msg, 500, c)
			return
		}
		accounts = append(accounts, derivedAccount.ephemeralResponse())

		i++
		if i == rpp || i == firstHardenedChildIndex-1 {
			break
		}
	}

	provide.Render(accounts, 200, c)
}
