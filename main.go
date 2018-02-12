package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
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
	r.POST("/api/v1/networks", createNetworkHandler)
	r.GET("/api/v1/networks/:id/addresses", networkAddressesHandler)
	r.GET("/api/v1/networks/:id/blocks", networkBlocksHandler)
	r.GET("/api/v1/networks/:id/contracts", networkContractsHandler)
	r.GET("/api/v1/networks/:id/status", networkStatusHandler)
	r.GET("/api/v1/networks/:id/transactions", networkTransactionsHandler)

	r.GET("/api/v1/prices", pricesHandler)

	r.GET("/api/v1/contracts", contractsListHandler)
	r.POST("/api/v1/contracts/:id/execute", contractExecutionHandler)

	r.GET("/api/v1/oracles", oraclesListHandler)
	r.POST("/api/v1/oracles", createOracleHandler)
	r.GET("/api/v1/oracles/:id", oracleDetailsHandler)

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

func createNetworkHandler(c *gin.Context) {
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

	network := &Network{}
	err = json.Unmarshal(buf, network)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	network.ApplicationID = appID

	if network.Create() {
		render(network, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		render(obj, 422, c)
	}
}

func networksListHandler(c *gin.Context) {
	var networks []Network
	query := DatabaseConnection().Where("networks.application_id IS NULL").Order("created_at ASC")

	appID := authorizedSubjectId(c, "application")
	if appID != nil {
		query = query.Or("networks.application_id = ?", appID)
	}

	query.Order("created_at ASC").Find(&networks)
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

func networkStatusHandler(c *gin.Context) {
	var network = &Network{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		renderError("network not found", 404, c)
		return
	}
	status, err := network.Status()
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve network status; %s", err.Error())
		renderError(msg, 500, c)
		return
	}
	render(status, 200, c)
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

	query := DatabaseConnection().Where("contracts.application_id = ?", appID)

	filterTokens := strings.ToLower(c.Query("filter_tokens")) == "true"
	if filterTokens {
		query = query.Joins("LEFT OUTER JOIN tokens ON tokens.contract_id = contracts.id").Where("symbol IS NULL")
	}

	var contracts []Contract
	query.Order("created_at ASC").Find(&contracts)
	render(contracts, 200, c)
}

func contractExecutionHandler(c *gin.Context) {
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

	var contract = &Contract{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&contract)

	if contract == nil || contract.ID == uuid.Nil {
		renderError("contract not found", 404, c)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	}

	execution := &ContractExecution{}
	err = json.Unmarshal(buf, execution)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	executionResponse, err := contract.Execute(execution.WalletID, execution.Value, execution.Method, execution.Params)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	render(executionResponse, 202, c) // returns 202 Accepted status to indicate the contract invocation is pending
}

// oracles

func oraclesListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := DatabaseConnection().Where("oracles.application_id = ?", appID)

	var oracles []Oracle
	query.Order("created_at ASC").Find(&oracles)
	render(oracles, 200, c)
}

func oracleDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func createOracleHandler(c *gin.Context) {
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

	oracle := &Oracle{}
	err = json.Unmarshal(buf, oracle)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	oracle.ApplicationID = appID

	if oracle.Create() {
		render(oracle, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = oracle.Errors
		render(obj, 422, c)
	}
}

// tokens

func tokensListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var tokens []Token
	DatabaseConnection().Where("application_id = ?", appID).Order("created_at ASC").Find(&tokens)
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

	var query *gorm.DB
	if appID != nil {
		query = DatabaseConnection().Where("transactions.application_id = ?", appID)
	} else if userID != nil {
		query = DatabaseConnection().Where("transactions.user_id = ?", userID)
	}

	filterContractCreationTx := strings.ToLower(c.Query("filter_contract_creations")) == "true"
	if filterContractCreationTx {
		query = query.Where("transactions.to IS NULL")
	}

	var txs []Transaction
	query.Order("created_at DESC").Find(&txs)
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
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var tx = &Transaction{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		renderError("transaction not found", 404, c)
		return
	} else if appID != nil && &tx.ApplicationID != &appID {
		renderError("forbidden", 403, c)
		return
	}
	err := tx.RefreshDetails()
	if err != nil {
		renderError("internal server error", 500, c)
		return
	}
	render(tx, 200, c)
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
	if wallet == nil || wallet.ID == uuid.Nil {
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
