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

ALTER TABLE ONLY accounts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY bridges ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY connectors ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY contracts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY filters ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY load_balancers ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY networks ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY nodes ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY oracles ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY tokens ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY transactions ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY wallets ALTER COLUMN created_at SET DEFAULT now();
