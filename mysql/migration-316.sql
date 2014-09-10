INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-visit-submitted', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'New Spruce Visit',
        'You have a new patient visit waiting.', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-treatment-plan-created', 'Default', (SELECT id FROM email_sender LIMIT 1),
        '{{if eq .Role ".PATIENT"}}Your doctor has reviewed your Spruce case{{else}}A Spruce treatment plan was created for a patient{{end}}',
        '{{if eq .Role ".PATIENT"}}Your doctor has reviewed your Spruce case{{else}}A Spruce treatment plan was created for a patient{{end}}', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-new-message', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'You have a new message on Spruce',
        'You have a new message on Spruce', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-case-assigned', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'A patient Spruce case has been assigned to you.',
        'A patient Spruce case has been assigned to you.', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-visit-routed', 'Default', (SELECT id FROM email_sender LIMIT 1),
        'A patient has submitted a Spruce visit.',
        'A patient has submitted a Spruce visit.', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-rx-transmission', 'Default', (SELECT id FROM email_sender LIMIT 1),
        '[SPRUCE] There was an error routing prescription to pharmacy',
        '[SPRUCE] There was an error routing prescription to pharmacy', '', 1);

INSERT INTO email_template (type, name, sender_id, subject_template,
        body_text_template, body_html_template, active)
    VALUES ('notify-refill-rx-created', 'Default', (SELECT id FROM email_sender LIMIT 1),
        '[SPRUCE] You have a new refill request from a patient',
        '[SPRUCE] You have a new refill request from a patient', '', 1);
