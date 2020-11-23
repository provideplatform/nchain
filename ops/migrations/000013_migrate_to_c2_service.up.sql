DROP TABLE connectors_load_balancers;
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
