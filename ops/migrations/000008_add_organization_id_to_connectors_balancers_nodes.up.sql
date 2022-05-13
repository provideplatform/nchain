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

ALTER TABLE ONLY connectors ADD COLUMN organization_id uuid;
CREATE INDEX idx_connectors_organization_id ON connectors USING btree (organization_id);

ALTER TABLE ONLY load_balancers ADD COLUMN organization_id uuid;
CREATE INDEX idx_load_balancers_organization_id ON load_balancers USING btree (organization_id);

ALTER TABLE ONLY nodes ADD COLUMN organization_id uuid;
CREATE INDEX idx_nodes_organization_id ON nodes USING btree (organization_id);
