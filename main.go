package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

func main() {
	migrateSchema()

	runConsumers()
	runAPIUsageDaemon()
	cacheTxFilters()

	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(provide.CORSMiddleware())
	r.Use(provide.TrackAPICalls())

	r.GET("/api/v1/networks", networksListHandler)
	r.GET("/api/v1/networks/:id", networkDetailsHandler)
	r.PUT("/api/v1/networks/:id", updateNetworkHandler)
	r.POST("/api/v1/networks", createNetworkHandler)
	r.GET("/api/v1/networks/:id/addresses", networkAddressesListHandler)
	r.GET("/api/v1/networks/:id/blocks", networkBlocksListHandler)
	r.GET("/api/v1/networks/:id/bridges", networkBridgesListHandler)
	r.GET("/api/v1/networks/:id/connectors", networkConnectorsListHandler)
	r.GET("/api/v1/networks/:id/contracts", networkContractsListHandler)
	r.GET("/api/v1/networks/:id/contracts/:contractId", networkContractDetailsHandler)
	r.GET("/api/v1/networks/:id/nodes", networkNodesListHandler)
	r.POST("/api/v1/networks/:id/nodes", createNetworkNodeHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId", networkNodeDetailsHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId/logs", networkNodeLogsHandler)
	r.DELETE("/api/v1/networks/:id/nodes/:nodeId", deleteNetworkNodeHandler)
	r.GET("/api/v1/networks/:id/oracles", networkOraclesListHandler)
	r.GET("/api/v1/networks/:id/status", networkStatusHandler)
	r.GET("/api/v1/networks/:id/tokens", networkTokensListHandler)
	r.GET("/api/v1/networks/:id/transactions", networkTransactionsListHandler)
	r.GET("/api/v1/networks/:id/transactions/:transactionId", networkTransactionDetailsHandler)

	r.GET("/api/v1/prices", pricesHandler)

	r.GET("/api/v1/connectors", connectorsListHandler)
	r.POST("/api/v1/connectors", createConnectorHandler)
	r.GET("/api/v1/connectors/:id", connectorDetailsHandler)
	r.DELETE("/api/v1/connectors/:id", deleteConnectorHandler)

	r.GET("/api/v1/contracts", contractsListHandler)
	r.GET("/api/v1/contracts/:id", contractDetailsHandler)
	r.POST("/api/v1/contracts", createContractHandler)
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
					return nil, fmt.Errorf("JWT claims specified non-%s subject: %s", subject, subprts[0])
				}
				id = subprts[1]
			}
		}
		return nil, nil
	}
	provide.ParseBearerAuthorizationHeader(c, &keyfn)
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

	network := &Network{}
	err = json.Unmarshal(buf, network)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	network.ApplicationID = appID
	network.UserID = userID

	if network.Create() {
		render(network, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		render(obj, 422, c)
	}
}

func updateNetworkHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	network := &Network{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network.ID == uuid.Nil {
		renderError("network not found", 404, c)
		return
	}

	if userID != nil && network.UserID != nil && *userID != *network.UserID {
		renderError("forbidden", 403, c)
		return
	}

	err = json.Unmarshal(buf, network)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	if network.Update() {
		render(nil, 204, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		render(obj, 422, c)
	}
}

func networksListHandler(c *gin.Context) {
	var networks []Network
	query := DatabaseConnection().Where("networks.enabled = true")

	if strings.ToLower(c.Query("cloneable")) == "true" {
		query = query.Where("networks.cloneable = true")
	} else if strings.ToLower(c.Query("cloneable")) == "false" {
		query = query.Where("networks.cloneable = false")
	}

	if strings.ToLower(c.Query("public")) == "true" {
		query = query.Where("networks.application_id IS NULL AND networks.user_id IS NULL")
	} else {
		appID := authorizedSubjectId(c, "application")
		if appID != nil {
			query = query.Where("networks.application_id = ?", appID)
		} else {
			query = query.Where("networks.application_id IS NULL")
		}

		userID := authorizedSubjectId(c, "user")
		if userID != nil {
			query = query.Where("networks.user_id = ?", userID)
		} else {
			query = query.Where("networks.user_id IS NULL")
		}
	}

	query = query.Order("networks.created_at ASC")
	provide.Paginate(c, query, &Network{}).Find(&networks)
	render(networks, 200, c)
}

func networkDetailsHandler(c *gin.Context) {
	var network = &Network{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		renderError("network not found", 404, c)
		return
	}
	render(network, 200, c)
}

func networkAddressesListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkBlocksListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkBridgesListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkConnectorsListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkContractsListHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := ContractListQuery()
	query = query.Where("contracts.network_id = ? AND contracts.application_id IS NULL", c.Param("id"))

	filterTokens := strings.ToLower(c.Query("filter_tokens")) == "true"
	if filterTokens {
		query = query.Joins("LEFT OUTER JOIN tokens ON tokens.contract_id = contracts.id").Where("symbol IS NULL")
	}

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("contracts.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("contracts.created_at ASC")
	}

	var contracts []Contract
	query = query.Order("contracts.created_at ASC")
	provide.Paginate(c, query, &Contract{}).Find(&contracts)
	render(contracts, 200, c)
}

