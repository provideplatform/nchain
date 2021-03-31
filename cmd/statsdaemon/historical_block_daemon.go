package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jinzhu/gorm"
	logger "github.com/kthomas/go-logger"
	natsutil "github.com/kthomas/go-natsutil"
	redisutil "github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
	provide "github.com/provideservices/provide-go/api/nchain"
	providecrypto "github.com/provideservices/provide-go/crypto"
)

// HistoricalBlockDataSource provides JSON-RPC polling (http) only
// interfaces for a network
type HistoricalBlockDataSource struct {
	Network *network.Network
	Poll    func(chan *provide.NetworkStatus) error // JSON-RPC polling -- implementations should be blocking
}

type HistoricalBlockDaemon struct {
	attempt    uint32
	backoff    int64
	dataSource *HistoricalBlockDataSource

	// natsConnection *stan.Conn
	db  *gorm.DB
	log *logger.Logger

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	queue chan *provide.NetworkStatus

	Network *network.Network
	Poll    func(chan *provide.NetworkStatus) error // websocket -- implementations should be blocking

	recentBlocks          []interface{}
	recentBlockTimestamps []uint64
	stats                 *provide.NetworkStatus
}

type jsonRPCRequest struct {
	ID      uuid.UUID     `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type jsonRPCResponse struct {
	ID      uuid.UUID       `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   interface{}     `json:"error"` //check what's in here
}

// EthereumHistoricalBlockDataSourceFactory builds and returns a JSON-RPC
// data source which is used by historical block daemon instances to consume historical blocks
func EthereumHistoricalBlockDataSourceFactory(network *network.Network) *HistoricalBlockDataSource {
	return &HistoricalBlockDataSource{
		Network: network,

		// wait, we get run  eth_getBlockByNumber via websocket - likely easier to copy the code!
		Poll: func(ch chan *provide.NetworkStatus) error {
			// json rpc call to eth_getBlockByNumber
			jsonRpcURL := network.RPCURL()
			if jsonRpcURL == "" {
				err := new(jsonRpcNotSupported)
				return *err
			}

			client, err := rpc.DialHTTP(jsonRpcURL)
			if err != nil {
				common.Log.Errorf("Failed to establish historical blocks RPC connection to %s; %s", jsonRpcURL, err.Error())
				return err
			}

			defer client.Close()

			// let's get a new id (likely used in nats)
			//id, _ := uuid.NewV4()

			//let's get block 9951220
			blockNumber := fmt.Sprintf("0x%x", 9951220)
			// request := jsonRPCRequest{
			// 	ID:      id,
			// 	JSONRPC: "2.0",
			// 	Method:  "eth_getBlockByNumber",
			// 	Params:  []interface{}{blockNumber, true},
			// }
			//args := []interface{}{blockNumber, true}

			// body, err := json.Marshal(request)
			// if err != nil {
			// 	return err
			// }
			type response struct {
				Result interface{}
			}
			//var result interface{}
			var resp interface{}
			err = client.Call(&resp, "eth_getBlockByNumber", blockNumber, true)
			if err != nil {
				return err
			}
			common.Log.Debugf("got result: %+v", resp)

			//return new(jsonRpcNotSupported)
			return err
		},
	}

}

// Consume the websocket stream; attempts to fallback to JSON-RPC if websocket stream fails or is not available for the network
func (sd *HistoricalBlockDaemon) consume() []error {
	errs := make([]error, 0)
	sd.log.Debugf("Attempting to consume configured historical block daemon data source; attempt #%v", sd.attempt)

	var err error
	if sd.dataSource != nil {
		err = sd.dataSource.Poll(sd.queue)
	} else {
		err = errors.New("Configured hsitorical blocks daemon does not have a configured data source")
	}

	if err != nil {
		errs = append(errs, err)
		switch err.(type) {
		case jsonRpcNotSupported:
			sd.log.Warningf("Configured historical block  daemon data source does not support JSON-RPC; attempting to upgrade to websocket stream for network id: %s", sd.dataSource.Network.ID)
			// no web socket here!
		case websocketNotSupported:
			sd.log.Warningf("Configured historical block  daemon data source does not support streaming via websocket; attempting to fallback to JSON-RPC long polling using stats daemon for network id: %s", sd.dataSource.Network.ID)
			err := sd.dataSource.Poll(sd.queue)
			if err != nil {
				errs = append(errs, err)
				sd.log.Warningf("Configured historical block  daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		}
	}
	return errs
}

func (sd *HistoricalBlockDaemon) ingest(response interface{}) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("Recovered from failed historical blocks daemon message ingestion attempt; %s", r)
		}
	}()

	if sd.dataSource.Network.IsBcoinNetwork() {
		// nop
	} else if sd.dataSource.Network.IsEthereumNetwork() {
		sd.ingestEthereum(response)
	}
}

