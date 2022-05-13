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

// +build unit

package tx_test

import (
	"github.com/kthomas/go-pgputil"
)

func init() {
	pgputil.RequirePGP()
}

// func TestTransaction_Create(t *testing.T) {
// 	accountID, _ := uuid.FromString("1be0f75c-c05d-42d7-85fd-0c406466a95c")
// 	appID, _ := uuid.FromString("146ab73e-b2eb-4386-8c6f-93663792c741")
// 	networkID, _ := uuid.FromString("66d44f30-9092-4182-a3c4-bc02736d6ae5")
// 	txVal := tx.NewTxValue(int64(200))
// 	tx := &tx.Transaction{
// 		NetworkID:     networkID,
// 		ApplicationID: &appID,
// 		AccountID:     &accountID,
// 		To:            common.StringOrNil("0x0E6081223ACCE2f7f402edE17ED2B0ABDe4E9D0c"),
// 		Value:         txVal,
// 		Data:          common.StringOrNil("0x"),
// 	}
// 	if !tx.Create(dbconf.DatabaseConnection()) {
// 		t.Errorf("tx create() failed; %s", *tx.Errors[0].Message)
// 	}
// }
