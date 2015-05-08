INSERT INTO app_text (app_text_tag, comment) VALUES
	('txt_feedback_screen_title', 'In-app feedback screen title'),
	('txt_feedback_rating_prompt', 'In-app feedback rating prompt'),
	('txt_feedback_comment_placeholder', 'In-app feedback comment placeholder'),
	('txt_feedback_submit_button', 'In-app feedback submit button');
INSERT INTO localized_text (language_id, app_text_id, ltext) VALUES
	(1, (SELECT id FROM app_text WHERE app_text_tag = 'txt_feedback_screen_title'), 'How did we do?'),
	(1, (SELECT id FROM app_text WHERE app_text_tag = 'txt_feedback_rating_prompt'), 'Please rate your Spruce experience so far:'),
	(1, (SELECT id FROM app_text WHERE app_text_tag = 'txt_feedback_comment_placeholder'), 'Anything else you''d like to tell us?'),
	(1, (SELECT id FROM app_text WHERE app_text_tag = 'txt_feedback_submit_button'), 'Done');

CREATE TABLE patient_feedback (
	id INT UNSIGNED NOT NULL AUTO_INCREMENT,
	feedback_for VARCHAR(32) NOT NULL,
	patient_id INT UNSIGNED NOT NULL,
	rating INT NOT NULL,
	comment TEXT,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY (patient_id, feedback_for),
    CONSTRAINT patient_feedback_patient FOREIGN KEY (patient_id) REFERENCES patient (id)
);
