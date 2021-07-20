package connector

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
	c2 "github.com/provideservices/provide-go/api/c2"
	provide "github.com/provideservices/provide-go/common"
	util "github.com/provideservices/provide-go/common/util"
)

// InstallConnectorsAPI installs the handlers using the given gin Engine
func InstallConnectorsAPI(r *gin.Engine) {
	r.GET("/api/v1/connectors", connectorsListHandler)
	r.POST("/api/v1/connectors", createConnectorHandler)
	r.GET("/api/v1/connectors/:id", connectorDetailsHandler)
	r.DELETE("/api/v1/connectors/:id", deleteConnectorHandler)

	r.GET("/api/v1/connectors/:id/entities", connectorEntitiesListHandler)
	r.POST("/api/v1/connectors/:id/entities", connectorEntityCreateHandler)
	r.GET("/api/v1/connectors/:id/entities/:entityId", connectorEntityDetailsHandler)
	r.PUT("/api/v1/connectors/:id/entities/:entityId", updateConnectorEntityHandler)
	r.DELETE("/api/v1/connectors/:id/entities/:entityId", deleteConnectorEntityHandler)

	r.GET("/api/v1/connectors/:id/load_balancers", connectorLoadBalancersListHandler)
	r.GET("/api/v1/connectors/:id/nodes", connectorNodesListHandler)
}

func connectorsListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection()

	if appID != nil {
		query = query.Where("connectors.application_id = ?", appID)
	}

	if orgID != nil {
		query = query.Where("connectors.organization_id = ?", orgID)
	}

	if c.Query("type") != "" {
		query = query.Where("connectors.type = ?", c.Query("type"))
	}

	var connectors []*Connector
	query = query.Order("connectors.created_at ASC")
	provide.Paginate(c, query, &Connector{}).Find(&connectors)
	provide.Render(connectors, 200, c)
}

func connectorDetailsHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")

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
	if appID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && *orgID != *connector.OrganizationID {
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
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
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

	if orgID != nil {
		// HACK!!! FIXME!!! this should not happen conditionally
		connector.OrganizationID = orgID
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
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.ApplicationID != nil && *orgID != *connector.ApplicationID {
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
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	connector := &Connector{}

	query := db.Where("connectors.id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("connectors.application_id = ?", appID)
	}
	if orgID != nil {
		query = query.Where("connectors.organization_id = ?", orgID)
	}
	query.Find(&connector)

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

	loadBalancers := make([]*c2.LoadBalancer, 0)
	db.Model(&connector).Association("LoadBalancers").Find(&loadBalancers)
	provide.Render(loadBalancers, 200, c)
}

func connectorNodesListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	connector := &Connector{}

	query := db.Where("connectors.id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("connectors.application_id = ?", appID)
	}
	if orgID != nil {
		query = query.Where("connectors.organization_id = ?", orgID)
	}
	query.Find(&connector)

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

func connectorEntitiesListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.OrganizationID != nil && *orgID != *connector.OrganizationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	params := map[string]interface{}{}
	queryParams := c.Request.URL.Query()
	for k := range queryParams {
		params[k] = queryParams[k]
	}

	resp, err := connector.listEntities(params)
	if err != nil {
		provide.RenderError(err.Error(), 500, c)
		return
	}

	provide.Render(resp, 200, c)
}

func connectorEntityCreateHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	params := map[string]interface{}{}
	err = json.Unmarshal(buf, &params)
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.OrganizationID != nil && *orgID != *connector.OrganizationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	resp, err := connector.createEntity(params)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	provide.Render(resp, 201, c)
}

func connectorEntityDetailsHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.OrganizationID != nil && *orgID != *connector.OrganizationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	resp, err := connector.findEntity(c.Param("entityId"))
	if err != nil {
		provide.RenderError(err.Error(), 500, c)
		return
	}

	provide.Render(resp, 200, c)
}

func updateConnectorEntityHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	params := map[string]interface{}{}
	err = json.Unmarshal(buf, &params)
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.OrganizationID != nil && *orgID != *connector.OrganizationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	err = connector.updateEntity(c.Param("entityId"), params)
	if err != nil {
		provide.RenderError(err.Error(), 500, c)
		return
	}

	provide.Render(nil, 204, c)
}

func deleteConnectorEntityHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var connector = &Connector{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		provide.RenderError("connector not found", 404, c)
		return
	}
	if appID != nil && connector.ApplicationID != nil && *appID != *connector.ApplicationID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	if orgID != nil && connector.OrganizationID != nil && *orgID != *connector.OrganizationID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	err := connector.deleteEntity(c.Param("entityId"))
	if err != nil {
		provide.RenderError(err.Error(), 500, c)
		return
	}

	provide.Render(nil, 204, c)
}
