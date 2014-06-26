-- Specifically indicating the timestamp to be nullable so that default properties for mysql are supressed
alter table patient_care_provider_assignment modify column expires timestamp(6) NULL;
alter table patient_case_care_provider_assignment modify column expires timestamp(6) NULL;