ALTER TABLE ONLY transactions RENAME COLUMN network_latency TO broadcast_latency;

ALTER TABLE ONLY transactions DROP COLUMN wallet_id;
ALTER TABLE ONLY transactions DROP COLUMN hd_derivation_path;

DROP INDEX idx_transactions_wallet_id;
ALTER TABLE ONLY transactions DROP CONSTRAINT transactions_wallet_id_wallets_id_foreign;
