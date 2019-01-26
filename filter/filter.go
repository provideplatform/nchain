package filter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/consumer"
	provide "github.com/provideservices/provide-go"
)

const natsStreamingTxFilterExecSubjectPrefix = "ml.filter.exec"
const streamingTxFilterReturnTimeout = time.Millisecond * 50

func init() {
	db := dbconf.DatabaseConnection()
	db.AutoMigrate(&Filter{})
	db.Model(&Filter{}).AddIndex("idx_filters_application_id", "application_id")
	db.Model(&Filter{}).AddIndex("idx_filters_network_id", "network_id")
	db.Model(&Filter{}).AddForeignKey("network_id", "networks(id)", "SET NULL", "CASCADE")
}

// Filter instances must be associated with an application identifier.
type Filter struct {
	provide.Model
	ApplicationID *uuid.UUID       `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID        `sql:"not null;type:uuid" json:"network_id"`
	Name          *string          `sql:"not null" json:"name"`
	Priority      uint8            `sql:"not null;default:0" json:"priority"`
	Lang          *string          `sql:"not null" json:"lang"`
	Source        *string          `sql:"not null" json:"source"`
	Params        *json.RawMessage `sql:"type:json" json:"params"`
}

// Create and persist a new filter
func (f *Filter) Create() bool {
	if !f.Validate() {
		return false
	}

	db := dbconf.DatabaseConnection()

	if db.NewRecord(f) {
		result := db.Create(&f)
		rowsAffected := result.RowsAffected
		errors := result.GetErrors()
		if len(errors) > 0 {
			for _, err := range errors {
				f.Errors = append(f.Errors, &provide.Error{
					Message: common.StringOrNil(err.Error()),
				})
			}
		}
		if !db.NewRecord(f) {
			success := rowsAffected > 0
			if success {
				go f.cache()
			}
			return success
		}
	}
	return false
}

func CacheTxFilters() {
	db := dbconf.DatabaseConnection()
	var filters []Filter
	db.Find(&filters)
	for _, filter := range filters {
		appFilters := common.TxFilters[filter.ApplicationID.String()]
		if appFilters == nil {
			appFilters = make([]interface{}, 0)
			common.TxFilters[filter.ApplicationID.String()] = appFilters
		}
		appFilters = append(appFilters, &filter)
	}
}

// cache the filter in memory
func (f *Filter) cache() {
	appFilters := common.TxFilters[f.ApplicationID.String()]
	if appFilters == nil {
		appFilters = make([]interface{}, 0)
		common.TxFilters[f.ApplicationID.String()] = appFilters
	}
	appFilters = append(appFilters, f)
}

// ParseParams - parse the original JSON params used for filter creation
func (f *Filter) ParseParams() map[string]interface{} {
	params := map[string]interface{}{}
	if f.Params != nil {
		err := json.Unmarshal(*f.Params, &params)
		if err != nil {
			common.Log.Warningf("Failed to unmarshal filter params; %s", err.Error())
			return nil
		}
	}
	return params
}

// Invoke a filter for the given tx payload
func (f *Filter) Invoke(txPayload []byte) *float64 {
	subjectUUID, _ := uuid.NewV4()
	natsStreamingTxFilterReturnSubject := fmt.Sprintf("%s.return.%s", natsStreamingTxFilterExecSubjectPrefix, subjectUUID.String())

	natsMsg := map[string]interface{}{
		"sub":     natsStreamingTxFilterReturnSubject,
		"payload": txPayload,
	}
	natsPayload, _ := json.Marshal(natsMsg)

	natsConnection := consumer.GetNatsStreamingConnection()
	natsConnection.Publish(natsStreamingTxFilterExecSubjectPrefix, natsPayload)

	natsConn := natsutil.GetNatsConnection()
	defer natsConn.Close()

	sub, err := natsConn.SubscribeSync(natsStreamingTxFilterReturnSubject)
	if err != nil {
		common.Log.Warningf("Failed to create a synchronous NATS subscription to subject: %s; %s", natsStreamingTxFilterReturnSubject, err.Error())
		return nil
	}

	var confidence *float64
	msg, err := sub.NextMsg(streamingTxFilterReturnTimeout)
	if err != nil {
		common.Log.Warningf("Failed to parse confidence from streaming tx filter; %s", err.Error())
		return nil
	}
	_confidence, err := strconv.ParseFloat(string(msg.Data), 64)
	if err != nil {
		common.Log.Warningf("Failed to parse confidence from streaming tx filter; %s", err.Error())
		return nil
	}
	confidence = &_confidence
	return confidence
}

// Validate a filter for persistence
func (f *Filter) Validate() bool {
	f.Errors = make([]*provide.Error, 0)
	return len(f.Errors) == 0
}
