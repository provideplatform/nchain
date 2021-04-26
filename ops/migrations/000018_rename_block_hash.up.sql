ALTER TABLE blocks RENAME COLUMN transactions_hash TO hash;
CREATE INDEX idx_blocks_hash ON blocks USING btree (hash);
