start transaction;
insert into question_type (qtype) values ('q_type_photo');
update question set qtype_id = (select id from question_type where qtype='q_type_photo') where question_tag in ('q_face_photo_intake', 'q_chest_photo_intake', 'q_back_photo_intake', 'q_other_photo_intake', 'q_neck_photo_intake');
commit;