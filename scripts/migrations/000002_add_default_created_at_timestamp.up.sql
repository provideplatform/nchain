ALTER TABLE ONLY accounts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY bridges ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY connectors ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY contracts ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY filters ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY load_balancers ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY networks ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY nodes ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY oracles ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY tokens ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY transactions ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE ONLY wallets ALTER COLUMN created_at SET DEFAULT now();
