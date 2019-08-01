package token

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
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
	appID := provide.AuthorizedSubjectID(c, "application")
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
	userID := provide.AuthorizedSubjectID(c, "user")
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
