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
	dbconf "github.com/kthomas/go-db-config"
	logger "github.com/kthomas/go-logger"
	natsutil "github.com/kthomas/go-natsutil"
	redisutil "github.com/kthomas/go-redisutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/provideapp/nchain/common"
	"github.com/provideapp/nchain/network"
	providego "github.com/provideservices/provide-go/api"
	provide "github.com/provideservices/provide-go/api/nchain"
	providecrypto "github.com/provideservices/provide-go/crypto"
)

// add some historical block consts
const defaultHistoricalBlockDaemonQueueSize = 8

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
			// here would be a good place to implement a next block getter
			// first check the network
			// if there's no block, then we're not worried about historical blocks,
			// q: do we need this, if we're worried about gaps?
			// a: yes, because we have to start there and put at least one block into the
			// blocks table so we can start getting gaps

			// but we do care about gaps (if the statsdaemon was down)
			// so query the blocks table for gaps
			// if there's no gaps, then adios
			// if there are gaps, then process them into a list of block numbers
			// and iterate through grabbing all the block infos
			// and pushing them onto nats for the block finalizer to grab
			// and update the block finalizer5 so it populates the blocks/tx table
			// block to begin with (must check where the txs come from!)

			// data type for blocks table
			type Block struct {
				providego.Model
				NetworkID uuid.UUID `sql:"type:uuid" json:"network_id"`
				Block     int       `sql:"type:int8" json:"block"`
				TxHash    string    `sql:"type:text" json:"transaction_hash"` //FIXME should be block hash
			}

			type BlockGap struct {
				Block         int
				PreviousBlock int
			}

			var blockGaps []BlockGap

			// TODO: check if there's the block in the network table
			// and if there is, check if that block exists for that network in the blocks table
			// if it doesn't, just pull that block info from the json rpc
			// and exit
			// it can grab the missing blocks (between the network.block table and the other blocks)
			// if there aren't any gaps, the statsdaemon will start making them... organically :)

			var missingBlocks []int
			db := dbconf.DatabaseConnection()
			db.Raw("select * from (select block, lag(block,1) over (order by block) as previous_block from blocks where network_id = ?) list where block - previous_block > 1", network.ID).Scan(&blockGaps)
			common.Log.Debugf("block: %+v", blockGaps)
			// block gaps is in the structure
			// block - previousblock, where there is a gap
			// so we iterate through it to get an array of blockNumbers we're missing
			for _, blockGap := range blockGaps {
				endBlock := blockGap.Block - 1
				startBlock := blockGap.PreviousBlock + 1
				gap := endBlock - startBlock
				for looper := 0; looper <= gap; looper++ {
					missingBlocks = append(missingBlocks, startBlock+looper)
				}
			}

			for _, missingBlock := range missingBlocks {
				var resp interface{}
				blockNumber := fmt.Sprintf("0x%x", missingBlock)
				//TODO use the providego method like Kyle hinted at :)
				err = client.Call(&resp, "eth_getBlockByNumber", blockNumber, true)
				if err != nil {
					return err
				}
				if resultJSON, err := json.Marshal(resp); err == nil {
					header := &types.Header{}
					err := json.Unmarshal(resultJSON, header)
					if err != nil {
						common.Log.Warningf("Failed to stringify result JSON in otherwise valid message received on network stats websocket: %s; %s", resp, err.Error())
					} else if header != nil && header.Number != nil {
						ch <- &provide.NetworkStatus{
							Meta: map[string]interface{}{
								"last_block_header": resp,
							},
						}
					}
				}
			}
			return err
		},
	}

}

