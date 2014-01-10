start transaction;
update localized_text set ltext = 'Where are you experiencing symptoms?' where app_text_id = (select qtext_app_text_id from question where question_tag='q_acne_location');

insert into app_text (app_text_tag, comment) values ('txt_neck', 'option for acne location');
insert into localized_text (language_id, ltext, app_text_id) values (1, 'Neck', (select id from app_text where app_text_tag='txt_neck'));
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_acne_location'), (select id from app_text where app_text_tag='txt_neck'), (select id from answer_type where atype='a_type_multiple_choice'),'a_neck_acne_location',5,'ACTIVE');
update potential_answer set ordering=4 where potential_answer_tag='a_face_acne_location';
update potential_answer set ordering=6 where potential_answer_tag='a_chest_acne_location';
update potential_answer set ordering=7 where potential_answer_tag='a_back_acne_location';
update potential_answer set ordering=8 where potential_answer_tag='a_other_acne_location';

insert into answer_type (atype) values ('a_type_photo_entry_neck');
insert into question (qtype_id, question_tag) values ((select id from question_type where qtype='q_type_multiple_photo'), 'q_neck_photo_intake');
insert into potential_answer (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, status) values ((select id from question where question_tag='q_neck_photo_intake'), (select id from app_text where app_text_tag='txt_neck'), (select id from answer_type where atype='a_type_photo_entry_neck'),'a_neck_photo_intake',0,'ACTIVE');


commit;