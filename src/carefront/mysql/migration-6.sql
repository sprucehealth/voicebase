-- Dropping foreign key constraint for app text pertaining to potential answer and then any app text id that is not needed for a potential answer
start transaction;

alter table potential_answer drop foreign key potential_answer_ibfk_3;
alter table potential_answer drop key question_id;

update potential_answer set answer_localized_text_id=NULL where id in (3, 11, 15, 16, 17, 22, 23);

commit;