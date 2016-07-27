
INSERT INTO thread_entities (thread_id, entity_id, member)
SELECT id, organization_id, true FROM threads WHERE type IN ('EXTERNAL', 'SECURE_EXTERNAL', 'SETUP', 'SUPPORT', 'LEGACY_TEAM')
ON DUPLICATE KEY UPDATE member = true;
