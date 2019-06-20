package consumer

import (
	"encoding/json"
	"time"

	"github.com/kthomas/go-natsutil"

	stan "github.com/nats-io/stan.go"
	"github.com/provideapp/goldmine/common"
	provide "github.com/provideservices/provide-go"
)

const apiUsageDaemonBufferSize = 1024 * 25
const apiUsageDaemonFlushInterval = 10000

const natsAPIUsageEventNotificationSubject = "api.usage.event"
const natsAPIUsageEventNotificationMaxInFlight = 32

type apiUsageDelegate struct {
	natsConnection *stan.Conn
}

// Track receives an API call from the API daemon's underlying buffered channel for local processing
func (d *apiUsageDelegate) Track(apiCall *provide.APICall) {
	payload, _ := json.Marshal(apiCall)
	if d.natsConnection != nil {
		(*d.natsConnection).PublishAsync(natsAPIUsageEventNotificationSubject, payload, func(_ string, err error) {
			if err != nil {
				common.Log.Warningf("Failed to asnychronously publish %s; %s", natsAPIUsageEventNotificationSubject, err.Error())
				d.initNatsStreamingConnection()
				defer d.Track(apiCall)
			}
		})
	} else {
		common.Log.Warningf("Failed to asnychronously publish %s; no NATS streaming connection", natsAPIUsageEventNotificationSubject)
	}
}

func (d *apiUsageDelegate) initNatsStreamingConnection() {
	natsConnection, err := natsutil.GetNatsStreamingConnection(time.Second*10, func(_ stan.Conn, err error) {
		d.initNatsStreamingConnection()
	})
	if err != nil {
		common.Log.Warningf("Failed to establish NATS connection for API usage delegate; %s", err.Error())
		return
	}
	d.natsConnection = natsConnection
}

// RunAPIUsageDaemon runs the usage daemon
func RunAPIUsageDaemon() {
	delegate := new(apiUsageDelegate)
	delegate.initNatsStreamingConnection()
	provide.RunAPIUsageDaemon(apiUsageDaemonBufferSize, apiUsageDaemonFlushInterval, delegate)
}
