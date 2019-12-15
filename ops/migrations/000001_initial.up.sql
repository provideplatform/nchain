--
-- PostgreSQL database dump
--

-- Dumped from database version 10.6
-- Dumped by pg_dump version 10.11 (Ubuntu 10.11-1.pgdg16.04+1)

-- The following portion of the pg_dump output should not run during migrations:
-- SET statement_timeout = 0;
-- SET lock_timeout = 0;
-- SET idle_in_transaction_session_timeout = 0;
-- SET client_encoding = 'UTF8';
-- SET standard_conforming_strings = on;
-- SELECT pg_catalog.set_config('search_path', '', false);
-- SET check_function_bodies = false;
-- SET xmloption = content;
-- SET client_min_messages = warning;
-- SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner:
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner:
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner:
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';

ALTER USER current_user WITH NOSUPERUSER;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: accounts; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.accounts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    user_id uuid,
    network_id uuid NOT NULL,
    address text NOT NULL,
    private_key bytea NOT NULL,
    accessed_at timestamp with time zone,
    wallet_id uuid,
    hd_derivation_path text,
    public_key bytea
);


ALTER TABLE public.accounts OWNER TO current_user;

--
-- Name: bridges; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.bridges (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL
);


ALTER TABLE public.bridges OWNER TO current_user;

--
-- Name: connectors; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.connectors (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL,
    name text NOT NULL,
    type text NOT NULL,
    config json,
    accessed_at timestamp with time zone,
    encrypted_config bytea,
    status text DEFAULT 'init'::text NOT NULL,
    description text
);


ALTER TABLE public.connectors OWNER TO current_user;

--
-- Name: connectors_load_balancers; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.connectors_load_balancers (
    connector_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    load_balancer_id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);


ALTER TABLE public.connectors_load_balancers OWNER TO current_user;

--
-- Name: connectors_nodes; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.connectors_nodes (
    connector_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    node_id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);


ALTER TABLE public.connectors_nodes OWNER TO current_user;

--
-- Name: contracts; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.contracts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL,
    transaction_id uuid,
    name text NOT NULL,
    address text NOT NULL,
    params json,
    accessed_at timestamp with time zone,
    contract_id uuid
);


ALTER TABLE public.contracts OWNER TO current_user;

--
-- Name: filters; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.filters (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL,
    name text NOT NULL,
    priority integer DEFAULT 0 NOT NULL,
    lang text NOT NULL,
    source text NOT NULL,
    params json
);


ALTER TABLE public.filters OWNER TO current_user;

--
-- Name: load_balancers; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.load_balancers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
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


ALTER TABLE public.load_balancers OWNER TO current_user;

--
-- Name: load_balancers_nodes; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.load_balancers_nodes (
    load_balancer_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    node_id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);


ALTER TABLE public.load_balancers_nodes OWNER TO current_user;

--
-- Name: networks; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.networks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    name text NOT NULL,
    description text,
    is_production boolean NOT NULL,
    sidechain_id uuid,
    config json,
    application_id uuid,
    enabled boolean NOT NULL,
    cloneable boolean NOT NULL,
    network_id uuid,
    user_id uuid,
    chain_id text NOT NULL
);


ALTER TABLE public.networks OWNER TO current_user;

--
-- Name: nodes; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.nodes (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone,
    network_id uuid,
    user_id uuid,
    host text,
    description text,
    config json,
    status text,
    bootnode boolean,
    role text,
    ipv4 text,
    ipv6 text,
    private_ipv4 text,
    private_ipv6 text,
    application_id uuid,
    encrypted_config bytea
);


ALTER TABLE public.nodes OWNER TO current_user;

--
-- Name: oracles; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.oracles (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL,
    contract_id uuid NOT NULL,
    name text NOT NULL,
    params json,
    attachment_ids uuid[]
);


ALTER TABLE public.oracles OWNER TO current_user;

--
-- Name: tokens; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.tokens (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    network_id uuid NOT NULL,
    contract_id uuid,
    sale_contract_id uuid,
    name text NOT NULL,
    symbol text NOT NULL,
    decimals bigint NOT NULL,
    address text NOT NULL,
    sale_address text,
    accessed_at timestamp with time zone
);


