CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

--
-- Name: networks; Type: TABLE; Schema: public; Owner: goldmine
--

CREATE TABLE public.networks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    application_id uuid,
    user_id uuid,
    name text NOT NULL,
    description text,
    is_production boolean NOT NULL,
    cloneable boolean NOT NULL,
    enabled boolean NOT NULL,
    chain_id text,
    sidechain_id uuid,
    network_id uuid,
    config json NOT NULL
);
