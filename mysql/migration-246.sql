update question set required=0 where question_tag in ('q_anything_else_prev_acne_otc', 'q_anything_else_prev_acne_prescription', 'q_acne_otc_product_tried');

set @language_id = (select id from languages_supported where language='en');

-- Include question summaries for subquestions
insert into app_text (app_text_tag) values ('txt_currently_using');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_currently_using'), "Currently Using");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_currently_using') where question_tag='q_using_prev_acne_prescription';
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_currently_using') where question_tag='q_using_prev_acne_otc';

insert into app_text (app_text_tag) values ('txt_effective');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_effective'), "Effective");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_effective') where question_tag='q_how_effective_prev_acne_prescription';
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_effective') where question_tag='q_how_effective_prev_acne_otc';

insert into app_text (app_text_tag) values ('txt_used_three_plus_months');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_used_three_plus_months'), "Used for 3+ months");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_used_three_plus_months') where question_tag='q_use_more_three_months_prev_acne_prescription';

insert into app_text (app_text_tag) values ('txt_irritating');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_irritating'), "Irritating");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_irritating') where question_tag='q_irritate_skin_prev_acne_prescription';
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_irritating') where question_tag='q_irritate_skin_prev_acne_otc';

insert into app_text (app_text_tag) values ('txt_comments');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_comments'), "Comments");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_comments') where question_tag='q_anything_else_prev_acne_prescription';
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_comments') where question_tag='q_anything_else_prev_acne_otc';	

insert into app_text (app_text_tag) values ('txt_which_product');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_which_product'), "Which product");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_which_product') where question_tag='q_acne_otc_product_tried';

insert into app_text (app_text_tag) values ('txt_how_long');
insert into localized_text (language_id, app_text_id, ltext) values (@language_id, (select id from app_text where app_text_tag='txt_how_long'), "How Long");
update question set qtext_short_text_id = (select id from app_text where app_text_tag='txt_how_long') where question_tag='q_length_current_medication';











