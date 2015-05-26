CREATE INDEX status_enqueue_date ON doctor_queue (status, enqueue_date);
CREATE INDEX doctor_status_enqueue_date ON doctor_queue (doctor_id, status, enqueue_date);
DROP INDEX doctor_id ON doctor_queue;
ALTER TABLE patient_feedback MODIFY comment TEXT CHARSET utf8mb4;