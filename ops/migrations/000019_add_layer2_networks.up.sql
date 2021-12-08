ALTER TABLE ONLY networks ADD COLUMN layer2 boolean DEFAULT false;
UPDATE networks SET layer2 = false;
ALTER TABLE ONLY networks ALTER COLUMN layer2 SET NOT NULL;
CREATE INDEX idx_networks_layer2 ON public.networks USING btree (layer2);
