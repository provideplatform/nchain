ALTER TABLE ONLY transactions ADD COLUMN nonce int;
CREATE INDEX idx_transactions_nonce ON transactions USING btree (nonce);
