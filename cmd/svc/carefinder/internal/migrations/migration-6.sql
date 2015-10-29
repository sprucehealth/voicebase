CREATE TABLE doctor_master_description (
	doctor_id text not null references carefinder_doctor_info(id),
	first_name text not null,
	last_name text not null,
	editorial_description text not null,
	seo_description text not null
);

ALTER TABLE carefinder_doctor_info ADD COLUMN seo_description TEXT;

