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

ALTER TABLE ONLY transactions RENAME COLUMN network_latency TO broadcast_latency;

ALTER TABLE ONLY transactions DROP COLUMN wallet_id;
ALTER TABLE ONLY transactions DROP COLUMN hd_derivation_path;

DROP INDEX idx_transactions_wallet_id;
ALTER TABLE ONLY transactions DROP CONSTRAINT transactions_wallet_id_wallets_id_foreign;
