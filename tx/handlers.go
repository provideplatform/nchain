package tx

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/filter"
	"github.com/provideapp/goldmine/network"
	"github.com/provideapp/goldmine/wallet"
	provide "github.com/provideservices/provide-go"
)

// InstallTransactionsAPI installs the handlers using the given gin Engine
func InstallTransactionsAPI(r *gin.Engine) {
	r.GET("/api/v1/transactions", transactionsListHandler)
	r.POST("/api/v1/transactions", createTransactionHandler)
	r.GET("/api/v1/transactions/:id", transactionDetailsHandler)
	r.GET("/api/v1/networks/:id/transactions", networkTransactionsListHandler)
	r.GET("/api/v1/networks/:id/transactions/:transactionId", networkTransactionDetailsHandler)

	r.POST("/api/v1/contracts/:id/execute", contractExecutionHandler)
}

func transactionsListHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var query *gorm.DB
	if appID != nil {
		query = dbconf.DatabaseConnection().Where("transactions.application_id = ?", appID)
	} else if userID != nil {
		query = dbconf.DatabaseConnection().Where("transactions.user_id = ?", userID)
	}

	filterContractCreationTx := strings.ToLower(c.Query("filter_contract_creations")) == "true"
	if filterContractCreationTx {
		query = query.Where("transactions.to IS NULL")
	}

	if c.Query("status") != "" {
		query = query.Where("transactions.status IN ?", strings.Split(c.Query("status"), ","))
	}

	if c.Query("to") != "" {
		query = query.Where("transactions.to = ?", c.Query("to"))
	}

	if c.Query("from") != "" {
		from := &wallet.Account{}
		db := dbconf.DatabaseConnection()
		fromQuery := db.Where("address = ?", from)
		if c.Query("network_id") != "" {
			fromQuery = fromQuery.Where("network_id = ?", c.Query("network_id"))
		}
		fromQuery.Find(&from)
		if from != nil && from.ID != uuid.Nil {
			query = query.Where("transactions.account_id= ?", from.ID)
		} else {
			provide.RenderError("from address not resolved to a known signer", 404, c)
		}
	}

	if c.Query("network_id") != "" {
		query = query.Where("transactions.network_id = ?", c.Query("network_id"))
	}

	if c.Query("account_id") != "" {
		query = query.Where("transactions.account_id= ?", c.Query("account_id"))
	}

	var txs []Transaction
	query = query.Order("created_at DESC")
	provide.Paginate(c, query, &Transaction{}).Find(&txs)
	provide.Render(txs, 200, c)
}

func createTransactionHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	tx := &Transaction{}
	err = json.Unmarshal(buf, tx)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	tx.ApplicationID = appID
	tx.UserID = userID

	db := dbconf.DatabaseConnection()

	if tx.Signer != nil {
		if tx.AccountID != nil {
			provide.RenderError("provided signer and account_id to tx creation, which is ambiguous", 400, c)
			return
		}

		wallet := &wallet.Account{}
		db.Where("address = ?", tx.Signer).Find(&wallet)
		if wallet == nil || wallet.ID == uuid.Nil {
			provide.RenderError("failed to resolve signer address to wallet", 404, c)
			return
		}
		tx.AccountID = &wallet.ID
	}

	if tx.Create(db) {
		provide.Render(tx, 201, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = tx.Errors
		provide.Render(obj, 422, c)
	}
}

func transactionDetailsHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	if appID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()

	var tx = &Transaction{}
	db.Where("id = ?", c.Param("id")).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		db.Where("ref = ?", c.Param("id")).Find(&tx)
		if tx == nil || tx.ID == uuid.Nil {
			provide.RenderError("transaction not found", 404, c)
			return
		}
	} else if appID != nil && tx.ApplicationID != nil && *tx.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}
	err := tx.RefreshDetails()
	if err != nil {
		provide.RenderError("internal server error", 500, c)
		return
	}
	provide.Render(tx, 200, c)
}

func networkTransactionsListHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := dbconf.DatabaseConnection().Where("transactions.network_id = ? AND transactions.application_id IS NULL", c.Param("id"))

	filterContractCreationTx := strings.ToLower(c.Query("filter_contract_creations")) == "true"
	if filterContractCreationTx {
		query = query.Where("transactions.to IS NULL")
	}

	if c.Query("status") != "" {
		query = query.Where("transactions.status IN ?", strings.Split(c.Query("status"), ","))
	}

	var txs []Transaction
	query = query.Order("created_at DESC")
	provide.Paginate(c, query, &Transaction{}).Find(&txs)
	provide.Render(txs, 200, c)
}

