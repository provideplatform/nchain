package prices

import (
	"github.com/gin-gonic/gin"
	"github.com/provideapp/goldmine/common"
)

// InstallPricesAPI installs the handlers using the given gin Engine
func InstallPricesAPI(r *gin.Engine) {
	r.GET("/api/v1/prices", pricesHandler)
}

func pricesHandler(c *gin.Context) {
	common.Render(CurrentPrices, 200, c)
}
