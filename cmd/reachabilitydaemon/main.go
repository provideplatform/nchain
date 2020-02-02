package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	pgputil "github.com/kthomas/go-pgputil"
	redisutil "github.com/kthomas/go-redisutil"

	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/connector"
	_ "github.com/provideapp/goldmine/connector"
	_ "github.com/provideapp/goldmine/contract"
	"github.com/provideapp/goldmine/network"
	_ "github.com/provideapp/goldmine/tx"
)

const runloopTickerInterval = 5 * time.Second
const runloopSleepInterval = 250 * time.Millisecond

var (
	cancelF     context.CancelFunc
	closing     uint32
	shutdownCtx context.Context

	mutex sync.Mutex

	connectors    []*connector.Connector
	loadBalancers []*network.LoadBalancer
)

func init() {
	pgputil.RequirePGP()
	redisutil.RequireRedis()
}

func main() {
	common.Log.Debug("Installing signal handlers for reachabilitydaemon")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

	requireConnectorReachabilityDaemonInstances()
	requireLoadBalancerReachabilityDaemonInstances()

	common.Log.Debugf("Running reachabilitydaemon main()")
	timer := time.NewTicker(runloopTickerInterval)
	defer timer.Stop()

	for !shuttingDown() {
		select {
		case <-timer.C:
			// TODO: check reachability and statsdaemon statuses
		case sig := <-sigs:
			common.Log.Infof("Received signal: %s", sig)
			shutdown()
		case <-shutdownCtx.Done():
			close(sigs)
		default:
			time.Sleep(runloopSleepInterval)
		}
	}

	common.Log.Debug("Exiting reachabilitydaemon main()")
	cancelF()
}

func requireConnectorReachabilityDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	db := dbconf.DatabaseConnection()

	connectors = make([]*connector.Connector, 0)
	db.Find(&connectors)

	for _, connector := range connectors {
		host := connector.Host(db)
		if host != nil {
			reachableFn := func() {
				connector.UpdateStatus(db, "available", nil)
			}
			unreachableFn := func() {
				connector.UpdateStatus(db, "unavailable", nil)
			}

			cfg := connector.ParseConfig()
			if port, portOk := cfg["port"].(float64); portOk {
				RequireReachabilityDaemon(&endpoint{
					network:     "tcp",
					addr:        fmt.Sprintf("%s:%v", *host, port),
					reachable:   reachableFn,
					unreachable: unreachableFn,
				})
			}
			if apiPort, apiPortOk := cfg["api_port"].(float64); apiPortOk {
				RequireReachabilityDaemon(&endpoint{
					network:     "tcp",
					addr:        fmt.Sprintf("%s:%v", *host, apiPort),
					reachable:   reachableFn,
					unreachable: unreachableFn,
				})
			}
		}
	}
}

func requireLoadBalancerReachabilityDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	db := dbconf.DatabaseConnection()

	loadBalancers = make([]*network.LoadBalancer, 0)
	db.Find(&loadBalancers)

	for _, lb := range loadBalancers {
		if lb.Host != nil {
			reachableFn := func() {
				lb.UpdateStatus(db, "active", nil)
			}
			unreachableFn := func() {
				lb.UpdateStatus(db, "unreachable", nil)
			}

			cfg := lb.ParseConfig()
			if security, securityOk := cfg["security"].(map[string]interface{}); securityOk {
				if ingress, ingressOk := security["ingress"]; ingressOk {
					switch ingress.(type) {
					case map[string]interface{}:
						ingressCfg := ingress.(map[string]interface{})
						for cidr := range ingressCfg {
							if tcpPorts, tcpPortsOk := ingressCfg[cidr].(map[string]interface{})["tcp"].([]interface{}); tcpPortsOk {
								for i := range tcpPorts {
									RequireReachabilityDaemon(&endpoint{
										network:     "tcp",
										addr:        fmt.Sprintf("%s:%v", *lb.Host, tcpPorts[i]),
										reachable:   reachableFn,
										unreachable: unreachableFn,
									})
								}
							}

							if udpPorts, udpPortsOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpPortsOk {
								for i := range udpPorts {
									RequireReachabilityDaemon(&endpoint{
										network:     "udp",
										addr:        fmt.Sprintf("%s:%v", *lb.Host, udpPorts[i]),
										reachable:   reachableFn,
										unreachable: unreachableFn,
									})
								}
							}
						}
					}
				}
			}
		}
	}
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("Shutting down reachabilitydaemon")
		cancelF()
	}
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}
