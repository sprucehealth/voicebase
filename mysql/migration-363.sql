ALTER TABLE diagnosis_details_intake DROP FOREIGN KEY diagnosis_details_intake_ibfk_3;
ALTER TABLE diagnosis_details_intake ADD FOREIGN KEY (layout_version_id) REFERENCES diagnosis_details_layout(id);