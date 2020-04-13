package connector

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/network"
	provide "github.com/provideservices/provide-go"
)

// InstallConnectorsAPI installs the handlers using the given gin Engine
func InstallConnectorsAPI(r *gin.Engine) {
	r.GET("/api/v1/connectors", connectorsListHandler)
	r.POST("/api/v1/connectors", createConnectorHandler)
	r.GET("/api/v1/connectors/:id", connectorDetailsHandler)
	r.DELETE("/api/v1/connectors/:id", deleteConnectorHandler)

	r.GET("/api/v1/connectors/:id/load_balancers", connectorLoadBalancersListHandler)
	r.GET("/api/v1/connectors/:id/nodes", connectorNodesListHandler)
}

func connectorsListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("connectors.application_id = ?", appID)

	if c.Query("type") != "" {
		query = query.Where("connectors.type = ?", c.Query("type"))
	}

	var connectors []*Connector
	query = query.Order("connectors.created_at ASC")
	provide.Paginate(c, query, &Connector{}).Find(&connectors)
	provide.Render(connectors, 200, c)
}

func connectorDetailsHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	enrichment := common.StringOrNil(c.Query("enrichment"))
	connector.enrich(enrichment, map[string]interface{}{
		"objects": strings.Split(c.Query("objects"), ","),
	})

	provide.Render(connector, 200, c)
}

func createConnectorHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	connector := &Connector{}
	err = json.Unmarshal(buf, connector)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	connector.ApplicationID = appID

	if connector.OrganizationID != nil {
		common.Log.Warningf("setting organization_id on connector via creation API; needs audit! at this time, org authorization not yet implemented in goldmine")
	}

	if connector.Create() {
		provide.Render(connector, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = connector.Errors
		provide.Render(obj, 422, c)
	}
}

func deleteConnectorHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if !connector.Delete() {
		provide.RenderError("connector not deleted", 500, c)
		return
	}
	provide.Render(nil, 204, c)
}

func connectorLoadBalancersListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	connector := &Connector{}

	query := db.Where("connectors.application_id = ?", appID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}

	if appID != nil {
		query = query.Where("load_balancers.application_id = ?", appID)
	}

	if c.Query("region") != "" {
		query = query.Where("load_balancers.region = ?", c.Query("region"))
	}
	if c.Query("status") != "" {
		query = query.Where("load_balancers.status = ?", c.Query("status"))
	}
	if c.Query("type") != "" {
		query = query.Where("load_balancers.type = ?", c.Query("type"))
	}

	loadBalancers := make([]*network.LoadBalancer, 0)
	db.Model(&connector).Association("LoadBalancers").Find(&loadBalancers)
	provide.Render(loadBalancers, 200, c)
}

func connectorNodesListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	connector := &Connector{}

	query := db.Where("connectors.application_id = ?", appID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}

	if appID != nil {
		query = query.Where("nodes.application_id = ?", appID)
	}

	nodes := make([]*network.Node, 0)
	db.Model(&connector).Association("Nodes").Find(&nodes)
	provide.Render(nodes, 200, c)
}
