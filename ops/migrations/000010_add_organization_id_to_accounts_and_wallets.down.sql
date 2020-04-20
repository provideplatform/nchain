DROP INDEX idx_accounts_organization_id;
ALTER TABLE ONLY accounts DROP COLUMN organization_id;

DROP INDEX idx_wallets_organization_id;
ALTER TABLE ONLY wallets DROP COLUMN organization_id;
