package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	dbconf "github.com/kthomas/go-db-config"
	logger "github.com/kthomas/go-logger"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
	provide "github.com/provideplatform/provide-go/api/nchain"
)

const natsLogTransceiverEmitSubject = "nchain.logs.emit"

const defaultLogTransceiverQueueSize = 512
const defaultLogTransceiverMaximumBackoffMillis = 12800

var currentLogTransceivers = map[string]*LogTransceiver{}
var currentLogTransceiversMutex = &sync.Mutex{}

// LogTransceiver struct
type LogTransceiver struct {
	attempt uint32
	backoff int64

	// natsConnection *stan.Conn
	db  *gorm.DB
	log *logger.Logger

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	queue chan *[]byte

	Network *network.Network
	Stream  func(chan *[]byte) error // websocket -- implementations should be blocking
}

// EthereumLogTransceiverFactory builds and returns a streaming logs transceiver which is
// used to efficiently propagate interesting log events to our low-latency message daemnon
func EthereumLogTransceiverFactory(network *network.Network) *LogTransceiver {
	return &LogTransceiver{
		Network: network,

		Stream: func(ch chan *[]byte) error {
			websocketURL := network.WebsocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}
			var wsDialer websocket.Dialer
			wsConn, _, err := wsDialer.Dial(websocketURL, nil)
			if err != nil {
				common.Log.Errorf("failed to establish network logs websocket connection to %s; %s", websocketURL, err.Error())
			} else {
				defer wsConn.Close()
				id, _ := uuid.NewV4()
				// xxx let's try out the blockNumber param
				payload := map[string]interface{}{
					"method":  "eth_subscribe",
					"params":  []interface{}{"logs", map[string]interface{}{}},
					"id":      id.String(),
					"jsonrpc": "2.0",
				}
				if err := wsConn.WriteJSON(payload); err != nil {
					common.Log.Errorf("failed to write subscribe message to network logs websocket connection")
				} else {
					common.Log.Debugf("subscribed to network logs websocket: %s", websocketURL)

					for {
						_, message, err := wsConn.ReadMessage()
						if err != nil {
							common.Log.Errorf("failed to receive event on network logs websocket; %s", err)
							break
						} else {
							common.Log.Debugf("received %d-byte event on network logs websocket for network: %s", len(message), *network.Name)

							response := &provide.EthereumWebsocketSubscriptionResponse{}
							err := json.Unmarshal(message, response)
							if err != nil {
								common.Log.Warningf("failed to unmarshal event received on network logs websocket: %s; %s", message, err.Error())
							} else {
								if result, ok := response.Params["result"].(map[string]interface{}); ok {
									result["block_hash"] = result["blockHash"]
									result["type"] = result["contractType"]
									result["transaction_hash"] = result["transactionHash"]
									result["network_id"] = network.ID.String()

									if resultJSON, err := json.Marshal(result); err == nil {
										ch <- &resultJSON
									}
								}
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
func (lt *LogTransceiver) consume() error {
	lt.log.Debugf("Attempting to consume configured network log transceiver; attempt #%v", lt.attempt)

	var err error
	if lt.Stream != nil {
		err = lt.Stream(lt.queue)
	} else {
		err = errors.New("Configured log transceiver does not have a configured Stream impl")
	}

	return err
}

func (lt *LogTransceiver) ingest(logmsg []byte) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed log transceiver event ingestion attempt; %s", r)
		}
	}()

	if lt.Network.IsEthereumNetwork() {
		lt.ingestEthereum(logmsg)
	}
}

func (lt *LogTransceiver) ingestEthereum(logmsg []byte) {
	err := natsutil.NatsStreamingPublish(natsLogTransceiverEmitSubject, logmsg)
	if err != nil {
		common.Log.Warningf("log transceiver failed to publish %d-byte log emission message; %s", len(logmsg), err.Error())
	}
}

// loop is responsible for processing new messages received by daemon
func (lt *LogTransceiver) loop() error {
	for {
		select {
		case evt := <-lt.queue:
			lt.ingest(*evt)

		case <-lt.shutdownCtx.Done():
			lt.log.Debugf("closing log transceiver on shutdown")
			return nil
		}
	}
}

// EvictNetworkLogTransceiver evicts a single, previously-initialized log transceiver instance {
func EvictNetworkLogTransceiver(network *network.Network) error {
	currentLogTransceiversMutex.Lock()
	defer currentLogTransceiversMutex.Unlock()

	if daemon, ok := currentLogTransceivers[network.ID.String()]; ok {
		common.Log.Debugf("evicting log transceiver instance for network: %s; id: %s", *network.Name, network.ID)
		daemon.shutdown()
		delete(currentLogTransceivers, network.ID.String())
		return nil
	}

	return fmt.Errorf("unable to evict log transceiver instance for network: %s; id; %s", *network.Name, network.ID)
}

// RequireNetworkLogTransceiver ensures a single log transceiver instance is running for
// the given network; if no log transceiver instance has been started for the network,
// the instance is configured and started immediately, caching real-time network logs.
func RequireNetworkLogTransceiver(network *network.Network) *LogTransceiver {
	var daemon *LogTransceiver
	if daemon, ok := currentLogTransceivers[network.ID.String()]; ok {
		common.Log.Debugf("Cached log transceiver instance found for network: %s; id: %s", *network.Name, network.ID)
		return daemon
	}

	currentLogTransceiversMutex.Lock()
	defer currentLogTransceiversMutex.Unlock()

	common.Log.Infof("Initializing new log transceiver instance for network: %s; id: %s", *network.Name, network.ID)
	daemon = NewNetworkLogTransceiver(common.Log, network)
	if daemon != nil {
		currentLogTransceivers[network.ID.String()] = daemon
		go daemon.run()
	}

	return daemon
}

// NewNetworkLogTransceiver initializes a new network logs daemon instance using
// a network-specific factory method to construct the daemon's steraming data source
func NewNetworkLogTransceiver(lg *logger.Logger, network *network.Network) *LogTransceiver {
	var lt *LogTransceiver
	if network.IsEthereumNetwork() {
		lt = EthereumLogTransceiverFactory(network)
	}

	if lt != nil {
		lt.db = dbconf.DatabaseConnection()
		lt.log = lg.Clone()
		lt.shutdownCtx, lt.cancelF = context.WithCancel(context.Background())
		lt.queue = make(chan *[]byte, defaultStatsDaemonQueueSize)
	}

	return lt
}

// Run the configured log transceiver instance
func (lt *LogTransceiver) run() error {
	go func() {
		for !lt.shuttingDown() {
			lt.attempt++
			common.Log.Debugf("Stepping into main runloop of log transceiver instance; attempt #%v", lt.attempt)
			err := lt.consume()
			if err != nil {
				common.Log.Warningf("Configured network log transceiver failed to consume network log events; %s", err.Error())
				if lt.backoff == 0 {
					lt.backoff = 100
				} else {
					lt.backoff *= 2
				}
				if lt.backoff > defaultLogTransceiverMaximumBackoffMillis {
					lt.backoff = 0
				}
				time.Sleep(time.Duration(lt.backoff) * time.Millisecond)
				lt.Network.Reload()
			}
		}
	}()

	err := lt.loop()

	if err == nil {
		lt.log.Info("Network log transceiver exited cleanly")
	} else {
		if !lt.shuttingDown() {
			common.Log.Errorf("Forcing shutdown of log transceiver due to error; %s", err)
			lt.shutdown()
		}
	}

	return err
}

func (lt *LogTransceiver) handleSignals() {
	common.Log.Debug("Installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			lt.shutdown()
		case <-lt.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (lt *LogTransceiver) shutdown() {
	if atomic.AddUint32(&lt.closing, 1) == 1 {
		common.Log.Debugf("Shutting down log transceiver instance for network: %s", *lt.Network.Name)
		lt.cancelF()
	}
}

func (lt *LogTransceiver) shuttingDown() bool {
	return (atomic.LoadUint32(&lt.closing) > 0)
}
