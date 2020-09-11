ALTER TABLE ONLY accounts DROP COLUMN private_key;

ALTER TABLE ONLY accounts ADD COLUMN vault_id uuid NOT NULL;
ALTER TABLE ONLY accounts ADD COLUMN key_id uuid NOT NULL;
CREATE INDEX idx_accounts_vault_id_key_id ON accounts USING btree (vault_id, key_id);

ALTER TABLE ONLY wallets DROP COLUMN mnemonic;
ALTER TABLE ONLY wallets DROP COLUMN private_key;
ALTER TABLE ONLY wallets DROP COLUMN seed;

ALTER TABLE ONLY wallets ADD COLUMN vault_id uuid NOT NULL;
ALTER TABLE ONLY wallets ADD COLUMN key_id uuid NOT NULL;
CREATE INDEX idx_wallets_vault_id_key_id ON wallets USING btree (vault_id, key_id);
