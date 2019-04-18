package contract

import (
	"encoding/json"
	"sync"
	"time"

	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/provideapp/goldmine/common"
	"github.com/provideapp/goldmine/consumer"
)

const natsContractCompilerInvocationTimeout = time.Minute * 1
const natsContractCompilerInvocationSubject = "goldmine.contract.compiler-invocation"
const natsContractCompilerInvocationMaxInFlight = 32

const natsNetworkContractCreateInvocationTimeout = time.Minute * 1
const natsNetworkContractCreateInvocationSubject = "goldmine.contract.persist"
const natsNetworkContractCreateInvocationMaxInFlight = 32

var waitGroup sync.WaitGroup

func init() {
	natsConnection := consumer.GetNatsStreamingConnection()
	if natsConnection == nil {
		return
	}

	createNatsContractCompilerInvocationSubscriptions(natsConnection, &waitGroup)
	createNatsNetworkContractCreateInvocationSubscriptions(natsConnection, &waitGroup)
}

func createNatsContractCompilerInvocationSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			contractCompilerInvocationSubscription, err := natsConnection.QueueSubscribe(natsContractCompilerInvocationSubject, natsContractCompilerInvocationSubject, consumeContractCompilerInvocationMsg, stan.SetManualAckMode(), stan.AckWait(natsContractCompilerInvocationTimeout), stan.MaxInflight(natsContractCompilerInvocationMaxInFlight), stan.DurableName(natsContractCompilerInvocationSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsContractCompilerInvocationSubject)
				wg.Done()
				return
			}
			defer contractCompilerInvocationSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsContractCompilerInvocationSubject)

			wg.Wait()
		}()
	}
}

func createNatsNetworkContractCreateInvocationSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			networkContractCreateInvocationSubscription, err := natsConnection.QueueSubscribe(natsNetworkContractCreateInvocationSubject, natsNetworkContractCreateInvocationSubject, consumeNetworkContractCreateInvocationMsg, stan.SetManualAckMode(), stan.AckWait(natsNetworkContractCreateInvocationTimeout), stan.MaxInflight(natsNetworkContractCreateInvocationMaxInFlight), stan.DurableName(natsNetworkContractCreateInvocationSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsNetworkContractCreateInvocationSubject)
				wg.Done()
				return
			}
			defer networkContractCreateInvocationSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsNetworkContractCreateInvocationSubject)

			wg.Wait()
		}()
	}
}

func consumeContractCompilerInvocationMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS contract compiler invocation message: %s", msg)
	var contract *Contract

	err := json.Unmarshal(msg.Data, &contract)
	if err != nil {
		common.Log.Warningf("Failed to umarshal contract compiler invocation message; %s", err.Error())
		consumer.Nack(msg)
		return
	}

	_, err = contract.Compile()
	if err != nil {
		common.Log.Warningf("Failed to compile contract; %s", err.Error())
		consumer.Nack(msg)
	} else {
		common.Log.Debugf("Contract compiler invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	}
}

func consumeNetworkContractCreateInvocationMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS network contract creation invocation message: %s", msg)

	var params map[string]interface{}
	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("Failed to umarshal network contract creation invocation invocation message; %s", err.Error())
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
		common.Log.Warningf("Failed to persist network contract with address: %s; %s", addr)
		consumer.Nack(msg)
	}
}
