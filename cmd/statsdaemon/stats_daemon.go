package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
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
	natsutil "github.com/kthomas/go-natsutil"
	redisutil "github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/network"
	"github.com/provideplatform/provide-go/api/nchain"
	provide "github.com/provideplatform/provide-go/api/nchain"
	providecrypto "github.com/provideplatform/provide-go/crypto"
)

const blockchainInfoWebsocketURL = "wss://ws.blockchain.info/inv"
const defaultChainpointBufferSize = 64
const defaultChainpointFlushInterval = time.Millisecond * 60000
const defaultChainpointProofInterval = time.Millisecond * 60500
const defaultStatsDaemonQueueSize = 8
const defaultStatsTTL = time.Minute * 60
const natsBlockFinalizedSubject = "nchain.block.finalized"
const networkStatsJsonRpcPollingTickerInterval = time.Millisecond * 2500
const networkStatsMaxRecentBlockCacheSize = 8
const networkStatsMinimumRecentBlockCacheSize = 3
const statsDaemonMaximumBackoffMillis = 12800

var currentNetworkStats = map[string]*StatsDaemon{}
var currentNetworkStatsMutex = &sync.Mutex{}

// NetworkStatsDataSource provides JSON-RPC polling (http) and streaming (websocket)
// interfaces for a network
type NetworkStatsDataSource struct {
	Network *network.Network
	Poll    func(chan *provide.NetworkStatus) error // JSON-RPC polling -- implementations should be blocking
	Stream  func(chan *provide.NetworkStatus) error // websocket -- implementations should be blocking
}

// StatsDaemon struct
type StatsDaemon struct {
	attempt    uint32
	backoff    int64
	dataSource *NetworkStatsDataSource
	// natsConnection *stan.Conn

	log *logger.Logger

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	queue chan *provide.NetworkStatus

	recentBlocks          []interface{}
	recentBlockTimestamps []uint64
	stats                 *provide.NetworkStatus
}

type natsBlockFinalizedMsg struct {
	NetworkID *string `json:"network_id"`
	Block     uint64  `json:"block"`
	BlockHash *string `json:"blockhash"`
	Timestamp uint64  `json:"timestamp"`
}

type jsonRpcNotSupported string
type websocketNotSupported string

func (err jsonRpcNotSupported) Error() string {
	return "JSON-RPC not supported"
}

func (err websocketNotSupported) Error() string {
	return "Websocket not supported"
}

// BaseledgerNetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume EVM-based network statistics
func BaseledgerNetworkStatsDataSourceFactory(network *network.Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Network: network,

		Poll: func(ch chan *provide.NetworkStatus) error {
			return new(jsonRpcNotSupported)
		},

		Stream: func(ch chan *provide.NetworkStatus) error {
			websocketURL := network.WebsocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}
			var wsDialer websocket.Dialer
			wsConn, _, err := wsDialer.Dial(websocketURL, nil)
			if err != nil {
				common.Log.Errorf("Failed to establish network stats websocket connection to %s; %s", websocketURL, err.Error())
			} else {
				defer wsConn.Close()
				// { "jsonrpc": "2.0", "method": "subscribe", "params": ["tm.event='NewBlock'"], "id": 1 }
				payload := map[string]interface{}{
					"method":  "subscribe",
					"params":  []string{"tm.event='NewBlockHeader'"},
					"id":      1,
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
							common.Log.Debugf("Received %d-byte message on network stats websocket for network: %s", len(message), *network.Name)
							response := &provide.RPCResponse{}
							err := json.Unmarshal(message, response)
							if err != nil {
								common.Log.Warningf("Failed to unmarshal message received on network stats websocket: %s; %s", message, err.Error())
							} else {
								if result, ok := response.Result["data"].(map[string]interface{}); ok {
									if resultJSON, err := json.Marshal(result); err == nil {
										headerResponse := &nchain.BaseledgerBlockHeaderResponse{}
										err := json.Unmarshal(resultJSON, headerResponse)
										if err != nil {
											common.Log.Warningf("Failed to stringify result JSON in otherwise valid message received on network stats websocket: %s; %s", response, err.Error())
										} else if headerResponse != nil {
											common.Log.Debugf("Block header received %v\n", headerResponse.Value.Header)
											ch <- &provide.NetworkStatus{
												Meta: map[string]interface{}{
													"last_block_header": headerResponse.Value.Header,
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

// BcoinNetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume bcoin network statistics
func BcoinNetworkStatsDataSourceFactory(network *network.Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Network: network,

		Poll: func(ch chan *provide.NetworkStatus) error {
			return new(jsonRpcNotSupported)
		},

		Stream: func(ch chan *provide.NetworkStatus) error {
			websocketURL := network.WebsocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}

			networkCfg := network.ParseConfig()
			var rpcAPIUser *string
			var rpcAPIKey *string
			if rpcUser, rpcUserOk := networkCfg["rpc_api_user"].(string); rpcUserOk {
				rpcAPIUser = &rpcUser
			}
			if rpcKey, rpcKeyOk := networkCfg["rpc_api_key"].(string); rpcKeyOk {
				rpcAPIKey = &rpcKey
			}

			useBCInfoWebsocket := false

			if websocketURL == blockchainInfoWebsocketURL {
				common.Log.Debugf("enabling blockchain.info websocket on network stats websocket for configured network: %s", *network.Name)
				useBCInfoWebsocket = true
			}

			if useBCInfoWebsocket {
				websocketURL := network.WebsocketURL()
				if websocketURL == "" {
					err := new(websocketNotSupported)
					return *err
				}
				var wsDialer websocket.Dialer
				wsConn, _, err := wsDialer.Dial(websocketURL, nil)
				if err != nil {
					common.Log.Errorf("failed to establish network stats websocket connection to %s; %s", websocketURL, err.Error())
				} else {
					defer wsConn.Close()
					payload := map[string]interface{}{
						"op": "ping_block",
					}
					if err := wsConn.WriteJSON(payload); err != nil {
						common.Log.Errorf("failed to write ping_block message to blockchain.info network stats websocket connection")
						return err
					}

					payload = map[string]interface{}{
						"op": "blocks_sub",
					}
					if err := wsConn.WriteJSON(payload); err != nil {
						common.Log.Errorf("failed to write block subcription message to blockchain.info network stats websocket connection")
					} else {
						common.Log.Debugf("subscribed to block headers from blockchain.info network stats websocket: %s", websocketURL)

						for {
							_, message, err := wsConn.ReadMessage()
							if err != nil {
								common.Log.Errorf("failed to receive message on network stats websocket; %s", err)
								break
							} else {
								common.Log.Tracef("received %d-byte message on network stats websocket for network: %s", len(message), *network.Name)
								response := map[string]interface{}{}
								err := json.Unmarshal(message, &response)
								if err != nil {
									common.Log.Warningf("failed to unmarshal message received on network stats websocket: %s; %s", message, err.Error())
								} else {
									if op, opok := response["op"].(string); opok && op == "block" {
										if header, headerOk := response["x"].(map[string]interface{}); headerOk {
											common.Log.Tracef("received block header on blockchain.info network stats websocket subscription: %s", websocketURL)
											ch <- &provide.NetworkStatus{
												Meta: map[string]interface{}{
													blockchainInfoWebsocketURL: true,
													"last_block_header":        header,
												},
											}
										}
									}
								}
							}
						}
					}
				}
			} else {
				wsEndpoint := "ws"
				if strings.HasPrefix(websocketURL, "wss://") {
					websocketURL = strings.Split(websocketURL, "wss://")[len(strings.Split(websocketURL, "wss://"))-1]
				} else {
					websocketURL = strings.Split(websocketURL, "ws://")[len(strings.Split(websocketURL, "ws://"))-1]
				}

				cfg := &rpcclient.ConnConfig{
					Host:     websocketURL,
					Endpoint: wsEndpoint,
				}

				if rpcAPIUser != nil && rpcAPIKey != nil {
					cfg.User = *rpcAPIUser
					cfg.Pass = *rpcAPIKey
				}

				var client *rpcclient.Client
				var err error

				client, err = rpcclient.New(cfg, &rpcclient.NotificationHandlers{
					OnClientConnected: func() {
						common.Log.Debugf("bitcoin websocket client connected on configured network stats websocket for network: %s", *network.Name)

						// Register for block connect and disconnect notifications.
						if err := client.NotifyBlocks(); err != nil {
						} else {
							if err != nil {
								common.Log.Errorf("failed to establish network stats websocket subscription to %s for network: %s; %s", websocketURL, *network.Name, err.Error())
								client.Disconnect()
								return
							}
						}
					},

					OnFilteredBlockConnected: func(height int32, header *wire.BlockHeader, txns []*btcutil.Tx) {
						common.Log.Debugf("received block header on network stats websocket for network: %s; height: %d", *network.Name, height)
						common.Log.Debugf("block connected: %v (%d) %v", header.BlockHash(), height, header.Timestamp)

						ch <- &provide.NetworkStatus{
							Meta: map[string]interface{}{
								"last_block_header": header,
							},
						}
					},

					OnFilteredBlockDisconnected: func(height int32, header *wire.BlockHeader) {
						common.Log.Debugf("received block disconnected header on network stats websocket for network: %s; height: %d", *network.Name, height)
						common.Log.Debugf("block disconnected: %v (%d) %v", header.BlockHash(), height, header.Timestamp)
					},

					OnUnknownNotification: func(method string, params []json.RawMessage) {
						common.Log.Warningf("unknown notification received on bitcoin network stats websocket; method: %s; %s", method, params)
					},
				})

				if err != nil {
					common.Log.Errorf("failed to establish network stats websocket connection to %s for network: %s; %s", websocketURL, *network.Name, err.Error())
					return err
				}

				common.Log.Debugf("subscribed to network stats websocket: %s", websocketURL)
				client.WaitForShutdown()
			}

			return nil
		},
	}
}

// EthereumNetworkStatsDataSourceFactory builds and returns a JSON-RPC and streaming websocket
// data source which is used by stats daemon instances to consume EVM-based network statistics
func EthereumNetworkStatsDataSourceFactory(network *network.Network) *NetworkStatsDataSource {
	return &NetworkStatsDataSource{
		Network: network,

		Poll: func(ch chan *provide.NetworkStatus) error {
			return new(jsonRpcNotSupported)
		},

		Stream: func(ch chan *provide.NetworkStatus) error {
			websocketURL := network.WebsocketURL()
			if websocketURL == "" {
				err := new(websocketNotSupported)
				return *err
			}
			var wsDialer websocket.Dialer
			wsConn, _, err := wsDialer.Dial(websocketURL, nil)
			if err != nil {
				common.Log.Errorf("failed to establish network stats websocket connection to %s; %s", websocketURL, err.Error())
			} else {
				defer wsConn.Close()
				id, _ := uuid.NewV4()
				payload := map[string]interface{}{
					"method":  "eth_subscribe",
					"params":  []string{"newHeads"},
					"id":      id.String(),
					"jsonrpc": "2.0",
				}
				if err := wsConn.WriteJSON(payload); err != nil {
					common.Log.Errorf("failed to write subscribe message to network stats websocket connection")
				} else {
					common.Log.Debugf("subscribed to network stats websocket: %s", websocketURL)

					for {
						_, message, err := wsConn.ReadMessage()
						if err != nil {
							common.Log.Errorf("failed to receive message on network stats websocket; %s", err)
							break
						} else {
							common.Log.Tracef("received %d-byte message on network stats websocket for network: %s", len(message), *network.Name)
							response := &provide.EthereumWebsocketSubscriptionResponse{}
							err := json.Unmarshal(message, response)
							if err != nil {
								common.Log.Warningf("failed to unmarshal message received on network stats websocket: %s; %s", message, err.Error())
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
											common.Log.Warningf("failed to stringify result JSON in otherwise valid message received on network stats websocket: %s; %s", response, err.Error())
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
	sd.log.Debugf("attempting to consume configured stats daemon data source; attempt #%v", sd.attempt)

	var err error
	if sd.dataSource != nil {
		err = sd.dataSource.Stream(sd.queue)
	} else {
		err = errors.New("configured stats daemon does not have a configured data source")
	}

	if err != nil {
		errs = append(errs, err)
		switch err.(type) {
		case jsonRpcNotSupported:
			sd.log.Warningf("configured stats daemon data source does not support JSON-RPC; attempting to upgrade to websocket stream for network id: %s", sd.dataSource.Network.ID)
			err := sd.dataSource.Stream(sd.queue)
			if err != nil {
				errs = append(errs, err)
				sd.log.Warningf("configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		case websocketNotSupported:
			sd.log.Warningf("configured stats daemon data source does not support streaming via websocket; attempting to fallback to JSON-RPC long polling using stats daemon for network id: %s", sd.dataSource.Network.ID)
			err := sd.dataSource.Poll(sd.queue)
			if err != nil {
				errs = append(errs, err)
				sd.log.Warningf("configured stats daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		}
	}
	return errs
}

func (sd *StatsDaemon) ingest(response interface{}) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("recovered from failed stats daemon message ingestion attempt; %s", r)
		}
	}()

	if sd.dataSource.Network.IsBcoinNetwork() {
		sd.ingestBcoin(response)
	} else if sd.dataSource.Network.IsEthereumNetwork() {
		sd.ingestEthereum(response)
	}
}

func (sd *StatsDaemon) ingestBcoin(response interface{}) {
	switch response.(type) {
	case *provide.NetworkStatus:
		resp := response.(*provide.NetworkStatus)
		if resp != nil && resp.Meta != nil {
			header, headerOk := resp.Meta["last_block_header"].(map[string]interface{})
			isBCInfoWebsocket, isBCInfoWebsocketOk := resp.Meta[blockchainInfoWebsocketURL].(bool)
			chainInfo, chainInfoOk := resp.Meta["chain_info"].(map[string]interface{})
			if headerOk && isBCInfoWebsocketOk && isBCInfoWebsocket {
				if height, heightOk := header["height"].(float64); heightOk {
					sd.stats.Block = uint64(height)

					sd.stats.State = nil
					sd.stats.Syncing = sd.stats.Block == 0

					if sd.stats.Block == 0 {
						common.Log.Debugf("Ignoring genesis header")
						return
					}
				}

				var lastBlockAt uint64
				if timestamp, timestampOk := header["time"].(float64); timestampOk {
					lastBlockAt = uint64(timestamp) * 1000.0
					sd.stats.LastBlockAt = &lastBlockAt
				}

				sd.stats.Meta["last_block_header"] = header

				merkleRoot, _ := header["mrklRoot"].(string)

				// chainptID := fmt.Sprintf("provide.%s.block", sd.dataSource.Network.ID)
				// chainptHash := []byte(merkleRoot)
				// providechainpoint.ImmortalizeHashes(chainptID, []*[]byte{&chainptHash})

				if len(sd.recentBlocks) == 0 || sd.recentBlocks[len(sd.recentBlocks)-1].(map[string]interface{})["mrklRoot"].(string) != merkleRoot {
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
			} else if headerOk && chainInfoOk {
				if resp.Height != nil {
					sd.stats.Block = *resp.Height

					sd.stats.State = nil
					sd.stats.Syncing = sd.stats.Block == 0

					if sd.stats.Block == 0 {
						common.Log.Debugf("ignoring genesis header")
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
					// This is pretty naive but gives us an avg. time before we have >= 3 recent blocks;
					// can take some time after statsdaemon starts monitoring a PoW network...
					sd.stats.Meta["average_blocktime"] = (float64(time.Now().Unix()) - medianTime) / (11.0 / 2.0)
				}
			} else {
				common.Log.Warningf("failed to parse last_block_header from *provide.NetworkStats meta; dropping message...")
			}
		} else {
			common.Log.Warningf("received malformed *provide.NetworkStats message; dropping message...")
		}
	}

	sd.publish()
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
						common.Log.Warningf("failed to stringify result JSON in otherwise valid message received via JSON-RPC: %s; %s", response, err.Error())
					} else if hdr != nil && hdr.Number != nil {
						sd.ingest(hdr)
					}
				}
			} else {
				common.Log.Warningf("failed to parse last_block_header from *provide.NetworkStats meta; dropping message...")
			}
		} else {
			common.Log.Warningf("received malformed *provide.NetworkStats message; dropping message...")
		}
	case *types.Header:
		header := response.(*types.Header)
		sd.stats.Block = header.Number.Uint64()
		sd.stats.State = nil
		sd.stats.Syncing = sd.stats.Block == 0

		if sd.stats.Block == 0 {
			common.Log.Debugf("ignoring genesis header")
			return
		}

		lastBlockAt := header.Time * 1000.0
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
		natsutil.NatsJetstreamPublish(natsBlockFinalizedSubject, natsPayload)

		common.Log.Debugf("processed block %d (%s) on network: %s", header.Number.Uint64(), blockHash, *sd.dataSource.Network.Name)
	}

	sd.publish()
}

// loop is responsible for processing new messages received by daemon
func (sd *StatsDaemon) loop() error {
	for {
		select {
		case msg := <-sd.queue:
			sd.ingest(msg)

		case <-sd.shutdownCtx.Done():
			sd.log.Debugf("closing stats daemon on shutdown")
			return nil
		}
	}
}

// publish stats atomically to in-memory network namespace
func (sd *StatsDaemon) publish() error {
	payload, _ := json.Marshal(sd.stats)
	ttl := defaultStatsTTL
	err := redisutil.Set(sd.dataSource.Network.StatsKey(), string(payload), &ttl)
	if err != nil {
		common.Log.Warningf("failed to set network stats on key: %s; %s", sd.dataSource.Network.StatsKey(), err.Error())
	} else {
		natsutil.NatsPublish(sd.dataSource.Network.StatusKey(), payload)
	}
	return err
}

// EvictNetworkStatsDaemon evicts a single, previously-initialized stats daemon instance {
func EvictNetworkStatsDaemon(network *network.Network) error {
	currentNetworkStatsMutex.Lock()
	defer currentNetworkStatsMutex.Unlock()

	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		common.Log.Debugf("evicting stats daemon instance for network: %s; id: %s", *network.Name, network.ID)
		daemon.shutdown()
		delete(currentNetworkStats, network.ID.String())
		return nil
	}

	return fmt.Errorf("unable to evict stats daemon instance for network: %s; id; %s", *network.Name, network.ID)
}

// RequireNetworkStatsDaemon ensures a single stats daemon instance is running for
// the given network; if no stats daemon instance has been started for the network,
// the instance is configured and started immediately, caching real-time network stats.
func RequireNetworkStatsDaemon(network *network.Network) *StatsDaemon {
	var daemon *StatsDaemon
	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		common.Log.Debugf("cached stats daemon instance found for network: %s; id: %s", *network.Name, network.ID)
		return daemon
	}

	common.Log.Infof("initializing new stats daemon instance for network: %s; id: %s", *network.Name, network.ID)
	daemon = NewNetworkStatsDaemon(common.Log, network)
	if daemon != nil {
		currentNetworkStats[network.ID.String()] = daemon
		go daemon.run()
	}

	return daemon
}

// NewNetworkStatsDaemon initializes a new network stats daemon instance using
// NetworkStatsDataSourceFactory to construct the daemon's its data source
func NewNetworkStatsDaemon(lg *logger.Logger, network *network.Network) *StatsDaemon {
	currentNetworkStatsMutex.Lock()
	defer currentNetworkStatsMutex.Unlock()

	sd := new(StatsDaemon)
	sd.attempt = 0
	sd.log = lg.Clone()
	sd.shutdownCtx, sd.cancelF = context.WithCancel(context.Background())
	sd.queue = make(chan *provide.NetworkStatus, defaultStatsDaemonQueueSize)

	if network.IsBcoinNetwork() {
		sd.dataSource = BcoinNetworkStatsDataSourceFactory(network)
	} else if network.IsEthereumNetwork() {
		sd.dataSource = EthereumNetworkStatsDataSourceFactory(network)
	} else if network.IsBaseledgerNetwork() {
		sd.dataSource = BaseledgerNetworkStatsDataSourceFactory(network)
	}
	// sd.handleSignals()

	if sd.dataSource == nil {
		return nil
	}

	chainID := network.ChainID
	if chainID == nil {
		chn, err := providecrypto.EVMGetChainID(network.ID.String(), network.RPCURL())
		if err != nil {
			common.Log.Warningf("failed to retrieve chain id for %s network. Error: %s", network.ID.String(), err.Error())
			return nil
		}
		_chainID := hexutil.EncodeBig(chn)
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
			common.Log.Debugf("stepping into main runloop of stats daemon instance; attempt #%v", sd.attempt)
			errs := sd.consume()
			if len(errs) > 0 {
				common.Log.Warningf("configured stats daemon data source returned %v error(s) while attempting to consume configured data source", len(errs))
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
		sd.log.Info("stats daemon exited cleanly")
	} else {
		if !sd.shuttingDown() {
			common.Log.Errorf("forcing shutdown of stats daemon due to error; %s", err)
			sd.shutdown()
		}
	}

	return err
}

func (sd *StatsDaemon) handleSignals() {
	common.Log.Debug("installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			common.Log.Infof("received signal: %s", sig)
			sd.shutdown()
		case <-sd.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (sd *StatsDaemon) shutdown() {
	if atomic.AddUint32(&sd.closing, 1) == 1 {
		common.Log.Debugf("shutting down stats daemon instance for network: %s", *sd.dataSource.Network.Name)
		sd.cancelF()
	}
}

func (sd *StatsDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&sd.closing) > 0)
}
