package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

	r.GET("/api/v1/tokens", tokensListHandler)
	r.POST("/api/v1/tokens", createTokenHandler)
	r.DELETE("/api/v1/tokens/:id", deleteTokenHandler)

	r.GET("/api/v1/transactions", transactionsListHandler)
	r.POST("/api/v1/transactions", createTransactionHandler)
	r.GET("/api/v1/transactions/:id", transactionDetailsHandler)

	r.GET("/api/v1/wallets", walletsListHandler)
	r.POST("/api/v1/wallets", createWalletHandler)
	r.GET("/api/v1/wallets/:id", walletDetailsHandler)
	r.DELETE("/api/v1/wallets/:id", deleteWalletHandler)

	r.GET("/status", statusHandler)

	if shouldServeTLS() {
		r.RunTLS(ListenAddr, CertificatePath, PrivateKeyPath)
	} else {
		r.Run(ListenAddr)
	}
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

// tokens

func tokensListHandler(c *gin.Context) {
	var tokens []Token
	DatabaseConnection().Find(&tokens)
	render(tokens, 200, c)
}

func createTokenHandler(c *gin.Context) {
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

	if token.Create() {
		render(token, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = token.Errors
		render(obj, 422, c)
	}
}

func deleteTokenHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

// transactions

func transactionsListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func createTransactionHandler(c *gin.Context) {

}

func transactionDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

// wallets

func walletsListHandler(c *gin.Context) {
	var wallets []Wallet
	DatabaseConnection().Find(&wallets)
	render(wallets, 200, c)
}

func walletDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func createWalletHandler(c *gin.Context) {
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

	if wallet.Create() {
		render(wallet, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = wallet.Errors
		render(obj, 422, c)
	}
}

func deleteWalletHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}
