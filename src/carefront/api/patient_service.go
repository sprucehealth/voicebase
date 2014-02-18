package api

import (
	"carefront/common"
	"carefront/libs/pharmacy"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"
)

func (d *DataService) RegisterPatient(accountId int64, firstName, lastName, gender, zipCode, city, state, phone, phoneType string, dob time.Time) (*common.Patient, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(`insert into patient (account_id, first_name, last_name, gender, dob, status) 
								values (?, ?, ?,  ?, ?, 'REGISTERED')`, accountId, firstName, lastName, gender, dob)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return nil, err
	}

	_, err = tx.Exec(`insert into patient_phone (patient_id, phone, phone_type, status) values (?,?,?, 'ACTIVE')`, lastId, phone, phoneType)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec(`insert into patient_location (patient_id, zip_code, city, state, status) 
									values (?, ?, ?, ?, 'ACTIVE')`, lastId, zipCode, city, state)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.GetPatientFromId(lastId)
}
func (d *DataService) GetPatientIdFromAccountId(accountId int64) (int64, error) {
	var patientId int64
	err := d.DB.QueryRow("select id from patient where account_id = ?", accountId).Scan(&patientId)
	return patientId, err
}

func (d *DataService) CheckCareProvidingElligibility(shortState string, healthConditionId int64) (doctorId int64, err error) {
	rows, err := d.DB.Query(`select provider_id from care_provider_state_elligibility 
								inner join care_providing_state on care_providing_state_id = care_providing_state.id 
								inner join provider_role on provider_role_id = provider_role.id 
									where state = ? and health_condition_id = ? and provider_tag='DOCTOR'`, shortState, healthConditionId)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	doctorIds := make([]int64, 0)
	for rows.Next() {
		var doctorId int64
		rows.Scan(&doctorId)
		doctorIds = append(doctorIds, doctorId)
	}

	if len(doctorIds) == 0 {
		return 0, nil
	}

	return doctorIds[0], nil
}

func (d *DataService) UpdatePatientWithERxPatientId(patientId, erxPatientId int64) error {
	_, err := d.DB.Exec(`update patient set erx_patient_id = ? where id = ? `, erxPatientId, patientId)
	return err
}

func (d *DataService) GetCareTeamForPatient(patientId int64) (*common.PatientCareProviderGroup, error) {
	rows, err := d.DB.Query(`select patient_care_provider_group.id as group_id, patient_care_provider_assignment.id as assignment_id, provider_tag, 
								created_date, modified_date,provider_id, patient_care_provider_group.status as group_status, 
								patient_care_provider_assignment.status as assignment_status from patient_care_provider_assignment 
									inner join patient_care_provider_group on assignment_group_id = patient_care_provider_group.id 
									inner join provider_role on provider_role.id = provider_role_id 
									where patient_care_provider_group.patient_id=?`, patientId)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var careTeam *common.PatientCareProviderGroup
	for rows.Next() {
		var groupId, assignmentId, providerId int64
		var providerTag, groupStatus, assignmentStatus string
		var createdDate, modifiedDate mysql.NullTime
		rows.Scan(&groupId, &assignmentId, &providerTag, &createdDate, &modifiedDate, &providerId, &groupStatus, &assignmentStatus)
		if careTeam == nil {
			careTeam = &common.PatientCareProviderGroup{}
			careTeam.Id = groupId
			careTeam.PatientId = patientId
			if createdDate.Valid {
				careTeam.CreationDate = createdDate.Time
			}
			if modifiedDate.Valid {
				careTeam.ModifiedDate = modifiedDate.Time
			}
			careTeam.Status = groupStatus
			careTeam.Assignments = make([]*common.PatientCareProviderAssignment, 0)
		}

		patientCareProviderAssignment := &common.PatientCareProviderAssignment{
			Id:           assignmentId,
			ProviderRole: providerTag,
			ProviderId:   providerId,
			Status:       assignmentStatus,
		}

		careTeam.Assignments = append(careTeam.Assignments, patientCareProviderAssignment)
	}

	return careTeam, nil
}

func (d *DataService) CreateCareTeamForPatientWithPrimaryDoctor(patientId, doctorId int64) (*common.PatientCareProviderGroup, error) {
	var providerRoleId int64
	err := d.DB.QueryRow(`select id from provider_role where provider_tag=?`, DOCTOR_ROLE).Scan(&providerRoleId)
	if err != nil {
		return nil, err
	}

	return d.createProviderAssignmentForPatient(patientId, doctorId, providerRoleId)
}

