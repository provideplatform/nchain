package main

import (
	"testing"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
)

const natsMsgTimeout = time.Millisecond * 50

func natsGuaranteeDelivery(sub string, t *testing.T) {

	go func() {

		//deliveries = map[string][]*stan.Msg{}
		RunConsumers()

		natsConn := getNatsStreamingConnection()

		// TODO: use a mutex if we need to detect > 1 delivery on sub

		ch := make(chan *stan.Msg, 1)

		// wg := sync.WaitGroup{}
		// wg.Add(1)

		defer natsConn.Close()

		natsSub, err := natsConn.QueueSubscribe(sub, sub, func(msg *stan.Msg) {
			Log.Debugf("subject: " + msg.MsgProto.Subject)
			Log.Debugf("message: " + msg.MsgProto.Reply)
			ch <- msg
			//wg.Done()
		})
		// }, stan.DurableName(sub), stan.DeliverAllAvailable())

		if err != nil {
			Log.Debugf("test failure")
			t.Errorf("Failed to subscribe to NATS subject: %s", sub)
			// wg.Done()
			// return
		}
		defer natsSub.Unsubscribe()
		Log.Debugf("Subscribed to NATS subject: %s", sub)

		// wg.Wait()

		startedAt := time.Now().UnixNano()
		ticker := time.NewTicker(natsMsgTimeout / 5)
		for {
			select {
			case <-ticker.C:
				elapsedMillis := (time.Now().UnixNano() - startedAt) / 1000000
				Log.Debugf("ticker: %d", elapsedMillis)
				if elapsedMillis >= int64(natsMsgTimeout/1000000) {
					ticker.Stop()
					Log.Debugf("test failure")
					t.Errorf("Failed to consume message on NATS subject: %s; timed out after %dms", sub, elapsedMillis)
					// return fmt.Errorf("Failed to consume message on NATS subject: %s; timed out after %dms", sub, elapsedMillis)
				}
			case msg := <-ch:
				Log.Debugf("Guaranteed delivery of NATS message on subject: %s; msg: %s", sub, msg)
				// wg.Done()
				ticker.Stop()
				// return nil
			default:
				// no-op
			}
		}
	}()
}
