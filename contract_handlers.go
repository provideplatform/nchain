package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

// InstallContractsAPI installs the handlers using the given gin Engine
func InstallContractsAPI(r *gin.Engine) {
	r.GET("/api/v1/contracts", contractsListHandler)
	r.GET("/api/v1/contracts/:id", contractDetailsHandler)
	r.POST("/api/v1/contracts", createContractHandler)
	r.POST("/api/v1/contracts/:id/execute", contractExecutionHandler)
}

func contractsListHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	query := ContractListQuery()

	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL")
	}

	filterTokens := strings.ToLower(c.Query("filter_tokens")) == "true"
	if filterTokens {
		query = query.Joins("LEFT OUTER JOIN tokens ON tokens.contract_id = contracts.id").Where("symbol IS NULL")
	}

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("contracts.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("contracts.created_at ASC")
	}

	var contracts []Contract
	provide.Paginate(c, query, &Contract{}).Find(&contracts)
	render(contracts, 200, c)
}

func contractDetailsHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	var contract = &Contract{}

	query := db.Where("id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL", userID)
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		renderError("contract not found", 404, c)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	}

	render(contract, 200, c)
}

func createContractHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	if appID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	contract := &Contract{}
	err = json.Unmarshal(buf, contract)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	contract.ApplicationID = appID

	params := contract.ParseParams()
	if contract.Name == nil {
		if constructor, constructorOk := params["constructor"].(string); constructorOk {
			contract.Name = &constructor
		} else if name, nameOk := params["name"].(string); nameOk {
			contract.Name = &name
		}
	}

	_, rawSourceOk := params["raw_source"].(string)
	if rawSourceOk && contract.Address == nil {
		contract.Address = StringOrNil("0x")
	}

	if contract.Create() {
		if rawSourceOk {
			render(contract, 202, c)
		} else {
			render(contract, 201, c)
		}
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = contract.Errors
		render(obj, 422, c)
	}
}

func contractArbitraryExecutionHandler(c *gin.Context, db *gorm.DB, buf []byte) {
	userID := authorizedSubjectId(c, "user")
	if userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	wallet := &Wallet{} // signer for the tx

	params := map[string]interface{}{}
	err := json.Unmarshal(buf, &params)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	publicKey, publicKeyOk := params["public_key"].(string)
	privateKey, privateKeyOk := params["private_key"].(string)
	gas, gasOk := params["gas"].(float64)

	ref, err := uuid.NewV4()
	if err != nil {
		Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &ContractExecution{
		Ref: StringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying a wallet_id and wallet")
			renderError(err.Error(), 422, c)
			return
		}
		wallet.setID(*execution.WalletID)
	} else if publicKeyOk && privateKeyOk {
		wallet.Address = publicKey
		wallet.PrivateKey = StringOrNil(privateKey)
	}
	execution.Wallet = wallet

	if gasOk {
		execution.Gas = &gas
	}

	network := &Network{}
	if execution.NetworkID != nil && *execution.NetworkID != uuid.Nil {
		db.Where("id = ?", execution.NetworkID).Find(&network)
	}

	if network == nil || network.ID == uuid.Nil {
		renderError("network not found for arbitrary contract execution", 404, c)
		return
	}

	params = map[string]interface{}{
		"abi": execution.ABI,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		renderError("failed to marshal ephemeral contract params containing ABI", 422, c)
		return
	}
	paramsMsg := json.RawMessage(paramsJSON)

	ephemeralContract := &Contract{
		NetworkID: network.ID,
		Address:   StringOrNil(c.Param("id")),
		Params:    &paramsMsg,
	}

	_gas, _ := big.NewFloat(gas).Uint64()
	resp, err := ephemeralContract.Execute(execution.Ref, execution.Wallet, execution.Value, execution.Method, execution.Params, _gas)
	if err == nil {
		render(resp, 202, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = []string{err.Error()}
		render(obj, 422, c)
	}
}

func arbitraryRPCExecutionHandler(db *gorm.DB, networkID *uuid.UUID, params map[string]interface{}, c *gin.Context) {
	network := &Network{}
	db.Where("id = ?", networkID).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		renderError("not found", 404, c)
		return
	}
	method := params["method"].(string)
	authorizedMethod := false
	cfg := network.ParseConfig()
	if whitelist, whitelistOk := cfg["rpc_method_whitelist"].([]interface{}); whitelistOk {
		for _, mthd := range whitelist {
			mthdStr := mthd.(string)
			authorizedMethod = mthdStr == method
			if authorizedMethod {
				break
			}
		}
	}
	if !authorizedMethod {
		renderError(fmt.Sprintf("forbidden rpc method %s", method), 403, c)
		return
	}
	Log.Debugf("%s", params)
	resp, err := network.InvokeJSONRPC(method, params["params"].([]interface{}))
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	render(resp, 200, c)
}

