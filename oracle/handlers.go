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

package oracle

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	provide "github.com/provideplatform/provide-go/common"
	util "github.com/provideplatform/provide-go/common/util"
)

// InstallOraclesAPI installs the handlers using the given gin Engine
func InstallOraclesAPI(r *gin.Engine) {
	r.GET("/api/v1/oracles", oraclesListHandler)
	r.POST("/api/v1/oracles", createOracleHandler)
	r.GET("/api/v1/oracles/:id", oracleDetailsHandler)
}

func oraclesListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("oracles.application_id = ?", appID)

	var oracles []Oracle
	query = query.Order("created_at ASC")
	provide.Paginate(c, query, &Oracle{}).Find(&oracles)
	provide.Render(oracles, 200, c)
}

func oracleDetailsHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func createOracleHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	oracle := &Oracle{}
	err = json.Unmarshal(buf, oracle)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	oracle.ApplicationID = appID

	if oracle.Create() {
		provide.Render(oracle, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = oracle.Errors
		provide.Render(obj, 422, c)
	}
}
