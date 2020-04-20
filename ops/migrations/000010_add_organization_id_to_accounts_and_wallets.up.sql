ALTER TABLE ONLY accounts ADD COLUMN organization_id uuid;
CREATE INDEX idx_accounts_organization_id ON accounts USING btree (organization_id);

ALTER TABLE ONLY wallets ADD COLUMN organization_id uuid;
CREATE INDEX idx_wallets_organization_id ON accounts USING btree (organization_id);
