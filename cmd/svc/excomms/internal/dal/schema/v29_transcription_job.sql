CREATE TABLE transcription_job (
	media_id VARCHAR(255) NOT NULL,
	job_id VARCHAR(255) NOT NULL,
	created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	completed TINYINT(1) NOT NULL DEFAULT 0, 
	timed_out TINYINT(1) NOT NULL DEFAULT 0,
	available_after TIMESTAMP NOT NULL,
	completed_timestamp TIMESTAMP,
	PRIMARY KEY (media_id)
);
