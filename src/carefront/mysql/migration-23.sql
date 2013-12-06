-- Removing potential_answers from free_text and autocomplete type questions since they are not needed for those
-- questions as we can submit answers against those questions without requiring the potential_answer_id

delete from potential_answer where question_id in (select id from question where qtype_id in (select id from question_type where qtype in ('q_type_autocomplete', 'q_type_free_text') ));