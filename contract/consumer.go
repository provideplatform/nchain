package contract

import (
	"encoding/json"
	"time"

	natsutil "github.com/kthomas/go-natsutil"
	"github.com/nats-io/go-nats-streaming"

	"github.com/provideapp/goldmine/common"
)

const natsContractCompilerInvocationTimeout = time.Minute * 1
const natsContractCompilerInvocationSubject = "goldmine.contract.compiler-invocation"
const natsContractCompilerInvocationMaxInFlight = 32

func createNatsContractCompilerInvocationSubscriptions(natsConnection stan.Conn) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		waitGroup.Add(1)
		go func() {
			defer natsConnection.Close()

			contractCompilerInvocationSubscription, err := natsConnection.QueueSubscribe(natsContractCompilerInvocationSubject, natsContractCompilerInvocationSubject, consumeContractCompilerInvocationMsg, stan.SetManualAckMode(), stan.AckWait(natsContractCompilerInvocationTimeout), stan.MaxInflight(natsContractCompilerInvocationMaxInFlight), stan.DurableName(natsContractCompilerInvocationSubject))
			if err != nil {
				common.Log.Warningf("Failed to subscribe to NATS subject: %s", natsContractCompilerInvocationSubject)
				waitGroup.Done()
				return
			}
			defer contractCompilerInvocationSubscription.Unsubscribe()
			common.Log.Debugf("Subscribed to NATS subject: %s", natsContractCompilerInvocationSubject)

			waitGroup.Wait()
		}()
	}
}

func consumeContractCompilerInvocationMsg(msg *stan.Msg) {
	common.Log.Debugf("Consuming NATS contract compiler invocation message: %s", msg)
	var contract *Contract

	err := json.Unmarshal(msg.Data, &contract)
	if err != nil {
		common.Log.Warningf("Failed to umarshal contract compiler invocation message; %s", err.Error())
		return
	}

	_, err = contract.Compile()
	if err != nil {
		common.Log.Warningf("Failed to compile contract; %s", err.Error())
	} else {
		common.Log.Debugf("Contract compiler invocation succeeded; ACKing NATS message for contract: %s", contract.ID)
		msg.Ack()
	}
}
