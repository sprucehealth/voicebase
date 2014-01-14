update potential_answer set answer_localized_text_id =  answer_summary_text_id where potential_answer_tag='a_six_twelve_months_ago';
update localized_text set ltext = 'List all other medications you are currently taking.' where app_text_id = (select id from app_text where app_text_tag='txt_list_medications');
update localized_text set ltext = 'Alopecia (hair loss)' where app_text_id = (select id from app_text where app_text_tag='txt_alopecia_diagnosis');

