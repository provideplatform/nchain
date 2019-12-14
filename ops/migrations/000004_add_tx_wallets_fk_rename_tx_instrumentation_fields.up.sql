ALTER TABLE ONLY transactions RENAME COLUMN broadcast_latency TO network_latency;

ALTER INDEX idx_transactions_wallet_id RENAME TO idx_transactions_account_id;

ALTER TABLE ONLY transactions ADD COLUMN wallet_id uuid;
ALTER TABLE ONLY transactions ADD COLUMN hd_derivation_path text;

CREATE INDEX idx_transactions_wallet_id ON transactions USING btree (wallet_id);
ALTER TABLE ONLY transactions ADD CONSTRAINT transactions_wallet_id_wallets_id_foreign FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON UPDATE CASCADE ON DELETE SET NULL;
