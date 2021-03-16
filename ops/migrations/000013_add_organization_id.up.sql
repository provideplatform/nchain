ALTER TABLE ONLY contracts ADD COLUMN organization_id uuid;
CREATE INDEX idx_contracts_organization_id ON contracts USING btree (organization_id);