ALTER TABLE public.tokens OWNER TO current_user;

--
-- Name: transactions; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.transactions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone NOT NULL,
    application_id uuid,
    user_id uuid,
    network_id uuid NOT NULL,
    account_id uuid,
    "to" text,
    value text DEFAULT 0 NOT NULL,
    data text,
    hash text,
    status text DEFAULT 'pending'::text NOT NULL,
    ref text,
    description text,
    finalized_at timestamp with time zone,
    block bigint,
    broadcast_at timestamp with time zone,
    published_at timestamp with time zone,
    queue_latency bigint,
    broadcast_latency bigint,
    e2_e_latency bigint,
    block_timestamp timestamp with time zone
);


ALTER TABLE public.transactions OWNER TO current_user;

--
-- Name: wallets; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.wallets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    wallet_id uuid,
    application_id uuid,
    user_id uuid,
    purpose integer DEFAULT 44 NOT NULL,
    mnemonic bytea NOT NULL,
    seed bytea NOT NULL,
    public_key bytea NOT NULL,
    private_key bytea NOT NULL
);


ALTER TABLE public.wallets OWNER TO current_user;

--
-- Name: wallets_accounts; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.wallets_accounts (
    wallet_id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid DEFAULT public.uuid_generate_v4() NOT NULL
);


ALTER TABLE public.wallets_accounts OWNER TO current_user;

--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: bridges bridges_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.bridges
    ADD CONSTRAINT bridges_pkey PRIMARY KEY (id);


--
-- Name: connectors_load_balancers connectors_load_balancers_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_pkey PRIMARY KEY (connector_id, load_balancer_id);


--
-- Name: connectors_nodes connectors_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_nodes
    ADD CONSTRAINT connectors_nodes_pkey PRIMARY KEY (connector_id, node_id);


--
-- Name: connectors connectors_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors
    ADD CONSTRAINT connectors_pkey PRIMARY KEY (id);


--
-- Name: contracts contracts_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_pkey PRIMARY KEY (id);


--
-- Name: filters filters_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.filters
    ADD CONSTRAINT filters_pkey PRIMARY KEY (id);


--
-- Name: load_balancers_nodes load_balancers_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.load_balancers_nodes
    ADD CONSTRAINT load_balancers_nodes_pkey PRIMARY KEY (load_balancer_id, node_id);


--
-- Name: load_balancers load_balancers_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.load_balancers
    ADD CONSTRAINT load_balancers_pkey PRIMARY KEY (id);


--
-- Name: networks networks_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_pkey PRIMARY KEY (id);


--
-- Name: nodes nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT nodes_pkey PRIMARY KEY (id);


--
-- Name: oracles oracles_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.oracles
    ADD CONSTRAINT oracles_pkey PRIMARY KEY (id);


