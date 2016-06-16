CREATE TABLE diagnosis_code (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL,
	name TEXT NOT NULL,
	billable BOOLEAN NOT NULL,
	CONSTRAINT unique_code UNIQUE (code)
);

CREATE TABLE diagnosis_includes_notes (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

CREATE TABLE diagnosis_inclusion_term (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

CREATE TABLE diagnosis_excludes1_note (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

CREATE TABLE diagnosis_excludes2_note (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

CREATE TABLE diagnosis_use_additional_code_note (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

CREATE TABLE diagnosis_code_first_note (
	id SERIAL PRIMARY KEY,
	diagnosis_code_id TEXT NOT NULL REFERENCES diagnosis_code(id) ON DELETE CASCADE,
	note TEXT NOT NULL
);

-- GIN Index for searching against ICD10 diagnosis codes 
CREATE INDEX code_search_index ON diagnosis_code USING gin(to_tsvector('english', code));

-- Searching against document containing name, includes notes, and inclusion terms (that are appropriately weighted)
CREATE MATERIALIZED VIEW diagnosis_search_index AS 
SELECT dc.id as did, dc.name as name, dc.code as code, dc.billable as billable,
		   setweight(to_tsvector('english', dc.name), 'A') || 
		   setweight(to_tsvector('english', dc.code), 'B') ||  
		   to_tsvector('english', coalesce(string_agg(din.note, ' '),'')) || 
		   to_tsvector('english', coalesce(string_agg(dit.note, ' '), '')) as document
	FROM diagnosis_code dc
	LEFT OUTER JOIN diagnosis_includes_notes din ON dc.id = din.diagnosis_code_id
	LEFT OUTER JOIN diagnosis_inclusion_term dit ON dc.id = dit.diagnosis_code_id
	GROUP BY dc.id;

CREATE INDEX idx_fts_search ON diagnosis_search_index USING gin(document);

-- Using extension for fuzzy string matching as fallback
create extension pg_trgm;

-- Creating a list of unique lexemes in each document to do a similarity search against
CREATE MATERIALIZED VIEW diagnosis_unique_lexeme AS
SELECT word FROM ts_stat('SELECT to_tsvector(''english'', dc.name) || 
    to_tsvector(''english'', coalesce(string_agg(din.note, '' ''),'''')) ||
    to_tsvector(''english'', coalesce(string_agg(dit.note, '' ''),''''))
	FROM diagnosis_code dc
	LEFT OUTER JOIN diagnosis_includes_notes din ON dc.id = din.diagnosis_code_id
	LEFT OUTER JOIN diagnosis_inclusion_term dit ON dc.id = dit.diagnosis_code_id
	GROUP BY dc.id ');

CREATE INDEX idx_diagnosis_unique_lexeme_search ON diagnosis_unique_lexeme USING gin(word gin_trgm_ops);





