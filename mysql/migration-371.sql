-- Create language ID column and FK
ALTER TABLE question 
ADD COLUMN language_id INT(10) UNSIGNED DEFAULT 1,
ADD CONSTRAINT fk_question_languages_supported_id 
FOREIGN KEY(language_id)
REFERENCES languages_supported(id);

-- Create language ID column and FK
ALTER TABLE potential_answer 
ADD COLUMN language_id INT(10) UNSIGNED DEFAULT 1,
ADD CONSTRAINT fk_potential_answer_languages_supported_id 
FOREIGN KEY(language_id)
REFERENCES languages_supported(id);

-- Create question version record and the versioning constraint
ALTER TABLE question
ADD COLUMN version INT(10) UNSIGNED NOT NULL DEFAULT 1,
ADD UNIQUE unique_question_question_tag_version(question_tag, version, language_id);

-- Create the answer set id column then populate using the question id then add non null constraint
ALTER TABLE potential_answer 
ADD COLUMN answer_set_id INT(10) UNSIGNED;
UPDATE potential_answer SET answer_set_id = question_id;
ALTER TABLE potential_answer 
MODIFY answer_set_id INT(10) NOT NULL;

-- Create question version record and the versioning constraint
ALTER TABLE potential_answer
ADD COLUMN version INT(10) UNSIGNED NOT NULL DEFAULT 1,
ADD UNIQUE unique_potential_answer_tag_version(potential_answer_tag, answer_set_id, version, language_id);

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
ALTER TABLE question ADD COLUMN title_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.qtext_short_text_id = app_text_id
SET
  title_text = ltext;

-- Create subtitle_text column and populate from existing data
ALTER TABLE question ADD COLUMN subtitle_text VARCHAR(600);
UPDATE question q
LEFT JOIN localized_text lt ON
  q.subtext_app_text_id = app_text_id
SET
  subtitle_text = ltext;

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