// FIXME-- DRY this up
func networkContractDetailsHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	db := DatabaseConnection()
	var contract = &Contract{}

	query := db.Where("contracts.network_id = ? AND contracts.id = ?", c.Param("id"), c.Param("contractId"))
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL")
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("contracts.network_id = ? AND contracts.address = ?", c.Param("id"), c.Param("contractId")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		renderError("contract not found", 404, c)
		return
	}

	render(contract, 200, c)
}

func networkNodesListHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := DatabaseConnection().Where("network_nodes.network_id = ? AND network_nodes.user_id = ?", c.Param("id"), userID)

	var nodes []NetworkNode
	query = query.Order("network_nodes.created_at ASC")
	provide.Paginate(c, query, &NetworkNode{}).Find(&nodes)
	render(nodes, 200, c)
}

func networkNodeDetailsHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		renderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		renderError("forbidden", 403, c)
		return
	}

	render(node, 200, c)
}

func networkNodeLogsHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		renderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		renderError("forbidden", 403, c)
		return
	}

	logs, err := node.Logs()
	if err != nil {
		renderError(fmt.Sprintf("log retrieval failed; %s", err.Error()), 500, c)
		return
	}

	render(logs, 200, c)
}

func createNetworkNodeHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	networkID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		renderError(err.Error(), 400, c)
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	node := &NetworkNode{}
	err = json.Unmarshal(buf, node)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	node.Status = stringOrNil("pending")
	node.UserID = userID
	node.NetworkID = networkID

	var network = &Network{}
	DatabaseConnection().Model(node).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		renderError("network not found", 404, c)
		return
	}

	if network.UserID != nil && *network.UserID != *userID {
		renderError("forbidden", 403, c)
		return
	}

	if node.Create() {
		render(node, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = node.Errors
		render(obj, 422, c)
	}
}

func deleteNetworkNodeHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("nodeId")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		renderError("network node not found", 404, c)
		return
	}
	if *userID != *node.UserID {
		renderError("forbidden", 403, c)
		return
	}
	if !node.Delete() {
		renderError("network node not deleted", 500, c)
		return
	}
	render(nil, 204, c)
}

func networkStatusHandler(c *gin.Context) {
	var network = &Network{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		renderError("network not found", 404, c)
		return
	}
	status, err := network.Status(false)
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve network status; %s", err.Error())
		renderError(msg, 500, c)
		return
	}
	render(status, 200, c)
}

func networkOraclesListHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func networkTokensListHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := DatabaseConnection().Where("tokens.network_id = ? AND tokens.application_id IS NULL", c.Param("id"))

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("tokens.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("tokens.created_at ASC")
	}

	var tokens []Token
	provide.Paginate(c, query, &Token{}).Find(&tokens)
	render(tokens, 200, c)
}

func networkTransactionsListHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := DatabaseConnection().Where("transactions.network_id = ? AND transactions.application_id IS NULL", c.Param("id"))

	filterContractCreationTx := strings.ToLower(c.Query("filter_contract_creations")) == "true"
	if filterContractCreationTx {
		query = query.Where("transactions.to IS NULL")
	}

	if c.Query("status") != "" {
		query = query.Where("transactions.status IN ?", strings.Split(c.Query("status"), ","))
	}

	var txs []Transaction
	query = query.Order("created_at DESC")
	provide.Paginate(c, query, &Transaction{}).Find(&txs)
	render(txs, 200, c)
}

