-- Remove unused table
DROP TABLE photo_tips;

-- Deprecated but don't remove the now unused object_storage so that we don't lose data.
-- Remove the foreign key and index since they're dead weight now that the table is not used.
ALTER TABLE info_intake DROP FOREIGN KEY info_intake_ibfk_6;
ALTER TABLE info_intake DROP INDEX object_storage_id;
ALTER TABLE object_storage RENAME TO deprecated_object_storage;
