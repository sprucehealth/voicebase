package api

import (
	"carefront/common"
	"carefront/libs/pharmacy"
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"log"
	"time"
)

func (d *DataService) RegisterPatient(accountId int64, firstName, lastName, gender, zipCode, city, state, phone string, dob time.Time) (*common.Patient, error) {
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

	_, err = tx.Exec(`insert into patient_phone (patient_id, phone, phone_type, status) values (?,?,?, 'ACTIVE')`, lastId, phone, patient_phone_type)
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

	tx.Commit()
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

	tx.Commit()
	return d.GetCareTeamForPatient(patientId)
}

func (d *DataService) GetPatientFromAccountId(accountId int64) (*common.Patient, error) {
	return d.getPatientBasedOnQuery(`select patient.id, account_id, first_name, last_name, zip_code,city,state, phone, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.account_id = ? and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
								and (zip_code is null or patient_location.status='ACTIVE')`, accountId, patient_phone_type)
}

func (d *DataService) GetPatientFromId(patientId int64) (*common.Patient, error) {
	return d.getPatientBasedOnQuery(`select patient.id, account_id, first_name, last_name, zip_code, city, state, phone, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.id = ? and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
								and (zip_code is null or patient_location.status='ACTIVE') `, patientId, patient_phone_type)
}

func (d *DataService) GetPatientFromPatientVisitId(patientVisitId int64) (*common.Patient, error) {
	var patient common.Patient
	var phone, zipCode sql.NullString
	var dob mysql.NullTime
	err := d.DB.QueryRow(`select patient.id, account_id, first_name, last_name, zip_code, phone, gender, dob, patient.status from patient_visit
							left outer join patient_phone on patient_phone.patient_id = patient_visit.patient_id
							left outer join patient_location on patient_location.patient_id = patient_visit.patient_id
							inner join patient on patient_visit.patient_id = patient.id where patient_visit.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, patientVisitId, patient_phone_type,
	).Scan(
		&patient.PatientId, &patient.AccountId, &patient.FirstName, &patient.LastName,
		&zipCode, &phone, &patient.Gender, &dob, &patient.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if phone.Valid {
		patient.Phone = phone.String
	}
	if dob.Valid {
		patient.Dob = dob.Time
	}
	if zipCode.Valid {
		patient.ZipCode = zipCode.String
	}

	return &patient, nil
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
	tx.Commit()
	return nil
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

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_id, source, name, address, city, state, zip_code, phone,lat,lng, status) values (?,?,?,?,?,?,?,?,?,?,?,?)`, patientId, pharmacyDetails.Id, pharmacyDetails.Source, pharmacyDetails.Name, pharmacyDetails.Address, pharmacyDetails.City, pharmacyDetails.State, pharmacyDetails.Postal, pharmacyDetails.Phone, pharmacyDetails.Latitude, pharmacyDetails.Longitude, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (d *DataService) GetPatientPharmacySelection(patientId int64) (pharmacySelection *pharmacy.PharmacyData, err error) {
	var id, sourceType, name, address, phone, city, state, zipCode, lat, lng sql.NullString
	err = d.DB.QueryRow(`select pharmacy_id, source, name, address, city, state, zip_code, phone,lat,lng from patient_pharmacy_selection where patient_id = ? and status=?`, patientId, status_active).Scan(&id, &sourceType, &name, &address, &city, &state, &zipCode, &phone, &lat, &lng)
	if err == sql.ErrNoRows {
		err = NoRowsError
		return
	}

	pharmacySelection = &pharmacy.PharmacyData{}
	pharmacySelection.Id = id.String
	pharmacySelection.Source = sourceType.String

	if address.Valid {
		pharmacySelection.Address = address.String
	}

	if city.Valid {
		pharmacySelection.City = city.String
	}

	if state.Valid {
		pharmacySelection.State = state.String
	}

	if zipCode.Valid {
		pharmacySelection.Postal = zipCode.String
	}

	if lat.Valid {
		pharmacySelection.Latitude = lat.String
	}

	if lng.Valid {
		pharmacySelection.Longitude = lng.String
	}

	if phone.Valid {
		pharmacySelection.Phone = phone.String
	}

	if name.Valid {
		pharmacySelection.Name = name.String
	}
	return
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

	tx.Commit()
	return nil
}

func (d *DataService) getPatientBasedOnQuery(queryStr string, queryParams ...interface{}) (*common.Patient, error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var phone, zipCode, city, state sql.NullString
	var patientId, accountId int64
	err := d.DB.QueryRow(queryStr, queryParams...).Scan(&patientId, &accountId, &firstName, &lastName, &zipCode, &city, &state, &phone, &gender, &dob, &status)
	if err != nil {
		return nil, err
	}
	patient := &common.Patient{
		PatientId: patientId,
		FirstName: firstName,
		LastName:  lastName,
		Status:    status,
		Gender:    gender,
		AccountId: accountId,
	}
	if phone.Valid {
		patient.Phone = phone.String
	}
	if dob.Valid {
		patient.Dob = dob.Time
	}
	if zipCode.Valid {
		patient.ZipCode = zipCode.String
	}
	if city.Valid {
		patient.City = city.String
	}
	if state.Valid {
		patient.State = state.String
	}
	return patient, nil
}
