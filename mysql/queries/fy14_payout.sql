--- Below are sql queries used to identify the list of patient encounters
--- to consider in the FY2014 payout to doctors

-- List of all encounters in 2014 including the doctor name and state of patient encounter
SELECT dt.id, concat('"', d.first_name, ' ' , d.last_name, '"'), pl.state, sku_id, created 
FROM doctor_transaction dt
INNER JOIN doctor d ON d.id = doctor_id 
INNER JOIN patient_location pl ON pl.patient_id = dt.patient_id 
WHERE created >= ('2014-09-25') and created < ('2015-01-01');

-- Group the encounters by state 
SELECT concat(d.first_name, ' ' , d.last_name) as name, pl.state as state, COUNT(state) as count
FROM doctor_transaction dt
INNER JOIN doctor d ON d.id = doctor_id 
INNER JOIN patient_location pl ON pl.patient_id = dt.patient_id 
WHERE created >= ('2014-09-25') and created < ('2015-01-01')
GROUP BY state, name 
ORDER BY name;


