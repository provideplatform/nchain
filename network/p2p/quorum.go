package p2p

// QuorumP2PProvider is a network.P2PAPI implementing the quorum API
type QuorumP2PProvider struct {
	rpcClientKey string
	rpcURL       string
}

// InitQuorumP2PProvider initializes and returns the quorum p2p provider
func InitQuorumP2PProvider(host string, port uint) *QuorumP2PProvider {
	return nil
}
