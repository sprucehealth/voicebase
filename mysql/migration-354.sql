-- Move message into treatment plan since it's unique (only one) and always used along with the treatment plan
ALTER TABLE treatment_plan ADD COLUMN note TEXT;
UPDATE treatment_plan tp SET note = (SELECT message FROM doctor_treatment_message m WHERE m.treatment_plan_id = tp.id);
DROP TABLE doctor_treatment_message;

-- Copy the doctor's saved message into all existing favorite treatment plans
ALTER TABLE dr_favorite_treatment_plan ADD COLUMN note TEXT;
UPDATE dr_favorite_treatment_plan tp SET note = (SELECT message FROM doctor_saved_case_message m WHERE m.doctor_id = tp.doctor_id);
DROP TABLE doctor_saved_case_message;