func networkTransactionDetailsHandler(c *gin.Context) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var tx = &Transaction{}
	DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("transactionId")).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		renderError("transaction not found", 404, c)
		return
	}
	err := tx.RefreshDetails()
	if err != nil {
		renderError("internal server error", 500, c)
		return
	}
	render(tx, 200, c)
}

// prices

func pricesHandler(c *gin.Context) {
	render(CurrentPrices, 200, c)
}

// connectors

func connectorsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := DatabaseConnection().Where("connectors.application_id = ?", appID)

	var connectors []Connector
	query = query.Order("created_at ASC")
	provide.Paginate(c, query, &Connector{}).Find(&connectors)
	render(connectors, 200, c)
}

func connectorDetailsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func createConnectorHandler(c *gin.Context) {
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

	connector := &Connector{}
	err = json.Unmarshal(buf, connector)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	connector.ApplicationID = appID

	if connector.Create() {
		render(connector, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = connector.Errors
		render(obj, 422, c)
	}
}

func deleteConnectorHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		renderError("connector not found", 404, c)
		return
	}
	if *appID != *connector.ApplicationID {
		renderError("forbidden", 403, c)
		return
	}
	if !connector.Delete() {
		renderError("connector not deleted", 500, c)
		return
	}
	render(nil, 204, c)
}

// contracts

func contractsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := ContractListQuery()

	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL")
	}

	filterTokens := strings.ToLower(c.Query("filter_tokens")) == "true"
	if filterTokens {
		query = query.Joins("LEFT OUTER JOIN tokens ON tokens.contract_id = contracts.id").Where("symbol IS NULL")
	}

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("contracts.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("contracts.created_at ASC")
	}

	var contracts []Contract
	provide.Paginate(c, query, &Contract{}).Find(&contracts)
	render(contracts, 200, c)
}

func contractDetailsHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	db := DatabaseConnection()
	var contract = &Contract{}

	query := db.Where("id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL", userID)
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		renderError("contract not found", 404, c)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	}

	render(contract, 200, c)
}

func createContractHandler(c *gin.Context) {
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

	contract := &Contract{}
	err = json.Unmarshal(buf, contract)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	contract.ApplicationID = appID

	params := contract.ParseParams()
	if contract.Name == nil {
		if constructor, constructorOk := params["constructor"].(string); constructorOk {
			contract.Name = &constructor
		} else if name, nameOk := params["name"].(string); nameOk {
			contract.Name = &name
		}
	}

	_, rawSourceOk := params["raw_source"].(string)
	if rawSourceOk && contract.Address == nil {
		contract.Address = stringOrNil("0x")
	}

	if contract.Create() {
		if rawSourceOk {
			render(contract, 202, c)
		} else {
			render(contract, 201, c)
		}
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = contract.Errors
		render(obj, 422, c)
	}
}

func contractArbitraryExecutionHandler(c *gin.Context, db *gorm.DB, buf []byte) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	wallet := &Wallet{} // signer for the tx

	params := map[string]interface{}{}
	err := json.Unmarshal(buf, &params)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	publicKey, publicKeyOk := params["public_key"].(string)
	privateKey, privateKeyOk := params["private_key"].(string)
	gas, gasOk := params["gas"].(float64)

	ref, err := uuid.NewV4()
	if err != nil {
		Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &ContractExecution{
		Ref: stringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying a wallet_id and wallet")
			renderError(err.Error(), 422, c)
			return
		}
		wallet.setID(*execution.WalletID)
	} else if publicKeyOk && privateKeyOk {
		wallet.Address = publicKey
		wallet.PrivateKey = stringOrNil(privateKey)
	}
	execution.Wallet = wallet

	if gasOk {
		execution.Gas = &gas
	}

	network := &Network{}
	if execution.NetworkID != nil && *execution.NetworkID != uuid.Nil {
		db.Where("id = ?", execution.NetworkID).Find(&network)
	}

	if network == nil || network.ID == uuid.Nil {
		renderError("network not found for arbitrary contract execution", 404, c)
		return
	}

	params = map[string]interface{}{
		"abi": execution.ABI,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		renderError("failed to marshal ephemeral contract params containing ABI", 422, c)
		return
	}
	paramsMsg := json.RawMessage(paramsJSON)

	ephemeralContract := &Contract{
		NetworkID: network.ID,
		Address:   stringOrNil(c.Param("id")),
		Params:    &paramsMsg,
	}

	_gas, _ := big.NewFloat(gas).Uint64()
	resp, err := ephemeralContract.Execute(execution.Ref, execution.Wallet, execution.Value, execution.Method, execution.Params, _gas)
	if err == nil {
		render(resp, 202, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = []string{err.Error()}
		render(obj, 422, c)
	}
}

