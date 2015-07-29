INSERT INTO app_text (app_text_tag, comment)
VALUES
    ('parental_consent_completed_notification', 'Text for notification sent to minor patient when parent completes cosent flow'),
    ('parental_consent_request_sms', 'Prefill content for SMS composed to parents when a child requests consent for treatment');

INSERT INTO localized_text (language_id, app_text_id, ltext)
VALUES
    (1, (SELECT id FROM app_text WHERE app_text_tag = 'parental_consent_completed_notification'), 'Your parent has authorized your visit. Complete your visit and get a personalized treatment plan.'),
    (1, (SELECT id FROM app_text WHERE app_text_tag = 'parental_consent_request_sms'), 'Hey, I''d like to see a dermatologist for my acne. With Spruce I can see a board-certified dermatologist from my phone but need your approval: %s');
