DROP INDEX idx_connectors_organization_id;
ALTER TABLE ONLY connectors DROP COLUMN organization_id;

DROP INDEX idx_load_balancers_organization_id;
ALTER TABLE ONLY load_balancers DROP COLUMN organization_id;

DROP INDEX idx_nodes_organization_id;
ALTER TABLE ONLY nodes DROP COLUMN organization_id;
