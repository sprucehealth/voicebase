
-- Purpose of this table is to store the content to use for each of the different
-- feedback types supported (feedback:good_freetext, feedback:bad_freetext, feedback:multiple_choice, feedback:app_store, etc.)
-- At any given time there will be just a single active template for each of the types. Versioning each of the types
-- updates the current template to be inactive and the new one to be active. At any given time only the active one is 
-- considered.
CREATE TABLE feedback_template (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  tag VARCHAR(64) NOT NULL,
  type VARCHAR(256) NOT NULL,
  json_data blob NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  active BOOL NOT NULL,
  INDEX (tag, active),
  INDEX (active),
  PRIMARY KEY (id)
);

-- Purpose of this table is to store the structured response to feedback by the patient. Feedback
-- input is stored in blob form given the varied feedback types supported (free text, multiple choice, etc.)
-- Doing any sort of querying on the data (to determine how many people said that their communication
-- with their doctor could have been better, for instance) requires parsing the json data. But the idea is that
-- such parsing would happen on redshift rather than directly in the application database so the querying
-- shouldn't be so bad. Note that such querying would require joining against the feedback_template table
-- to ensure that we are only looking at feedback data for the type that was active at the time of input.
-- We will, however, try to store all the necessary information required to analyze input within this blob
-- (the question, along with the answer, for instance).
-- Storing the information in a blob also means that any validation of the data would have to happen at the application 
-- layer. This feels like an okay tradeoff given that the feedaback input should be fairly minimal and straightforward.
CREATE TABLE patient_structured_feedback (
  feedback_template_id INT UNSIGNED NOT NULL,
  patient_feedback_id INT UNSIGNED NOT NULL,
  patient_id BIGINT UNSIGNED NOT NULL,
  json_data blob NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ,
  PRIMARY KEY (feedback_template_id, patient_feedback_id),
  FOREIGN KEY (patient_id) REFERENCES patient(id),
  FOREIGN KEY (patient_feedback_id) REFERENCES patient_feedback(id),
  FOREIGN KEY (feedback_template_id) REFERENCES feedback_template(id)
);

-- This change will make it possible to stage patient feedback that is intended
-- to be collected by the patient but has yet to be collected. The idea 
-- is that an entry gets inserted when it is time to get feedback (like when the treatment plan has been viewed by the patient)
-- from the patient for a particular reason. If the patient dismisses the feedback prompt then "dismissed" 
-- changes to true. If the patient gives us a rating, then "pending" changes to false.
-- Note that this does come at the cost of making the rating nullable, but it feels like this is okay
-- because what we are gaining is knowing when a feedback prompt was intended to be shown to the patient
-- and then what the patient actually did with that prompt.
ALTER TABLE patient_feedback ADD COLUMN pending BOOL default false;
ALTER TABLE patient_feedback ADD COLUMN dismissed BOOL default false;
ALTER TABLE patient_feedback MODIFY column rating INT UNSIGNED;
ALTER TABLE patient_feedback ADD UNIQUE KEY (feedback_for);


CREATE TABLE feedback_template_config (
  rating INT NOT NULL,
  template_tags_csv VARCHAR(256) NOT NULL,
  last_modified_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (rating)
)

