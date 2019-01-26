package network

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

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/websocket"
	logger "github.com/kthomas/go-logger"
	"github.com/provideapp/goldmine/common"
	"github.com/provideservices/provide-go"
)

const defaultStatsDaemonQueueSize = 8
const networkStatsJsonRpcPollingTickerInterval = time.Millisecond * 2500
const networkStatsMaxRecentBlockCacheSize = 32
const networkStatsMinimumRecentBlockCacheSize = 3
const statsDaemonMaximumBackoffMillis = 12800

var currentNetworkStats = map[string]*StatsDaemon{}
var currentNetworkStatsMutex = &sync.Mutex{}

// NetworkStatsDataSource provides JSON-RPC polling (http) and streaming (websocket)
// interfaces for a network
type NetworkStatsDataSource struct {
	Network *Network
	Poll    func(chan *provide.NetworkStatus) error // JSON-RPC polling -- implementations should be blocking
	Stream  func(chan *provide.NetworkStatus) error // websocket -- implementations should be blocking
}

// StatsDaemon struct
type StatsDaemon struct {
	attempt    uint32
	backoff    int64
	dataSource *NetworkStatsDataSource

	log *logger.Logger

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	queue chan *provide.NetworkStatus

	recentBlocks          []interface{}
	recentBlockTimestamps []uint64
	stats                 *provide.NetworkStatus
}

type jsonRpcNotSupported string
type websocketNotSupported string

func (err jsonRpcNotSupported) Error() string {
	return "JSON-RPC not supported"
}

func (err websocketNotSupported) Error() string {
	return "Websocket not supported"
}

// BcoinNetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume bcoin network statistics
func BcoinNetworkStatsDataSourceFactory(network *Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Network: network,

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
					status, err := network.Status(true)
					if err != nil {
						common.Log.Errorf("Failed to retrieve network status via JSON-RPC: %s; %s", rpcURL, err)
						ticker.Stop()
						return nil
					}

					ch <- status
				}
			}
		},

		Stream: func(ch chan *provide.NetworkStatus) error {
			websocketURL := network.websocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}

			networkCfg := network.ParseConfig()
			var rpcAPIUser string
			var rpcAPIKey string
			if rpcUser, rpcUserOk := networkCfg["rpc_api_user"].(string); rpcUserOk {
				rpcAPIUser = rpcUser
			}
			if rpcKey, rpcKeyOk := networkCfg["rpc_api_key"].(string); rpcKeyOk {
				rpcAPIKey = rpcKey
			}

			cfg := &rpcclient.ConnConfig{
				Host:     websocketURL,
				Endpoint: "ws",
				User:     rpcAPIUser,
				Pass:     rpcAPIKey,
			}

			client, err := rpcclient.New(cfg, &rpcclient.NotificationHandlers{
				OnFilteredBlockConnected: func(height int32, header *wire.BlockHeader, txns []*btcutil.Tx) {
					common.Log.Debugf("Received message on network stats websocket; height: %d", height)
					common.Log.Debugf("Block connected: %v (%d) %v", header.BlockHash(), height, header.Timestamp)

				},
				OnFilteredBlockDisconnected: func(height int32, header *wire.BlockHeader) {
					common.Log.Debugf("Received message on network stats websocket; height: %d", height)
					common.Log.Debugf("Block disconnected: %v (%d) %v", header.BlockHash(), height, header.Timestamp)
				},
			})

			if err != nil {
				common.Log.Errorf("Failed to establish network stats websocket connection to %s", websocketURL)
				return err
			}

			// Register for block connect and disconnect notifications.
			if err := client.NotifyBlocks(); err != nil {
				common.Log.Errorf("Failed to write subscribe message to network stats websocket connection")
				return err
			} else {
				common.Log.Debugf("Subscribed to network stats websocket: %s", websocketURL)
			}
			client.WaitForShutdown()
			return nil
		},
	}
}

// EthereumNetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume EVM-based network statistics
func EthereumNetworkStatsDataSourceFactory(network *Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Network: network,

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
					status, err := network.Status(true)
					if err != nil {
						common.Log.Errorf("Failed to retrieve network status via JSON-RPC: %s; %s", rpcURL, err)
						ticker.Stop()
						return nil
					}

					ch <- status
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
				common.Log.Errorf("Failed to establish network stats websocket connection to %s", websocketURL)
			} else {
				defer wsConn.Close()
				var chainID string
				if network.ChainID != nil {
					chainID = *network.ChainID
				}
				payload := map[string]interface{}{
					"method":  "eth_subscribe",
					"params":  []string{"newHeads"},
					"id":      chainID,
					"jsonrpc": "2.0",
				}
				if err := wsConn.WriteJSON(payload); err != nil {
					common.Log.Errorf("Failed to write subscribe message to network stats websocket connection")
				} else {
					common.Log.Debugf("Subscribed to network stats websocket: %s", websocketURL)

					for {
						_, message, err := wsConn.ReadMessage()
						if err != nil {
							common.Log.Errorf("Failed to receive message on network stats websocket; %s", err)
							break
						} else {
							common.Log.Debugf("Received message on network stats websocket: %s", message)
							response := &provide.EthereumWebsocketSubscriptionResponse{}
							err := json.Unmarshal(message, response)
							if err != nil {
								common.Log.Warningf("Failed to unmarshal message received on network stats websocket: %s; %s", message, err.Error())
							} else {
								if result, ok := response.Params["result"].(map[string]interface{}); ok {
									if _, mixHashOk := result["mixHash"]; !mixHashOk {
										result["mixHash"] = ethcommon.HexToHash("0x")
									}
									if _, nonceOk := result["nonce"]; !nonceOk {
										result["nonce"] = types.EncodeNonce(0)
									}
									if resultJSON, err := json.Marshal(result); err == nil {
										header := &types.Header{}
										err := json.Unmarshal(resultJSON, header)
										if err != nil {
											common.Log.Warningf("Failed to stringify result JSON in otherwise valid message received on network stats websocket: %s; %s", response, err.Error())
										} else if header != nil && header.Number != nil {
											ch <- &provide.NetworkStatus{
												Meta: map[string]interface{}{
													"last_block_header": result,
												},
											}
										}
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
func (sd *StatsDaemon) consume() []error {
	errs := make([]error, 0)
	common.Log.Debugf("Attempting to consume configured stats daemon data source; attempt #%v", sd.attempt)

	var err error
	if sd.dataSource != nil {
		err = sd.dataSource.Stream(sd.queue)
	} else {
		err = errors.New("Configured stats daemon does not have a configured data source")
	}

	if err != nil {
		errs = append(errs, err)
		switch err.(type) {
		case jsonRpcNotSupported:
			sd.log.Warningf("Configured stats daemon data source does not support JSON-RPC: %s; attempting to upgrade to websocket stream for network id: %s", sd.dataSource.Network.ID)
			err := sd.dataSource.Stream(sd.queue)
			if err != nil {
				errs = append(errs, err)
				sd.log.Warningf("Configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		case websocketNotSupported:
			sd.log.Warningf("Configured stats daemon data source does not support streaming via websocket; attempting to fallback to JSON-RPC long polling using stats daemon for network id: %s", sd.dataSource.Network.ID)
			err := sd.dataSource.Poll(sd.queue)
			if err != nil {
				errs = append(errs, err)
				sd.log.Warningf("Configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		}
	}
	return errs
}

func (sd *StatsDaemon) ingest(response interface{}) {
	if sd.dataSource.Network.isBcoinNetwork() {
		sd.ingestBcoin(response)
	} else if sd.dataSource.Network.isLcoinNetwork() {
		sd.ingestLcoin(response)
	} else if sd.dataSource.Network.isEthereumNetwork() {
		sd.ingestEthereum(response)
	}
}

func (sd *StatsDaemon) ingestBcoin(response interface{}) {
	switch response.(type) {
	case *provide.NetworkStatus:
		resp := response.(*provide.NetworkStatus)
		if resp != nil && resp.Meta != nil {
			header, headerOk := resp.Meta["last_block_header"].(map[string]interface{})
			chainInfo, chainInfoOk := resp.Meta["chain_info"].(map[string]interface{})
			if headerOk && chainInfoOk {
				if resp.Height != nil {
					sd.stats.Block = *resp.Height

					sd.stats.State = nil
					sd.stats.Syncing = sd.stats.Block == 0

					if sd.stats.Block == 0 {
						common.Log.Debugf("Ignoring genesis header")
						return
					}
				}

				var lastBlockAt uint64
				if resp.LastBlockAt != nil {
					lastBlockAt = *resp.LastBlockAt * 1000.0
					sd.stats.LastBlockAt = &lastBlockAt
				}

				sd.stats.Meta["last_block_header"] = header

				merkleRoot, _ := header["merkleroot"].(string)

				if len(sd.recentBlocks) == 0 || sd.recentBlocks[len(sd.recentBlocks)-1].(map[string]interface{})["merkleroot"].(string) != merkleRoot {
					sd.recentBlocks = append(sd.recentBlocks, header)
					sd.recentBlockTimestamps = append(sd.recentBlockTimestamps, lastBlockAt)
				}

				for len(sd.recentBlocks) > networkStatsMaxRecentBlockCacheSize {
					i := len(sd.recentBlocks) - 1
					sd.recentBlocks = append(sd.recentBlocks[:i], sd.recentBlocks[i+1:]...)
				}

				if len(sd.recentBlocks) >= networkStatsMinimumRecentBlockCacheSize {
					blocktimes := make([]float64, 0)
					timedelta := float64(0)
					i := 0
					for i < len(sd.recentBlocks)-1 {
						currentBlocktime := sd.recentBlockTimestamps[i]
						nextBlocktime := sd.recentBlockTimestamps[i+1]
						blockDelta := float64(nextBlocktime-currentBlocktime) / 1000.0
						if blockDelta != 0 {
							blocktimes = append(blocktimes, blockDelta)
							timedelta += blockDelta
							i++
						}
					}

					if len(blocktimes) > 0 {
						sd.stats.Meta["average_blocktime"] = timedelta / float64(len(blocktimes))
						sd.stats.Meta["blocktimes"] = blocktimes
						sd.stats.Meta["last_block_hash"] = merkleRoot
					}
				} else if medianTime, medianTimeOk := chainInfo["mediantime"].(float64); medianTimeOk {
					// This is pretty naive but gives us an avg. time before we have >= 3 recent blocks; can take some time after statsdaemon starts monitoring a PoW network...
					sd.stats.Meta["average_blocktime"] = (float64(time.Now().Unix()) - medianTime) / (11.0 / 2.0)
				}
			} else {
				common.Log.Warningf("Failed to parse last_block_header from *provide.NetworkStats meta; dropping message...")
			}
		} else {
			common.Log.Warningf("Received malformed *provide.NetworkStats message; dropping message...")
		}
	}
}

func (sd *StatsDaemon) ingestLcoin(response interface{}) {
	switch response.(type) {
	default:
		common.Log.Warningf("Lcoin ingest functionality not yet implemented in stats daemon")
	}
}

func (sd *StatsDaemon) ingestEthereum(response interface{}) {
	switch response.(type) {
	case *provide.NetworkStatus:
		resp := response.(*provide.NetworkStatus)
		if resp != nil && resp.Meta != nil {
			if header, headerOk := resp.Meta["last_block_header"].(map[string]interface{}); headerOk {
				if _, mixHashOk := header["mixHash"]; !mixHashOk {
					header["mixHash"] = ethcommon.HexToHash("0x")
				}
				if _, nonceOk := header["nonce"]; !nonceOk {
					header["nonce"] = types.EncodeNonce(0)
				}

				if headerJSON, err := json.Marshal(header); err == nil {
					hdr := &types.Header{}
					err := json.Unmarshal(headerJSON, hdr)
					if err != nil {
						common.Log.Warningf("Failed to stringify result JSON in otherwise valid message received via JSON-RPC: %s; %s", response, err.Error())
					} else if hdr != nil && hdr.Number != nil {
						sd.ingest(hdr)
					}
				}
			} else {
				common.Log.Warningf("Failed to parse last_block_header from *provide.NetworkStats meta; dropping message...")
			}
		} else {
			common.Log.Warningf("Received malformed *provide.NetworkStats message; dropping message...")
		}
	case *types.Header:
		header := response.(*types.Header)
		sd.stats.Block = header.Number.Uint64()
		sd.stats.State = nil
		sd.stats.Syncing = sd.stats.Block == 0

		if sd.stats.Block == 0 {
			common.Log.Debugf("Ignoring genesis header")
			return
		}

		lastBlockAt := header.Time.Uint64() * 1000.0
		sd.stats.LastBlockAt = &lastBlockAt

		sd.stats.Meta["last_block_header"] = header

		blockHash := header.Hash().String()

		if len(sd.recentBlocks) == 0 || sd.recentBlocks[len(sd.recentBlocks)-1].(*types.Header).Hash().String() != blockHash {
			sd.recentBlocks = append(sd.recentBlocks, header)
			sd.recentBlockTimestamps = append(sd.recentBlockTimestamps, lastBlockAt)
		}

		for len(sd.recentBlocks) > networkStatsMaxRecentBlockCacheSize {
			i := len(sd.recentBlocks) - 1
			sd.recentBlocks = append(sd.recentBlocks[:i], sd.recentBlocks[i+1:]...)
		}

		if len(sd.recentBlocks) >= networkStatsMinimumRecentBlockCacheSize {
			blocktimes := make([]float64, 0)
			timedelta := float64(0)
			i := 0
			for i < len(sd.recentBlocks)-1 {
				currentBlocktime := sd.recentBlockTimestamps[i]
				nextBlocktime := sd.recentBlockTimestamps[i+1]
				blockDelta := float64(nextBlocktime-currentBlocktime) / 1000.0
				blocktimes = append(blocktimes, blockDelta)
				timedelta += blockDelta
				i++
			}

			if len(blocktimes) > 0 {
				sd.stats.Meta["average_blocktime"] = timedelta / float64(len(blocktimes))
				sd.stats.Meta["blocktimes"] = blocktimes
				sd.stats.Meta["last_block_hash"] = blockHash
			}
		}

		natsPayload, _ := json.Marshal(&natsBlockFinalizedMsg{
			NetworkID: common.StringOrNil(sd.dataSource.Network.ID.String()),
			Block:     header.Number.Uint64(),
			BlockHash: common.StringOrNil(blockHash),
			Timestamp: lastBlockAt,
		})

		natsConnection := common.GetDefaultNatsStreamingConnection()
		natsConnection.Publish(natsBlockFinalizedSubject, natsPayload)
	}
}

// This loop is responsible for processing new messages received by daemon
func (sd *StatsDaemon) loop() error {
	for {
		select {
		case msg := <-sd.queue:
			sd.ingest(msg)

		case <-sd.shutdownCtx.Done():
			sd.log.Debugf("Closing stats daemon on shutdown")
			return nil
		}
	}
}

// EvictNetworkStatsDaemon evicts a single, previously-initialized stats daemon instance {
func EvictNetworkStatsDaemon(network *Network) error {
	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		common.Log.Debugf("Evicting stats daemon instance for network: %s; id: %s", *network.Name, network.ID)
		daemon.shutdown()
		currentNetworkStatsMutex.Lock()
		delete(currentNetworkStats, network.ID.String())
		currentNetworkStatsMutex.Unlock()
		return nil
	}
	return fmt.Errorf("Unable to evict stats daemon instance for network: %s; id; %s", *network.Name, network.ID)
}

// RequireNetworkStatsDaemon ensures a single stats daemon instance is running for
// the given network; if no stats daemon instance has been started for the network,
// the instance is configured and started immediately, caching real-time network stats.
func RequireNetworkStatsDaemon(network *Network) *StatsDaemon {
	// if network.AvailablePeerCount() == 0 {
	// 	common.Log.Debugf("Stats daemon instance not initialized for network: %s; no available peers", *network.Name)
	// 	return nil
	// }

	var daemon *StatsDaemon
	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		common.Log.Debugf("Cached stats daemon instance found for network: %s; id: %s", *network.Name, network.ID)
		return daemon
	}

	currentNetworkStatsMutex.Lock()
	common.Log.Infof("Initializing new stats daemon instance for network: %s; id: %s", *network.Name, network.ID)
	daemon = NewNetworkStatsDaemon(common.Log, network)
	if daemon != nil {
		currentNetworkStats[network.ID.String()] = daemon
		go daemon.run()
	}
	currentNetworkStatsMutex.Unlock()

	return daemon
}

// NewNetworkStatsDaemon initializes a new network stats daemon instance using
// NetworkStatsDataSourceFactory to construct the daemon's its data source
func NewNetworkStatsDaemon(lg *logger.Logger, network *Network) *StatsDaemon {
	sd := new(StatsDaemon)
	sd.attempt = 0
	sd.log = lg.Clone()
	sd.shutdownCtx, sd.cancelF = context.WithCancel(context.Background())
	sd.queue = make(chan *provide.NetworkStatus, defaultStatsDaemonQueueSize)

	if network.isBcoinNetwork() {
		sd.dataSource = BcoinNetworkStatsDataSourceFactory(network)
	} else if network.isEthereumNetwork() {
		sd.dataSource = EthereumNetworkStatsDataSourceFactory(network)
	}
	//sd.handleSignals()

	if sd.dataSource == nil {
		return nil
	}

	chainID := network.ChainID
	if chainID == nil {
		_chainID := hexutil.EncodeBig(provide.EVMGetChainID(network.ID.String(), network.rpcURL()))
		chainID = &_chainID
	}
	sd.stats = &provide.NetworkStatus{
		ChainID: chainID,
		Meta:    map[string]interface{}{},
		State:   common.StringOrNil("configuring"),
	}

	return sd
}

// Run the configured stats daemon instance
func (sd *StatsDaemon) run() error {
	go func() {
		for !sd.shuttingDown() {
			sd.attempt++
			common.Log.Debugf("Stepping into main runloop of stats daemon instance; attempt #%v", sd.attempt)
			errs := sd.consume()
			if len(errs) > 0 {
				common.Log.Warningf("Configured stats daemon data source returned %v error(s) while attempting to consume configured data source", len(errs))
				if sd.backoff == 0 {
					sd.backoff = 100
				} else {
					sd.backoff *= 2
				}
				if sd.backoff > statsDaemonMaximumBackoffMillis {
					sd.backoff = 0
				}
				time.Sleep(time.Duration(sd.backoff) * time.Millisecond)
				sd.dataSource.Network.Reload()
			}
		}
	}()

	err := sd.loop()

	if err == nil {
		sd.log.Info("Stats daemon exited cleanly")
	} else {
		if !sd.shuttingDown() {
			common.Log.Errorf("Forcing shutdown of stats daemon due to error; %s", err)
			sd.shutdown()
		}
	}

	return err
}

func (sd *StatsDaemon) handleSignals() {
	common.Log.Debug("Installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			sd.shutdown()
		case <-sd.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (sd *StatsDaemon) shutdown() {
	if atomic.AddUint32(&sd.closing, 1) == 1 {
		common.Log.Debugf("Shutting down stats daemon instance for network: %s", *sd.dataSource.Network.Name)
		sd.cancelF()
	}
}

func (sd *StatsDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&sd.closing) > 0)
}
