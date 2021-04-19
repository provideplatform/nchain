ALTER TABLE ONLY transactions ADD COLUMN organization_id uuid;
CREATE INDEX idx_transactions_organization_id ON transactions USING btree (organization_id);

ALTER TABLE ONLY tokens ADD COLUMN organization_id uuid;
CREATE INDEX idx_tokens_organization_id ON tokens USING btree (organization_id);
