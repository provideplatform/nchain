ALTER TABLE ONLY networks ADD COLUMN block int8;
CREATE INDEX idx_network_block ON networks USING btree (block);