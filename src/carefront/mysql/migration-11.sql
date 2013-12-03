use carefront_db;

start transaction;

alter table potential_answer modify answer_localized_text_id int unsigned;

update potential_answer set answer_localized_text_id=NULL where answer_localized_text_id=0;

commit;