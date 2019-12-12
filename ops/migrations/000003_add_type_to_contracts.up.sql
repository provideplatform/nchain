ALTER TABLE ONLY contracts ADD COLUMN type char(32);
CREATE INDEX idx_contracts_type ON contracts USING btree (type);
