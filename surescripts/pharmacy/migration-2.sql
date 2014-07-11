-- Purpose of these migrations is to create a mapping between the surescripts database pharmacy ids and 
-- the dosespot staging environment ids so that we are able to successfully route prescriptions to the test
-- pharmacies in non-production environments, and the code is mostly agnostic to how we map the data. 
-- This gives us the ability to play with real pharmacies on the patient app in non-production environments
create table pharmacy_test_data_mapping (
	pharmacy_id integer references pharmacy(id),
	dosespot_test_id integer
);

\copy pharmacy_test_data_mapping FROM '/Users/kunaljham/Dropbox/personal/workspace/backend/surescripts_pharmacy/dosespot_surescripts_test_id_mapping.out';

alter table pharmacy_test_data_mapping add column ncpdpid integer;

update pharmacy_test_data_mapping set ncpdpid = pharmacy.ncpdpid from pharmacy 
	where pharmacy.id = pharmacy_test_data_mapping.pharmacy_id; 