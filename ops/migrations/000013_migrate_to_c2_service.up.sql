ALTER TABLE ONLY connectors_load_balancers DROP CONSTRAINT connectors_load_balancers_pkey;
ALTER TABLE ONLY connectors_load_balancers DROP CONSTRAINT connectors_load_balancers_connector_id_connectors_id_foreign;
ALTER TABLE ONLY connectors_load_balancers DROP CONSTRAINT connectors_load_balancers_load_balancer_id_load_balancers_id_fo;
DROP TABLE connectors_load_balancers;

ALTER TABLE ONLY load_balancers_nodes DROP CONSTRAINT load_balancers_load_balancer_id_load_balancers_id_foreign;
DROP TABLE load_balancers_nodes;

DROP TABLE load_balancers;

ALTER TABLE ONLY nodes DROP COLUMN host;
ALTER TABLE ONLY nodes DROP COLUMN ipv4;
ALTER TABLE ONLY nodes DROP COLUMN ipv6;
ALTER TABLE ONLY nodes DROP COLUMN private_ipv4;
ALTER TABLE ONLY nodes DROP COLUMN private_ipv6;
ALTER TABLE ONLY nodes DROP COLUMN description;
ALTER TABLE ONLY nodes DROP COLUMN status;
ALTER TABLE ONLY nodes DROP COLUMN config;
ALTER TABLE ONLY nodes DROP COLUMN encrypted_config;

ALTER TABLE ONLY nodes ADD COLUMN c2_node_id uuid NOT NULL;
CREATE INDEX idx_network_nodes_c2_node_id ON nodes USING btree (c2_node_id);
