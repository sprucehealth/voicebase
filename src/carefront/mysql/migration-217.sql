-- Formatted field tags does not have to have a value
alter table question modify column formatted_field_tags varchar(150);

-- Update title of photo question
update localized_text set ltext = 'Where do your breakouts occur?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_location');

-- Remove neck option from photo question
update potential_answer set status='INACTIVE' where potential_answer_tag='a_neck_acne_location';

-- Add new question for profile left
insert into question (qtype_id, question_tag, required) values ((select id from question_type where qtype='q_type_photo'), 'q_face_left_photo_intake', 1);
update potential_answer set potential_answer_tag = 'a_face_left_photo_intake', question_id = (select id from question where question_tag='q_face_left_photo_intake') where potential_answer_tag = 'a_face_left_phota_intake';


-- Add new question for profile right
insert into question (qtype_id, question_tag, required) values ((select id from question_type where qtype='q_type_photo'), 'q_face_right_photo_intake', 1);
update potential_answer set potential_answer_tag = 'a_face_right_photo_intake', question_id = (select id from question where question_tag='q_face_right_photo_intake') where potential_answer_tag = 'a_face_right_phota_intake';

