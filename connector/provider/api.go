package provider

import (
	"time"

	"github.com/provideservices/provide-go"
)

const ElasticsearchConnectorProvider = "elasticsearch"
const IPFSConnectorProvider = "ipfs"
const MongoDBConnectorProvider = "mongodb"
const RedisConnectorProvider = "redis"
const TableauConnectorProvider = "tableau"
const ZokratesConnectorProvider = "zokrates"

const natsConnectorDenormalizeConfigSubject = "goldmine.connector.config.denormalize"
const natsConnectorResolveReachabilitySubject = "goldmine.connector.reachability.resolve"
const natsLoadBalancerBalanceNodeSubject = "goldmine.node.balance"
const natsLoadBalancerDeprovisioningSubject = "goldmine.loadbalancer.deprovision"

// API defines a provider interface for third-party software and service connectors,
// providing an interface for infrastructure provisioning and deprovisioning and an
// interface for data exchange (i.e., CRUD, search, etc.)
type API interface {
	// infrastructure-specific
	Deprovision() error
	DeprovisionNode() error
	Provision() error
	ProvisionNode() error
	Reachable() bool

	// "data-like" connector-specific resource apis, starting with CRUD (i.e.,
	// this is a proxy interface to the underlying provider such as IPFS)
	Create(params map[string]interface{}) (*ConnectedEntity, error)
	Read(id string) (*ConnectedEntity, error)
	Update(id string, params map[string]interface{}) error
	Delete(id string) error

	List(params map[string]interface{}) ([]*ConnectedEntity, error)
	Query(q string) (interface{}, error)
}

// ConnectedEntity is a generic representation for single object entities that provider APIs map
// data-related API calls into; this is a point for adopting standards as such develop...
type ConnectedEntity struct {
	// core model
	ID         *string                `json:"id,omitempty"`
	CreatedAt  *time.Time             `json:"created_at,omitempty"`
	DataURL    *string                `json:"data_url,omitempty"`
	Errors     []*provide.Error       `json:"errors,omitempty"`
	Type       *string                `json:"type,omitempty"`
	Hash       *string                `json:"hash,omitempty"`
	Href       *string                `json:"href,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ModifiedAt *time.Time             `json:"modified_at,omitempty"`
	Filename   *string                `json:"filename,omitempty"`
	Name       *string                `json:"name,omitempty"`
	Raw        *string                `json:"raw,omitempty"`
	Size       *uint64                `json:"size,omitempty"`

	// relations
	Parent   *ConnectedEntity    `json:"parent,omitempty"`
	Children *[]*ConnectedEntity `json:"children,omitempty"`
}
