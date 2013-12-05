namespace go carefront.thrift.api

include "common.thrift"

struct Patient {
	1: required i64 patient_id
	2: required i64 account_id
	3: optional string first_name
	4: optional string last_name
	5: optional i64 dob
	6: optional string gender
	7: optional string zipcode
}

exception DoesNotExist {

}

service PatientAPI {
	list<Patient> get_patients_for_account(
		1: required i64 account_id,
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity)

	Patient get_patient_from_id(
		1: required i64 patient_id
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity,
		4: DoesNotExist does_not_exist)

	// Return patient ID
	i64 register_patient(
		1: required Patient patient // patient_id in struct is ignored
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity)

	i64 create_new_patient_visit(
		1: required i64 patient_id,
		2: required i64 health_condition_id,
		3: required i64 layout_version_id
	) throws (
		1: common.InternalServerError error,
		2: common.AccessDenied access_denied,
		3: common.OverCapacity over_capacity)
}
