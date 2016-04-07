-- Need this index to efficiently lookup the support thread for the org
CREATE INDEX threads_type_deleted ON threads (organization_id, type, deleted);

-- Delete old setup threads, will be recreated through a migration cli
UPDATE threads SET deleted = 1 WHERE type = 'SETUP';
