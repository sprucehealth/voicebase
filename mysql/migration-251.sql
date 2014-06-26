alter table photo_slot add column ordering int unsigned not null;
update photo_slot set ordering = 0 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_face_front') and question_id = (select id from question where question_tag='q_face_photo_section');
update photo_slot set ordering = 1 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_face_right') and question_id = (select id from question where question_tag='q_face_photo_section');
update photo_slot set ordering = 2 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_face_left') and question_id = (select id from question where question_tag='q_face_photo_section');
update photo_slot set ordering = 3 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_other') and question_id = (select id from question where question_tag='q_face_photo_section');

update photo_slot set ordering = 0 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_chest') and question_id = (select id from question where question_tag='q_chest_photo_section');
update photo_slot set ordering = 1 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_other') and question_id = (select id from question where question_tag='q_chest_photo_section');

update photo_slot set ordering = 0 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_back') and question_id = (select id from question where question_tag='q_back_photo_section');
update photo_slot set ordering = 1 where slot_type_id = (select id from photo_slot_type where slot_type = 'photo_slot_other') and question_id = (select id from question where question_tag='q_back_photo_section');

set @language_id = (select id from languages_supported where language='en');
insert into app_text (app_text_tag) values ('txt_face_side');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_face_side'), "Face Side");	
update photo_slot set slot_name_app_text_id = (select id from app_text where app_text_tag='txt_face_side') where slot_type_id in (select id from photo_slot_type where slot_type in ('photo_slot_face_right','photo_slot_face_left'));




