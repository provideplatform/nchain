package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/provideservices/provide-go"

	"github.com/gorilla/websocket"
	logger "github.com/kthomas/go-logger"
)

const networkStatsJsonRpcPollingTickerInterval = time.Millisecond * 5000

var currentNetworkStats = map[string]*StatsDaemon{}

type NetworkStatsDataSource struct {
	Poll   func(chan *provide.NetworkStatus) error // JSON-RPC polling -- implementations should be blocking
	Stream func(chan *provide.NetworkStatus) error // websocket -- implementations should be blocking
}

type StatsDaemon struct {
	attempt    uint32
	dataSource *NetworkStatsDataSource

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	log   *logger.Logger
	queue chan *provide.NetworkStatus
	wg    sync.WaitGroup
}

type jsonRpcNotSupported string
type websocketNotSupported string

func (err jsonRpcNotSupported) Error() string {
	return "JSON-RPC not supported"
}

func (err websocketNotSupported) Error() string {
	return "Websocket not supported"
}

// NetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume network statistics
func NetworkStatsDataSourceFactory(network *Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Poll: func(ch chan *provide.NetworkStatus) error {
			rpcURL := network.rpcURL()
			if rpcURL == "" {
				err := new(jsonRpcNotSupported)
				return *err
			}
			ticker := time.NewTicker(networkStatsJsonRpcPollingTickerInterval)
			for {
				select {
				case <-ticker.C:
					status, err := network.Status()
					if err != nil {
						Log.Errorf("Failed to retrieve network status via JSON-RPC: %s; %s", rpcURL, err)
						ticker.Stop()
						return nil
					} else {
						Log.Debugf("Received network status via JSON-RPC: %s; %s", rpcURL, status)
						ch <- status
					}
				}
			}
		},

		Stream: func(ch chan *provide.NetworkStatus) error {
			websocketURL := network.websocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}
			var wsDialer websocket.Dialer
			wsConn, _, err := wsDialer.Dial(websocketURL, nil)
			if err != nil {
				Log.Errorf("Failed to establish network stats websocket connection to %s", websocketURL)
			} else {
				defer wsConn.Close()
				subscribe := map[string]interface{}{
					"type": "subscribe",
				}
				if err := wsConn.WriteJSON(subscribe); err != nil {
					Log.Errorf("Failed to write subscribe message to network stats websocket connection")
				} else {
					Log.Debugf("Subscribed to %s network stats websocket: %s", websocketURL)

					for {
						_, message, err := wsConn.ReadMessage()
						if err != nil {
							Log.Errorf("Failed to receive message on network stats websocket; %s", err)
							break
						} else {
							Log.Debugf("Received message on network stats websocket: %s", message)
							status := &provide.NetworkStatus{}
							err := json.Unmarshal(message, status)
							if err != nil {
								Log.Warningf("Failed to unmarshal message received on network stats websocket: %s; %s", message, err.Error())
							} else {
								ch <- status
							}
						}
					}
				}
			}
			return err
		},
	}
}

// Consume the websocket stream; attempts to fallback to JSON-RPC if websocket stream fails or is not available for the network
func (sd *StatsDaemon) consumeAsync() {
	sd.wg.Add(1)
	go func() {
		defer sd.wg.Done()
		sd.consume()
	}()
	sd.wg.Wait()
}

// Consume the websocket stream; attempts to fallback to JSON-RPC if websocket stream fails or is not available for the network
func (sd *StatsDaemon) consume() {
	for {
		err := sd.dataSource.Stream(sd.queue)
		if err != nil {
			switch err.(type) {
			case jsonRpcNotSupported:
				sd.log.Warningf("Configured stats daemon data source does not support JSON-RPC: %s; attempting to upgrade to websocket stream...", sd)
				err := sd.dataSource.Stream(sd.queue)
				if err != nil {
					sd.log.Warningf("Configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
					// FIXME-- this could mean the stats daemon is incapable of getting stats at this time for the network in question...
				}
			case websocketNotSupported:
				sd.log.Warningf("Configured stats daemon data source does not support streaming via websocket; attempting to fallback to JSON-RPC long polling using stats daemon: %s", sd)
				err := sd.dataSource.Poll(sd.queue)
				if err != nil {
					sd.log.Warningf("Configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
					// FIXME-- this could mean the stats daemon is incapable of getting stats at this time for the network in question...
				}
			default:
				sd.log.Warningf("Configured stats daemon data source returned error while consuming websocket stream: %s; restarting stream...", err.Error())
			}
		}
	}
}

// This loop is responsible for processing new messages received by daemon
func (sd *StatsDaemon) loop() error {
	for {
		select {
		case msg := <-sd.queue:
			sd.log.Debugf("Stats daemon runloop received network stats msg: %s", msg)

		case <-sd.shutdownCtx.Done():
			sd.log.Debugf("Closing stats daemon on shutdown")
			return nil
		}
	}
}

// RequireNetworkStatsDaemon ensures a single stats daemon instance is running for
// the given network; if no stats daemon instance has been started for the network,
// the instance is configured and started immediately, caching real-time network stats.
func RequireNetworkStatsDaemon(network *Network) *StatsDaemon {
	var daemon *StatsDaemon
	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		Log.Debugf("Cached stats daemon instance found for network: %s; id: %s", *network.Name, network.ID)
		return daemon
	}

	Log.Infof("Initializing new stats daemon instance for network: %s; id: %s", *network.Name, network.ID)
	daemon = NewNetworkStatsDaemon(Log, network)
	currentNetworkStats[network.ID.String()] = daemon

	daemon.wg.Add(1)
	go func() {
		var err error

		for !daemon.shuttingDown() {
			daemon.attempt++
			Log.Debugf("Stepping into main runloop of stats daemon instance; attempt #%v", daemon.attempt)
			err = daemon.run()
		}

		if !daemon.shuttingDown() {
			Log.Errorf("Forcing shutdown of stats daemon due to error; %s", err)
			daemon.shutdown()
		}
	}()

	return daemon
}

// NewNetworkStatsDaemon initializes a new network stats daemon instance using
// NetworkStatsDataSourceFactory to construct the daemon's its data source
func NewNetworkStatsDaemon(lg *logger.Logger, network *Network) *StatsDaemon {
	sd := new(StatsDaemon)
	sd.attempt = 0
	sd.log = lg.Clone()
	sd.shutdownCtx, sd.cancelF = context.WithCancel(context.Background())
	sd.dataSource = NetworkStatsDataSourceFactory(network)
	sd.queue = make(chan *provide.NetworkStatus, 32)
	sd.handleSignals()
	return sd
}

// Run the configured stats daemon instance
func (sd *StatsDaemon) run() error {
	sd.consumeAsync()
	err := sd.loop()

	if err == nil {
		sd.log.Info("StatsDaemon exited cleanly")
	} else {
		sd.log.Warningf("StatsDaemon exited; %s", err)
	}

	return err
}

func (sd *StatsDaemon) handleSignals() {
	Log.Debug("Installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sd.wg.Add(1)
	go func() {
		defer sd.wg.Done()
		select {
		case sig := <-sigs:
			Log.Infof("Received signal: %s", sig)
			sd.shutdown()
		case <-sd.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (sd *StatsDaemon) shutdown() {
	if atomic.AddUint32(&sd.closing, 1) == 1 {
		Log.Debug("Shutdown broadcast")
		sd.cancelF()
	}
}

func (sd *StatsDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&sd.closing) > 0)
}
