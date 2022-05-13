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

CREATE TABLE public.blocks (
	id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
	network_id uuid NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	block int8 NULL,
	transaction_hash text NULL
);

ALTER TABLE public.accounts OWNER TO current_user;

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (id);

CREATE INDEX idx_blocks_network_id ON public.blocks USING btree (network_id);

CREATE INDEX idx_blocks_block ON public.blocks USING btree (block);

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE CASCADE;
