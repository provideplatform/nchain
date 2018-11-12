package main

import (
	"fmt"
	"net"
	"sync"
)

var (
	streamingTxFilterPoolWaitGroup sync.WaitGroup
	txFilterConnectionPools        map[string]*streamingConnPool
)

type streamingConnPool struct {
	// mu    sync.RWMutex
	conns   chan net.Conn
	factory func() (net.Conn, error)
}

func (pool *streamingConnPool) Close() {
	conns := pool.conns
	pool.conns = nil

	close(conns)
	for conn := range conns {
		conn.Close()
	}
}

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

func (pool *streamingConnPool) leaseConnection() (net.Conn, error) {
	select {
	case conn := <-pool.conns:
		if conn == nil {
			return nil, fmt.Errorf("Failed to lease a connection from the tx filter connection pool")
		}
		return conn, nil
	default:
		conn, err := pool.factory()
		if err != nil {
			Log.Warningf("Failed to create a pooled streaming tx filter connection; %s", err.Error())
			return nil, err
		}
		return conn, nil
	}
}
