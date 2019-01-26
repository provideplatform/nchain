package main

import "github.com/gin-gonic/gin"

// InstallPricesAPI installs the handlers using the given gin Engine
func InstallPricesAPI(r *gin.Engine) {
	r.GET("/api/v1/prices", pricesHandler)
}

func pricesHandler(c *gin.Context) {
	render(CurrentPrices, 200, c)
}
