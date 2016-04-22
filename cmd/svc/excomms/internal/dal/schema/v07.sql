ALTER TABLE outgoing_call_request ADD COLUMN caller_entity_id VARCHAR(64) NOT NULL;
ALTER TABLE outgoing_call_request ADD COLUMN callee_entity_id VARCHAR(64) NOT NULL;