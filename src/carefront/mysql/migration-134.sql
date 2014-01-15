update question set qtype_id = (select id from question_type where qtype='q_type_multiple_choice') where question_tag in ('q_acne_type', 'q_acne_rosacea_type');

