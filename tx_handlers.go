package main

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

// InstallTxAPI installs the handlers using the given gin Engine
func InstallTransactionsAPI(r *gin.Engine) {
	r.GET("/api/v1/transactions", transactionsListHandler)
	r.POST("/api/v1/transactions", createTransactionHandler)
	r.GET("/api/v1/transactions/:id", transactionDetailsHandler)
}

func transactionsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	var query *gorm.DB
	if appID != nil {
		query = dbconf.DatabaseConnection().Where("transactions.application_id = ?", appID)
	} else if userID != nil {
		query = dbconf.DatabaseConnection().Where("transactions.user_id = ?", userID)
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

	db := dbconf.DatabaseConnection()

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
