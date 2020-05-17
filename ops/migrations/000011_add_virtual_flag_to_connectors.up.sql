ALTER TABLE ONLY connectors ADD COLUMN is_virtual boolean DEFAULT false;
UPDATE connectors SET is_virtual = false;
