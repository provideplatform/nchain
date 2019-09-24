package contract

import (
	"encoding/json"
	"sync"
	"time"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/stan.go"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/consumer"
)

const natsNetworkContractCreateInvocationTimeout = time.Minute * 1
const natsNetworkContractCreateInvocationSubject = "goldmine.contract.create"
const natsNetworkContractCreateInvocationMaxInFlight = 32

var waitGroup sync.WaitGroup

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Contract package consumer configured to skip NATS streaming subscription setup")
		return
	}

	createNatsNetworkContractCreateInvocationSubscriptions(&waitGroup)
}

func createNatsNetworkContractCreateInvocationSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsStreamingSubscription(wg,
			natsNetworkContractCreateInvocationTimeout,
			natsNetworkContractCreateInvocationSubject,
			natsNetworkContractCreateInvocationSubject,
			consumeNetworkContractCreateInvocationMsg,
			natsNetworkContractCreateInvocationTimeout,
			natsNetworkContractCreateInvocationMaxInFlight,
		)
	}
}

func consumeNetworkContractCreateInvocationMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS network contract creation invocation message: %s", msg)

	var params map[string]interface{}
	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal network contract creation invocation message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	addr, addrOk := params["address"].(string)
	networkID, networkIDOk := params["network_id"].(string)
	networkUUID, networkUUIDErr := uuid.FromString(networkID)
	contractName, contractNameOk := params["name"].(string)
	_, abiOk := params["abi"].([]interface{})

	if !addrOk {
		common.Log.Warningf("Failed to create network contract; no contract address provided")
		consumer.Nack(msg)
		return
	}
	if !networkIDOk || networkUUIDErr != nil {
		common.Log.Warningf("Failed to create network contract; invalid or no network id provided")
		consumer.Nack(msg)
		return
	}
	if !contractNameOk {
		common.Log.Warningf("Failed to create network contract; no contract name provided")
		consumer.Nack(msg)
		return
	}
	if !abiOk {
		common.Log.Warningf("Failed to create network contract; no ABI provided")
		consumer.Nack(msg)
		return
	}
	contract := &Contract{
		ApplicationID: nil,
		NetworkID:     networkUUID,
		TransactionID: nil,
		Name:          common.StringOrNil(contractName),
		Address:       common.StringOrNil(addr),
		Params:        nil,
	}
	contract.setParams(params)

	if contract.Create() {
		common.Log.Debugf("Network contract creation invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	} else {
		common.Log.Warningf("Failed to persist network contract with address: %s", addr)
		consumer.Nack(msg)
	}
}
