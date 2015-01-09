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

-- Create question version record and the versioning constraint
ALTER TABLE potential_answer
ADD COLUMN version INT(10) UNSIGNED NOT NULL DEFAULT 1,
ADD UNIQUE unique_potential_answer_tag_version(potential_answer_tag, question_id, version, language_id);

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

ALTER TABLE question ADD COLUMN question_type VARCHAR(60);
UPDATE question q
LEFT JOIN question_type qt on
  q.qtype_id = qt.id
SET
  question_type = qt.qtype;
ALTER TABLE question MODIFY question_type VARCHAR(60) NOT NULL;

ALTER TABLE potential_answer ADD COLUMN answer_type VARCHAR(60);
UPDATE potential_answer a
LEFT JOIN answer_type atype on
  atype.id = atype_id
SET
  answer_type = atype.atype;
ALTER TABLE potential_answer MODIFY answer_type VARCHAR(60) NOT NULL;

COMMIT;
