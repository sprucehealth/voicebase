-- Identifying patients for the abandoned cart email
SELECT email, patient.first_name, patient.last_name, account.registration_date
FROM patient_visit v
INNER JOIN patient ON patient.id = v.patient_id
INNEr JOIN account ON account.id = account_id
WHERE v.creation_date >= '2014-12-04 07:59:00'
    AND v.creation_date < '2014-12-15 07:59:00'
    AND v.submitted_date IS NULL
    AND v.status = 'OPEN'
    AND v.sku_id = (select id from sku where type='acne_visit');


-- Inserting account credit for patients in the abandoned cart
start transaction;

INSERT INTO account_credit_history (account_id, credit, description)
    SELECT p.account_id, 1000, 'ac_20141204_20141215'
    FROM patient_visit v
    INNER JOIN patient p ON p.id = v.patient_id
    WHERE v.creation_date >= ('2014-12-04 07:59:00')
        AND v.creation_date < ('2014-12-15 07:59:00')
        AND v.submitted_date IS NULL
        AND v.status = 'OPEN'
        AND last_name not like '%Test%';


INSERT IGNORE INTO account_credit (account_id, credit, last_checked_account_credit_history_id)
    SELECT h.account_id, 1000, h.id 
    FROM account_credit_history h
    WHERE description = 'ac_20141204_20141215';

commit;
