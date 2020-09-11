DROP INDEX idx_accounts_vault_id_key_id;
ALTER TABLE ONLY accounts DROP COLUMN vault_id;
ALTER TABLE ONLY accounts ADD COLUMN private_key bytea;

DROP INDEX idx_wallets_vault_id_key_id;
ALTER TABLE ONLY wallets DROP COLUMN vault_id;
ALTER TABLE ONLY wallets DROP COLUMN key_id;
ALTER TABLE ONLY wallets ADD COLUMN mnemonic bytea;
ALTER TABLE ONLY wallets ADD COLUMN private_key bytea;
ALTER TABLE ONLY wallets ADD COLUMN seed bytea;
