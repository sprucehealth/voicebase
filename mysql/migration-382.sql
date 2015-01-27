ALTER TABLE app_version_layout_mapping DROP KEY layout_major;
ALTER TABLE app_version_layout_mapping ADD UNIQUE KEY `major_platform_role_sku_pathway_key` (layout_major, platform, role, purpose, sku_id, clinical_pathway_id);

ALTER TABLE patient_doctor_layout_mapping DROP KEY dr_major;
ALTER TABLE patient_doctor_layout_mapping ADD UNIQUE KEY `dr_patient_sku_pathway` (dr_major, dr_minor, patient_major, patient_minor, sku_id, clinical_pathway_id);

-- Add the ability to mark a doctor as being unavailable in a state so that we have the flexibility to 
-- turn doctors off on a per state as well as global basis
ALTER TABLE care_provider_state_elligibility ADD COLUMN unavailable TINYINT(1) NOT NULL DEFAULT 0;

-- Adding a key for quick lookup of a list of available registered for a pathway in a particular state
ALTER TABLE care_provider_state_elligibility ADD KEY `eligible_doctor_lookup` (role_type_id, care_providing_state_id, unavailable);	
