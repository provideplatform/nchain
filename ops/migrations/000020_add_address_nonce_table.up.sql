CREATE TABLE public.address_nonces (
	id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
	address string NOT NULL,
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	pending_nonce int4 NULL,
	broadcast_nonce int4 NULL,
);

ALTER TABLE public.address_nonces OWNER TO current_user;

ALTER TABLE ONLY public.address_nonces
    ADD CONSTRAINT address_nonces_pkey PRIMARY KEY (id);

CREATE INDEX idx_address_nonces_address ON public.address_nonces USING btree (address);
