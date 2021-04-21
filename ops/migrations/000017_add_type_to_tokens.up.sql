ALTER TABLE ONLY tokens ADD COLUMN type varchar(32);
CREATE INDEX idx_tokens_type ON tokens USING btree (type);
