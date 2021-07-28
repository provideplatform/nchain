package prices

import (
	"github.com/gin-gonic/gin"
	provide "github.com/provideplatform/provide-go/common"
)

// InstallPricesAPI installs the handlers using the given gin Engine
func InstallPricesAPI(r *gin.Engine) {
	r.GET("/api/v1/prices", pricesHandler)
}

func pricesHandler(c *gin.Context) {
	provide.Render(CurrentPrices, 200, c)
}
