CREATE INDEX idx_blocks_hash;
ALTER TABLE blocks RENAME COLUMN hash TO transactions_hash;
