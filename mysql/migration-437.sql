-- Copy update for Parental Consent flow
UPDATE localized_text
SET ltext = 'Hey, I''d like to see a dermatologist for my acne. With Spruce I can see a board-certified dermatologist but need your approval before a doctor can treat me (since it’s remote treatment). Here’s the link %s - all the info you need should be there along with contact details for Spruce.'
WHERE app_text_id = (SELECT id FROM app_text WHERE app_text_tag = 'parental_consent_request_sms');