// Consume the websocket stream; attempts to fallback to JSON-RPC if websocket stream fails or is not available for the network
func (hbd *HistoricalBlockDaemon) consume() []error {
	errs := make([]error, 0)
	hbd.log.Debugf("Attempting to consume configured historical block daemon data source; attempt #%v", hbd.attempt)

	var err error
	if hbd.dataSource != nil {
		err = hbd.dataSource.Poll(hbd.queue)
	} else {
		err = errors.New("Configured hsitorical blocks daemon does not have a configured data source")
	}

	if err != nil {
		errs = append(errs, err)
		switch err.(type) {
		case jsonRpcNotSupported:
			hbd.log.Warningf("Configured historical block  daemon data source does not support JSON-RPC; attempting to upgrade to websocket stream for network id: %s", hbd.dataSource.Network.ID)
			// no web socket here!
		case websocketNotSupported:
			hbd.log.Warningf("Configured historical block  daemon data source does not support streaming via websocket; attempting to fallback to JSON-RPC long polling using stats daemon for network id: %s", hbd.dataSource.Network.ID)
			err := hbd.dataSource.Poll(hbd.queue)
			if err != nil {
				errs = append(errs, err)
				hbd.log.Warningf("Configured historical block  daemon data source returned error while consuming JSON-RPC endpoint: %s; restarting stream...", err.Error())
			}
		}
	}
	return errs
}

func (hbd *HistoricalBlockDaemon) ingest(response interface{}) {
	defer func() {
		if r := recover(); r != nil {
			common.Log.Warningf("Recovered from failed historical blocks daemon message ingestion attempt; %s", r)
		}
	}()

	if hbd.dataSource.Network.IsBcoinNetwork() {
		// nop
	} else if hbd.dataSource.Network.IsEthereumNetwork() {
		hbd.ingestEthereum(response)
	}
}

func (hbd *HistoricalBlockDaemon) ingestEthereum(response interface{}) {
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
						hbd.ingest(hdr)
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
		hbd.stats.Block = header.Number.Uint64()
		hbd.stats.State = nil
		hbd.stats.Syncing = hbd.stats.Block == 0

		if hbd.stats.Block == 0 {
			common.Log.Debugf("Ignoring genesis header")
			return
		}

		lastBlockAt := header.Time * 1000.0
		hbd.stats.LastBlockAt = &lastBlockAt

		hbd.stats.Meta["last_block_header"] = header

		blockHash := header.Hash().String()

		if len(hbd.recentBlocks) == 0 || hbd.recentBlocks[len(hbd.recentBlocks)-1].(*types.Header).Hash().String() != blockHash {
			hbd.recentBlocks = append(hbd.recentBlocks, header)
			hbd.recentBlockTimestamps = append(hbd.recentBlockTimestamps, lastBlockAt)
		}

		for len(hbd.recentBlocks) > networkStatsMaxRecentBlockCacheSize {
			i := len(hbd.recentBlocks) - 1
			hbd.recentBlocks = append(hbd.recentBlocks[:i], hbd.recentBlocks[i+1:]...)
		}

		if len(hbd.recentBlocks) >= networkStatsMinimumRecentBlockCacheSize {
			blocktimes := make([]float64, 0)
			timedelta := float64(0)
			i := 0
			for i < len(hbd.recentBlocks)-1 {
				currentBlocktime := hbd.recentBlockTimestamps[i]
				nextBlocktime := hbd.recentBlockTimestamps[i+1]
				blockDelta := float64(nextBlocktime-currentBlocktime) / 1000.0
				blocktimes = append(blocktimes, blockDelta)
				timedelta += blockDelta
				i++
			}

			if len(blocktimes) > 0 {
				hbd.stats.Meta["average_blocktime"] = timedelta / float64(len(blocktimes))
				hbd.stats.Meta["blocktimes"] = blocktimes
				hbd.stats.Meta["last_block_hash"] = blockHash
			}
		}

		common.Log.Debugf("network: %s", *hbd.dataSource.Network.Name)
		common.Log.Debugf("block hash processed: %s", blockHash)
		common.Log.Debugf("block number processed: %v", header.Number.Uint64())
		natsPayload, _ := json.Marshal(&natsBlockFinalizedMsg{
			NetworkID: common.StringOrNil(hbd.dataSource.Network.ID.String()),
			Block:     header.Number.Uint64(),
			BlockHash: common.StringOrNil(blockHash),
			Timestamp: lastBlockAt,
		})

		natsutil.NatsStreamingPublish(natsBlockFinalizedSubject, natsPayload)
	}

	hbd.publish()
}

