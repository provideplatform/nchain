ALTER TABLE ONLY connectors ADD COLUMN organization_id uuid;
CREATE INDEX idx_connectors_organization_id ON connectors USING btree (organization_id);

ALTER TABLE ONLY load_balancers ADD COLUMN organization_id uuid;
CREATE INDEX idx_load_balancers_organization_id ON load_balancers USING btree (organization_id);

ALTER TABLE ONLY nodes ADD COLUMN organization_id uuid;
CREATE INDEX idx_nodes_organization_id ON nodes USING btree (organization_id);
