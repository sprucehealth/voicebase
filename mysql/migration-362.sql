SET @en=(SELECT id FROM languages_supported WHERE language='en');

INSERT INTO app_text (app_text_tag) values ('txt_severity');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Severity', (SELECT id FROM app_text WHERE app_text_tag='txt_severity'));

INSERT INTO app_text (app_text_tag) values ('txt_mild');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Mild', (SELECT id FROM app_text WHERE app_text_tag='txt_mild'));

INSERT INTO app_text (app_text_tag) values ('txt_moderate');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Moderate', (SELECT id FROM app_text WHERE app_text_tag='txt_moderate'));

INSERT INTO app_text (app_text_tag) values ('txt_severe');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Severe', (SELECT id FROM app_text WHERE app_text_tag='txt_severe'));

INSERT INTO app_text (app_text_tag) values ('txt_type');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Type', (SELECT id FROM app_text WHERE app_text_tag='txt_type'));

INSERT INTO app_text (app_text_tag) values ('txt_cystic');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Cystic', (SELECT id FROM app_text WHERE app_text_tag='txt_cystic'));

INSERT INTO app_text (app_text_tag) values ('txt_hormonal');
INSERT INTO localized_text (language_id, ltext, app_text_id) VALUES (@en, 'Hormonal', (SELECT id FROM app_text WHERE app_text_tag='txt_hormonal'));


INSERT INTO question (qtype_id, qtext_app_text_id, question_tag, required) values
	(
	(SELECT id FROM question_type WHERE qtype= 'q_type_single_select'),
	(SELECT id FROM app_text WHERE app_text_tag='txt_severity'),
	'q_diagnosis_severity',
	0);

INSERT INTO potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status)
	VALUES (
		(SELECT id FROM question WHERE question_tag='q_diagnosis_severity'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_mild'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_severity_mild',
		0,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_severity'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_moderate'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_severity_moderate',
		1,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_severity'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_severe'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_severity_severe',
		2,
		'ACTIVE'
		);

INSERT INTO question (qtype_id, qtext_app_text_id, question_tag, required) values
	(
	(SELECT id FROM question_type WHERE qtype= 'q_type_multiple_choice'),
	(SELECT id FROM app_text WHERE app_text_tag='txt_type'),
	'q_diagnosis_acne_vulgaris_type',
	0);

INSERT INTO potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status)
	VALUES (
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_vulgaris_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_comedonal'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_vulgaris_type_comedonal',
		0,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_vulgaris_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_acne_inflammatory'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_vulgaris_type_inflammatory',
		1,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_vulgaris_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_cystic'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_vulgaris_type_cystic',
		2,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_vulgaris_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_hormonal'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_vulgaris_type_hormonal',
		3,
		'ACTIVE'
		);

INSERT INTO question (qtype_id, qtext_app_text_id, question_tag, required) values
	(
	(SELECT id FROM question_type WHERE qtype= 'q_type_multiple_choice'),
	(SELECT id FROM app_text WHERE app_text_tag='txt_type'),
	'q_diagnosis_acne_rosacea_type',
	0);

INSERT INTO potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status)
	VALUES (
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_rosacea_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_erythematotelangiectatic_rosacea'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_rosacea_type_erythematotelangiectatic',
		0,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_rosacea_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_papulopstular_rosacea'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_rosacea_type_papulopstular',
		1,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_rosacea_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_rhinophyma_rosacea'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_rosacea_type_rhinophyma',
		2,
		'ACTIVE'
		),(
		(SELECT id FROM question WHERE question_tag='q_diagnosis_acne_rosacea_type'),
		(SELECT id FROM app_text WHERE app_text_tag='txt_ocular_rosacea'),
		(SELECT id FROM answer_type WHERE atype='a_type_multiple_choice'),
		'a_diagnosis_acne_rosacea_type_ocular',
		3,
		'ACTIVE'
		);






