package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

// InstallNetworksAPI installs the handlers using the given gin Engine
func InstallNetworksAPI(r *gin.Engine) {
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
}

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
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
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
	query := dbconf.DatabaseConnection().Where("networks.enabled = true")

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
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
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

	db := dbconf.DatabaseConnection()
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

	query := dbconf.DatabaseConnection().Where("network_nodes.network_id = ? AND network_nodes.user_id = ?", c.Param("id"), userID)

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
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
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
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
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
	node.Status = StringOrNil("pending")
	node.UserID = userID
	node.NetworkID = networkID

	var network = &Network{}
	dbconf.DatabaseConnection().Model(node).Related(&network)
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
	dbconf.DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("nodeId")).Find(&node)
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
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
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

	query := dbconf.DatabaseConnection().Where("tokens.network_id = ? AND tokens.application_id IS NULL", c.Param("id"))

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

	query := dbconf.DatabaseConnection().Where("transactions.network_id = ? AND transactions.application_id IS NULL", c.Param("id"))

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
	dbconf.DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("transactionId")).Find(&tx)
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