// loop is responsible for processing new messages received by daemon
func (hbd *HistoricalBlockDaemon) loop() error {
	for {
		select {
		case msg := <-hbd.queue:
			hbd.ingest(msg)

		case <-hbd.shutdownCtx.Done():
			hbd.log.Debugf("Closing historical block daemon on shutdown")
			return nil
		}
	}
}

// publish stats atomically to in-memory network namespace
func (hbd *HistoricalBlockDaemon) publish() error {
	payload, _ := json.Marshal(hbd.stats)
	ttl := defaultStatsTTL
	err := redisutil.Set(hbd.dataSource.Network.StatsKey(), string(payload), &ttl)
	if err != nil {
		common.Log.Warningf("failed to set network stats on key: %s; %s", hbd.dataSource.Network.StatsKey(), err.Error())
	} else {
		natsutil.NatsPublish(hbd.dataSource.Network.StatusKey(), payload)
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
	hbd := new(HistoricalBlockDaemon)
	hbd.attempt = 0
	hbd.log = lg.Clone()
	hbd.shutdownCtx, hbd.cancelF = context.WithCancel(context.Background())
	hbd.queue = make(chan *provide.NetworkStatus, defaultHistoricalBlockDaemonQueueSize)

	if network.IsEthereumNetwork() {
		hbd.dataSource = EthereumHistoricalBlockDataSourceFactory(network)
	}
	//sd.handleSignals()

	if hbd.dataSource == nil {
		return nil
	}

	chainID := network.ChainID
	if chainID == nil {
		chn, err := providecrypto.EVMGetChainID(network.ID.String(), network.RPCURL())
		if err != nil {
			common.Log.Debugf("Error getting chain ID of %s network. Error: %s", network.ID.String(), err.Error())
			return nil
		}
		_chainID := hexutil.EncodeBig(chn)
		chainID = &_chainID
	}
	hbd.stats = &provide.NetworkStatus{
		ChainID: chainID,
		Meta:    map[string]interface{}{},
		State:   common.StringOrNil("configuring"),
	}

	return hbd
}

// Run the configured stats daemon instance
func (hbd *HistoricalBlockDaemon) run() error {
	go func() {
		for !hbd.shuttingDown() {
			hbd.attempt++
			common.Log.Debugf("Stepping into main runloop of historical block daemon instance; attempt #%v", hbd.attempt)
			errs := hbd.consume()
			if len(errs) > 0 {
				common.Log.Warningf("Configured historical block daemon data source returned %v error(s) while attempting to consume configured data source", len(errs))
				if hbd.backoff == 0 {
					hbd.backoff = 100
				} else {
					hbd.backoff *= 2
				}
				if hbd.backoff > statsDaemonMaximumBackoffMillis {
					hbd.backoff = 0
				}
				time.Sleep(time.Duration(hbd.backoff) * time.Millisecond)
				hbd.dataSource.Network.Reload()
			}
		}
	}()

	err := hbd.loop()

	if err == nil {
		hbd.log.Info("Stats daemon exited cleanly")
	} else {
		if !hbd.shuttingDown() {
			common.Log.Errorf("Forcing shutdown of historical block daemon due to error; %s", err)
			hbd.shutdown()
		}
	}

	return err
}

func (hbd *HistoricalBlockDaemon) handleSignals() {
	common.Log.Debug("Installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			hbd.shutdown()
		case <-hbd.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (hbd *HistoricalBlockDaemon) shutdown() {
	if atomic.AddUint32(&hbd.closing, 1) == 1 {
		common.Log.Debugf("Shutting down historical block daemon instance for network: %s", *hbd.dataSource.Network.Name)
		hbd.cancelF()
	}
}

func (hbd *HistoricalBlockDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&hbd.closing) > 0)
}
