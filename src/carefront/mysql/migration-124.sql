update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag = 'txt_erythematotelangiectatic_rosacea') where potential_answer_tag="a_acne_erythematotelangiectatic_rosacea";
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag = 'txt_papulopstular_rosacea') where potential_answer_tag="a_acne_papulopstular_rosacea";
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag = 'txt_rhinophyma_rosacea') where potential_answer_tag="a_acne_rhinophyma_rosacea";
update potential_answer set answer_localized_text_id = (select id from app_text where app_text_tag = 'txt_ocular_rosacea') where potential_answer_tag="a_acne_ocular_rosacea";