func contractExecutionHandler(c *gin.Context) {
	appID := authorizedSubjectId(c, "application")
	userID := authorizedSubjectId(c, "user")
	if appID == nil && userID == nil {
		renderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		renderError(err.Error(), 400, c)
		return
	}

	db := dbconf.DatabaseConnection()

	contractID := c.Param("id")
	rpcHack := strings.Index(contractID, "rpc:") == 0
	if rpcHack {
		rpcNetworkIDStr := contractID[4:]
		rpcNetworkID, err := uuid.FromString(rpcNetworkIDStr)
		if err != nil {
			err = fmt.Errorf("Failed to parse RPC network id as valid uuid: %s; %s", rpcNetworkIDStr, err.Error())
			renderError(err.Error(), 400, c)
			return
		}
		params := map[string]interface{}{}
		err = json.Unmarshal(buf, &params)
		if err != nil {
			err = fmt.Errorf("Failed to parse JSON-RPC params; %s", err.Error())
			renderError(err.Error(), 400, c)
			return
		}
		Log.Debugf("Attempting arbitrary, non-permissioned contract execution on behalf of user with id: %s", userID)
		arbitraryRPCExecutionHandler(db, &rpcNetworkID, params, c)
		return
	}
	// HACK

	var contract = &Contract{}

	db.Where("id = ?", contractID).Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		if appID != nil {
			renderError("contract not found", 404, c)
			return
		}

		Log.Debugf("Attempting arbitrary, non-permissioned contract execution on behalf of user with id: %s", userID)
		contractArbitraryExecutionHandler(c, db, buf)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		renderError("forbidden", 403, c)
		return
	}

	ref, err := uuid.NewV4()
	if err != nil {
		Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &ContractExecution{
		Ref: StringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}
	execution.Contract = contract
	execution.ContractID = &contract.ID
	if execution.WalletID != nil && *execution.WalletID != uuid.Nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying a wallet_id and wallet")
			renderError(err.Error(), 422, c)
			return
		}
		wallet := &Wallet{}
		wallet.setID(*execution.WalletID)
		execution.Wallet = wallet
	}

	executionResponse, err := execution.Execute()
	if err != nil {
		renderError(err.Error(), 422, c)
		return
	}

	switch executionResponse.(type) {
	case *ContractExecutionResponse:
		executionResponse = map[string]interface{}{
			"response": executionResponse.(*ContractExecutionResponse).Response,
		}
		render(executionResponse, 200, c) // returns 200 OK status to indicate the contract invocation was able to return a syncronous response
	default:
		confidence := invokeTxFilters(appID, buf, db)
		executionResponse = map[string]interface{}{
			"confidence": confidence,
			"ref":        executionResponse.(*ContractExecution).Ref,
		}
		render(executionResponse, 202, c) // returns 202 Accepted status to indicate the contract invocation is pending
	}
}

func invokeTxFilters(applicationID *uuid.UUID, payload []byte, db *gorm.DB) *float64 {
	if applicationID == nil {
		Log.Warningf("Tx filters are not currently supported for transactions outside of the scope of an application context")
		return nil
	}

	if _, hasConfiguredFilter := TxFilters[applicationID.String()]; !hasConfiguredFilter {
		Log.Debugf("No tx filters to invoke for application: %s", applicationID.String())
		return nil
	}

	var confidence *float64
	var filters []Filter
	query := db.Where("application_id = ?", applicationID).Order("priority ASC") // TODO: load the filters into memory
	query.Find(&filters)
	for _, filter := range filters {
		if confidence == nil {
			_confidence := float64(0.0)
			confidence = &_confidence
		}
		confidence = filter.Invoke(payload) // TODO: discuss order and priority of filters
	}
	return confidence
}
