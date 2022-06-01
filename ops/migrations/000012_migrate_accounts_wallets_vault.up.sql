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

ALTER TABLE ONLY accounts DROP COLUMN private_key;

ALTER TABLE ONLY accounts ADD COLUMN vault_id uuid NOT NULL;
ALTER TABLE ONLY accounts ADD COLUMN key_id uuid NOT NULL;
CREATE INDEX idx_accounts_vault_id_key_id ON accounts USING btree (vault_id, key_id);

ALTER TABLE ONLY wallets DROP COLUMN mnemonic;
ALTER TABLE ONLY wallets DROP COLUMN private_key;
ALTER TABLE ONLY wallets DROP COLUMN seed;

ALTER TABLE ONLY wallets ADD COLUMN vault_id uuid NOT NULL;
ALTER TABLE ONLY wallets ADD COLUMN key_id uuid NOT NULL;
CREATE INDEX idx_wallets_vault_id_key_id ON wallets USING btree (vault_id, key_id);
