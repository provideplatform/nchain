package network

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

const defualtNetworkNodeLogRPP = int64(100)

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
	r.GET("/api/v1/networks/:id/nodes", networkNodesListHandler)
	r.POST("/api/v1/networks/:id/nodes", createNetworkNodeHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId", networkNodeDetailsHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId/logs", networkNodeLogsHandler)
	r.DELETE("/api/v1/networks/:id/nodes/:nodeId", deleteNetworkNodeHandler)
	r.GET("/api/v1/networks/:id/oracles", networkOraclesListHandler)
	r.GET("/api/v1/networks/:id/status", networkStatusHandler)
}

func createNetworkHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	userID := common.AuthorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}

	network := &Network{}
	err = json.Unmarshal(buf, network)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}
	network.ApplicationID = appID
	network.UserID = userID

	if network.Create() {
		common.Render(network, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		common.Render(obj, 422, c)
	}
}

func updateNetworkHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	if userID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}

	network := &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network.ID == uuid.Nil {
		common.RenderError("network not found", 404, c)
		return
	}

	if userID != nil && network.UserID != nil && *userID != *network.UserID {
		common.RenderError("forbidden", 403, c)
		return
	}

	err = json.Unmarshal(buf, network)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}

	if network.Update() {
		common.Render(nil, 204, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		common.Render(obj, 422, c)
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
		appID := common.AuthorizedSubjectId(c, "application")
		if appID != nil {
			query = query.Where("networks.application_id = ?", appID)
		} else {
			query = query.Where("networks.application_id IS NULL")
		}

		userID := common.AuthorizedSubjectId(c, "user")
		if userID != nil {
			query = query.Where("networks.user_id = ?", userID)
		} else {
			query = query.Where("networks.user_id IS NULL")
		}
	}

	query = query.Order("networks.created_at ASC")
	provide.Paginate(c, query, &Network{}).Find(&networks)
	common.Render(networks, 200, c)
}

func networkDetailsHandler(c *gin.Context) {
	var network = &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		common.RenderError("network not found", 404, c)
		return
	}
	common.Render(network, 200, c)
}

func networkAddressesListHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func networkBlocksListHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func networkBridgesListHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func networkConnectorsListHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func networkNodesListHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	appID := common.AuthorizedSubjectId(c, "application")
	if userID == nil && appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("network_nodes.network_id = ?", c.Param("id"))

	if userID != nil {
		query = query.Where("network_nodes.user_id = ?", userID)
	}

	if appID != nil {
		query = query.Where("network_nodes.application_id = ?", appID)
	}

	var nodes []NetworkNode
	query = query.Order("network_nodes.created_at ASC")
	provide.Paginate(c, query, &NetworkNode{}).Find(&nodes)
	common.Render(nodes, 200, c)
}

func networkNodeDetailsHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	appID := common.AuthorizedSubjectId(c, "application")
	if userID == nil && appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		common.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	}

	common.Render(node, 200, c)
}

func networkNodeLogsHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	appID := common.AuthorizedSubjectId(c, "application")
	if userID == nil && appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		common.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	}

	page := c.Query("page")
	rpp := c.Query("rpp")
	var limit int64
	limit, err := strconv.ParseInt(rpp, 10, 64)
	if err != nil {
		limit = defualtNetworkNodeLogRPP
	}

	logs, err := node.Logs(false, &limit, common.StringOrNil(page))
	if err != nil {
		common.RenderError(fmt.Sprintf("log retrieval failed; %s", err.Error()), 500, c)
		return
	}

	common.Render(logs, 200, c)
}

func createNetworkNodeHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	appID := common.AuthorizedSubjectId(c, "application")
	if userID == nil && appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	networkID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		common.RenderError(err.Error(), 400, c)
	}

	buf, err := c.GetRawData()
	if err != nil {
		common.RenderError(err.Error(), 400, c)
		return
	}

	node := &NetworkNode{}
	err = json.Unmarshal(buf, node)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}
	node.Status = common.StringOrNil("pending")
	node.NetworkID = networkID
	node.UserID = userID
	node.ApplicationID = appID

	var network = &Network{}
	dbconf.DatabaseConnection().Model(node).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		common.RenderError("network not found", 404, c)
		return
	}

	if network.UserID != nil && userID != nil && *network.UserID != *userID {
		common.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	}

	if node.Create() {
		common.Render(node, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = node.Errors
		common.Render(obj, 422, c)
	}
}

func deleteNetworkNodeHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	appID := common.AuthorizedSubjectId(c, "application")
	if userID == nil && appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	var node = &NetworkNode{}
	dbconf.DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("nodeId")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		common.RenderError("network node not found", 404, c)
		return
	}
	if userID != nil && node.UserID != nil && *userID != *node.UserID {
		common.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && node.ApplicationID != nil && *node.ApplicationID != *appID {
		common.RenderError("forbidden", 403, c)
		return
	}

	if !node.Delete() {
		common.RenderError("network node not deleted", 500, c)
		return
	}
	common.Render(nil, 204, c)
}

func networkStatusHandler(c *gin.Context) {
	var network = &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		common.RenderError("network not found", 404, c)
		return
	}
	status, err := network.Status(false)
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve network status; %s", err.Error())
		common.RenderError(msg, 500, c)
		return
	}
	common.Render(status, 200, c)
}

func networkOraclesListHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}
