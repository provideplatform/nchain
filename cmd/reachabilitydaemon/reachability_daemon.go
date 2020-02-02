package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	logger "github.com/kthomas/go-logger"
	"github.com/provideapp/goldmine/common"
)

const defaultReachabilityGracePeriod = uint32(2) // this is the number of unreachable attempts prior to marking an endpoint as unreachable

const reachabilityDaemonSleepInterval = 250 * time.Millisecond
const reachabilityDaemonTickerInterval = 15 * time.Second
const reachabilityDaemonReachabilityTimeout = time.Millisecond * 2500

var currentReachabilityDaemons = map[string]*ReachabilityDaemon{}
var currentReachabilityDaemonsMutex = &sync.Mutex{}

// endpoint implements the net.Addr interface
type endpoint struct {
	network string
	addr    string

	gracePeriod uint32 // number of attempts prior to firing endpoint unreachable callback

	reachable   func()
	unreachable func()
}

func (e *endpoint) Network() string {
	return e.network
}

func (e *endpoint) String() string {
	return e.addr
}

// ReachabilityDaemon struct
type ReachabilityDaemon struct {
	addr  net.Addr
	endpt *endpoint

	attempt           uint32
	isReachable       bool
	lastReachableTime *time.Time

	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	log *logger.Logger
}

// EvictReachabilityDaemon evicts a single, previously-initialized reachability daemon instance
func EvictReachabilityDaemon(addr net.Addr) error {
	key := fmt.Sprintf("%s:%s", addr.Network(), addr.String())
	if daemon, ok := currentReachabilityDaemons[key]; ok {
		common.Log.Debugf("Evicting reachability daemon instance: %s", key)
		daemon.shutdown()
		currentReachabilityDaemonsMutex.Lock()
		delete(currentReachabilityDaemons, key)
		currentReachabilityDaemonsMutex.Unlock()
		return nil
	}
	return fmt.Errorf("Unable to evict reachability daemon instance: %s", key)
}

// RequireReachabilityDaemon ensures a single reachability daemon instance is running for
// the given network address; if no reachability daemon instance has been started for the address,
// the instance is configured and started immediately, caching real-time reachability status.
func RequireReachabilityDaemon(endpt *endpoint) *ReachabilityDaemon {
	key := fmt.Sprintf("%s:%s", endpt.Network(), endpt.String())

	var daemon *ReachabilityDaemon
	if daemon, ok := currentReachabilityDaemons[key]; ok {
		common.Log.Debugf("Cached reachability daemon instance found for addr: %s", key)
		return daemon
	}

	currentReachabilityDaemonsMutex.Lock()
	common.Log.Infof("Initializing new reachability daemon instance for addr: %s", key)
	daemon = NewReachabilityDaemon(common.Log, endpt)
	if daemon != nil {
		currentReachabilityDaemons[key] = daemon
		go daemon.run()
	}
	currentReachabilityDaemonsMutex.Unlock()

	return daemon
}

// NewReachabilityDaemon initializes a new reachability daemon instance
func NewReachabilityDaemon(lg *logger.Logger, endpt *endpoint) *ReachabilityDaemon {
	rd := new(ReachabilityDaemon)
	rd.log = lg.Clone()
	rd.shutdownCtx, rd.cancelF = context.WithCancel(context.Background())

	if endpt.gracePeriod == 0 {
		endpt.gracePeriod = defaultReachabilityGracePeriod
	}
	rd.endpt = endpt

	return rd
}

// reachable returns true if the configured reachability daemon endpoint is... reachable
func (rd *ReachabilityDaemon) reachable() bool {
	ntwrk := rd.endpt.Network()
	addr := rd.endpt.String()
	conn, err := net.DialTimeout(ntwrk, addr, reachabilityDaemonReachabilityTimeout)
	if err == nil {
		common.Log.Debugf("%s %s is reachable", ntwrk, addr)
		defer conn.Close()
		return true
	}
	common.Log.Debugf("%s %s is unreachable", ntwrk, addr)
	return false
}

// Run the configured reachability daemon instance
func (rd *ReachabilityDaemon) run() {
	go func() {
		timer := time.NewTicker(reachabilityDaemonTickerInterval)
		defer timer.Stop()

		for !rd.shuttingDown() {
			select {
			case <-timer.C:
				rd.attempt++

				if rd.reachable() {
					common.Log.Debugf("reachability daemon endpoint %s %s is reachable", rd.endpt.Network(), rd.endpt.String())
					rd.attempt = 0
					if !rd.isReachable && rd.endpt.reachable != nil {
						rd.isReachable = true
						lastReachableTime := time.Now()
						rd.lastReachableTime = &lastReachableTime
						rd.endpt.reachable()
					}
				} else if rd.attempt == rd.endpt.gracePeriod+1 {
					common.Log.Warningf("reachability daemon endpoint %s %s is not reachable after %v attempts; grace period: %v attempts", rd.endpt.Network(), rd.endpt.String(), rd.attempt, rd.endpt.gracePeriod)
					if (rd.isReachable || rd.lastReachableTime == nil) && rd.endpt.unreachable != nil {
						rd.isReachable = false
						rd.endpt.unreachable()
					}
				}
			default:
				time.Sleep(reachabilityDaemonSleepInterval)
			}
		}
	}()
}

func (rd *ReachabilityDaemon) handleSignals() {
	common.Log.Debug("Installing SIGINT and SIGTERM signal handlers")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			rd.shutdown()
		case <-rd.shutdownCtx.Done():
			close(sigs)
		}
	}()
}

func (rd *ReachabilityDaemon) shutdown() {
	if atomic.AddUint32(&rd.closing, 1) == 1 {
		common.Log.Debugf("Shutting down reachability daemon instance for %s endpoint %s", rd.endpt.Network(), rd.endpt.String())
		rd.cancelF()
	}
}

func (rd *ReachabilityDaemon) shuttingDown() bool {
	return (atomic.LoadUint32(&rd.closing) > 0)
}
