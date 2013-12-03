-- Adding missing localized text for the summary of the question pertaining to adding medications that patient is allergic to
start transaction;

set @en_language_id = (select id from languages_supported where language="en");

insert into localized_text (language_id, app_text_id, ltext) values (@en_language_id, (select id from app_text where app_text_tag = 'txt_short_allergic_medications_list'), 'Add a medication');

commit;