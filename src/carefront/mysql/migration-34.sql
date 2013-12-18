start transaction;

insert into app_text (app_text_tag, comment) values ('txt_select_all_apply', 'select all apply');
insert into localized_text (app_text_id, language_id, ltext) values ((select id from app_text where app_text_tag = 'txt_select_all_apply'), 1, "(select all that apply)");

update question set subtext_app_text_id=(select id from app_text where app_text_tag='txt_select_all_apply') where question_tag='q_acne_type';

commit;
