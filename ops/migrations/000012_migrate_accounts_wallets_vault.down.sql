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

DROP INDEX idx_accounts_vault_id_key_id;
ALTER TABLE ONLY accounts DROP COLUMN vault_id;
ALTER TABLE ONLY accounts ADD COLUMN private_key bytea;

DROP INDEX idx_wallets_vault_id_key_id;
ALTER TABLE ONLY wallets DROP COLUMN vault_id;
ALTER TABLE ONLY wallets DROP COLUMN key_id;
ALTER TABLE ONLY wallets ADD COLUMN mnemonic bytea;
ALTER TABLE ONLY wallets ADD COLUMN private_key bytea;
ALTER TABLE ONLY wallets ADD COLUMN seed bytea;
