package token

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// InstallTokensAPI installs the handlers using the given gin Engine
func InstallTokensAPI(r *gin.Engine) {
	r.GET("/api/v1/tokens", tokensListHandler)
	r.GET("/api/v1/tokens/:id", tokenDetailsHandler)
	r.POST("/api/v1/tokens", createTokenHandler)
	r.GET("/api/v1/networks/:id/tokens", networkTokensListHandler)

}

func tokensListHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	if appID == nil {
		common.RenderError("unauthorized", 401, c)
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
	common.Render(tokens, 200, c)
}

func tokenDetailsHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func createTokenHandler(c *gin.Context) {
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

	token := &Token{}
	err = json.Unmarshal(buf, token)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}
	token.ApplicationID = appID

	if token.Create() {
		common.Render(token, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = token.Errors
		common.Render(obj, 422, c)
	}
}

func networkTokensListHandler(c *gin.Context) {
	userID := common.AuthorizedSubjectId(c, "user")
	if userID == nil {
		common.RenderError("unauthorized", 401, c)
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
	common.Render(tokens, 200, c)
}
