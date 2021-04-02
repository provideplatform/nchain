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

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE CASCADE;