--
-- Name: tokens tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_pkey PRIMARY KEY (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: wallets_accounts wallets_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.wallets_accounts
    ADD CONSTRAINT wallets_accounts_pkey PRIMARY KEY (wallet_id, account_id);


--
-- Name: wallets wallets_pkey; Type: CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_pkey PRIMARY KEY (id);


--
-- Name: idx_accounts_accessed_at; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_accounts_accessed_at ON public.accounts USING btree (accessed_at);


--
-- Name: idx_accounts_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_accounts_application_id ON public.accounts USING btree (application_id);


--
-- Name: idx_accounts_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_accounts_network_id ON public.accounts USING btree (network_id);


--
-- Name: idx_accounts_user_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_accounts_user_id ON public.accounts USING btree (user_id);


--
-- Name: idx_accounts_wallet_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_accounts_wallet_id ON public.accounts USING btree (wallet_id);


--
-- Name: idx_bridges_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_bridges_application_id ON public.bridges USING btree (application_id);


--
-- Name: idx_bridges_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_bridges_network_id ON public.bridges USING btree (network_id);


--
-- Name: idx_chain_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE UNIQUE INDEX idx_chain_id ON public.networks USING btree (chain_id);


--
-- Name: idx_connectors_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_connectors_application_id ON public.connectors USING btree (application_id);


--
-- Name: idx_connectors_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_connectors_network_id ON public.connectors USING btree (network_id);


--
-- Name: idx_connectors_type; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_connectors_type ON public.connectors USING btree (type);


--
-- Name: idx_contracts_accessed_at; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_contracts_accessed_at ON public.contracts USING btree (accessed_at);


--
-- Name: idx_contracts_address; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_contracts_address ON public.contracts USING btree (address);


--
-- Name: idx_contracts_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_contracts_application_id ON public.contracts USING btree (application_id);


--
-- Name: idx_contracts_contract_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_contracts_contract_id ON public.contracts USING btree (contract_id);


--
-- Name: idx_contracts_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_contracts_network_id ON public.contracts USING btree (network_id);


--
-- Name: idx_contracts_transaction_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE UNIQUE INDEX idx_contracts_transaction_id ON public.contracts USING btree (transaction_id);


--
-- Name: idx_filters_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_filters_application_id ON public.filters USING btree (application_id);


--
-- Name: idx_filters_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_filters_network_id ON public.filters USING btree (network_id);


--
-- Name: idx_load_balancers_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_load_balancers_application_id ON public.load_balancers USING btree (application_id);


--
-- Name: idx_load_balancers_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_load_balancers_network_id ON public.load_balancers USING btree (network_id);


--
-- Name: idx_load_balancers_region; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_load_balancers_region ON public.load_balancers USING btree (region);


--
-- Name: idx_load_balancers_status; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_load_balancers_status ON public.load_balancers USING btree (status);


--
-- Name: idx_load_balancers_type; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_load_balancers_type ON public.load_balancers USING btree (type);


--
-- Name: idx_network_nodes_type; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_network_nodes_type ON public.load_balancers USING btree (type);


--
-- Name: idx_networks_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_networks_application_id ON public.networks USING btree (application_id);


--
-- Name: idx_networks_cloneable; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_networks_cloneable ON public.networks USING btree (cloneable);


--
-- Name: idx_networks_enabled; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_networks_enabled ON public.networks USING btree (enabled);


--
-- Name: idx_networks_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_networks_network_id ON public.networks USING btree (network_id);


--
-- Name: idx_networks_user_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_networks_user_id ON public.networks USING btree (user_id);


--
-- Name: idx_nodes_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_application_id ON public.nodes USING btree (application_id);


--
-- Name: idx_nodes_bootnode; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_bootnode ON public.nodes USING btree (bootnode);


--
-- Name: idx_nodes_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_network_id ON public.nodes USING btree (network_id);


--
-- Name: idx_nodes_role; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_role ON public.nodes USING btree (role);


--
-- Name: idx_nodes_status; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_status ON public.nodes USING btree (status);


--
-- Name: idx_nodes_user_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_nodes_user_id ON public.nodes USING btree (user_id);


--
-- Name: idx_oracles_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_oracles_application_id ON public.oracles USING btree (application_id);


--
-- Name: idx_oracles_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_oracles_network_id ON public.oracles USING btree (network_id);


--
-- Name: idx_tokens_accessed_at; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_tokens_accessed_at ON public.tokens USING btree (accessed_at);


--
-- Name: idx_tokens_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_tokens_application_id ON public.tokens USING btree (application_id);


--
-- Name: idx_tokens_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_tokens_network_id ON public.tokens USING btree (network_id);


--
-- Name: idx_transactions_account_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_account_id ON public.transactions USING btree (account_id);


--
-- Name: idx_transactions_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_application_id ON public.transactions USING btree (application_id);


--
-- Name: idx_transactions_created_at; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_created_at ON public.transactions USING btree (created_at);


--
-- Name: idx_transactions_hash; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE UNIQUE INDEX idx_transactions_hash ON public.transactions USING btree (hash);


--
-- Name: idx_transactions_network_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_network_id ON public.transactions USING btree (network_id);


--
-- Name: idx_transactions_ref; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_ref ON public.transactions USING btree (ref);


--
-- Name: idx_transactions_status; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_status ON public.transactions USING btree (status);


--
-- Name: idx_transactions_user_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_transactions_user_id ON public.transactions USING btree (user_id);


--
-- Name: idx_wallets_application_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_wallets_application_id ON public.wallets USING btree (application_id);


--
-- Name: idx_wallets_user_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_wallets_user_id ON public.wallets USING btree (user_id);


--
-- Name: idx_wallets_wallet_id; Type: INDEX; Schema: public; Owner: goldmine
--

CREATE INDEX idx_wallets_wallet_id ON public.wallets USING btree (wallet_id);


--
-- Name: accounts accounts_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: accounts accounts_wallet_id_wallets_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_wallet_id_wallets_id_foreign FOREIGN KEY (wallet_id) REFERENCES public.wallets(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: bridges bridges_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.bridges
    ADD CONSTRAINT bridges_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: connectors_load_balancers connectors_load_balancers_connector_id_connectors_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES public.connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: connectors_load_balancers connectors_load_balancers_load_balancer_id_load_balancers_id_fo; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_load_balancers
    ADD CONSTRAINT connectors_load_balancers_load_balancer_id_load_balancers_id_fo FOREIGN KEY (load_balancer_id) REFERENCES public.load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: connectors connectors_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors
    ADD CONSTRAINT connectors_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: connectors_nodes connectors_nodes_connector_id_connectors_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_nodes
    ADD CONSTRAINT connectors_nodes_connector_id_connectors_id_foreign FOREIGN KEY (connector_id) REFERENCES public.connectors(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: connectors_nodes connectors_nodes_node_id_nodes_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.connectors_nodes
    ADD CONSTRAINT connectors_nodes_node_id_nodes_id_foreign FOREIGN KEY (node_id) REFERENCES public.nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: contracts contracts_contract_id_contracts_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_contract_id_contracts_id_foreign FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: contracts contracts_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: contracts contracts_transaction_id_transactions_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.contracts
    ADD CONSTRAINT contracts_transaction_id_transactions_id_foreign FOREIGN KEY (transaction_id) REFERENCES public.transactions(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: filters filters_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.filters
    ADD CONSTRAINT filters_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: load_balancers_nodes load_balancers_load_balancer_id_load_balancers_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.load_balancers_nodes
    ADD CONSTRAINT load_balancers_load_balancer_id_load_balancers_id_foreign FOREIGN KEY (load_balancer_id) REFERENCES public.load_balancers(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: load_balancers load_balancers_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.load_balancers
    ADD CONSTRAINT load_balancers_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: load_balancers_nodes load_balancers_node_id_nodes_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.load_balancers_nodes
    ADD CONSTRAINT load_balancers_node_id_nodes_id_foreign FOREIGN KEY (node_id) REFERENCES public.nodes(id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: networks networks_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: networks networks_sidechain_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.networks
    ADD CONSTRAINT networks_sidechain_id_networks_id_foreign FOREIGN KEY (sidechain_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: nodes nodes_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT nodes_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: oracles oracles_contract_id_contracts_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.oracles
    ADD CONSTRAINT oracles_contract_id_contracts_id_foreign FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: oracles oracles_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.oracles
    ADD CONSTRAINT oracles_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: tokens tokens_contract_id_contracts_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_contract_id_contracts_id_foreign FOREIGN KEY (contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: tokens tokens_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: tokens tokens_sale_contract_id_contracts_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_sale_contract_id_contracts_id_foreign FOREIGN KEY (sale_contract_id) REFERENCES public.contracts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: transactions transactions_account_id_accounts_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_account_id_accounts_id_foreign FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: transactions transactions_network_id_networks_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_network_id_networks_id_foreign FOREIGN KEY (network_id) REFERENCES public.networks(id) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: wallets wallets_wallet_id_wallets_id_foreign; Type: FK CONSTRAINT; Schema: public; Owner: goldmine
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_wallet_id_wallets_id_foreign FOREIGN KEY (wallet_id) REFERENCES public.wallets(id) ON UPDATE CASCADE ON DELETE SET NULL;

--
-- PostgreSQL database dump complete
--
