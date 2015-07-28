-- Making question_id nullable given that we are now adding alerts at the visit level
-- that are not direclty linked to question_id
ALTER TABLE patient_alerts DROP FOREIGN KEY patient_alerts_ibfk_3;
ALTER TABLE patient_alerts MODIFY COLUMN question_id INT UNSIGNED;
ALTER TABLE patient_alerts ADD FOREIGN KEY (question_id) REFERENCES question(id);
