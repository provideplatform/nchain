package main

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func Test_streamingConnPool_leaseConnection(t *testing.T) {
	streamingTxFilterPoolMaxConnectionCount = 1

	addr := fmt.Sprintf("%s:%d", "spark.provide.services", uint64(7078))
	// Log.Debugf("Attempting to start streaming tx filter pool: %s; filter id: %s", addr, "id")

	pool := &streamingConnPool{
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
	c, err := pool.leaseConnection(time.Millisecond * 50)
	fmt.Printf("pool size: %d", pool.Size())
	if err != nil {
		fmt.Printf(err.Error())
	}
	params := "{\"wallet_id\": \"e6851e51-1ff9-4a9d-b2ea-e739ca29e169\", \"method\": \"send\",\"params\": [\"<insert a hash unique to the accountholder>|<insert a hash that is unique to the ATM>|<insert 10-digit phone number>|<insert first and last name>|<insert tx amount or 0>|<insert latitude as float>|<insert longitude as float>\"], \"value\": 0}"
	c.Write([]byte(params))
	c.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
	resp := []byte{}
	size, err := c.Read(resp)
	if err != nil {
		fmt.Printf(err.Error())
	}
	// c2, err := pool.leaseConnection(time.Millisecond * 50)
	// fmt.Printf("%s err? %s", c2, err)
	fmt.Printf("%d bytes read...: %s", size, resp)
}
