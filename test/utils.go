package test

import (
	"time"

	stan "github.com/nats-io/stan.go"
	"github.com/provideapp/goldmine/common"
)

const natsMsgTimeout = time.Millisecond * 50

// pollingToStrChFuncType is the type representing functions which use chFunc to send data to the channel `ch` during 100ms by default.
// timeout can be passed as 4th param, polling interval as 5th one.
type pollingToStrChFuncType func(ch chan string, chFunc func(ch chan string) error, endingMsg *string, durations ...time.Duration) error

// PollingToStrChFunc is the function helping tests to get info from goroutines
var PollingToStrChFunc pollingToStrChFuncType = func(
	ch chan string,
	chFunc func(ch chan string) error,
	endingMsg *string,
	durations ...time.Duration) error {

	startedAt := time.Now().UnixNano()
	interval := (time.Millisecond * 10)
	timeout := (time.Millisecond * 350)
	// fmt.Printf("durations len: %v\n", len(durations))
	if len(durations) > 0 {
		timeout = durations[0]
		if len(durations) > 1 {
			interval = durations[1]
		}
	}
	// fmt.Printf("polling timeout: %v\n", timeout)
	ticker := time.NewTicker(interval)
	timer := time.NewTimer(timeout)

	go func() error {
		for {
			select {
			case <-ticker.C:
				elapsedMillis := (time.Now().UnixNano() - startedAt) / 1000000
				// fmt.Printf("time elapsed: %v\n", elapsedMillis)

				if elapsedMillis >= int64(timeout) {
					ticker.Stop()
				}

				chFunc(ch)
				// dont return!
			case <-timer.C:
				//ch <- *endingMsg
				ch <- "timeout"
				return nil
			default:
				// no-op
			}
		}
	}()
	return nil
}

func NatsGuaranteeDelivery(sub string) {

	go func() {

		//deliveries = map[string][]*stan.Msg{}
		// RunConsumers()

		natsConn, _ := natsutil.GetSharedNatsStreamingConnection()

		// TODO: use a mutex if we need to detect > 1 delivery on sub

		ch := make(chan *stan.Msg, 1)

		// wg := sync.WaitGroup{}
		// wg.Add(1)

		defer (*natsConn).Close()

		natsSub, err := (*natsConn).QueueSubscribe(sub, sub, func(msg *stan.Msg) {
			common.Log.Debugf("subject: " + msg.MsgProto.Subject)
			common.Log.Debugf("message: " + msg.MsgProto.Reply)
			ch <- msg
			//wg.Done()
		})
		// }, stan.DurableName(sub), stan.DeliverAllAvailable())

		if err != nil {
			common.Log.Debugf("test failure")
			common.Log.Errorf("Failed to subscribe to NATS subject: %s", sub)
			// wg.Done()
			// return
		}
		defer natsSub.Unsubscribe()
		common.Log.Debugf("Subscribed to NATS subject: %s", sub)

		// wg.Wait()

		startedAt := time.Now().UnixNano()
		ticker := time.NewTicker(natsMsgTimeout / 5)
		for {
			select {
			case <-ticker.C:
				elapsedMillis := (time.Now().UnixNano() - startedAt) / 1000000
				if elapsedMillis >= int64(natsMsgTimeout/1000000) {
					ticker.Stop()
					common.Log.Errorf("Failed to consume message on NATS subject: %s; timed out after %dms", sub, elapsedMillis)
				}
			case msg := <-ch:
				common.Log.Debugf("Guaranteed delivery of NATS message on subject: %s; msg: %s", sub, msg)
				// wg.Done()
				ticker.Stop()
				// return nil
			default:
				// no-op
			}
		}
	}()
}
