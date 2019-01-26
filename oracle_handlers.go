package main

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	provide "github.com/provideservices/provide-go"
)

// InstallOraclesAPI installs the handlers using the given gin Engine
func InstallOraclesAPI(r *gin.Engine) {
	r.GET("/api/v1/oracles", oraclesListHandler)
	r.POST("/api/v1/oracles", createOracleHandler)
	r.GET("/api/v1/oracles/:id", oracleDetailsHandler)
}

func oraclesListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("oracles.application_id = ?", appID)

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
