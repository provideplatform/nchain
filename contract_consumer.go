package main

import (
	"encoding/json"
	"sync"
	"time"

	natsutil "github.com/kthomas/go-natsutil"
	"github.com/nats-io/go-nats-streaming"
)

const natsContractCompilerInvocationTimeout = time.Minute * 1
const natsContractCompilerInvocationSubject = "goldmine.contract.compiler-invocation"
const natsContractCompilerInvocationMaxInFlight = 32

func createNatsContractCompilerInvocationSubscriptions(natsConnection stan.Conn, wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer natsConnection.Close()

			contractCompilerInvocationSubscription, err := natsConnection.QueueSubscribe(natsContractCompilerInvocationSubject, natsContractCompilerInvocationSubject, consumeContractCompilerInvocationMsg, stan.SetManualAckMode(), stan.AckWait(natsContractCompilerInvocationTimeout), stan.MaxInflight(natsContractCompilerInvocationMaxInFlight), stan.DurableName(natsContractCompilerInvocationSubject))
			if err != nil {
				Log.Warningf("Failed to subscribe to NATS subject: %s", natsContractCompilerInvocationSubject)
				wg.Done()
				return
			}
			defer contractCompilerInvocationSubscription.Unsubscribe()
			Log.Debugf("Subscribed to NATS subject: %s", natsContractCompilerInvocationSubject)

			wg.Wait()
		}()
	}
}

func consumeContractCompilerInvocationMsg(msg *stan.Msg) {
	Log.Debugf("Consuming NATS contract compiler invocation message: %s", msg)
	var contract *Contract

	err := json.Unmarshal(msg.Data, &contract)
	if err != nil {
		Log.Warningf("Failed to umarshal contract compiler invocation message; %s", err.Error())
		return
	}

	_, err = contract.Compile()
	if err != nil {
		Log.Warningf("Failed to compile contract; %s", err.Error())
	} else {
		Log.Debugf("Contract compiler invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	}
}
