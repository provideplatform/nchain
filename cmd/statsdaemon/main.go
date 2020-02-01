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
	networks      []*network.Network
)

func init() {
	pgputil.RequirePGP()
	redisutil.RequireRedis()
}

func main() {
	common.Log.Debug("Installing signal handlers for daemon")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	shutdownCtx, cancelF = context.WithCancel(context.Background())

	requireConnectorReachabilityDaemonInstances()
	requireLoadBalancerReachabilityDaemonInstances()
	requireNetworkStatsDaemonInstances()

	common.Log.Debugf("Running daemon main()")
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

	common.Log.Debug("Exiting daemon main()")
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
			cfg := connector.ParseConfig()
			if port, portOk := cfg["port"].(float64); portOk {
				RequireReachabilityDaemon(&endpoint{
					network: "tcp",
					addr:    fmt.Sprintf("%s:%v", *host, port),
				})
			}
			if apiPort, apiPortOk := cfg["api_port"].(float64); apiPortOk {
				RequireReachabilityDaemon(&endpoint{
					network: "tcp",
					addr:    fmt.Sprintf("%s:%v", *host, apiPort),
				})
			}
		}
	}
}

func requireLoadBalancerReachabilityDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	loadBalancers = make([]*network.LoadBalancer, 0)
	dbconf.DatabaseConnection().Find(&loadBalancers)

	for _, lb := range loadBalancers {
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
									network: "tcp",
									addr:    fmt.Sprintf("%s:%v", *lb.Host, tcpPorts[i]),
								})
							}
						}

						if udpPorts, udpPortsOk := ingressCfg[cidr].(map[string]interface{})["udp"].([]interface{}); udpPortsOk {
							for i := range udpPorts {
								RequireReachabilityDaemon(&endpoint{
									network: "udp",
									addr:    fmt.Sprintf("%s:%v", *lb.Host, udpPorts[i]),
								})
							}
						}
					}
				}
			}
		}
	}
}

func requireNetworkStatsDaemonInstances() {
	mutex.Lock()
	defer mutex.Unlock()

	networks = make([]*network.Network, 0)
	dbconf.DatabaseConnection().Where("user_id IS NULL AND enabled IS TRUE").Find(&networks)

	for _, ntwrk := range networks {
		RequireNetworkStatsDaemon(ntwrk)
	}
}

func shutdown() {
	if atomic.AddUint32(&closing, 1) == 1 {
		common.Log.Debug("Shutting down daemon")
		cancelF()
	}
}

func shuttingDown() bool {
	return (atomic.LoadUint32(&closing) > 0)
}
