package main

import (
	"encoding/json"

	provide "github.com/provideservices/provide-go"
)

const apiUsageDaemonBufferSize = 1024 * 25
const apiUsageDaemonFlushInterval = 10000

const natsAPIUsageEventNotificationSubject = "api.usage.event"
const natsAPIUsageEventNotificationMaxInFlight = 32

type apiUsageDelegate struct{}

// Track receives an API call from the API daemon's underlying buffered channel for local processing
func (d *apiUsageDelegate) Track(apiCall *provide.APICall) {
	payload, _ := json.Marshal(apiCall)
	natsConnection := getNatsStreamingConnection()
	natsConnection.Publish(natsAPIUsageEventNotificationSubject, payload)
}

// RunAPIUsageDaemon initializes a delegate and runs the API usage daemon
func RunAPIUsageDaemon() {
	delegate := new(apiUsageDelegate)
	provide.RunAPIUsageDaemon(apiUsageDaemonBufferSize, apiUsageDaemonFlushInterval, delegate)
}
