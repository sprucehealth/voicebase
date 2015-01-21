-- Create language ID column and FK
ALTER TABLE question 
ADD COLUMN language_id INT(10) UNSIGNED DEFAULT 1,
ADD CONSTRAINT fk_question_languages_supported_id 
FOREIGN KEY(language_id)
REFERENCES languages_supported(id);

-- Create language ID column and FK
ALTER TABLE potential_answer 
ADD COLUMN language_id INT(10) UNSIGNED NOT NULL DEFAULT 1,
ADD CONSTRAINT fk_potential_answer_languages_supported_id 
FOREIGN KEY(language_id)
REFERENCES languages_supported(id);

-- Create question version record and the versioning constraint
ALTER TABLE question
ADD COLUMN version INT(10) UNSIGNED NOT NULL DEFAULT 1,
ADD UNIQUE unique_question_question_tag_version(question_tag, version, language_id);

-- Create constraints
ALTER TABLE potential_answer
ADD UNIQUE unique_potential_answer_tag_quid_order(potential_answer_tag, question_id, ordering, language_id);

-- Create answer_text column and populate from existing data
ALTER TABLE potential_answer ADD COLUMN answer_text VARCHAR(600);
UPDATE potential_answer pa
LEFT JOIN localized_text lt ON
  pa.answer_localized_text_id = app_text_id
SET
  answer_text = ltext;

-- Create answer_summary_text column and populate from existing data
ALTER TABLE potential_answer ADD COLUMN answer_summary_text VARCHAR(600);
UPDATE potential_answer pa
LEFT JOIN localized_text lt ON
  pa.answer_summary_text_id = app_text_id
SET
  answer_summary_text = ltext;

-- Create title_text column and populate from existing data
ALTER TABLE question ADD COLUMN summary_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.qtext_short_text_id = app_text_id
SET
  summary_text = ltext;

-- Create subtitle_text column and populate from existing data
ALTER TABLE question ADD COLUMN subtext_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.subtext_app_text_id = app_text_id
SET
  subtext_text = ltext;

-- Create question_text column and populate from existing data
ALTER TABLE question ADD COLUMN question_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.qtext_app_text_id = app_text_id
SET
  question_text = ltext;

-- Create alert_text column and populate from existing data
ALTER TABLE question ADD COLUMN alert_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.alert_app_text_id = app_text_id
SET
  alert_text = ltext;

-- Create question_type column and poopulate from existing data
ALTER TABLE question ADD COLUMN question_type VARCHAR(60);
UPDATE question q
LEFT JOIN question_type qt on
  q.qtype_id = qt.id
SET
  question_type = qt.qtype;
ALTER TABLE question MODIFY question_type VARCHAR(60) NOT NULL;

-- Create answer_type column and populate from existing data
ALTER TABLE potential_answer ADD COLUMN answer_type VARCHAR(60);
UPDATE potential_answer a
LEFT JOIN answer_type atype on
  atype.id = atype_id
SET
  answer_type = atype.atype;
ALTER TABLE potential_answer MODIFY answer_type VARCHAR(60) NOT NULL;

-- We no longer need unique contraints on tags
DROP INDEX potential_outcome_tag on potential_answer;
DROP INDEX question_tag on question;

-- Drop the old unique constraint on question_id ordering
ALTER TABLE potential_answer DROP FOREIGN KEY potential_answer_ibfk_2;
DROP INDEX question_id_2 on potential_answer;

-- Recreate the FK
ALTER TABLE potential_answer 
ADD CONSTRAINT fk_question_question_id 
FOREIGN KEY(question_id)
REFERENCES question(id);

-- Add language ID and rename to additional_question_fields
ALTER TABLE extra_question_fields ADD COLUMN language_id INT(10) UNSIGNED DEFAULT 1;
RENAME TABLE extra_question_fields TO additional_question_fields;

-- Drop the unique question ID and FK that's locking it
ALTER TABLE additional_question_fields DROP FOREIGN KEY additional_question_fields_ibfk_1;
DROP INDEX question_id ON additional_question_fields;

-- Rebuild the FK
ALTER TABLE additional_question_fields 
ADD CONSTRAINT fk_additional_answer_fields_question_id 
FOREIGN KEY(question_id)
REFERENCES question(id);

-- Merge the old table into the renamed table
INSERT INTO additional_question_fields (question_id, json, language_id)
SELECT question_id, CAST(CONCAT('{"', question_field, '":"', ltext, '"}') AS BINARY) json, language_id FROM question_fields qf JOIN localized_text lt ON qf.app_text_id = lt.id;

-- Drop the old table
DROP TABLE question_fields;

-- Migrate our new permissions
INSERT INTO account_available_permission (name) VALUES ('layout.view'), ('layout.edit');
INSERT IGNORE INTO account_group_permission (group_id, permission_id)
    SELECT (SELECT id FROM account_group WHERE name = 'superuser'), id
    FROM account_available_permission;

-- Drop the unique potential_answer FK
ALTER TABLE potential_answer DROP FOREIGN KEY potential_answer_ibfk_1;
ALTER TABLE potential_answer DROP COLUMN atype_id;

-- Drop the unique questoin FK
ALTER TABLE question DROP FOREIGN KEY question_ibfk_1;
ALTER TABLE question DROP COLUMN qtype_id;