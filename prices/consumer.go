/*
 * Copyright 2017-2022 Provide Technologies Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package prices

// exchangeConsumer "github.com/kthomas/exchange-consumer"

// // RunExchangeConsumer runs a real-time consumer of pricing data for the given currency pair
// func RunExchangeConsumer(currencyPair string, wg *sync.WaitGroup) {
// 	wg.Add(1)
// 	go func() {
// 		consumer := exchangeConsumer.GdaxMessageConsumerFactory(common.Log, priceTick, currencyPair)
// 		err := consumer.Run()
// 		if err != nil {
// 			common.Log.Warningf("Consumer exited unexpectedly; %s", err)
// 		} else {
// 			common.Log.Debug("Exiting exchange consumer...")
// 		}
// 	}()
// }

// func priceTick(msg *exchangeConsumer.GdaxMessage) error {
// 	if msg.Type == "match" && msg.Price != "" {
// 		price, err := strconv.ParseFloat(msg.Price, 64)
// 		if err == nil {
// 			SetPrice(msg.ProductId, msg.Sequence, price)
// 		}
// 	} else {
// 		common.Log.Debugf("Dropping GDAX message; seq: %d", msg.Sequence)
// 	}
// 	return nil
// }
