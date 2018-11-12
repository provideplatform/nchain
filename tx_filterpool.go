package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	streamingTxFilterPoolWaitGroup sync.WaitGroup
	txFilterConnectionPools        = map[string]*streamingConnPool{}
)

type streamingConnPool struct {
	closed  bool
	conns   chan net.Conn
	factory func() (net.Conn, error)
}

func (pool *streamingConnPool) Close() {
	conns := pool.conns
	pool.conns = nil
	pool.closed = true

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

// RunStreamingTxFilterConnectionPools is the entry point that bootstraps a pool
// for each application tx filter that is configured with a streaming tx filter
func RunStreamingTxFilterConnectionPools() {
	db := DatabaseConnection()
	var filters []Filter
	db.Find(&filters)
	for _, filter := range filters {
		filter.StartTxStreamingConnectionPool()
	}
}

func (pool *streamingConnPool) Size() int {
	return len(pool.conns)
}

func hasInMemoryStreamingTxConnectionPool(applicationID string) bool {
	return txFilterConnectionPools[applicationID] != nil
}

func (pool *streamingConnPool) leaseConnection(timeout time.Duration) (net.Conn, error) {
	if pool.closed {
		return nil, fmt.Errorf("Failed to lease a connection from the tx filter connection pool; connection pool is closed")
	}

	select {
	case conn := <-pool.conns:
		if conn == nil {
			return nil, fmt.Errorf("Failed to lease a connection from the tx filter connection pool")
		}
		return conn, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("Failed to lease a connection from the tx filter connection pool; timeout reached")
	default:
		if pool.Size() < int(streamingTxFilterPoolMaxConnectionCount) {
			conn, err := pool.factory()
			if err != nil {
				Log.Warningf("Failed to create a pooled streaming tx filter connection; %s", err.Error())
				return nil, err
			}
			return conn, nil
		}
		return nil, fmt.Errorf("Failed to create a pooled streaming tx filter connection; pool contains maximum number of connections (%d)", streamingTxFilterPoolMaxConnectionCount)
	}
}
