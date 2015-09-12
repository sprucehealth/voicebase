CREATE TABLE ftp_pathway_membership (
	dr_favorite_treatment_plan_id INT UNSIGNED NOT NULL,
	pathway_id INT UNSIGNED NOT NULL,
	creation_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (dr_favorite_treatment_plan_id, pathway_id),
	CONSTRAINT dr_favorite_treatment_plan_id  FOREIGN KEY (dr_favorite_treatment_plan_id) REFERENCES dr_favorite_treatment_plan(id),
	CONSTRAINT pathway_id FOREIGN KEY (pathway_id) REFERENCES clinical_pathway(id)
);