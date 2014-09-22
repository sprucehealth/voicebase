update localized_text set ltext = 'Formed deep, hard lumps' 
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_deep_lumps');

update localized_text set ltext = 'Hair products' 
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_acne_worse_hair_products');

update localized_text set ltext = 'Hormonal changes'
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_acne_worse_hormonal_changes');

update localized_text set ltext = 'How many months have you been taking this medication?' 
	where app_text_id = (select qtext_app_text_id from question where question_tag='q_length_current_medication');


update potential_answer set status = 'ACTIVE' 
	where potential_answer_tag='a_combination_skin';

update potential_answer set ordering = 6
	where potential_answer_tag='a_normal_skin';
update potential_answer set ordering = 7
	where potential_answer_tag='a_oil_skin';
update potential_answer set ordering = 8
	where potential_answer_tag='a_combination_skin';
update potential_answer set ordering = 9
	where potential_answer_tag='a_dry_skin';
update potential_answer set ordering = 10
	where potential_answer_tag='a_sensitive_skin';
update potential_answer set ordering = 11
	where potential_answer_tag='a_other_skin';


update localized_text set ltext = 'How does your skin compare to the photos you took?'
	where app_text_id = (select qtext_app_text_id from question where question_tag='q_skin_photo_comparison');

update localized_text set ltext = 'I usually have more acne blemishes.'
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_more_acne_blemishes_photo_comparison');

update localized_text set ltext = 'I usually have fewer acne blemishes.'
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_fewer_acne_blemishes_photo_comparison');

update localized_text set ltext = 'My skin usually looks about the same.'
	where app_text_id = (select answer_localized_text_id from potential_answer where potential_answer_tag='a_about_the_same_photo_comparison');





