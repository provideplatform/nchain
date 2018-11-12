package main

import (
	"fmt"
	"net"
	"testing"
)

func TestRunStreamingTxFilterConnectionPools(t *testing.T) {
	if _, filterPoolOk := txFilterConnectionPools["test"]; filterPoolOk {
		Log.Warningf("Attempting to start streaming tx filter pool that has already been allocated; filter id: %s", "test")
		return
	}

	addr := fmt.Sprintf("%s:%v", "spark.provide.services", 7078)
	// Log.Debugf("Attempting to start streaming tx filter pool: %s; filter id: %s", addr, "test")

	connPool := &streamingConnPool{
		conns: make(chan net.Conn, streamingTxFilterPoolMaxConnectionCount),
		factory: func() (net.Conn, error) {
			txFilterConn, err := net.Dial("tcp", addr)
			if err != nil {
				// Log.Warningf("Failed to establish streaming tx filter pool connection to %s; %s", addr, err.Error())
				return nil, fmt.Errorf("Failed to establish streaming tx filter pool connection to %s; %s", addr, err.Error())
			}
			return txFilterConn, nil
		},
	}

	conn, _ := connPool.leaseConnection()
	n, err := conn.Write([]byte("wtf!!!!!!!!!!"))
	fmt.Printf("%d bytes written; err: %s", n, err.Error())
}
