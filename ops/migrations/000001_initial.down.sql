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

--
-- FIXME: this probably won't work due to schema constraints. Change the order of these drop statements.
--

DROP TABLE accounts;
DROP TABLE bridges;
DROP TABLE connectors;
DROP TABLE connectors_load_balancers;
DROP TABLE connectors_nodes;
DROP TABLE contracts;
DROP TABLE filters;
DROP TABLE load_balancers;
DROP TABLE load_balancers_nodes;
DROP TABLE networks;
DROP TABLE nodes;
DROP TABLE oracles;
DROP TABLE tokens;
DROP TABLE transactions;
DROP TABLE wallets;
DROP TABLE wallets_accounts;
