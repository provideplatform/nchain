ALTER TABLE ONLY accounts ADD COLUMN type varchar(32);
CREATE INDEX idx_accounts_type ON accounts USING btree (type);
