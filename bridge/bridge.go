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

package bridge

import (
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideplatform/provide-go/api"
)

// Bridge instances are still in the process of being defined.
type Bridge struct {
	provide.Model
	ApplicationID *uuid.UUID `sql:"type:uuid" json:"application_id"`
	NetworkID     uuid.UUID  `sql:"not null;type:uuid" json:"network_id"`
}
