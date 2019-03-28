package oracle

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	_ "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

// InstallOraclesAPI installs the handlers using the given gin Engine
func InstallOraclesAPI(r *gin.Engine) {
	r.GET("/api/v1/oracles", oraclesListHandler)
	r.POST("/api/v1/oracles", createOracleHandler)
	r.GET("/api/v1/oracles/:id", oracleDetailsHandler)
}

func oraclesListHandler(c *gin.Context) {
	appID := common.AuthorizedSubjectId(c, "application")
	if appID == nil {
		common.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("oracles.application_id = ?", appID)

	var oracles []Oracle
	query = query.Order("created_at ASC")
	provide.Paginate(c, query, &Oracle{}).Find(&oracles)
	common.Render(oracles, 200, c)
}

func oracleDetailsHandler(c *gin.Context) {
	common.RenderError("not implemented", 501, c)
}

func createOracleHandler(c *gin.Context) {
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

	oracle := &Oracle{}
	err = json.Unmarshal(buf, oracle)
	if err != nil {
		common.RenderError(err.Error(), 422, c)
		return
	}
	oracle.ApplicationID = appID

	if oracle.Create() {
		common.Render(oracle, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = oracle.Errors
		common.Render(obj, 422, c)
	}
}
