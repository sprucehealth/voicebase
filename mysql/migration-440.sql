-- Insert the EMAIL communication preference for all patients.
INSERT INTO communication_preference (account_id, communication_type, status) SELECT account_id, 'EMAIL', 'ACTIVE' FROM patient;
