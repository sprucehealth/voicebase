update localized_text set ltext='Medication allergies' where app_text_id = (select id from app_text where app_text_tag='txt_allergic_medications');