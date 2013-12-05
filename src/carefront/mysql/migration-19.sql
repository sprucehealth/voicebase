-- Ensuring that only unique fields are added to the question_fields table
alter table question_fields add key (question_field, question_id);