func contractExecutionHandler(c *gin.Context) {
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

	db := DatabaseConnection()
	var contract = &Contract{}

	db.Where("id = ?", c.Param("id")).Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		if appID != nil {
			renderError("contract not found", 404, c)
			return
		}

		Log.Debugf("Attempting arbitrary, non-permissioned contract execution on behalf of user with id: %s", userID)
		contractArbitraryExecutionHandler(c, db, buf)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	}

	ref, err := uuid.NewV4()
	if err != nil {
		Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &ContractExecution{
		Ref: stringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	execution.Contract = contract
	execution.ContractID = &contract.ID
	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying a wallet_id and wallet")
			renderError(err.Error(), 422, c)
			return
		}
		wallet := &Wallet{}
		wallet.setID(*execution.WalletID)
		execution.Wallet = wallet
	}

	executionResponse, err := execution.Execute()
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	switch executionResponse.(type) {
	case *ContractExecutionResponse:
		executionResponse = map[string]interface{}{
			"response": executionResponse.(*ContractExecutionResponse).Response,
		}
		render(executionResponse, 200, c) // returns 200 OK status to indicate the contract invocation was able to return a syncronous response
	default:
		confidence := invokeTxFilters(appID, buf, db)
		executionResponse = map[string]interface{}{
			"confidence": confidence,
			"ref":        executionResponse.(*ContractExecution).Ref,
		}
		render(executionResponse, 202, c) // returns 202 Accepted status to indicate the contract invocation is pending
	}
}

func invokeTxFilters(applicationID *uuid.UUID, payload []byte, db *gorm.DB) *float64 {
	if applicationID == nil {
		Log.Warningf("Tx filters are not currently supported for transactions outside of the scope of an application context")
		return nil
	}

	if _, hasConfiguredFilter := txFilters[applicationID.String()]; !hasConfiguredFilter {
		Log.Debugf("No tx filters to invoke for application: %s", applicationID.String())
		return nil
	}

	var confidence *float64
	var filters []Filter
	query := db.Where("application_id = ?", applicationID).Order("priority ASC") // TODO: load the filters into memory
	query.Find(&filters)
	for _, filter := range filters {
		if confidence == nil {
			_confidence := float64(0.0)
			confidence = &_confidence
		}
		confidence = filter.Invoke(payload) // TODO: discuss order and priority of filters
	}
	return confidence
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
	query = query.Order("created_at ASC")
	provide.Paginate(c, query, &Oracle{}).Find(&oracles)
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

	query := DatabaseConnection().Where("tokens.application_id = ?", appID)

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("tokens.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("tokens.created_at ASC")
	}

	var tokens []Token
	provide.Paginate(c, query, &Token{}).Find(&tokens)
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

	if c.Query("status") != "" {
		query = query.Where("transactions.status IN ?", strings.Split(c.Query("status"), ","))
	}

	var txs []Transaction
	query = query.Order("created_at DESC")
	provide.Paginate(c, query, &Transaction{}).Find(&txs)
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

	db := DatabaseConnection()

	var tx = &Transaction{}
	db.Where("id = ?", c.Param("id")).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		db.Where("ref = ?", c.Param("id")).Find(&tx)
		if tx == nil || tx.ID == uuid.Nil {
			renderError("transaction not found", 404, c)
			return
		}
	} else if appID != nil && tx.ApplicationID != nil && *tx.ApplicationID != *appID {
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

	query := DatabaseConnection()

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
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
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
	DatabaseConnection().Where("id = ?", c.Param("id")).Find(&wallet)
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
