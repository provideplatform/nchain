package contract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	dbconf "github.com/kthomas/go-db-config"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	provide "github.com/provideservices/provide-go/common"
	util "github.com/provideservices/provide-go/common/util"

	"github.com/provideapp/ident/token"
)

// InstallContractsAPI installs the handlers using the given gin Engine
func InstallContractsAPI(r *gin.Engine) {
	r.GET("/api/v1/contracts", contractsListHandler)
	r.GET("/api/v1/contracts/:id", contractDetailsHandler)
	r.POST("/api/v1/contracts", createContractHandler)
	r.POST("/api/v1/contracts/:id/subscriptions", createContractSubscriptionTokenHandler)

	r.GET("/api/v1/networks/:id/contracts", networkContractsListHandler)
	r.GET("/api/v1/networks/:id/contracts/:contractId", networkContractDetailsHandler)
}

func contractsListHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := ContractListQuery()

	if appID == nil && c.Query("application_id") != "" {
		appIDString := c.Query("application_id")
		appUUID, err := uuid.FromString(appIDString)
		if err != nil {
			msg := fmt.Sprintf("malformed application_id provided; %s", err.Error())
			provide.RenderError(msg, 422, c)
			return
		}
		appID = &appUUID
	}

	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	} else if orgID != nil {
		query = query.Where("contracts.organization_id = ?", orgID)
	}

	filterTokens := strings.ToLower(c.Query("filter_tokens")) == "true"
	if filterTokens {
		query = query.Joins("LEFT OUTER JOIN tokens ON tokens.contract_id = contracts.id").Where("symbol IS NULL")
	}

	if c.Query("type") != "" {
		query = query.Where("contracts.type = ?", c.Query("type"))
	}

	sortByMostRecent := strings.ToLower(c.Query("sort")) == "recent"
	if sortByMostRecent {
		query = query.Order("contracts.accessed_at DESC NULLS LAST")
	} else {
		query = query.Order("contracts.created_at ASC")
	}

	var contracts []*Contract
	provide.Paginate(c, query, &Contract{}).Find(&contracts)
	for _, contract := range contracts {
		contract.enrich()
	}
	provide.Render(contracts, 200, c)
}

func contractDetailsHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	contract := &Contract{}

	query := db.Where("id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if orgID != nil {
		query = query.Where("contracts.organization_id = ?", orgID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL", userID)
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		provide.RenderError("contract not found", 404, c)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if orgID != nil && *contract.OrganizationID != *orgID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	contract.enrich()

	provide.Render(contract, 200, c)
}

func createContractHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && orgID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	buf, err := c.GetRawData()
	if err != nil {
		provide.RenderError(err.Error(), 400, c)
		return
	}

	contract := &Contract{}
	err = json.Unmarshal(buf, contract)
	if err != nil {
		provide.RenderError(err.Error(), 422, c)
		return
	}
	contract.ApplicationID = appID
	contract.OrganizationID = orgID

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
		contract.Address = common.StringOrNil("0x")
	}

	if contract.Create() {
		contract.enrich()

		if rawSourceOk {
			provide.Render(contract, 202, c)
		} else {
			provide.Render(contract, 201, c)
		}
	} else {
		obj := map[string]interface{}{}
		obj["errors"] = contract.Errors
		provide.Render(obj, 422, c)
	}
}

func createContractSubscriptionTokenHandler(c *gin.Context) {
	appID := util.AuthorizedSubjectID(c, "application")
	userID := util.AuthorizedSubjectID(c, "user")
	orgID := util.AuthorizedSubjectID(c, "organization")
	if appID == nil && userID == nil && orgID == nil {
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
		err = fmt.Errorf("failed to parse params; %s", err.Error())
		provide.RenderError(err.Error(), 400, c)
		return
	}

	db := dbconf.DatabaseConnection()
	contract := &Contract{}

	query := db.Where("id = ?", c.Param("id"))
	if appID != nil {
		query = query.Where("contracts.application_id = ?", appID)
	}
	if orgID != nil {
		query = query.Where("contracts.organization_id = ?", orgID)
	}
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL AND contracts.organization_id IS NULL", userID)
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("address = ?", c.Param("id")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		provide.RenderError("contract not found", 404, c)
		return
	} else if appID != nil && *contract.ApplicationID != *appID {
		provide.RenderError("forbidden", 403, c)
		return
	} else if orgID != nil && *contract.OrganizationID != *orgID {
		provide.RenderError("forbidden", 403, c)
		return
	}

	contract.enrich()
	if contract.PubsubPrefix == nil {
		provide.RenderError("forbidden", 403, c)
		return
	}

	var subject string
	if appID != nil {
		subject = fmt.Sprintf("application:%s", appID.String())
	} else if appID != nil {
		subject = fmt.Sprintf("organization:%s", appID.String())
	} else if userID != nil {
		subject = fmt.Sprintf("user:%s", userID.String())
	}

	allowedSubject := *contract.PubsubPrefix

	subscribeAllow := make([]string, 0)
	if subpart, subpartOk := params["subject"].(string); subpartOk {
		subjectSuffix := subpart
		if strings.HasSuffix(subjectSuffix, ".*") {
			subjectSuffix = fmt.Sprintf("%s.>", subjectSuffix[0:len(subjectSuffix)-2])
		}
		subscribeAllow = append(subscribeAllow, fmt.Sprintf("%s.%s", allowedSubject, subjectSuffix))
	} else {
		subscribeAllow = append(subscribeAllow, allowedSubject)
		subscribeAllow = append(subscribeAllow, fmt.Sprintf("%s.>", allowedSubject))
	}

	tkn, err := token.VendNatsBearerAuthorization(subject, []string{}, []string{}, subscribeAllow, []string{}, nil, nil)
	if err != nil {
		err = fmt.Errorf("failed to vend NATS bearer authorization; %s", err.Error())
		provide.RenderError(err.Error(), 500, c)
		return
	}

	provide.Render(tkn, 201, c)
}

func networkContractsListHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	query := ContractListQuery()
	query = query.Where("contracts.network_id = ? AND contracts.application_id IS NULL", c.Param("id"))

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

	var contracts []*Contract
	query = query.Order("contracts.created_at ASC")
	provide.Paginate(c, query, &Contract{}).Find(&contracts)
	for _, contract := range contracts {
		contract.enrich()
	}
	provide.Render(contracts, 200, c)
}

// FIXME-- DRY this up
func networkContractDetailsHandler(c *gin.Context) {
	userID := util.AuthorizedSubjectID(c, "user")
	if userID == nil {
		provide.RenderError("unauthorized", 401, c)
		return
	}

	db := dbconf.DatabaseConnection()
	var contract = &Contract{}

	query := db.Where("contracts.network_id = ? AND contracts.id = ?", c.Param("id"), c.Param("contractId"))
	if userID != nil {
		query = query.Where("contracts.application_id IS NULL")
	}

	query.Find(&contract)

	if contract == nil || contract.ID == uuid.Nil { // attempt to lookup the contract by address
		db.Where("contracts.network_id = ? AND contracts.address = ?", c.Param("id"), c.Param("contractId")).Find(&contract)
	}

	if contract == nil || contract.ID == uuid.Nil {
		provide.RenderError("contract not found", 404, c)
		return
	}

	contract.enrich()

	provide.Render(contract, 200, c)
}
