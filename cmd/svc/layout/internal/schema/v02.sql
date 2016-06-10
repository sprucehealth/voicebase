ALTER TABLE visit_layout ADD COLUMN internal_name TEXT NOT NULL DEFAULT 'dummy';
UPDATE visit_layout SET internal_name = name;
