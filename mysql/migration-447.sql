-- Move all transmission errors over to the care coordinator's account
SET @primary_cc_id = (SELECT id from doctor where account_id = (SELECT id from account where role_type_id = (select id from role_type where role_type_tag='MA') ORDER BY id limit 1));

UPDATE doctor_queue 
SET doctor_id = @primary_cc_id
WHERE status='PENDING'
AND event_type like '%TRANSMISSION_ERROR%';
