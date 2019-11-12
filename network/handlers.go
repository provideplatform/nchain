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

const defualtNodeLogRPP = int64(100)

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
	r.GET("/api/v1/networks/:id/status", networkStatusHandler)

	r.GET("/api/v1/networks/:id/load_balancers", loadBalancersListHandler)
	r.GET("/api/v1/networks/:id/load_balancers/:loadBalancerId", loadBalancerDetailsHandler)
	r.PUT("/api/v1/networks/:id/load_balancers/:loadBalancerId", updateLoadBalancerHandler)

	r.GET("/api/v1/networks/:id/nodes", nodesListHandler)
	r.POST("/api/v1/networks/:id/nodes", createNodeHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId", nodeDetailsHandler)
	r.PUT("/api/v1/networks/:id/nodes/:nodeId", updateNodeHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId/logs", nodeLogsHandler)
	r.DELETE("/api/v1/networks/:id/nodes/:nodeId", deleteNodeHandler)

	r.GET("/api/v1/networks/:id/oracles", networkOraclesListHandler)
}

func createNetworkHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	network := &Network{}
	err = json.Unmarshal(buf, network)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	network.ApplicationID = appID
	network.UserID = userID

	if network.Create() {
		provide.Render(network, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		provide.Render(obj, 422, c)
	}
}

func updateNetworkHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	network := &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network.ID == uuid.Nil {
		provide.RenderError("network not found", 404, c)
		return
	}

	if userID != nil && network.UserID != nil && *userID != *network.UserID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	err = json.Unmarshal(buf, network)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	if network.Update() {
		provide.Render(nil, 204, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = network.Errors
		provide.Render(obj, 422, c)
	}
}

func networksListHandler(c *gin.Context) {
	var networks []*Network
	query := NetworkListQuery()
	query = query.Where("networks.enabled = true")

	if strings.ToLower(c.Query("cloneable")) == "true" {
		query = query.Where("networks.cloneable = true")
	} else if strings.ToLower(c.Query("cloneable")) == "false" {
		query = query.Where("networks.cloneable = false")
	}

	if strings.ToLower(c.Query("public")) == "true" {
		query = query.Where("networks.application_id IS NULL AND networks.user_id IS NULL")
	} else {
		appID := provide.AuthorizedSubjectID(c, "application")
		if appID != nil {
			query = query.Where("networks.application_id = ?", appID)
		} else {
			query = query.Where("networks.application_id IS NULL")
		}

		userID := provide.AuthorizedSubjectID(c, "user")
		if userID != nil {
			query = query.Where("networks.user_id = ?", userID)
		} else {
			query = query.Where("networks.user_id IS NULL")
		}
	}

	query = query.Order("networks.created_at ASC")
	provide.Paginate(c, query, &Network{}).Find(&networks)
	for _, ntwrk := range networks {
		cfg := ntwrk.ParseConfig()
		delete(cfg, "chainspec")
		delete(cfg, "chainspec_abi")
		ntwrk.setConfig(cfg)
	}
	provide.Render(networks, 200, c)
}

func networkDetailsHandler(c *gin.Context) {
	var network = &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		provide.RenderError("network not found", 404, c)
		return
	}
	provide.Render(network, 200, c)
}

func networkAddressesListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func networkBlocksListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func networkBridgesListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func networkConnectorsListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func loadBalancersListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := LoadBalancerListQuery()
	query = query.Where("load_balancers.network_id = ?", c.Param("id"))
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

	var loadBalancers []LoadBalancer
	query = query.Order("load_balancers.created_at ASC")
	provide.Paginate(c, query, &Node{}).Find(&loadBalancers)
	provide.Render(loadBalancers, 200, c)
}

func loadBalancerDetailsHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func updateLoadBalancerHandler(c *gin.Context) {
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

	var loadBalancer = &LoadBalancer{}
	query := dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("loadBalancerId"), c.Param("id"))
	if appID != nil {
		query = query.Where("load_balancers.application_id = ?", appID)
	}
	query.Find(&loadBalancer)

	if loadBalancer == nil || loadBalancer.ID == uuid.Nil {
		provide.RenderError("load balancer not found", 404, c)
		return
	} else if appID != nil && loadBalancer.ApplicationID != nil && *loadBalancer.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if loadBalancer.ApplicationID == nil {
		// shouldn't be able to reach this branch, here for now to be safe
		provide.RenderError("forbidden", 403, c)
		return
	}

	var initialStatus string
	if loadBalancer.Status != nil {
		initialStatus = *loadBalancer.Status
	}

	err = json.Unmarshal(buf, loadBalancer)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	loadBalancer.Status = common.StringOrNil(initialStatus)

	if loadBalancer.Update() {
		provide.Render(nil, 204, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = loadBalancer.Errors
		provide.Render(obj, 422, c)
	}
}

func nodesListHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := NodeListQuery()
	query = query.Where("nodes.network_id = ?", c.Param("id"))

	if userID != nil {
		query = query.Where("nodes.user_id = ?", userID)
	}

	if appID != nil {
		query = query.Where("nodes.application_id = ?", appID)
	}

	var nodes []Node
	query = query.Order("nodes.created_at ASC")
	provide.Paginate(c, query, &Node{}).Find(&nodes)
	provide.Render(nodes, 200, c)
}

func nodeDetailsHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var node = &Node{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		provide.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && node.ApplicationID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else {
		provide.RenderError("forbidden", 403, c)
		return
	}

	provide.Render(node, 200, c)
}

func nodeLogsHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var node = &Node{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		provide.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	page := c.Query("page")
	rpp := c.Query("rpp")
	var limit int64
	limit, err := strconv.ParseInt(rpp, 10, 64)
	if err != nil {
		limit = defualtNodeLogRPP
	}

	logs, err := node.Logs(false, &limit, common.StringOrNil(page))
	if err != nil {
		provide.RenderError(fmt.Sprintf("log retrieval failed; %s", err.Error()), 500, c)
		return
	}

	provide.Render(logs, 200, c)
}

func createNodeHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	networkID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	node := &Node{}
	err = json.Unmarshal(buf, node)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	node.Status = common.StringOrNil("pending")
	node.NetworkID = networkID
	node.UserID = userID
	node.ApplicationID = appID

	var network = &Network{}
	dbconf.DatabaseConnection().Model(node).Related(&network)
	if network == nil || network.ID == uuid.Nil {
		provide.RenderError("network not found", 404, c)
		return
	}

	if network.UserID != nil && userID != nil && *network.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	if node.Create() {
		provide.Render(node, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = node.Errors
		provide.Render(obj, 422, c)
	}
}

func updateNodeHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	var node = &Node{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		provide.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && *node.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	var initialStatus string
	if node.Status != nil {
		initialStatus = *node.Status
	}

	err = json.Unmarshal(buf, node)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	node.Status = common.StringOrNil(initialStatus)

	if node.Update() {
		provide.Render(nil, 204, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = node.Errors
		provide.Render(obj, 422, c)
	}
}

func deleteNodeHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	appID := provide.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var node = &Node{}
	dbconf.DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("nodeId")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		provide.RenderError("network node not found", 404, c)
		return
	}
	if userID != nil && node.UserID != nil && *userID != *node.UserID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && node.ApplicationID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	if !node.Delete() {
		provide.RenderError("network node not deleted", 500, c)
		return
	}
	provide.Render(nil, 204, c)
}

func networkStatusHandler(c *gin.Context) {
	var network = &Network{}
	dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		provide.RenderError("network not found", 404, c)
		return
	}
	status, err := network.Status(false)
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve network status; %s", err.Error())
		provide.RenderError(msg, 500, c)
		return
	}
	provide.Render(status, 200, c)
}

func networkOraclesListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}
