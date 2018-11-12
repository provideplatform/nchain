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
		params := filter.ParseParams()
		host, hostOk := params["host"].(string)
		port, portOk := params["port"].(uint64)
		if hostOk && portOk {
			StartTxFilterPool(filter.ID.String(), host, port)
		}
	}
}

func (pool *streamingConnPool) Size() int {
	return len(pool.conns)
}

func hasInMemoryStreamingTxConnectionPool(applicationID string) bool {
	return txFilterConnectionPools[applicationID] != nil
}

// StartTxFilterPool launches a goroutine for the configured number of connections
// for the given filter id
func StartTxFilterPool(filterID string, host string, port uint64) { // FIXME: should this live as a member of *Filter?
	if _, filterPoolOk := txFilterConnectionPools[filterID]; filterPoolOk {
		Log.Warningf("Attempting to start streaming tx filter pool that has already been allocated; filter id: %s", filterID)
		return
	}

	addr := fmt.Sprintf("%s:%v", host, port)
	Log.Debugf("Attempting to start streaming tx filter pool: %s; filter id: %s", addr, filterID)

	txFilterConnectionPools[filterID] = &streamingConnPool{
		conns: make(chan net.Conn, streamingTxFilterPoolMaxConnectionCount),
		factory: func() (net.Conn, error) {
			txFilterConn, err := net.Dial("tcp", addr)
			if err != nil {
				Log.Warningf("Failed to establish streaming tx filter pool connection to %s; %s", addr, err.Error())
				return nil, fmt.Errorf("Failed to establish streaming tx filter pool connection to %s; %s", addr, err.Error())
			}
			return txFilterConn, nil
		},
	}
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
