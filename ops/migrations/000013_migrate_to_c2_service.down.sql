CREATE TABLE load_balancers (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    network_id uuid NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    host text,
    ipv4 text,
    ipv6 text,
    description text,
    config json,
    region text,
    status text DEFAULT 'provisioning'::text NOT NULL,
    encrypted_config bytea,
    application_id uuid
);

ALTER TABLE ONLY load_balancers
    ADD CONSTRAINT load_balancers_pkey PRIMARY KEY (id);

CREATE INDEX idx_load_balancers_application_id ON load_balancers USING btree (application_id);
CREATE INDEX idx_load_balancers_network_id ON load_balancers USING btree (network_id);
CREATE INDEX idx_load_balancers_region ON load_balancers USING btree (region);
CREATE INDEX idx_load_balancers_status ON load_balancers USING btree (status);
CREATE INDEX idx_load_balancers_type ON load_balancers USING btree (type);
CREATE INDEX idx_network_nodes_type ON load_balancers USING btree (type);

CREATE TABLE connectors_load_balancers (
    connector_id uuid DEFAULT uuid_generate_v4() NOT NULL,
    load_balancer_id uuid DEFAULT uuid_generate_v4() NOT NULL
);

ALTER TABLE ONLY connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_pkey PRIMARY KEY (connector_id, load_balancer_id);

ALTER TABLE ONLY connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE ONLY connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_load_balancer_id_load_balancers_id_fo FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;

CREATE TABLE public.load_balancers_nodes (
    load_balancer_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    node_id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);

ALTER TABLE ONLY load_balancers_nodes
    ADD CONSTRAINT load_balancers_nodes_pkey PRIMARY KEY (load_balancer_id, node_id);

ALTER TABLE ONLY public.load_balancers_nodes
    ADD CONSTRAINT load_balancers_nodes_pkey PRIMARY KEY (load_balancer_id, node_id);

ALTER TABLE ONLY nodes ADD COLUMN host text;
ALTER TABLE ONLY nodes ADD COLUMN ipv4 text;
ALTER TABLE ONLY nodes ADD COLUMN ipv6 text;
ALTER TABLE ONLY nodes ADD COLUMN private_ipv4 text;
ALTER TABLE ONLY nodes ADD COLUMN private_ipv6 text;
ALTER TABLE ONLY nodes ADD COLUMN description text;
ALTER TABLE ONLY nodes ADD COLUMN status text;
ALTER TABLE ONLY nodes ADD COLUMN config json;
ALTER TABLE ONLY nodes ADD COLUMN encrypted_config bytea;
