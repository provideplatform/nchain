DROP INDEX idx_transactions_organization_id;
ALTER TABLE ONLY transactions DROP COLUMN organization_id;

DROP INDEX idx_tokens_organization_id;
ALTER TABLE ONLY tokens DROP COLUMN organization_id;
