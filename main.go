package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/kthomas/go.uuid"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	gocore "github.com/provideapp/go-core"
)

func main() {
	bootstrap()
	migrateSchema()

	RunConsumers()

	r := gin.Default()

	r.GET("/api/v1/networks", networksListHandler)
	r.GET("/api/v1/networks/:id", networkDetailsHandler)
	r.GET("/api/v1/networks/:id/addresses", networkAddressesHandler)
	r.GET("/api/v1/networks/:id/blocks", networkBlocksHandler)
	r.GET("/api/v1/networks/:id/contracts", networkContractsHandler)
	r.GET("/api/v1/networks/:id/transactions", networkTransactionsHandler)

	r.GET("/api/v1/prices", pricesHandler)

	r.GET("/api/v1/contracts", contractsListHandler)

	r.GET("/api/v1/tokens", tokensListHandler)
	r.GET("/api/v1/tokens/:id", tokenDetailsHandler)
	r.POST("/api/v1/tokens", createTokenHandler)

	r.GET("/api/v1/transactions", transactionsListHandler)
	r.POST("/api/v1/transactions", createTransactionHandler)
	r.GET("/api/v1/transactions/:id", transactionDetailsHandler)

	r.GET("/api/v1/wallets", walletsListHandler)
	r.POST("/api/v1/wallets", createWalletHandler)
	r.GET("/api/v1/wallets/:id", walletDetailsHandler)
	r.GET("/api/v1/wallets/:id/balances/:tokenId", walletBalanceHandler)

	r.GET("/status", statusHandler)

	if shouldServeTLS() {
		r.RunTLS(ListenAddr, CertificatePath, PrivateKeyPath)
	} else {
		r.Run(ListenAddr)
	}
}

func authorizedSubjectId(c *gin.Context, subject string) *uuid.UUID {
	var id string
	keyfn := func(jwtToken *jwt.Token) (interface{}, error) {
		if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
			if sub, subok := claims["sub"].(string); subok {
				subprts := strings.Split(sub, ":")
				if len(subprts) != 2 {
					return nil, fmt.Errorf("JWT subject malformed; %s", sub)
				}
				if subprts[0] != subject {
					return nil, fmt.Errorf("JWT claims specified non-application subject: %s", subprts[0])
				}
				id = subprts[1]
			}
		}
		return nil, nil
	}
	gocore.ParseBearerAuthorizationHeader(c, &keyfn)
	uuidV4, err := uuid.FromString(id)
	if err != nil {
		return nil
	}
	return &uuidV4
}

func render(obj interface{}, status int, c *gin.Context) {
	c.Header("content-type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(status)
	if &obj != nil && status != http.StatusNoContent {
		encoder := json.NewEncoder(c.Writer)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(obj); err != nil {
			panic(err)
		}
	} else {
		c.Header("content-length", "0")
	}
}

func renderError(message string, status int, c *gin.Context) {
	err := map[string]*string{}
	err["message"] = &message
	render(err, status, c)
}

func requireParams(requiredParams []string, c *gin.Context) error {
	var errs []string
	for _, param := range requiredParams {
		if c.Query(param) == "" {
			errs = append(errs, param)
		}
	}
	if len(errs) > 0 {
		msg := strings.Trim(fmt.Sprintf("missing required parameters: %s", strings.Join(errs, ", ")), " ")
		renderError(msg, 400, c)
		return errors.New(msg)
	}
	return nil
}

func statusHandler(c *gin.Context) {
	render(nil, 204, c)
}

// networks

func networksListHandler(c *gin.Context) {
	var networks []Network
	DatabaseConnection().Find(&networks)
	render(networks, 200, c)
}

func networkDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkAddressesHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkBlocksHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkContractsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkTransactionsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

// prices

func pricesHandler(c *gin.Context) {
	render(CurrentPrices, 200, c)
}

// contracts

func contractsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var contracts []Contract
	DatabaseConnection().Where("application_id = ?", appID).Find(&contracts)
	render(contracts, 200, c)
}

// tokens

func tokensListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var tokens []Token
	DatabaseConnection().Where("application_id = ?", appID).Find(&tokens)
	render(tokens, 200, c)
}

func tokenDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func createTokenHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	token := &Token{}
	err = json.Unmarshal(buf, token)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	token.ApplicationID = appID

	if token.Create() {
		render(token, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = token.Errors
		render(obj, 422, c)
	}
}

// transactions

func transactionsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var txs []Transaction
	if appID != nil {
		DatabaseConnection().Where("application_id = ?", appID).Find(&txs)
	} else if userID != nil {
		DatabaseConnection().Where("user_id = ?", userID).Find(&txs)
	}
	render(txs, 200, c)
}

func createTransactionHandler(c *gin.Context) {
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

	tx := &Transaction{}
	err = json.Unmarshal(buf, tx)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	if appID != nil {
		tx.ApplicationID = appID
	}

	if userID != nil {
		tx.UserID = userID
	}

	if tx.Create() {
		render(tx, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = tx.Errors
		render(obj, 422, c)
	}
}

func transactionDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

// wallets

func walletsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var wallets []Wallet
	if appID != nil {
		DatabaseConnection().Where("application_id = ?", appID).Find(&wallets)
	} else if userID != nil {
		DatabaseConnection().Where("user_id = ?", userID).Find(&wallets)
	}
	render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func walletBalanceHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var wallet = &Wallet{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
	if wallet == nil {
		renderError("wallet not found", 404, c)
		return
	} else if appID != nil && &wallet.ApplicationID != &appID {
		renderError("forbidden", 403, c)
		return
	} else if userID != nil && &wallet.UserID != &userID {
		renderError("forbidden", 403, c)
		return
	}
	balance, err := wallet.TokenBalance(c.Param("tokenId"))
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}
	response := map[string]uint64{
		c.Param("tokenId"): balance,
	}
	render(response, 200, c)
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
