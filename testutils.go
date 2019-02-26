package main

import (
	"fmt"
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
)

// type promise interface {
// 	resolve(msg *nats.Msg) (interface{}, error)
// }

const natsMsgTimeout = time.Millisecond * 500

func natsGuaranteeDelivery(sub string) error {
	RunConsumers()

	natsConn := getNatsStreamingConnection()

	// TODO: use a mutex if we need to detect > 1 delivery on sub
	delivered := false

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer natsConn.Close()

		natsSub, err := natsConn.QueueSubscribe(sub, sub, func(msg *stan.Msg) {
			// _msg = msg
			Log.Debugf("GOT MESSAGE:::::%s", msg)
			delivered = true
		}, stan.DurableName(sub))

		if err != nil {
			Log.Warningf("Failed to subscribe to NATS subject: %s", sub)
			wg.Done()
			return
		}
		defer natsSub.Unsubscribe()
		Log.Debugf("Subscribed to NATS subject: %s", sub)

		wg.Wait()
	}()

	timer := time.NewTimer(natsMsgTimeout)
	select {
	case <-timer.C:
		Log.Debugf("Failed to guarantee delivery of NATS message on subject: %s", sub)
		break
	default:
		if delivered {
			Log.Debugf("Guaranteed delivery of NATS message on subject: %s", sub)
			break
		}

		Log.Debugf("Attempting to guarantee delivery of NATS message on subject: %s", sub)
	}

	if !delivered {
		return fmt.Errorf("Failed to consume message on NATS subject: %s; timed out after %dms", sub, natsMsgTimeout)
	}

	return nil
}
