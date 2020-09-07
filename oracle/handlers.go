package oracle

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	provide "github.com/provideservices/provide-go/common"
)

// InstallOraclesAPI installs the handlers using the given gin Engine
func InstallOraclesAPI(r *gin.Engine) {
	r.GET("/api/v1/oracles", oraclesListHandler)
	r.POST("/api/v1/oracles", createOracleHandler)
	r.GET("/api/v1/oracles/:id", oracleDetailsHandler)
}

func oraclesListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
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