func networkTransactionDetailsHandler(c *gin.Context) {
	userID := provide.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	var tx = &Transaction{}
	dbconf.DatabaseConnection().Where("network_id = ? AND id = ?", c.Param("id"), c.Param("transactionId")).Find(&tx)
	if tx == nil || tx.ID == uuid.Nil {
		provide.RenderError("transaction not found", 404, c)
		return
	}
	err := tx.RefreshDetails()
	if err != nil {
		provide.RenderError("internal server error", 500, c)
		return
	}
	provide.Render(tx, 200, c)
}

func contractArbitraryExecutionHandler(c *gin.Context, db *gorm.DB, buf []byte) {
	userID := provide.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	wal := &wallet.Account{} // signer for the tx

	params := map[string]interface{}{}
	err := json.Unmarshal(buf, &params)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	publicKey, publicKeyOk := params["public_key"].(string)
	privateKey, privateKeyOk := params["private_key"].(string)
	gas, gasOk := params["gas"].(float64)
	nonce, nonceOk := params["nonce"].(float64)

	ref, err := uuid.NewV4()
	if err != nil {
		common.Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &contract.Execution{
		Ref: common.StringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	if execution.AccountID != nil && *execution.AccountID != uuid.Nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying a account_idand wallet")
			provide.RenderError(err.Error(), 422, c)
			return
		}
		wal.SetID(*execution.AccountID)
	} else if publicKeyOk && privateKeyOk {
		wal.Address = publicKey
		wal.PrivateKey = common.StringOrNil(privateKey)
	}
	execution.Wallet = wal

	if gasOk {
		execution.Gas = &gas
	}

	if nonceOk {
		nonceUint := uint64(nonce)
		execution.Nonce = &nonceUint
	}

	ntwrk := &network.Network{}
	if execution.NetworkID != nil && *execution.NetworkID != uuid.Nil {
		db.Where("id = ?", execution.NetworkID).Find(&ntwrk)
	}

	if ntwrk == nil || ntwrk.ID == uuid.Nil {
		provide.RenderError("network not found for arbitrary contract execution", 404, c)
		return
	}

	params = map[string]interface{}{
		"abi": execution.ABI,
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		provide.RenderError("failed to marshal ephemeral contract params containing ABI", 422, c)
		return
	}
	paramsMsg := json.RawMessage(paramsJSON)

	ephemeralContract := &contract.Contract{
		NetworkID: ntwrk.ID,
		Address:   common.StringOrNil(c.Param("id")),
		Params:    &paramsMsg,
	}

	var tx Transaction
	txCreateFn := func(c *contract.Contract, network *network.Network, accountID *uuid.UUID, walletID *uuid.UUID, execution *contract.Execution, _txParamsJSON *json.RawMessage) (*contract.ExecutionResponse, error) {
		return txCreatefunc(&tx, c, network, accountID, walletID, execution, _txParamsJSON)
	}
	accountFn := func(a interface{}, txParams map[string]interface{}) *uuid.UUID {
		return afunc(a.(wallet.Account), txParams)
	}
	walletFn := func(w interface{}, txParams map[string]interface{}) *uuid.UUID {
		return wfunc(w.(wallet.Wallet), txParams)
	}

	resp, err := ephemeralContract.ExecuteFromTx(execution, accountFn, walletFn, txCreateFn)
	if err == nil {
		provide.Render(resp, 202, c)
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = []string{err.Error()}
		provide.Render(obj, 422, c)
	}
}

func arbitraryRPCExecutionHandler(db *gorm.DB, networkID *uuid.UUID, params map[string]interface{}, c *gin.Context) {
	network := &network.Network{}
	db.Where("id = ?", networkID).Find(&network)
	if network == nil || network.ID == uuid.Nil {
		provide.RenderError("not found", 404, c)
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
		provide.RenderError(fmt.Sprintf("forbidden rpc method %s", method), 403, c)
		return
	}
	common.Log.Debugf("%s", params)
	resp, err := network.InvokeJSONRPC(method, params["params"].([]interface{}))
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	provide.Render(resp, 200, c)
}