func (d *DataService) createProviderAssignmentForPatient(patientId, providerId, providerRoleId int64) (*common.PatientCareProviderGroup, error) {

	// create new group assignment for patient visit
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}

	res, err := tx.Exec(`insert into patient_care_provider_group (patient_id, status) values (?, ?)`, patientId, status_creating)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// create new assignment for patient
	_, err = tx.Exec("insert into patient_care_provider_assignment (patient_id, provider_role_id, provider_id, assignment_group_id, status) values (?, ?, ?, ?, 'PRIMARY')", patientId, providerRoleId, providerId, lastInsertId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// update group assignment to be the active group assignment for this patient visit
	_, err = tx.Exec(`update patient_care_provider_group set status='ACTIVE' where id=?`, lastInsertId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.GetCareTeamForPatient(patientId)
}

func (d *DataService) CreateCareTeamForPatient(patientId int64) (*common.PatientCareProviderGroup, error) {
	// identify providers in the state required. Assuming for now that we can only have one provider in the
	// state of CA. The reason for this assumption is that we have not yet figured out how best to deal with
	// multiple active doctors in how they will be assigned to the patient.
	// TODO : Update care team formation when we have more than 1 doctor that we can have as active in our system
	var providerId, providerRoleId int64
	err := d.DB.QueryRow(`select provider_id, provider_role_id from care_provider_state_elligibility 
					inner join care_providing_state on care_providing_state_id = care_providing_state.id
					where state = 'CA'`).Scan(&providerId, &providerRoleId)

	if err == sql.ErrNoRows {
		return nil, NoElligibileProviderInState
	} else if err != nil {
		return nil, err
	}

	return d.createProviderAssignmentForPatient(patientId, providerId, providerRoleId)
}

func (d *DataService) GetPatientFromAccountId(accountId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id, account_id, first_name, last_name, zip_code,city,state, phone, phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.account_id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (zip_code is null or patient_location.status='ACTIVE')`, accountId)
	if len(patients) > 0 {
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromId(patientId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id, account_id, first_name, last_name, zip_code, city, state, phone,phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (zip_code is null or patient_location.status='ACTIVE') `, patientId)

	if len(patients) > 0 {
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientsForIds(patientIds []int64) ([]*common.Patient, error) {
	if len(patientIds) == 0 {
		return nil, nil
	}

	return d.getPatientBasedOnQuery(fmt.Sprintf(`select patient.id, patient.erx_patient_id, account_id, first_name, last_name, zip_code, city, state, phone,phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.id in (%s) and (phone is null or (patient_phone.status='ACTIVE'))
								and (zip_code is null or patient_location.status='ACTIVE')`, enumerateItemsIntoString(patientIds)))
}

func (d *DataService) GetPatientFromTreatmentPlanId(treatmentPlanId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id, account_id, first_name, last_name, zip_code, city, state, phone,phone_type, gender, dob, patient.status from treatment_plan 
							inner join patient_visit on patient_visit_id = patient_visit.id
							inner join patient on patient.id = patient_visit.patient_id
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where treatment_plan.id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (zip_code is null or patient_location.status='ACTIVE') `, treatmentPlanId)
	if len(patients) > 0 {
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromPatientVisitId(patientVisitId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id, account_id, first_name, last_name, zip_code,city,state, phone, phone_type, gender, dob, patient.status from patient_visit
							inner join patient on patient_visit.patient_id = patient.id 
							left outer join patient_phone on patient_phone.patient_id = patient_visit.patient_id
							left outer join patient_location on patient_location.patient_id = patient_visit.patient_id
							where patient_visit.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE'))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, patientVisitId)
	if len(patients) > 0 {
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) UpdatePatientAddress(patientId int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// update any existing address for the address type as inactive
	_, err = tx.Exec(`update patient_address set status=? where patient_id = ? and address_type = ?`, status_inactive, addressType, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert new address
	if addressLine2 != "" {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, address_line_2, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?, ?)`, patientId, addressLine1, addressLine2, city, state, zipCode, addressType, status_active)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?)`, patientId, addressLine1, city, state, zipCode, addressType, status_active)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) UpdatePatientPharmacy(patientId int64, pharmacyDetails *pharmacy.PharmacyData) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update patient_pharmacy_selection set status=? where patient_id = ?`, status_inactive, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// lookup pharmacy by its id to see if it already exists in the database
	var existingPharmacyId int64
	err = tx.QueryRow(`select id from pharmacy_selection where pharmacy_id = ?`, pharmacyDetails.Id).Scan(&existingPharmacyId)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	if existingPharmacyId == 0 {
		lastId, err := tx.Exec(`insert into pharmacy_selection (pharmacy_id, source, address_line_1, city, state, zip_code, phone, name) values (?,?,?,?,?,?,?,?) `,
			pharmacyDetails.Id, pharmacyDetails.Source, pharmacyDetails.Address, pharmacyDetails.City, pharmacyDetails.State, pharmacyDetails.Postal, pharmacyDetails.Phone, pharmacyDetails.Name)
		if err != nil {
			tx.Rollback()
			return err
		}

		existingPharmacyId, err = lastId.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patientId, existingPharmacyId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPatientPharmacySelection(patientId int64) (pharmacySelection *pharmacy.PharmacyData, err error) {
	rows, err := d.DB.Query(`select pharmacy_selection.id, patient_id, pharmacy_selection.pharmacy_id, source, name, address_line_1, city, state, zip_code, phone,lat,lng 
		from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id 
				where patient_id = ? and status=?`, patientId, status_active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		pharmacySelection, err = getPharmacyFromCurrentRow(rows)
	}

	return
}

func (d *DataService) GetPharmacySelectionForPatients(patientIds []int64) ([]*pharmacy.PharmacyData, error) {
	if len(patientIds) == 0 {
		return nil, nil
	}

	rows, err := d.DB.Query(fmt.Sprintf(`select pharmacy_selection.id, patient_id,  pharmacy_selection.pharmacy_id, source, name, address_line_1, city, state, zip_code, phone,lat,lng 
			from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id where patient_id in (%s) and status=?`, enumerateItemsIntoString(patientIds)), status_active)
	if err != nil {
		return nil, err
	}

	pharmacies := make([]*pharmacy.PharmacyData, 0)
	for rows.Next() {
		pharmacySelection, err := getPharmacyFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		pharmacies = append(pharmacies, pharmacySelection)
	}

	return pharmacies, nil
}

func getPharmacyFromCurrentRow(rows *sql.Rows) (*pharmacy.PharmacyData, error) {
	var localId, patientId int64
	var id, sourceType, name, address, phone, city, state, zipCode, lat, lng sql.NullString
	err := rows.Scan(&localId, &patientId, &id, &sourceType, &name, &address, &city, &state, &zipCode, &phone, &lat, &lng)
	if err != nil {
		return nil, err
	}

	pharmacySelection := &pharmacy.PharmacyData{
		LocalId:   localId,
		PatientId: patientId,
		Id:        id.String,
		Source:    sourceType.String,
		Address:   address.String,
		City:      city.String,
		State:     state.String,
		Postal:    zipCode.String,
		Latitude:  lat.String,
		Longitude: lng.String,
		Phone:     phone.String,
		Name:      name.String,
	}

	return pharmacySelection, nil
}

func (d *DataService) TrackPatientAgreements(patientId int64, agreements map[string]bool) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for agreementType, agreed := range agreements {
		_, err = tx.Exec(`update patient_agreement set status=? where patient_id = ? and agreement_type = ?`, status_inactive, patientId, agreementType)
		if err != nil {
			tx.Rollback()
			return err
		}

		var agreedBit int64
		if agreed == true {
			agreedBit = 1
		}

		_, err = tx.Exec(`insert into patient_agreement (patient_id, agreement_type,agreed, status) values (?,?,?,?)`, patientId, agreementType, agreedBit, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) getPatientBasedOnQuery(queryStr string, queryParams ...interface{}) ([]*common.Patient, error) {
	rows, err := d.DB.Query(queryStr, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patients := make([]*common.Patient, 0)
	for rows.Next() {
		var firstName, lastName, status, gender string
		var dob mysql.NullTime
		var phone, phoneType, zipCode, city, state sql.NullString
		var erxPatientId sql.NullInt64
		var patientId, accountId int64
		err = rows.Scan(&patientId, &erxPatientId, &accountId, &firstName, &lastName, &zipCode, &city, &state, &phone, &phoneType, &gender, &dob, &status)
		if err != nil {
			return nil, err
		}

		patient := &common.Patient{
			PatientId:    common.NewObjectId(patientId),
			FirstName:    firstName,
			LastName:     lastName,
			Status:       status,
			Gender:       gender,
			AccountId:    common.NewObjectId(accountId),
			ERxPatientId: common.NewObjectId(erxPatientId.Int64),
			Phone:        phone.String,
			PhoneType:    phoneType.String,
			Dob:          dob.Time,
			ZipCode:      zipCode.String,
			City:         city.String,
			State:        state.String,
		}

		patients = append(patients, patient)
	}

	return patients, nil
}
