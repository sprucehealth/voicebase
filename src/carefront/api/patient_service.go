package api

import (
	"carefront/common"
	"carefront/libs/pharmacy"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func (d *DataService) RegisterPatient(patient *common.Patient) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	if err := createPatientWithStatus(patient, PATIENT_REGISTERED, tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CreateUnlinkedPatientFromRefillRequest(patient *common.Patient) error {
	tx, err := d.DB.Begin()

	// create an account with no email and password for the unmatched patient
	lastId, err := tx.Exec(`insert into account (email, password) values (NULL,NULL)`)
	if err != nil {
		tx.Rollback()
		return err
	}

	accountId, err := lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	patient.AccountId = common.NewObjectId(accountId)

	// create an account
	if err := createPatientWithStatus(patient, PATIENT_UNLINKED, tx); err != nil {
		tx.Rollback()
		return err
	}

	// create address for patient
	if patient.PatientAddress != nil {
		addressId, err := d.addAddress(tx, patient.PatientAddress)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_address_selection (address_id, patient_id, is_default, is_updated_by_doctor) values (?,?,1,0)`, addressId, patient.PatientId.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if patient.Pharmacy != nil {
		var existingPharmacyId int64
		err = tx.QueryRow(`select id from pharmacy_selection where pharmacy_id = ?`, patient.Pharmacy.SourceId).Scan(&existingPharmacyId)
		if err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return err
		}

		if existingPharmacyId == 0 {
			err = addPharmacy(patient.Pharmacy, tx)
			if err != nil {
				tx.Rollback()
				return err
			}
			existingPharmacyId = patient.Pharmacy.LocalId
		}

		_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patient.PatientId.Int64(), existingPharmacyId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// create additional phone numbers for patient
	if len(patient.PhoneNumbers) > 1 {
		for _, phoneNumber := range patient.PhoneNumbers[1:] {
			_, err = tx.Exec(`insert into patient_phone (patient_id, phone, phone_type, status) value (?,?,?,?)`, patient.PatientId.Int64(), phoneNumber.Phone, phoneNumber.PhoneType, STATUS_INACTIVE)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// assign the erx patient id to the patient
	_, err = tx.Exec(`update patient set erx_patient_id = ? where id = ?`, patient.ERxPatientId.Int64(), patient.PatientId.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func createPatientWithStatus(patient *common.Patient, status string, tx *sql.Tx) error {
	res, err := tx.Exec(`insert into patient (account_id, first_name, last_name, gender, dob, status) 
								values (?, ?, ?, ?, ?, ?)`, patient.AccountId.Int64(), patient.FirstName, patient.LastName, patient.Gender, patient.Dob, status)
	if err != nil {
		tx.Rollback()
		return err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return err
	}

	if len(patient.PhoneNumbers) > 0 {
		_, err = tx.Exec(`insert into patient_phone (patient_id, phone, phone_type, status) values (?,?,?, 'ACTIVE')`, lastId, patient.PhoneNumbers[0].Phone, patient.PhoneNumbers[0].PhoneType)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`insert into patient_location (patient_id, zip_code, city, state, status) 
									values (?, ?, ?, ?, 'ACTIVE')`, lastId, patient.ZipCode, patient.City, patient.State)
	if err != nil {
		tx.Rollback()
		return err
	}

	patient.PatientId = common.NewObjectId(lastId)
	return nil
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
	if rows.Err() != nil {
		return 0, rows.Err()
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

	return careTeam, rows.Err()
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

	res, err := tx.Exec(`insert into patient_care_provider_group (patient_id, status) values (?, ?)`, patientId, STATUS_CREATING)
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
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, 
													patient_location.zip_code, patient_location.city, patient_location.state, 
													phone, phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id		
							where patient.account_id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (patient_location.zip_code is null or patient_location.status='ACTIVE')`, accountId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromId(patientId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, 
													patient_location.zip_code, patient_location.city, patient_location.state, 
													phone, phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id
							where patient.id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (patient_location.zip_code is null or patient_location.status='ACTIVE')`, patientId)

	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientsForIds(patientIds []int64) ([]*common.Patient, error) {
	if len(patientIds) == 0 {
		return nil, nil
	}

	return d.getPatientBasedOnQuery(fmt.Sprintf(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, 
													patient_location.zip_code, patient_location.city, patient_location.state, 
													phone, phone_type, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id
							where patient.id in (%s) and (phone is null or (patient_phone.status='ACTIVE'))
								and (patient_location.zip_code is null or patient_location.status='ACTIVE') `, enumerateItemsIntoString(patientIds)))
}

func (d *DataService) GetPatientFromTreatmentPlanId(treatmentPlanId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, zip_code, city, state, phone,phone_type, gender, dob, patient.status from treatment_plan 
							inner join patient_visit on patient_visit_id = patient_visit.id
							inner join patient on patient.id = patient_visit.patient_id
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id
							where treatment_plan.id = ? and (phone is null or (patient_phone.status='ACTIVE'))
								and (zip_code is null or patient_location.status='ACTIVE') `, treatmentPlanId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromPatientVisitId(patientVisitId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email,first_name, middle_name, last_name, suffix, prefix, zip_code,city,state, phone, phone_type, gender, dob, patient.status from patient_visit
							inner join patient on patient_visit.patient_id = patient.id 
							left outer join patient_phone on patient_phone.patient_id = patient_visit.patient_id
							left outer join patient_location on patient_location.patient_id = patient_visit.patient_id
							left outer join account on account.id = patient.account_id							
							where patient_visit.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE'))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, patientVisitId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromErxPatientId(erxPatientId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, zip_code,city,state, phone, phone_type, gender, dob, patient.status from patient
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id							
							where patient.erx_patient_id = ? 
							and (phone is null or (patient_phone.status='ACTIVE'))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, erxPatientId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromRefillRequestId(refillRequestId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, zip_code,city,state, phone, phone_type, gender, dob, patient.status from rx_refill_request
							inner join patient on rx_refill_request.patient_id = patient.id 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id							
							where rx_refill_request.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE'))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, refillRequestId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
		return patients[0], err
	}

	return nil, err
}

func (d *DataService) GetPatientFromTreatmentId(treatmentId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery(`select patient.id, patient.erx_patient_id,patient.payment_service_customer_id, account_id,account.email, first_name, middle_name, last_name, suffix, prefix, zip_code,city,state, phone, phone_type, gender, dob, patient.status from treatment
							inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
							inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
							inner join patient on patient_visit.patient_id = patient.id
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							left outer join account on account.id = patient.account_id							
							where treatment.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE'))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, treatmentId)
	if len(patients) > 0 {
		err = d.getAddressAndPhoneNumbersForPatient(patients[0])
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
	_, err = tx.Exec(`update patient_address set status=? where patient_id = ? and address_type = ?`, STATUS_INACTIVE, addressType, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert new address
	if addressLine2 != "" {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, address_line_2, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?, ?)`, patientId, addressLine1, addressLine2, city, state, zipCode, addressType, STATUS_ACTIVE)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?)`, patientId, addressLine1, city, state, zipCode, addressType, STATUS_ACTIVE)
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

	_, err = tx.Exec(`update patient_pharmacy_selection set status=? where patient_id = ?`, STATUS_INACTIVE, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// lookup pharmacy by its id to see if it already exists in the database
	var existingPharmacyId int64
	err = tx.QueryRow(`select id from pharmacy_selection where pharmacy_id = ?`, pharmacyDetails.SourceId).Scan(&existingPharmacyId)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	if existingPharmacyId == 0 {
		err = addPharmacy(pharmacyDetails, tx)
		if err != nil {
			tx.Rollback()
			return err
		}
		existingPharmacyId = pharmacyDetails.LocalId
	}

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patientId, existingPharmacyId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) getPatientPharmacySelection(patientId int64) (pharmacySelection *pharmacy.PharmacyData, err error) {
	rows, err := d.DB.Query(`select pharmacy_selection.id, patient_id, pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
		from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id 
				where patient_id = ? and status=?`, patientId, STATUS_ACTIVE)
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

	rows, err := d.DB.Query(fmt.Sprintf(`select pharmacy_selection.id, patient_id,  pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
			from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id where patient_id in (%s) and status=?`, enumerateItemsIntoString(patientIds)), STATUS_ACTIVE)
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

	return pharmacies, rows.Err()
}

func (d *DataService) GetPharmacyBasedOnReferenceIdAndSource(pharmacyId, pharmacySource string) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var id int64
	err := d.DB.QueryRow(`select id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat,lng
		from pharmacy_selection where pharmacy_id = ? and source = ?`, pharmacyId, pharmacySource).
		Scan(&id, &addressLine1, &addressLine2, &city, &state, &country, &phone, &zipCode, &name, &lat, &lng)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	pharmacyToReturn := &pharmacy.PharmacyData{
		LocalId:      id,
		SourceId:     pharmacyId,
		Source:       pharmacySource,
		Name:         name.String,
		AddressLine1: addressLine1.String,
		AddressLine2: addressLine2.String,
		City:         city.String,
		State:        state.String,
		Country:      country.String,
		Postal:       zipCode.String,
		Phone:        phone.String,
	}

	if lat.Valid {
		latFloat, _ := strconv.ParseFloat(lat.String, 64)
		pharmacyToReturn.Latitude = latFloat
	}

	if lng.Valid {
		lngFloat, _ := strconv.ParseFloat(lng.String, 64)
		pharmacyToReturn.Longitude = lngFloat
	}

	return pharmacyToReturn, nil
}

func (d *DataService) GetPharmacyFromId(pharmacyLocalId int64) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var source, pharmacyReferenceId string
	err := d.DB.QueryRow(`select source, pharmacy_id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat,lng
		from pharmacy_selection where id = ?`, pharmacyLocalId).
		Scan(&source, &pharmacyReferenceId, &addressLine1, &addressLine2, &city, &state, &country, &phone, &zipCode, &name, &lat, &lng)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	pharmacyToReturn := &pharmacy.PharmacyData{
		LocalId:      pharmacyLocalId,
		SourceId:     pharmacyReferenceId,
		Source:       source,
		Name:         name.String,
		AddressLine1: addressLine1.String,
		AddressLine2: addressLine2.String,
		City:         city.String,
		State:        state.String,
		Country:      country.String,
		Postal:       zipCode.String,
		Phone:        phone.String,
	}

	if lat.Valid {
		latFloat, _ := strconv.ParseFloat(lat.String, 64)
		pharmacyToReturn.Latitude = latFloat
	}

	if lng.Valid {
		lngFloat, _ := strconv.ParseFloat(lng.String, 64)
		pharmacyToReturn.Longitude = lngFloat
	}

	return pharmacyToReturn, nil
}

func (d *DataService) AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	if err := addPharmacy(pharmacyDetails, tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func addPharmacy(pharmacyDetails *pharmacy.PharmacyData, tx *sql.Tx) error {
	columnsAndData := map[string]interface{}{
		"pharmacy_id":    pharmacyDetails.SourceId,
		"source":         pharmacyDetails.Source,
		"name":           pharmacyDetails.Name,
		"address_line_1": pharmacyDetails.AddressLine1,
		"city":           pharmacyDetails.City,
		"state":          pharmacyDetails.State,
		"zip_code":       pharmacyDetails.Postal,
		"phone":          pharmacyDetails.Phone,
	}

	if pharmacyDetails.AddressLine2 != "" {
		columnsAndData["address_line_2"] = pharmacyDetails.AddressLine2
	}

	if pharmacyDetails.Latitude != 0 {
		columnsAndData["lat"] = strconv.FormatFloat(pharmacyDetails.Latitude, 'f', -1, 64)
	}

	if pharmacyDetails.Longitude != 0 {
		columnsAndData["lng"] = strconv.FormatFloat(pharmacyDetails.Longitude, 'f', -1, 64)
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)

	lastId, err := tx.Exec(fmt.Sprintf("insert into pharmacy_selection (%s) values (%s)", strings.Join(columns, ","),
		nReplacements(len(columns))), dataForColumns...)

	if err != nil {
		return err
	}

	lastInsertId, err := lastId.LastInsertId()
	if err != nil {
		return err
	}

	pharmacyDetails.LocalId = lastInsertId
	return nil
}

func getPharmacyFromCurrentRow(rows *sql.Rows) (*pharmacy.PharmacyData, error) {
	var localId, patientId int64
	var id, sourceType, name, addressLine1, addressLine2, phone, city, state, zipCode, lat, lng sql.NullString
	err := rows.Scan(&localId, &patientId, &id, &sourceType, &name, &addressLine1, &addressLine2, &city, &state, &zipCode, &phone, &lat, &lng)
	if err != nil {
		return nil, err
	}

	pharmacySelection := &pharmacy.PharmacyData{
		LocalId:      localId,
		PatientId:    patientId,
		SourceId:     id.String,
		Source:       sourceType.String,
		AddressLine1: addressLine1.String,
		AddressLine2: addressLine2.String,
		City:         city.String,
		State:        state.String,
		Postal:       zipCode.String,
		Phone:        phone.String,
		Name:         name.String,
	}

	if lat.Valid {
		latFloat, _ := strconv.ParseFloat(lat.String, 64)
		pharmacySelection.Latitude = latFloat
	}

	if lng.Valid {
		lngFloat, _ := strconv.ParseFloat(lng.String, 64)
		pharmacySelection.Longitude = lngFloat
	}

	return pharmacySelection, nil
}

func (d *DataService) TrackPatientAgreements(patientId int64, agreements map[string]bool) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for agreementType, agreed := range agreements {
		_, err = tx.Exec(`update patient_agreement set status=? where patient_id = ? and agreement_type = ?`, STATUS_INACTIVE, patientId, agreementType)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_agreement (patient_id, agreement_type,agreed, status) values (?,?,?,?)`, patientId, agreementType, agreed, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) UpdatePatientWithPaymentCustomerId(patientId int64, paymentCustomerId string) error {
	_, err := d.DB.Exec("update patient set payment_service_customer_id = ? where id = ?", paymentCustomerId, patientId)
	return err
}

func (d *DataService) AddCardAndMakeDefaultForPatient(patientId int64, card *common.Card) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// add a new address to db
	addressId, err := d.addAddress(tx, card.BillingAddress)
	if err != nil {
		tx.Rollback()
		return err
	}

	card.BillingAddress.Id = addressId

	// undo all previous default cards for the patient
	_, err = tx.Exec(`update credit_card set is_default = 0 where patient_id = ?`, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// add new card as the default card
	lastId, err := tx.Exec(`insert into credit_card  (third_party_card_id,fingerprint, type, patient_id, address_id, is_default, label, status) values (?,?,?,?,?,?,?,?)`,
		card.ThirdPartyId, card.Fingerprint, card.Type, patientId, addressId, 1, card.Label, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	cardId, err := lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	card.Id = common.NewObjectId(cardId)
	return tx.Commit()
}

func (d *DataService) MakeCardDefaultForPatient(patientId int64, card *common.Card) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update credit_card set is_default = 0 where patient_id = ?`, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update credit_card set is_default = 1 where id = ?`, card.Id.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) MarkCardInactiveForPatient(patientId int64, card *common.Card) error {
	_, err := d.DB.Exec(`update credit_card set status =? where patient_id = ? and id = ?`, STATUS_DELETED, patientId, card.Id.Int64())
	return err
}

func (d *DataService) DeleteCardForPatient(patientId int64, card *common.Card) error {
	_, err := d.DB.Exec(`delete from credit_card where patient_id = ? and id = ?`, patientId, card.Id.Int64())
	return err
}

func (d *DataService) MakeLatestCardDefaultForPatient(patientId int64) (*common.Card, error) {
	var latestCardId int64
	err := d.DB.QueryRow(`select id from credit_card where patient_id = ? and status = ? order by creation_date desc limit 1`, patientId, STATUS_ACTIVE).Scan(&latestCardId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_, err = d.DB.Exec(`update credit_card set is_default = 1 where patient_id = ? and id = ?`, patientId, latestCardId)
	if err != nil {
		return nil, err
	}

	card, err := d.GetCardFromId(latestCardId)
	if err != nil {
		return nil, err
	}
	return card, err
}

func (d *DataService) addAddress(tx *sql.Tx, address *common.Address) (int64, error) {

	lastId, err := tx.Exec(`insert into address (address_line_1, address_line_2, city, state, zip_code, country) values (?,?,?,?,?,?)`,
		address.AddressLine1, address.AddressLine2, address.City, address.State, address.ZipCode, addressUsa)
	if err != nil {
		return 0, err
	}

	addressId, err := lastId.LastInsertId()
	if err != nil {
		return 0, err
	}

	return addressId, nil
}

func (d *DataService) GetCardsForPatient(patientId int64) ([]*common.Card, error) {
	cards := make([]*common.Card, 0)

	rows, err := d.DB.Query(`select id, third_party_card_id, fingerprint, type, is_default from credit_card where patient_id = ? and status = ?`, patientId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cardId int64
		var card common.Card

		if err := rows.Scan(&cardId, &card.ThirdPartyId, &card.Fingerprint, &card.Type, &card.IsDefault); err != nil {
			return nil, err
		}
		card.Id = common.NewObjectId(cardId)
		cards = append(cards, &card)
	}

	return cards, rows.Err()
}

func (d *DataService) GetCardFromId(cardId int64) (*common.Card, error) {
	var card common.Card
	var addressId int64
	err := d.DB.QueryRow(`select third_party_card_id, fingerprint, type, address_id, is_default from credit_card where id = ?`,
		cardId).Scan(&card.ThirdPartyId, &card.Fingerprint, &card.Type, &addressId, &card.IsDefault)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	card.Id = common.NewObjectId(cardId)
	var addressLine1, addressLine2, city, state, country, zipCode sql.NullString
	err = d.DB.QueryRow(`select address_line_1, address_line_2, city, state, zip_code, country from address where id = ?`, addressId).Scan(&addressLine1, &addressLine2, &city, &state, &zipCode, &country)
	if err != nil {
		if err == sql.ErrNoRows {
			return &card, nil
		}
		return nil, err
	}
	card.BillingAddress = &common.Address{
		Id:           addressId,
		AddressLine1: addressLine1.String,
		AddressLine2: addressLine2.String,
		City:         city.String,
		State:        state.String,
		ZipCode:      zipCode.String,
		Country:      country.String,
	}
	return &card, nil
}

func (d *DataService) UpdateDefaultAddressForPatient(patientId int64, address *common.Address) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	if address.Id == 0 {
		address.Id, err = d.addAddress(tx, address)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`delete from patient_address_selection where patient_id = ?`, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into patient_address_selection (patient_id, address_id, is_default, is_updated_by_doctor) values (?,?,1,0)`, patientId, address.Id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) DeleteAddress(addressId int64) error {
	_, err := d.DB.Exec(`delete from address where id = ?`, addressId)
	return err
}

func (d *DataService) CreatePendingTask(workType, status string, itemId int64) (int64, error) {
	lastId, err := d.DB.Exec(`insert into pending_task (type, item_id, status) values (?,?,?)`, workType, itemId, status)
	if err != nil {
		return 0, err
	}

	pendingTaskId, err := lastId.LastInsertId()
	if err != nil {
		return 0, err
	}

	return pendingTaskId, nil
}

func (d *DataService) DeletePendingTask(pendingTaskId int64) error {
	_, err := d.DB.Exec(`delete from pending_task where id = ?`, pendingTaskId)
	return err
}

func (d *DataService) GetFullNameForState(state string) (string, error) {
	var fullName string
	err := d.DB.QueryRow(`select full_name from state where full_name = ? or abbreviation = ?`, state, state).Scan(&fullName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return fullName, nil
}

func (d *DataService) getPatientBasedOnQuery(queryStr string, queryParams ...interface{}) ([]*common.Patient, error) {
	rows, err := d.DB.Query(queryStr, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patients := make([]*common.Patient, 0)
	for rows.Next() {
		var firstName, lastName, status, gender, suffix, prefix, middleName string
		var dob mysql.NullTime
		var phone, phoneType, zipCode, city, state, email, paymentServiceCustomerId sql.NullString
		var erxPatientId sql.NullInt64
		var patientId, accountId int64
		err = rows.Scan(&patientId, &erxPatientId, &paymentServiceCustomerId, &accountId, &email, &firstName, &middleName, &lastName, &suffix, &prefix,
			&zipCode, &city, &state, &phone, &phoneType, &gender, &dob, &status)
		if err != nil {
			return nil, err
		}

		patient := &common.Patient{
			PatientId:         common.NewObjectId(patientId),
			PaymentCustomerId: paymentServiceCustomerId.String,
			FirstName:         firstName,
			LastName:          lastName,
			Prefix:            prefix,
			Suffix:            suffix,
			MiddleName:        middleName,
			Email:             email.String,
			Status:            status,
			Gender:            gender,
			AccountId:         common.NewObjectId(accountId),
			Dob:               dob.Time,
			ZipCode:           zipCode.String,
			City:              city.String,
			State:             state.String,
			PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
				Phone:     phone.String,
				PhoneType: phoneType.String,
			},
			},
		}

		if erxPatientId.Valid {
			patient.ERxPatientId = common.NewObjectId(erxPatientId.Int64)
		}

		patient.IsUnlinked = status == PATIENT_UNLINKED

		patient.Pharmacy, err = d.getPatientPharmacySelection(patient.PatientId.Int64())
		if err != nil {
			return nil, err
		}

		patients = append(patients, patient)
	}

	return patients, rows.Err()
}

func (d *DataService) getAddressAndPhoneNumbersForPatient(patient *common.Patient) error {
	var defaultPatientAddress common.Address

	// get default address information (if exists) for each patient
	err := d.DB.QueryRow(`select address.id, address_line_1, address_line_2, city, state, zip_code, country from patient_address_selection
						inner join address on address_id = address.id
						where patient_id = ? and is_default=1`, patient.PatientId.Int64()).Scan(&defaultPatientAddress.Id, &defaultPatientAddress.AddressLine1, &defaultPatientAddress.AddressLine2, &defaultPatientAddress.City, &defaultPatientAddress.State, &defaultPatientAddress.ZipCode, &defaultPatientAddress.Country)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if defaultPatientAddress.AddressLine1 != "" {
		patient.PatientAddress = &defaultPatientAddress
	}

	rows, err := d.DB.Query(`select phone, phone_type from patient_phone where patient_id = ? and status = ?`, patient.PatientId.Int64(), STATUS_INACTIVE)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var phoneInformation common.PhoneInformation
		err = rows.Scan(&phoneInformation.Phone, &phoneInformation.PhoneType)
		if err != nil {
			return err
		}
		patient.PhoneNumbers = append(patient.PhoneNumbers, &phoneInformation)
	}

	return rows.Err()
}
