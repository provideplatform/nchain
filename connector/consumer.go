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

package connector

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dbconf "github.com/kthomas/go-db-config"
	natsutil "github.com/kthomas/go-natsutil"
	uuid "github.com/kthomas/go.uuid"
	"github.com/nats-io/nats.go"
	"github.com/provideplatform/nchain/common"
)

const defaultNatsStream = "nchain"

const natsConnectorDeprovisioningSubject = "nchain.connector.deprovision"
const natsConnectorDeprovisioningMaxInFlight = 64
const natsConnectorDeprovisioningInvocationTimeout = time.Second * 15
const natsConnectorDeprovisioningMaxDeliveries = 50

const natsConnectorProvisioningSubject = "nchain.connector.provision"
const natsConnectorProvisioningMaxInFlight = 64
const natsConnectorProvisioningInvocationTimeout = time.Second * 15
const natsConnectorProvisioningMaxDeliveries = 50

const natsConnectorResolveReachabilitySubject = "nchain.connector.reachability.resolve"
const natsConnectorResolveReachabilityMaxInFlight = 64
const natsConnectorResolveReachabilityInvocationTimeout = time.Second * 10
const natsConnectorResolveReachabilityMaxDeliveries = 200

const natsConnectorDenormalizeConfigSubject = "nchain.connector.config.denormalize"
const natsConnectorDenormalizeConfigMaxInFlight = 64
const natsConnectorDenormalizeConfigInvocationTimeout = time.Second * 5
const natsConnectorDenormalizeConfigMaxDeliveries = 12

var waitGroup sync.WaitGroup

func init() {
	if !common.ConsumeNATSStreamingSubscriptions {
		common.Log.Debug("Connector package consumer configured to skip NATS streaming subscription setup")
		return
	}

	natsutil.EstablishSharedNatsConnection(nil)
	natsutil.NatsCreateStream(defaultNatsStream, []string{
		fmt.Sprintf("%s.>", defaultNatsStream),
	})

	createNatsConnectorProvisioningSubscriptions(&waitGroup)
	createNatsConnectorResolveReachabilitySubscriptions(&waitGroup)
	createNatsConnectorDeprovisioningSubscriptions(&waitGroup)
	createNatsConnectorDenormalizeConfigSubscriptions(&waitGroup)
}

func createNatsConnectorProvisioningSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsConnectorProvisioningInvocationTimeout,
			natsConnectorProvisioningSubject,
			natsConnectorProvisioningSubject,
			natsConnectorProvisioningSubject,
			consumeConnectorProvisioningMsg,
			natsConnectorProvisioningInvocationTimeout,
			natsConnectorProvisioningMaxInFlight,
			natsConnectorProvisioningMaxDeliveries,
			nil,
		)
	}
}

func createNatsConnectorResolveReachabilitySubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsConnectorResolveReachabilityInvocationTimeout,
			natsConnectorResolveReachabilitySubject,
			natsConnectorResolveReachabilitySubject,
			natsConnectorResolveReachabilitySubject,
			consumeConnectorResolveReachabilityMsg,
			natsConnectorResolveReachabilityInvocationTimeout,
			natsConnectorResolveReachabilityMaxInFlight,
			natsConnectorResolveReachabilityMaxDeliveries,
			nil,
		)
	}
}

func createNatsConnectorDeprovisioningSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsConnectorDeprovisioningInvocationTimeout,
			natsConnectorDeprovisioningSubject,
			natsConnectorDeprovisioningSubject,
			natsConnectorDeprovisioningSubject,
			consumeConnectorDeprovisioningMsg,
			natsConnectorDeprovisioningInvocationTimeout,
			natsConnectorDeprovisioningMaxInFlight,
			natsConnectorDeprovisioningMaxDeliveries,
			nil,
		)
	}
}

func createNatsConnectorDenormalizeConfigSubscriptions(wg *sync.WaitGroup) {
	for i := uint64(0); i < natsutil.GetNatsConsumerConcurrency(); i++ {
		natsutil.RequireNatsJetstreamSubscription(wg,
			natsConnectorDenormalizeConfigInvocationTimeout,
			natsConnectorDenormalizeConfigSubject,
			natsConnectorDenormalizeConfigSubject,
			natsConnectorDenormalizeConfigSubject,
			consumeConnectorDenormalizeConfigMsg,
			natsConnectorDenormalizeConfigInvocationTimeout,
			natsConnectorDenormalizeConfigMaxInFlight,
			natsConnectorDenormalizeConfigMaxDeliveries,
			nil,
		)
	}
}

func consumeConnectorProvisioningMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS connector provisioning message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal connector provisioning message; %s", err.Error())
		msg.Nak()
		return
	}

	connectorID, connectorIDOk := params["connector_id"].(string)
	if !connectorIDOk {
		common.Log.Warningf("failed to provision connector; no connector id provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	connector := &Connector{}
	db.Where("id = ?", connectorID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		common.Log.Warningf("failed to provision connector; no connector resolved for id: %s", connectorID)
		msg.Term()
		return
	}

	err = connector.provision()
	if err != nil {
		common.Log.Warningf("failed to provision connector; %s", err.Error())
		msg.Nak()
	} else {
		common.Log.Debugf("Connector provisioning succeeded; ACKing NATS message for connector: %s", connector.ID)
		msg.Ack()
	}
}

func consumeConnectorDeprovisioningMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS connector deprovisioning message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal connector deprovisioning message; %s", err.Error())
		msg.Nak()
		return
	}

	connectorID, connectorIDOk := params["connector_id"].(string)
	if !connectorIDOk {
		common.Log.Warningf("failed to deprovision connector; no connector id provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	connector := &Connector{}
	db.Where("id = ?", connectorID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		common.Log.Warningf("failed to deprovision connector; no connector resolved for id: %s", connectorID)
		msg.Term()
		return
	}

	err = connector.deprovision()
	if err != nil {
		common.Log.Warningf("failed to deprovision connector; %s", err.Error())
		msg.Nak()
	} else {
		common.Log.Debugf("connector deprovisioning succeeded; ACKing NATS message for connector: %s", connector.ID)
		msg.Ack()
	}
}

func consumeConnectorDenormalizeConfigMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS connector denormalize config message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal connector denormalize config message; %s", err.Error())
		msg.Nak()
		return
	}

	connectorID, connectorIDOk := params["connector_id"].(string)
	if !connectorIDOk {
		common.Log.Warningf("failed to denormalize connector config; no connector id provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	connector := &Connector{}
	db.Where("id = ?", connectorID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		common.Log.Warningf("failed to denormalize connector config; no connector resolved for id: %s", connectorID)
		msg.Nak()
		return
	}

	err = connector.denormalizeConfig()
	if err != nil {
		common.Log.Warningf("failed to denormalize connector config; %s", err.Error())
		msg.Nak()
	} else {
		common.Log.Debugf("Connector config denormalized; ACKing NATS message for connector: %s", connector.ID)
		msg.Ack()
	}
}

func consumeConnectorResolveReachabilityMsg(msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			msg.Term()
		}
	}()

	common.Log.Debugf("consuming NATS connector resolve reachability message: %s", msg)
	var params map[string]interface{}

	err := json.Unmarshal(msg.Data, &params)
	if err != nil {
		common.Log.Warningf("failed to umarshal connector resolve reachability message; %s", err.Error())
		msg.Nak()
		return
	}

	connectorID, connectorIDOk := params["connector_id"].(string)
	if !connectorIDOk {
		common.Log.Warningf("failed to resolve connector reachability; no connector id provided")
		msg.Term()
		return
	}

	db := dbconf.DatabaseConnection()

	connector := &Connector{}
	db.Where("id = ?", connectorID).Find(&connector)
	if connector == nil || connector.ID == uuid.Nil {
		common.Log.Warningf("failed to resolve connector reachability; no connector resolved for id: %s", connectorID)
		msg.Term()
		return
	}

	if connector.reachable() {
		common.Log.Debugf("Connector reachability resolved; ACKing NATS message for connector: %s", connector.ID)
		connector.UpdateStatus(db, "available", nil)
		msg.Ack()
	} else {
		if connector.Status != nil && *connector.Status == "available" {
			connector.UpdateStatus(db, "unavailable", nil)
		}

		common.Log.Debugf("connector is not reachable: %s", connector.ID)
		msg.Nak()
	}
}