func (sd *HistoricalBlockDaemon) ingestEthereum(response interface{}) {
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

		common.Log.Debugf("network: %s", *sd.dataSource.Network.Name)
		common.Log.Debugf("block hash processed: %s", blockHash)
		common.Log.Debugf("block number processed: %v", header.Number.Uint64())
		natsPayload, _ := json.Marshal(&natsBlockFinalizedMsg{
			NetworkID: common.StringOrNil(sd.dataSource.Network.ID.String()),
			Block:     header.Number.Uint64(),
			BlockHash: common.StringOrNil(blockHash),
			Timestamp: lastBlockAt,
		})

		natsutil.NatsStreamingPublish(natsBlockFinalizedSubject, natsPayload)
	}

	sd.publish()
}

// loop is responsible for processing new messages received by daemon
func (sd *HistoricalBlockDaemon) loop() error {
	for {
		select {
		case msg := <-sd.queue:
			sd.ingest(msg)

		case <-sd.shutdownCtx.Done():
			sd.log.Debugf("Closing historical block daemon on shutdown")
			return nil
		}
	}
}

// publish stats atomically to in-memory network namespace
func (sd *HistoricalBlockDaemon) publish() error {
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
func EvictHistoricalBlocksDaemon(network *network.Network) error {
	if daemon, ok := currentNetworkStats[network.ID.String()]; ok {
		common.Log.Debugf("Evicting historical blocks daemon instance for network: %s; id: %s", *network.Name, network.ID)
		daemon.shutdown()
		currentNetworkStatsMutex.Lock()
		delete(currentNetworkStats, network.ID.String())
		currentNetworkStatsMutex.Unlock()
		return nil
	}
	return fmt.Errorf("Unable to evict historical daemon instance for network: %s; id; %s", *network.Name, network.ID)
}

var currentHistoricalBlocks = map[string]*HistoricalBlockDaemon{}

// RequireNetworkStatsDaemon ensures a single stats daemon instance is running for
// the given network; if no stats daemon instance has been started for the network,
// the instance is configured and started immediately, caching real-time network stats.
func RequireHistoricalBlockStatsDaemon(network *network.Network) *HistoricalBlockDaemon {
	var daemon *HistoricalBlockDaemon
	if daemon, ok := currentHistoricalBlocks[network.ID.String()]; ok {
		common.Log.Debugf("Cached historical daemon instance found for network: %s; id: %s", *network.Name, network.ID)
		return daemon
	}

	currentNetworkStatsMutex.Lock()
	common.Log.Infof("Initializing new historical block daemon instance for network: %s; id: %s", *network.Name, network.ID)
	daemon = NewHistoricalBlockStatsDaemon(common.Log, network)
	if daemon != nil {
		currentHistoricalBlocks[network.ID.String()] = daemon
		go daemon.run()
	}
	currentNetworkStatsMutex.Unlock()

	return daemon
}

// NewNetworkStatsDaemon initializes a new network stats daemon instance using
// NetworkStatsDataSourceFactory to construct the daemon's its data source
func NewHistoricalBlockStatsDaemon(lg *logger.Logger, network *network.Network) *HistoricalBlockDaemon {
	sd := new(HistoricalBlockDaemon)
	sd.attempt = 0
	sd.log = lg.Clone()
	sd.shutdownCtx, sd.cancelF = context.WithCancel(context.Background())
	sd.queue = make(chan *provide.NetworkStatus, defaultStatsDaemonQueueSize)

	if network.IsEthereumNetwork() {
		sd.dataSource = EthereumHistoricalBlockDataSourceFactory(network)
	}
	//sd.handleSignals()

	if sd.dataSource == nil {
		return nil
	}

	chainID := network.ChainID
	if chainID == nil {
		_chainID := hexutil.EncodeBig(providecrypto.EVMGetChainID(network.ID.String(), network.RPCURL()))
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
func (sd *HistoricalBlockDaemon) run() error {
	go func() {
		for !sd.shuttingDown() {
			sd.attempt++
			common.Log.Debugf("Stepping into main runloop of historical block daemon instance; attempt #%v", sd.attempt)
			errs := sd.consume()
			if len(errs) > 0 {
				common.Log.Warningf("Configured historical block daemon data source returned %v error(s) while attempting to consume configured data source", len(errs))
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
			common.Log.Errorf("Forcing shutdown of historical block daemon due to error; %s", err)
			sd.shutdown()
		}
	}

	return err
}

func (sd *HistoricalBlockDaemon) handleSignals() {
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

func (sd *HistoricalBlockDaemon) shutdown() {
	if atomic.AddUint32(&sd.closing, 1) == 1 {
		common.Log.Debugf("Shutting down historical block daemon instance for network: %s", *sd.dataSource.Network.Name)
		sd.cancelF()
	}
}

func (sd *HistoricalBlockDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&sd.closing) > 0)
}