func contractExecutionHandler(c *gin.Context) {
	appID := provide.AuthorizedSubjectID(c, "application")
	userID := provide.AuthorizedSubjectID(c, "user")
	if appID == nil && userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	params := map[string]interface{}{}
	err = json.Unmarshal(buf, &params)
	if err != nil {
		err = fmt.Errorf("Failed to parse JSON-RPC params; %s", err.Error())
		provide.RenderError(err.Error(), 400, c)
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
			provide.RenderError(err.Error(), 400, c)
			return
		}
		common.Log.Debugf("Attempting arbitrary, non-permissioned contract execution on behalf of user with id: %s", userID)
		arbitraryRPCExecutionHandler(db, &rpcNetworkID, params, c)
		return
	}
	// HACK

	var contractObj = &contract.Contract{}

	db.Where("id = ?", contractID).Find(&contractObj)

	if contractObj == nil || contractObj.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contractObj)
	}

	if contractObj == nil || contractObj.ID == uuid.Nil {
		if appID != nil {
			provide.RenderError("contract not found", 404, c)
			return
		}

		common.Log.Debugf("Attempting arbitrary, non-permissioned contract execution on behalf of user with id: %s", userID)
		contractArbitraryExecutionHandler(c, db, buf)
		return
	} else if appID != nil && *contractObj.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	ref, err := uuid.NewV4()
	if err != nil {
		common.Log.Warningf("Failed to generate ref id; %s", err.Error())
	}

	execution := &contract.Execution{
		Ref: common.StringOrNil(ref.String()),
	}

	err = json.Unmarshal(buf, execution)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	execution.Contract = contractObj
	execution.ContractID = &contractObj.ID
	if execution.AccountID != nil && *execution.AccountID != uuid.Nil {
		if execution.Account != nil {
			err := fmt.Errorf("invalid request specifying a account_address and account")
			provide.RenderError(err.Error(), 422, c)
			return
		}
		account := &wallet.Account{}
		account.SetID(*execution.AccountID)
		execution.Account = account
	} else if execution.WalletID != nil && *execution.WalletID != uuid.Nil && execution.HDPath != nil {
		if execution.Wallet != nil {
			err := fmt.Errorf("invalid request specifying hd_derivation_path and wallet")
			provide.RenderError(err.Error(), 422, c)
			return
		}
		wallet := &wallet.Wallet{}
		wallet.SetID(*execution.WalletID)
		wallet.Path = execution.HDPath
		execution.Wallet = wallet
	}

	gas, gasOk := params["gas"].(float64)
	nonce, nonceOk := params["nonce"].(float64)

	if gasOk {
		execution.Gas = &gas
	}

	if nonceOk {
		nonceUint := uint64(nonce)
		execution.Nonce = &nonceUint
	}

	var tx Transaction
	txCreateFn := func(c *contract.Contract, network *network.Network, accountID *uuid.UUID, walletID *uuid.UUID, execution *contract.Execution, _txParamsJSON *json.RawMessage) (*contract.ExecutionResponse, error) {
		return txCreatefunc(&tx, c, network, accountID, walletID, execution, _txParamsJSON)
	}
	accountFn := func(a interface{}, txParams map[string]interface{}) *uuid.UUID {
		if a == nil {
			return nil
		}
		return afunc(a.(*wallet.Account), txParams)
	}
	walletFn := func(w interface{}, txParams map[string]interface{}) *uuid.UUID {
		if w == nil {
			return nil
		}
		return wfunc(w.(*wallet.Wallet), txParams)
	}

	executionResponse, err := execution.ExecuteFromTx(accountFn, walletFn, txCreateFn)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}

	switch executionResponse.(type) {
	case *contract.ExecutionResponse:
		executionResponse = executionResponse.(*contract.ExecutionResponse).Response.(map[string]interface{})
		provide.Render(executionResponse, 200, c) // returns 200 OK status to indicate the contract invocation was able to return a syncronous response
	default:
		confidence := invokeTxFilters(appID, buf, db)
		executionResponse = map[string]interface{}{
			"confidence": confidence,
			"ref":        executionResponse.(*contract.Execution).Ref,
		}
		provide.Render(executionResponse, 202, c) // returns 202 Accepted status to indicate the contract invocation is pending
	}
}

func invokeTxFilters(applicationID *uuid.UUID, payload []byte, db *gorm.DB) *float64 {
	if applicationID == nil {
		common.Log.Warningf("Tx filters are not currently supported for transactions outside of the scope of an application context")
		return nil
	}

	if _, hasConfiguredFilter := common.TxFilters[applicationID.String()]; !hasConfiguredFilter {
		common.Log.Debugf("No tx filters to invoke for application: %s", applicationID.String())
		return nil
	}

	var confidence *float64
	var filters []filter.Filter
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
