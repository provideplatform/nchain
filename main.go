package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	bootstrap()
	migrateSchema()

	RunConsumers()

	r := gin.Default()

	r.GET("/api/v1/addresses", addressesHandler)
	r.GET("/api/v1/blocks", blocksHandler)
	r.GET("/api/v1/contracts", contractsHandler)
	r.GET("/api/v1/prices", pricesHandler)
	r.GET("/api/v1/tokens", tokensHandler)
	r.GET("/api/v1/transactions", transactionsHandler)

	r.GET("/status", statusHandler)

	if shouldServeTLS() {
		r.RunTLS(ListenAddr, CertificatePath, PrivateKeyPath)
	} else {
		r.Run(ListenAddr)
	}
}

func render(obj interface{}, status int, c *gin.Context) {
	c.Header("content-type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(status)
	if &obj != nil && status != http.StatusNoContent {
		encoder := json.NewEncoder(c.Writer)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(obj); err != nil {
			panic(err)
		}
	} else {
		c.Header("content-length", "0")
	}
}

func renderError(message string, status int, c *gin.Context) {
	err := map[string]*string{}
	err["message"] = &message
	render(err, status, c)
}

func requireParams(requiredParams []string, c *gin.Context) error {
	var errs []string
	for _, param := range requiredParams {
		if c.Query(param) == "" {
			errs = append(errs, param)
		}
	}
	if len(errs) > 0 {
		msg := strings.Trim(fmt.Sprintf("missing required parameters: %s", strings.Join(errs, ", ")), " ")
		renderError(msg, 400, c)
		return errors.New(msg)
	}
	return nil
}

func statusHandler(c *gin.Context) {
	render(nil, 204, c)
}

func addressesHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func blocksHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func contractsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func pricesHandler(c *gin.Context) {
	render(CurrentPrices, 200, c)
}

func tokensHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}

func transactionsHandler(c *gin.Context) {
	renderError("not implemented", 501, c)
}
