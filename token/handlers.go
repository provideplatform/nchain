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

package token

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	provide "github.com/provideplatform/provide-go/common"
	util "github.com/provideplatform/provide-go/common/util"
)

// InstallTokensAPI installs the handlers using the given gin Engine
func InstallTokensAPI(r *gin.Engine) {
	r.GET("/api/v1/tokens", tokensListHandler)
	r.GET("/api/v1/tokens/:id", tokenDetailsHandler)
	r.POST("/api/v1/tokens", createTokenHandler)
	r.GET("/api/v1/networks/:id/tokens", networkTokensListHandler)

}

func tokensListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("tokens.application_id = ?", appID)

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("tokens.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("tokens.created_at ASC")
	}

	var tokens []Token
	provide.Paginate(c, query, &Token{}).Find(&tokens)
	provide.Render(tokens, 200, c)
}

func tokenDetailsHandler(c *gin.Context) {
	provide.RenderError("not implemented", 501, c)
}

func createTokenHandler(c *gin.Context) {
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

	token := &Token{}
	err = json.Unmarshal(buf, token)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	token.ApplicationID = appID

	if token.Create() {
		provide.Render(token, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = token.Errors
		provide.Render(obj, 422, c)
	}
}

func networkTokensListHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
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
	provide.Render(tokens, 200, c)
}
