CREATE TABLE visit_answer (
  visit_id BIGINT UNSIGNED NOT NULL,
  question_id VARCHAR(255) NOT NULL,
  actor_entity_id VARCHAR(128) NOT NULL,
  data BLOB NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT visit_id FOREIGN KEY (visit_id) REFERENCES visit (id),
  PRIMARY KEY (visit_id, question_id)
);
