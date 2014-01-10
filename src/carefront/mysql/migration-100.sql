start transaction;
update potential_answer set answer_localized_text_id=(select id from app_text where app_text_tag='txt_back_acne_location') where potential_answer_tag='a_back_phota_intake';
commit;