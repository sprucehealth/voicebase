-- Create the base events tables to event storage
CREATE TABLE server_event (
  name VARCHAR(255) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  session_id INTEGER,
  account_id INTEGER,
  patient_id INTEGER,
  doctor_id INTEGER,
  visit_id INTEGER,
  case_id INTEGER,
  treatment_plan_id INTEGER,
  role VARCHAR(255),
  extra_json TEXT
);

-- Index these tablea on each dimension as we will want to query it in all sorts of different ways
CREATE INDEX name_idx ON server_event (name);
CREATE INDEX timestamp_idx ON server_event (timestamp);
CREATE INDEX session_id_idx ON server_event (session_id);
CREATE INDEX account_id_idx ON server_event (account_id);
CREATE INDEX patient_id_idx ON server_event (patient_id);
CREATE INDEX doctor_id_idx ON server_event (doctor_id);
CREATE INDEX visit_id_idx ON server_event (visit_id);
CREATE INDEX case_id_idx ON server_event (case_id);
CREATE INDEX treatment_plan_id_idx ON server_event (treatment_plan_id);
CREATE INDEX role_idx ON server_event (role);

CREATE TABLE client_event (
  name VARCHAR(255) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  error TEXT,
  session_id UUID NOT NULL,
  device_id UUID NOT NULL,
  account_id INTEGER,
  patient_id INTEGER,
  doctor_id INTEGER,
  visit_id INTEGER,
  case_id INTEGER,
  screen_id UUID,
  question_id INTEGER,
  time_spent DECIMAL,
  app_type VARCHAR(255),
  app_env VARCHAR(255),
  app_build VARCHAR(255),
  platform VARCHAR(255),
  platform_version VARCHAR(255),
  device_type VARCHAR(255),
  device_model VARCHAR(255),
  screen_width INTEGER,
  screen_height INTEGER,
  screen_resolution VARCHAR(255),
  extra_json TEXT
);

CREATE TABLE web_request_event (
  service VARCHAR(255) NOT NULL,
  path VARCHAR(255) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  request_id BIGINT NOT NULL,
  status_code INTEGER NOT NULL,
  method VARCHAR(255) NOT NULL,
  url VARCHAR(255) NOT NULL,
  remote_addr VARCHAR(255),
  content_type VARCHAR(255),
  user_agent VARCHAR(255),
  referrer VARCHAR(255),
  response_time INTEGER NOT NULL,
  server VARCHAR(255) NOT NULL,
  account_id INTEGER,
  device_id UUID
);