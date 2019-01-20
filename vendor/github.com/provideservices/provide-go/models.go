package provide

import (
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kthomas/go.uuid"
)

// Model base class
type Model struct {
	ID        uuid.UUID `sql:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt time.Time `sql:"not null;default:now()" json:"created_at"`
	Errors    []*Error  `sql:"-" json:"-"`
}

// Error struct
type Error struct {
	Message *string `json:"message"`
	Status  *int    `json:"status"`
}

// APICall struct
type APICall struct {
	Sub           string    `json:"sub"`
	Method        string    `json:"method"`
	Host          string    `json:"host"`
	Path          string    `json:"path"`
	RemoteAddr    string    `json:"remote_addr"`
	StatusCode    int       `json:"status_code"`
	ContentLength *uint     `json:"content_length"`
	Timestamp     time.Time `json:"timestamp"`
}

// CompiledArtifact represents compiled sourcecode
type CompiledArtifact struct {
	Name        string                 `json:"name"`
	ABI         []interface{}          `json:"abi"`
	Assembly    map[string]interface{} `json:"assembly"`
	Bytecode    string                 `json:"bytecode"`
	Deps        map[string]interface{} `json:"deps"`
	Opcodes     string                 `json:"opcodes"`
	Raw         json.RawMessage        `json:"raw"`
	Source      *string                `json:"source"`
	Fingerprint *string                `json:"fingerprint"`
}

// EthereumTxTraceResponse is returned upon successful contract execution
type EthereumTxTraceResponse struct {
	Result []struct {
		Action struct {
			CallType *string `json:"callType"`
			From     *string `json:"from"`
			Gas      *string `json:"gas"`
			Init     *string `json:"init"`
			Input    *string `json:"input"`
			To       *string `json:"to"`
			Value    *string `json:"value"`
		} `json:"action"`
		BlockHash   *string `json:"blockHash"`
		BlockNumber int     `json:"blockNumber"`
		Result      struct {
			Address *string `json:"address"`
			Code    *string `json:"code"`
			GasUsed *string `json:"gasUsed"`
			Output  *string `json:"output"`
		} `json:"result"`
		Error               *string       `json:"error"`
		Subtraces           int           `json:"subtraces"`
		TraceAddress        []interface{} `json:"traceAddress"`
		TransactionHash     *string       `json:"transactionHash"`
		TransactionPosition int           `json:"transactionPosition"`
		Type                *string       `json:"type"`
	} `json:"result"`
}

// EthereumJsonRpcResponse is a generic handler for ethereum JSON-RPC responses
type EthereumJsonRpcResponse struct {
	ID     uint64      `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

// EthereumWebsocketSubscriptionResponse is a generic handler for ethereum websocket subscription responses
type EthereumWebsocketSubscriptionResponse struct {
	ID     interface{}            `json:"id"`
	Params map[string]interface{} `json:"params"`
}

// NetworkStatus provides network-agnostic status
type NetworkStatus struct {
	Block           uint64                 `json:"block"`            // current block
	ChainID         *string                `json:"chain_id"`         // the chain id
	Height          *uint64                `json:"height"`           // total height of the blockchain; null after syncing completed
	LastBlockAt     *uint64                `json:"last_block_at"`    // unix timestamp of the last block; i.e., when the last block was collated
	PeerCount       uint64                 `json:"peer_count"`       // number of peers connected to the JSON-RPC client
	ProtocolVersion *string                `json:"protocol_version"` // protocol version
	State           *string                `json:"state"`            // i.e., syncing, synced, etc
	Syncing         bool                   `json:"syncing"`          // when true, the network is in the process of syncing the ledger; available functionaltiy will be network-specific
	Meta            map[string]interface{} `json:"meta"`             // network-specific metadata
}

// Paginate the given query given the page number and results per page;
// returns the update query and total results
func paginate(db *gorm.DB, model interface{}, page, rpp int64) (query *gorm.DB, totalResults *uint64) {
	db.Model(model).Count(&totalResults)
	query = db.Limit(rpp).Offset((page - 1) * rpp)
	return query, totalResults
}
