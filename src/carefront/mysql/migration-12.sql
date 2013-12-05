-- Adding the field to store summary data for answer
alter table potential_answer add answer_summary_text_id int unsigned;
alter table potential_answer add foreign key (answer_summary_text_id) references app_text(id);