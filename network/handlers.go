/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package network

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	c2 "github.com/provideplatform/provide-go/api/c2"
	api "github.com/provideplatform/provide-go/api/nchain"
	provide "github.com/provideplatform/provide-go/common"
	util "github.com/provideplatform/provide-go/common/util"
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
	r.GET("/api/v1/networks/:id/status", networkStatusHandler)

	r.GET("/api/v1/networks/:id/load_balancers", loadBalancersListHandler)
	r.GET("/api/v1/networks/:id/load_balancers/:loadBalancerId", loadBalancerDetailsHandler)

	r.GET("/api/v1/networks/:id/nodes", nodesListHandler)
	r.POST("/api/v1/networks/:id/nodes", createNodeHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId", nodeDetailsHandler)
	r.GET("/api/v1/networks/:id/nodes/:nodeId/logs", nodeLogsHandler)
	r.DELETE("/api/v1/networks/:id/nodes/:nodeId", deleteNodeHandler)

	r.GET("/api/v1/networks/:id/oracles", networkOraclesListHandler)
}

func createNetworkHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
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
	userID := util.AuthorizedSubjectID(c, "user")
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
	query := ListQuery()
	query = query.Where("networks.enabled = true")

	if strings.ToLower(c.Query("cloneable")) == "true" {
		query = query.Where("networks.cloneable = true")
	} else if strings.ToLower(c.Query("cloneable")) == "false" {
		query = query.Where("networks.cloneable = false")
	}

	if strings.ToLower(c.Query("layer2")) == "true" {
		query = query.Where("networks.layer2 IS TRUE")
	} else if strings.ToLower(c.Query("layer2")) == "false" {
		query = query.Where("networks.layer2 IS FALSE")
	}

	if strings.ToLower(c.Query("layer3")) == "true" {
		query = query.Where("networks.layer3 IS TRUE")
	} else if strings.ToLower(c.Query("layer3")) == "false" {
		query = query.Where("networks.layer3 IS FALSE")
	}

	if strings.ToLower(c.Query("public")) == "true" {
		query = query.Where("networks.application_id IS NULL AND networks.user_id IS NULL")
	} else {
		appID := util.AuthorizedSubjectID(c, "application")
		if appID != nil {
			query = query.Where("networks.application_id = ?", appID)
		} else {
			query = query.Where("networks.application_id IS NULL")
		}

		userID := util.AuthorizedSubjectID(c, "user")
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
		ntwrk.SetConfig(cfg)
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

	resp, err := c2.ListLoadBalancers(c.GetString("token"), params)
	if err != nil {
		provide.RenderError(err.Error(), 500, c)
		return
	}
	provide.Render(resp, 200, c)
}

func loadBalancerDetailsHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func nodesListHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	appID := util.AuthorizedSubjectID(c, "application")
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

	var nodes []*Node
	query = query.Order("nodes.created_at ASC")
	provide.Paginate(c, query, &Node{}).Find(&nodes)
	provide.Render(nodes, 200, c)
}

func nodeDetailsHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	appID := util.AuthorizedSubjectID(c, "application")
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
	}

	provide.Render(node, 200, c)
}

func nodeLogsHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	appID := util.AuthorizedSubjectID(c, "application")
	if userID == nil && appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var node = &Node{}
	dbconf.DatabaseConnection().Where("id = ? AND network_id = ?", c.Param("nodeId"), c.Param("id")).Find(&node)
	if node == nil || node.ID == uuid.Nil {
		provide.RenderError("network node not found", 404, c)
		return
	} else if userID != nil && node.UserID != nil && *node.UserID != *userID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if appID != nil && node.ApplicationID != nil && *node.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	resp, err := c2.GetNodeLogs(c.GetString("token"), node.ID.String(), map[string]interface{}{
		"page":            c.DefaultQuery("page", "1"),
		"rpp":             c.DefaultQuery("rpp", strconv.Itoa(int(defualtNodeLogRPP))),
		"start_from_head": c.DefaultQuery("start_from_head", "false"),
	})
	if err != nil {
		provide.RenderError(fmt.Sprintf("log retrieval failed; %s", err.Error()), 500, c)
		return
	}

	provide.Render(resp, 200, c)
}

func createNodeHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	appID := util.AuthorizedSubjectID(c, "application")
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

	params := map[string]interface{}{}
	err = json.Unmarshal(buf, &params)
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

	if node.OrganizationID != nil {
		provide.RenderError("unable to set organization_id on node creation API at this time; org authorization not yet implemented in nchain", 400, c)
		return
	}

	if node.Create(c.GetString("token")) {
		provide.Render(node, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = node.Errors
		provide.Render(obj, 422, c)
	}
}

func deleteNodeHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	appID := util.AuthorizedSubjectID(c, "application")
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

	if !node.Delete(c.GetString("token")) {
		provide.RenderError("network node not deleted", 500, c)
		return
	}
	provide.Render(nil, 204, c)
}

func networkStatusHandler(c *gin.Context) {
	networkID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		provide.RenderError("invalid network id provided", 400, c)
		return
	}
	stats, err := Stats(networkID)
	if err != nil {
		var network = &Network{}
		dbconf.DatabaseConnection().Where("id = ?", c.Param("id")).Find(&network)
		if network == nil || network.ID == uuid.Nil {
			provide.RenderError("network not found", 404, c)
			return
		}
		provide.Render(&api.NetworkStatus{
			ChainID: network.ChainID,
			State:   common.StringOrNil(networkStateGenesis),
			Syncing: true,
		}, 200, c)
		return
	}
	provide.Render(stats, 200, c)
}

func networkOraclesListHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}
