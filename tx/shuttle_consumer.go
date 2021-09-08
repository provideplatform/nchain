package tx

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	dbconf "github.com/kthomas/go-db-config"
	"github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/nats-io/nats.go"
	"github.com/provideplatform/nchain/common"
	"github.com/provideplatform/nchain/contract"
	"github.com/provideplatform/nchain/token"
	provide "github.com/provideplatform/provide-go/api"
)

const natsShuttleCircuitDeployedSubject = "shuttle.circuit.deployed"
const natsShuttleCircuitDeployedMaxInFlight = 1024
const natsShuttleCircuitDeployedInvocationTimeout = time.Second * 30
const natsShuttleCircuitDeployedMaxDeliveries = 30000

const natsShuttleContractDeployedSubject = "shuttle.contract.deployed"
const natsShuttleContractDeployedMaxInFlight = 1024
const natsShuttleContractDeployedInvocationTimeout = time.Second * 30
const natsShuttleContractDeployedMaxDeliveries = 30000

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Tx package shuttle consumer configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsConnection(nil)
	natsutil.NatsCreateStream(defaultNatsStream, []string{
		fmt.Sprintf("%s.>", defaultNatsStream),
	})

	createNatsShuttleCircuitDeployedSubject(&waitGroup)
	createNatsShuttleContractDeployedSubject(&waitGroup)
}

func createNatsShuttleCircuitDeployedSubject(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsShuttleCircuitDeployedInvocationTimeout,
			natsShuttleCircuitDeployedSubject,
			natsShuttleCircuitDeployedSubject,
			consumeShuttleCircuitDeployedMsg,
			natsShuttleCircuitDeployedInvocationTimeout,
			natsShuttleCircuitDeployedMaxInFlight,
			natsShuttleCircuitDeployedMaxDeliveries,
			nil,
		)
	}
}

func createNatsShuttleContractDeployedSubject(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsShuttleContractDeployedInvocationTimeout,
			natsShuttleContractDeployedSubject,
			natsShuttleContractDeployedSubject,
			consumeShuttleContractDeployedMsg,
			natsShuttleContractDeployedInvocationTimeout,
			natsShuttleContractDeployedMaxInFlight,
			natsShuttleContractDeployedMaxDeliveries,
			nil,
		)
	}
}

func consumeShuttleCircuitDeployedMsg(msg *nats.Msg) {
	common.Log.Debugf("Consuming NATS shuttle circuit deployed message: %s", msg)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal shuttle circuit deployed message; %s", err.Error())
		msg.Nak()
		return
	}

	common.Log.Debugf("shuttle.circuit.deployed message is currently a no-op")
	msg.Ack()
}

func consumeShuttleContractDeployedMsg(msg *nats.Msg) {
	common.Log.Debugf("Consuming NATS shuttle contract deployed message: %s", msg)

	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal shuttle contract deployed message; %s", err.Error())
		msg.Nak()
		return
	}

	address, addressOk := params["addr"].(string)
	byAddr, byOk := params["by"].(string)
	networkID, networkIDOk := params["network_id"].(string)
	name, nameOk := params["name"].(string) // name of the dependency
	txHash, txHashOk := params["tx_hash"].(string)
	contractType, _ := params["type"].(string)

	if !addressOk {
		common.Log.Warning("Failed to handle shuttle.contract.deployed message; contract address required")
		msg.Nak()
		return
	}

	if !byOk {
		common.Log.Warning("Failed to handle shuttle.contract.deployed message; by address required")
		msg.Nak()
		return
	}

	if !networkIDOk {
		common.Log.Warning("Failed to handle shuttle.contract.deployed message; contract network_id required")
		msg.Nak()
		return
	}

	if !nameOk {
		common.Log.Warning("Failed to handle shuttle.contract.deployed message; contract name required")
		msg.Nak()
		return
	}

	if !txHashOk {
		common.Log.Warning("Failed to handle shuttle.contract.deployed message; tx hash required")
		msg.Nak()
		return
	}

	cntrct := &contract.Contract{}
	db := dbconf.DatabaseConnection()
	db.Where("network_id = ? AND address = ?", networkID, byAddr).Find(&cntrct)

	if cntrct == nil || cntrct.ID == uuid.Nil {
		common.Log.Warningf("Failed to handle shuttle.contract.deployed message; contract not resolved for address: %s", byAddr)
		msg.Nak()
		return
	}

	network, err := cntrct.GetNetwork()
	if err != nil {
		common.Log.Warningf("Failed to handle shuttle.contract.deployed message; network not resolved for contract with address: %s; %s", byAddr, err.Error())
		msg.Nak()
		return
	}

	if network.IsEthereumNetwork() {
		address = ethcommon.HexToAddress(address).Hex()
		byAddr = ethcommon.HexToAddress(byAddr).Hex()
	}

	p2pAPI, err := network.P2PAPIClient()
	if err != nil {
		common.Log.Warningf("Failed to handle shuttle.contract.deployed message; network P2P API client not resolved for contract with address: %s; %s", byAddr, err.Error())
		msg.Nak()
		return
	}

	receipt, err := p2pAPI.FetchTxReceipt(*cntrct.Address, txHash)
	if err != nil {
		common.Log.Warningf("Failed to handle shuttle.contract.deployed message; failed to fetch tx receipt for contract with address: %s; %s", byAddr, err.Error())
		msg.Nak()
		return
	}

	dependency := cntrct.ResolveCompiledDependencyArtifact(name)
	if dependency == nil {
		common.Log.Warningf("Failed to handle shuttle.contract.deployed message; contract at address %s unable to resolved dependency: %s", byAddr, name)
		msg.Nak()
		return
	}

	contractParams, _ := json.Marshal(map[string]interface{}{
		"compiled_artifact": dependency,
	})

	rawParams := json.RawMessage(contractParams)

	internalContract := &contract.Contract{
		ApplicationID: cntrct.ApplicationID,
		NetworkID:     cntrct.NetworkID,
		ContractID:    &cntrct.ID,
		Name:          common.StringOrNil(name),
		Address:       common.StringOrNil(address),
		Params:        &rawParams,
		Type:          common.StringOrNil(contractType),
	}

	if internalContract.Create() {
		common.Log.Debugf("created contract %s for %s shuttle.contract.deployed event", internalContract.ID, *network.Name)

		internalContract.ResolveTokenContract(db, network, *cntrct.Address, receipt,
			func(c *contract.Contract, tokenType, name string, decimals *big.Int, symbol string) (createdToken bool, tokenID uuid.UUID, errs []*provide.Error) {
				common.Log.Debugf("resolved %s token: %s (%v decimals); symbol: %s", *network.Name, name, decimals, symbol)

				tok := &token.Token{
					ApplicationID: c.ApplicationID,
					NetworkID:     c.NetworkID,
					ContractID:    &c.ID,
					Type:          common.StringOrNil(tokenType),
					Name:          common.StringOrNil(name),
					Symbol:        common.StringOrNil(symbol),
					Decimals:      decimals.Uint64(),
					Address:       common.StringOrNil(string(receipt.ContractAddress)),
				}

				createdToken = tok.Create()
				tokenID = tok.ID
				errs = tok.Errors

				return createdToken, tokenID, errs
			})
	} else {
		common.Log.Warningf("failed to create contract for %s shuttle.contract.deployed", *network.Name)
	}

	msg.Ack()
}
