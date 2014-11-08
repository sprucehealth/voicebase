start transaction;

INSERT INTO account_credit_history (account_id, credit, description)
	SELECT p.account_id, 1000, 'abandoned cart' 
	FROM patient_visit v
	INNER JOIN patient p ON p.id = v.patient_id
	WHERE v.creation_date >= ('2014-10-10 19:31:34')
	    AND v.creation_date < ('2014-11-05 17:00:00')
	    AND v.submitted_date IS NULL
	    AND v.status = 'OPEN'
	    AND v.sku_id = (select id from sku where type='acne_visit');


INSERT INTO account_credit (account_id, credit, last_checked_account_credit_history_id)
	(SELECT h.account_id, 1000, h.id 
	FROM account_credit_history h
	WHERE description='abandoned cart')
	ON DUPLICATE KEY UPDATE account_credit.credit = account_credit.credit + 1000;


commit;