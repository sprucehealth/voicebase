UPDATE patient_case_message_attachment
SET item_type='followup_visit'
WHERE item_type='visit';
