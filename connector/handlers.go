package connector

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// InstallConnectorsAPI installs the handlers using the given gin Engine
func InstallConnectorsAPI(r *gin.Engine) {
	r.GET("/api/v1/connectors", connectorsListHandler)
	r.POST("/api/v1/connectors", createConnectorHandler)
	r.GET("/api/v1/connectors/:id", connectorDetailsHandler)
	r.DELETE("/api/v1/connectors/:id", deleteConnectorHandler)
}

func connectorsListHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	if appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("connectors.application_id = ?", appID)

	var connectors []Connector
	query = query.Order("created_at ASC")
	provide.Paginate(c, query, &Connector{}).Find(&connectors)
	common.Render(connectors, 200, c)
}

func connectorDetailsHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func createConnectorHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	if appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}

	connector := &Connector{}
	err = json.Unmarshal(buf, connector)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}
	connector.ApplicationID = appID

	if connector.Create() {
		common.Render(connector, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = connector.Errors
		common.Render(obj, 422, c)
	}
}

func deleteConnectorHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	if appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		common.RenderError("connector not found", 404, c)
		return
	}
	if *appID != *connector.ApplicationID {
		common.RenderError("forbidden", 403, c)
		return
	}
	if !connector.Delete() {
		common.RenderError("connector not deleted", 500, c)
		return
	}
	common.Render(nil, 204, c)
}