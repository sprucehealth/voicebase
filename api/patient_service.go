package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
)

func (d *DataService) RegisterPatient(patient *common.Patient) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.createPatientWithStatus(patient, PATIENT_REGISTERED, tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) updateTopLevelPatientInformation(db db, patient *common.Patient) error {
	patient.Gender = strings.ToLower(patient.Gender)

	// update top level patient details
	_, err := db.Exec(`
		UPDATE patient
		SET first_name=?, middle_name=?, last_name=?, prefix=?, suffix=?,
			dob_month=?, dob_day=?, dob_year=?, gender=?
		WHERE id = ?`,
		patient.FirstName, patient.MiddleName, patient.LastName, patient.Prefix,
		patient.Suffix, patient.DOB.Month, patient.DOB.Day, patient.DOB.Year,
		patient.Gender, patient.PatientId.Int64())
	if err != nil {
		return err
	}

	// getting the account id from the db to ensure that we have the account id for the right patient when udpating the patient data
	if err := d.db.QueryRow(`select account_id from patient where id = ?`, patient.PatientId.Int64()).Scan(&patient.AccountId); err == sql.ErrNoRows {
		return NoRowsError
	} else if err != nil {
		return err
	}

	// delete the existing numbers to add the new ones coming through
	_, err = db.Exec(`DELETE FROM account_phone WHERE account_id = ?`, patient.AccountId.Int64())
	if err != nil {
		return err
	}

	for i, phoneNumber := range patient.PhoneNumbers {
		status := STATUS_INACTIVE
		// save the first number as the main/default number
		if i == 0 {
			status = STATUS_ACTIVE
		}
		_, err = db.Exec(`INSERT INTO account_phone (phone, phone_type, account_id, status) values (?,?,?,?)`,
			phoneNumber.Phone.String(), phoneNumber.Type, patient.AccountId.Int64(), status)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DataService) UpdateTopLevelPatientInformation(patient *common.Patient) error {
	return d.updateTopLevelPatientInformation(d.db, patient)
}

func (d *DataService) UpdatePatientInformation(patient *common.Patient, updateFromDoctor bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.updateTopLevelPatientInformation(tx, patient); err != nil {
		tx.Rollback()
		return err
	}

	// update patient address if it exists
	if patient.PatientAddress != nil {

		addressId, err := d.addAddress(tx, patient.PatientAddress)
		if err != nil {
			tx.Rollback()
			return err
		}

		// remove any other address selection
		_, err = tx.Exec(`delete from patient_address_selection where patient_id = ?`, patient.PatientId.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_address_selection (address_id, patient_id, is_default, is_updated_by_doctor) values 
								(?,?,1,?)`, addressId, patient.PatientId.Int64(), updateFromDoctor)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, healthConditionId int64) error {
	tx, err := d.db.Begin()

	// create an account with no email and password for the unmatched patient
	lastId, err := tx.Exec(`insert into account (email, password, role_type_id) values (NULL,NULL, ?)`, d.roleTypeMapping[PATIENT_ROLE])
	if err != nil {
		tx.Rollback()
		return err
	}

	accountId, err := lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	patient.AccountId = encoding.NewObjectId(accountId)

	// create an account
	if err := d.createPatientWithStatus(patient, PATIENT_UNLINKED, tx); err != nil {
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
			_, err = tx.Exec(`INSERT INTO account_phone (account_id, phone, phone_type, status) VALUES (?,?,?,?)`,
				patient.AccountId.Int64(), phoneNumber.Phone.String(), phoneNumber.Type, STATUS_INACTIVE)
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

	patientCase := &common.PatientCase{
		PatientId:         patient.PatientId,
		HealthConditionId: encoding.NewObjectId(healthConditionId),
		Status:            common.PCStatusUnclaimed,
	}

	// create a case for the patient
	if err := d.createPatientCase(tx, patientCase); err != nil {
		tx.Rollback()
		return err
	}

	// assign the doctor to the case and patient
	if err := d.assignCareProviderToPatientFileAndCase(tx, doctor.DoctorId.Int64(), d.roleTypeMapping[DOCTOR_ROLE], patientCase); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) createPatientWithStatus(patient *common.Patient, status string, tx *sql.Tx) error {
	patient.Gender = strings.ToLower(patient.Gender)

	res, err := tx.Exec(`
		INSERT INTO patient
		(account_id, first_name, last_name, gender, dob_year, dob_month, dob_day, status, training)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patient.AccountId.Int64(),
		patient.FirstName,
		patient.LastName,
		patient.Gender,
		patient.DOB.Year,
		patient.DOB.Month,
		patient.DOB.Day,
		status,
		patient.Training)
	if err != nil {
		return err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return err
	}

	if len(patient.PhoneNumbers) > 0 {
		_, err = tx.Exec(`INSERT INTO account_phone (account_id, phone, phone_type, status) VALUES (?,?,?,?)`,
			patient.AccountId.Int64(), patient.PhoneNumbers[0].Phone.String(), patient.PhoneNumbers[0].Type, STATUS_ACTIVE)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO patient_location (patient_id, zip_code, city, state, status)
		VALUES (?, ?, ?, ?, ?)`, lastId, patient.ZipCode, patient.CityFromZipCode,
		patient.StateFromZipCode, STATUS_ACTIVE)
	if err != nil {
		return err
	}

	res, err = tx.Exec(`INSERT INTO person (role_type_id, role_id) VALUES (?, ?)`, d.roleTypeMapping[PATIENT_ROLE], lastId)
	if err != nil {
		return err
	}
	patient.PersonId, err = res.LastInsertId()
	if err != nil {
		return err
	}

	patient.PatientId = encoding.NewObjectId(lastId)
	return nil
}

func (d *DataService) GetPatientIdFromAccountId(accountId int64) (int64, error) {
	var patientId int64
	err := d.db.QueryRow("select id from patient where account_id = ?", accountId).Scan(&patientId)
	return patientId, err
}

func (d *DataService) IsEligibleToServePatientsInState(state string, healthConditionId int64) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT care_provider_state_elligibility.id 
		FROM care_provider_state_elligibility 
		INNER JOIN care_providing_state ON care_providing_state_id = care_providing_state.id 
		WHERE (state = ? OR long_state = ?) AND health_condition_id = ? AND role_type_id = ? LIMIT 1`, state, state,
		healthConditionId, d.roleTypeMapping[DOCTOR_ROLE]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}

func (d *DataService) UpdatePatientWithERxPatientId(patientId, erxPatientId int64) error {
	_, err := d.db.Exec(`update patient set erx_patient_id = ? where id = ? `, erxPatientId, patientId)
	return err
}

func (d *DataService) GetCareTeamForPatient(patientId int64) (*common.PatientCareTeam, error) {
	rows, err := d.db.Query(`select role_type_tag, creation_date, expires, provider_id, status, patient_id, health_condition_id
								from patient_care_provider_assignment 
									inner join role_type on role_type.id = role_type_id 
									where patient_id=?`, patientId)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var careTeam common.PatientCareTeam
	careTeam.Assignments = make([]*common.CareProviderAssignment, 0)
	for rows.Next() {
		var assignment common.CareProviderAssignment
		err := rows.Scan(&assignment.ProviderRole,
			&assignment.CreationDate,
			&assignment.Expires,
			&assignment.ProviderID,
			&assignment.Status,
			&assignment.PatientId,
			&assignment.HealthConditionId)
		if err != nil {
			return nil, err
		}
		careTeam.Assignments = append(careTeam.Assignments, &assignment)
	}

	return &careTeam, rows.Err()
}

func (d *DataService) CreateCareTeamForPatientWithPrimaryDoctor(patientId, healthConditionId, doctorId int64) (*common.PatientCareTeam, error) {
	// create new assignment for patient
	_, err := d.db.Exec(`
		REPLACE INTO patient_care_provider_assignment 
		(patient_id, health_condition_id, role_type_id, provider_id, status) 
		VALUES (?, ?, ?, ?, ?)`, patientId, healthConditionId, d.roleTypeMapping[DOCTOR_ROLE], doctorId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}

	return d.GetCareTeamForPatient(patientId)
}

func (d *DataService) AddDoctorToCareTeamForPatient(patientId, healthConditionId, doctorId int64) error {
	_, err := d.db.Exec(`insert into patient_care_provider_assignment (patient_id, health_condition_id, provider_id, role_type_id, status) values (?,?,?,?,?)`, patientId, healthConditionId, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_ACTIVE)
	return err
}

func (d *DataService) GetPatientFromAccountId(accountId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "", `
		patient.account_id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (patient_location.zip_code IS NULL OR patient_location.status = 'ACTIVE')`, accountId)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		return patients[0], d.getOtherInfoForPatient(patients[0])
	}
	return nil, NoRowsError
}

func (d *DataService) GetPatientFromId(patientId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "", `
		patient.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (patient_location.zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientId)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, NoRowsError
	}
	return nil, errors.New("Got more than 1 patient when expected just 1")
}

func (d *DataService) GetPatientsForIds(patientIds []int64) ([]*common.Patient, error) {
	if len(patientIds) == 0 {
		return nil, nil
	}
	return d.getPatientBasedOnQuery("patient", "",
		fmt.Sprintf(`
			patient.id IN (%s)
				AND (phone IS NULL OR (account_phone.status='ACTIVE'))
				AND (patient_location.zip_code IS NULL OR patient_location.status='ACTIVE')`,
			enumerateItemsIntoString(patientIds)))
}

func (d *DataService) GetPatientFromTreatmentPlanId(treatmentPlanId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("treatment_plan",
		`INNER JOIN patient ON patient.id = treatment_plan.patient_id`,
		`treatment_plan.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, treatmentPlanId)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromPatientVisitId(patientVisitId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient_visit",
		`INNER JOIN patient ON patient_visit.patient_id = patient.id`,
		`patient_visit.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientVisitId)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromErxPatientId(erxPatientId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "",
		`patient.erx_patient_id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, erxPatientId)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromRefillRequestId(refillRequestId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("rx_refill_request",
		`INNER JOIN patient ON rx_refill_request.patient_id = patient.id`,
		`rx_refill_request.id = ?
			AND (phone IS NULL OR (account_phone.status='ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, refillRequestId)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, NoRowsError
	}

	return nil, errors.New("Got more than 1 patient for refill request when expected just 1")
}

func (d *DataService) GetPatientFromTreatmentId(treatmentId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("treatment",
		`INNER JOIN treatment_plan ON treatment.treatment_plan_id = treatment_plan.id
		INNER JOIN patient ON treatment_plan.patient_id = patient.id`,
		`treatment.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, treatmentId)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, NoRowsError
	}

	return nil, errors.New("Got more than 1 patient for treatment when expected just 1")
}

func (d *DataService) GetPatientFromCaseId(patientCaseId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient_case",
		`INNER JOIN patient ON patient_case.patient_id = patient.id`,
		`patient_case.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientCaseId)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, NoRowsError
	}

	return nil, errors.New("Got more than 1 patient from patient_case when expected just 1")
}

func (d *DataService) GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("unlinked_dntf_treatment",
		`INNER JOIN patient ON patient_id = patient.id`,
		`unlinked_dntf_treatment.id = ?`, unlinkedDNTFTreatmentId)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, NoRowsError
	}

	return nil, errors.New("Got more than 1 patient for treatment when expected just 1")
}

func (d *DataService) GetPatientVisitsForPatient(patientId int64) ([]*common.PatientVisit, error) {
	rows, err := d.db.Query(`select id, patient_id, patient_case_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status 
		from patient_visit where patient_id = ?`, patientId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getPatientVisitFromRows(rows)
}

func (d *DataService) AnyVisitSubmitted(patientID int64) (bool, error) {
	var count int64
	if err := d.db.QueryRow(`
		SELECT count(*) 
		FROM patient_visit 
		WHERE patient_visit.status != ? AND patient_id = ? LIMIT 1`,
		common.PVStatusOpen, patientID).Scan(&count); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d *DataService) UpdatePatientAddress(patientId int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error {
	tx, err := d.db.Begin()
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
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update patient_pharmacy_selection set status=? where patient_id = ?`, STATUS_INACTIVE, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = addPharmacy(pharmacyDetails, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	existingPharmacyId := pharmacyDetails.LocalId

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patientId, existingPharmacyId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) getPatientPharmacySelection(patientId int64) (pharmacySelection *pharmacy.PharmacyData, err error) {
	rows, err := d.db.Query(`select pharmacy_selection.id, patient_id, pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
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

	rows, err := d.db.Query(fmt.Sprintf(`select pharmacy_selection.id, patient_id,  pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
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

func (d *DataService) GetPharmacyBasedOnReferenceIdAndSource(pharmacyId int64, pharmacySource string) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var id int64
	err := d.db.QueryRow(`select id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat,lng
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
	var source string
	var pharmacyReferenceId int64
	err := d.db.QueryRow(`select source, pharmacy_id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat,lng
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
	tx, err := d.db.Begin()
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
	var sourceType, name, addressLine1, addressLine2, phone, city, state, zipCode, lat, lng sql.NullString
	var id sql.NullInt64
	err := rows.Scan(&localId, &patientId, &id, &sourceType, &name, &addressLine1, &addressLine2, &city, &state, &zipCode, &phone, &lat, &lng)
	if err != nil {
		return nil, err
	}

	pharmacySelection := &pharmacy.PharmacyData{
		LocalId:      localId,
		PatientId:    patientId,
		SourceId:     id.Int64,
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
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for agreementType, agreed := range agreements {
		_, err = tx.Exec(`update patient_agreement set status=? where patient_id = ? and agreement_type = ?`, STATUS_INACTIVE, patientId, agreementType)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_agreement (patient_id, agreement_type, agreed, status) values (?,?,?,?)`, patientId, agreementType, agreed, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) PatientAgreements(patientID int64) (map[string]time.Time, error) {
	rows, err := d.db.Query(`
		SELECT agreement_type, agreement_date
		FROM patient_agreement
		WHERE patient_id = ? AND agreed = 1`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ag := make(map[string]time.Time)
	for rows.Next() {
		var atype string
		var adate time.Time
		if err := rows.Scan(&atype, &adate); err != nil {
			return nil, err
		}
		ag[atype] = adate
	}

	return ag, rows.Err()
}

func (d *DataService) UpdatePatientWithPaymentCustomerId(patientId int64, paymentCustomerId string) error {
	_, err := d.db.Exec("update patient set payment_service_customer_id = ? where id = ?", paymentCustomerId, patientId)
	return err
}

func (d *DataService) AddCardAndMakeDefaultForPatient(patientId int64, card *common.Card) error {
	tx, err := d.db.Begin()
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

	card.Id = encoding.NewObjectId(cardId)
	return tx.Commit()
}

func (d *DataService) MakeCardDefaultForPatient(patientId int64, card *common.Card) error {
	tx, err := d.db.Begin()
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
	_, err := d.db.Exec(`update credit_card set status =? where patient_id = ? and id = ?`, STATUS_DELETED, patientId, card.Id.Int64())
	return err
}

func (d *DataService) DeleteCardForPatient(patientId int64, card *common.Card) error {
	_, err := d.db.Exec(`delete from credit_card where patient_id = ? and id = ?`, patientId, card.Id.Int64())
	return err
}

func (d *DataService) MakeLatestCardDefaultForPatient(patientId int64) (*common.Card, error) {
	var latestCardId int64
	err := d.db.QueryRow(`select id from credit_card where patient_id = ? and status = ? order by creation_date desc limit 1`, patientId, STATUS_ACTIVE).Scan(&latestCardId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_, err = d.db.Exec(`update credit_card set is_default = 1 where patient_id = ? and id = ?`, patientId, latestCardId)
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

	rows, err := d.db.Query(`select id, third_party_card_id, fingerprint, type, is_default, creation_date from credit_card where patient_id = ? and status = ?`, patientId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cardId encoding.ObjectId
		var card common.Card

		if err := rows.Scan(&cardId, &card.ThirdPartyId, &card.Fingerprint, &card.Type, &card.IsDefault, &card.CreationDate); err != nil {
			return nil, err
		}
		card.Id = cardId
		cards = append(cards, &card)
	}

	return cards, rows.Err()
}

func (d *DataService) GetDefaultCardForPatient(patientId int64) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date from credit_card where patient_id = ? and is_default = 1`,
		patientId)
	return d.getCardFromRow(row)
}

func (d *DataService) GetCardFromId(cardId int64) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date from credit_card where id = ?`,
		cardId)
	return d.getCardFromRow(row)
}

func (d *DataService) GetCardFromThirdPartyId(thirdPartyId string) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date from credit_card where third_party_card_id = ?`,
		thirdPartyId)
	return d.getCardFromRow(row)
}

func (d *DataService) getCardFromRow(row *sql.Row) (*common.Card, error) {
	var card common.Card
	var addressId int64
	err := row.Scan(&card.Id, &card.ThirdPartyId, &card.Fingerprint, &card.Type, &addressId, &card.IsDefault, &card.CreationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NoRowsError
		}
		return nil, err
	}
	var addressLine1, addressLine2, city, state, country, zipCode sql.NullString
	err = d.db.QueryRow(`select address_line_1, address_line_2, city, state, zip_code, country from address where id = ?`, addressId).Scan(&addressLine1, &addressLine2, &city, &state, &zipCode, &country)
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
	tx, err := d.db.Begin()
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
	_, err := d.db.Exec(`delete from address where id = ?`, addressId)
	return err
}

func (d *DataService) CreatePendingTask(workType, status string, itemId int64) (int64, error) {
	lastId, err := d.db.Exec(`insert into pending_task (type, item_id, status) values (?,?,?)`, workType, itemId, status)
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
	_, err := d.db.Exec(`delete from pending_task where id = ?`, pendingTaskId)
	return err
}

func (d *DataService) AddAlertsForPatient(patientId int64, alerts []*common.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	fields := make([]string, 0, len(alerts))
	values := make([]interface{}, 0, 4*len(alerts))
	for _, alert := range alerts {
		values = append(values, alert.PatientId, alert.Message, alert.Source, alert.SourceId)
		fields = append(fields, "(?,?,?,?)")
	}

	_, err := d.db.Exec(`insert into patient_alerts (patient_id, alert, source, source_id) values `+strings.Join(fields, ","), values...)
	return err
}

func (d *DataService) GetAlertsForPatient(patientId int64) ([]*common.Alert, error) {
	rows, err := d.db.Query(`select id, patient_id, creation_date, alert, source, source_id from patient_alerts where patient_id = ?`, patientId)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var alerts []*common.Alert
	for rows.Next() {
		alert := &common.Alert{}
		if err := rows.Scan(&alert.Id, &alert.PatientId, &alert.CreationDate, &alert.Message, &alert.Source, &alert.SourceId); err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

func (d *DataService) UpdatePatientPCP(pcp *common.PCP) error {
	_, err := d.db.Exec(`replace into patient_pcp (patient_id, physician_name, phone_number, practice_name, email, fax_number) values (?,?,?,?,?,?)`, pcp.PatientID, pcp.PhysicianName, pcp.PhoneNumber,
		pcp.PracticeName, pcp.Email, pcp.FaxNumber)
	return err
}

func (d *DataService) GetPatientPCP(patientId int64) (*common.PCP, error) {
	var pcp common.PCP
	err := d.db.QueryRow(`select patient_id, physician_name, phone_number, practice_name, email, fax_number from patient_pcp where patient_id = ?`, patientId).Scan(
		&pcp.PatientID,
		&pcp.PhysicianName,
		&pcp.PhoneNumber,
		&pcp.PracticeName,
		&pcp.Email,
		&pcp.FaxNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &pcp, nil
}

func (d *DataService) DeletePatientPCP(patientId int64) error {
	_, err := d.db.Exec(`delete from patient_pcp where patient_id = ?`, patientId)
	return err
}

func (d *DataService) UpdatePatientEmergencyContacts(patientId int64, emergencyContacts []*common.EmergencyContact) error {
	tx, err := d.db.Begin()
	if err != nil {
		return nil
	}

	// delete any existing emergency contacts for the patient
	_, err = tx.Exec(`delete from patient_emergency_contact where patient_id = ?`, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// add all emergency contacts
	for _, eContact := range emergencyContacts {
		res, err := tx.Exec(`insert into patient_emergency_contact (patient_id, full_name, phone_number, relationship) values (?,?,?,?)`, patientId, eContact.FullName, eContact.PhoneNumber, eContact.Relationship)
		if err != nil {
			tx.Rollback()
			return err
		}

		eContact.ID, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) GetPatientEmergencyContacts(patientId int64) ([]*common.EmergencyContact, error) {
	rows, err := d.db.Query(`select id, patient_id, full_name, phone_number, relationship from patient_emergency_contact where patient_id = ?`, patientId)
	if err != nil {
		return nil, err
	}

	var emergencyContacts []*common.EmergencyContact
	for rows.Next() {
		var eContact common.EmergencyContact
		err := rows.Scan(&eContact.ID,
			&eContact.PatientID,
			&eContact.FullName,
			&eContact.PhoneNumber,
			&eContact.Relationship)
		if err != nil {
			return nil, err
		}
		emergencyContacts = append(emergencyContacts, &eContact)
	}

	return emergencyContacts, rows.Err()
}

func (d *DataService) GetActiveMembersOfCareTeamForPatient(patientId int64, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_care_provider_assignment 
		inner join role_type on role_type_id = role_type.id
		where status = ? and patient_id = ?`, STATUS_ACTIVE, patientId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getMembersOfCareTeam(rows, fillInDetails)
}

func (d *DataService) getMembersOfCareTeam(rows *sql.Rows, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	var assignments []*common.CareProviderAssignment
	for rows.Next() {
		var assignment common.CareProviderAssignment
		if err := rows.Scan(&assignment.ProviderID, &assignment.ProviderRole, &assignment.Status, &assignment.CreationDate); err != nil {
			return nil, err
		}

		if fillInDetails {
			switch assignment.ProviderRole {
			case DOCTOR_ROLE, MA_ROLE:
				doctor, err := d.GetDoctorFromId(assignment.ProviderID)
				if err != nil {
					return nil, err
				}
				assignment.FirstName = doctor.FirstName
				assignment.LastName = doctor.LastName
				assignment.ShortTitle = doctor.ShortTitle
				assignment.LongTitle = doctor.LongTitle
				assignment.ShortDisplayName = doctor.ShortDisplayName
				assignment.LongDisplayName = doctor.LongDisplayName
				assignment.SmallThumbnailID = doctor.SmallThumbnailID
				assignment.LargeThumbnailID = doctor.LargeThumbnailID
				assignment.SmallThumbnailURL = doctor.SmallThumbnailURL
				assignment.LargeThumbnailURL = doctor.LargeThumbnailURL
			}
		}

		assignments = append(assignments, &assignment)

	}

	// sort by role so that the doctors are shown first in the care team
	sort.Sort(ByRole(assignments))
	return assignments, rows.Err()
}

func (d *DataService) getPatientBasedOnQuery(table, joins, where string, queryParams ...interface{}) ([]*common.Patient, error) {
	queryStr := fmt.Sprintf(`
		SELECT patient.id, patient.erx_patient_id, patient.payment_service_customer_id, patient.account_id,
			account.email, first_name, middle_name, last_name, suffix, prefix, zip_code, city, state, phone,
			phone_type, gender, dob_year, dob_month, dob_day, patient.status, patient.training, person.id
		FROM %s
		%s
		INNER JOIN person ON role_type_id = %d AND role_id = patient.id
		LEFT OUTER JOIN account_phone ON account_phone.account_id = patient.account_id
		LEFT OUTER JOIN patient_location ON patient_location.patient_id = patient.id
		LEFT OUTER JOIN account ON account.id = patient.account_id
		WHERE %s`, table, joins, d.roleTypeMapping[PATIENT_ROLE], where)
	rows, err := d.db.Query(queryStr, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patients := make([]*common.Patient, 0)
	for rows.Next() {
		var firstName, lastName, status, gender string
		var phoneType, zipCode, city, state, email, paymentServiceCustomerId, suffix, prefix, middleName sql.NullString
		var phone common.Phone
		var patientId, accountId, erxPatientId encoding.ObjectId
		var dobMonth, dobYear, dobDay int
		var personId int64
		var training bool
		err = rows.Scan(&patientId, &erxPatientId, &paymentServiceCustomerId, &accountId, &email, &firstName, &middleName, &lastName, &suffix, &prefix,
			&zipCode, &city, &state, &phone, &phoneType, &gender, &dobYear, &dobMonth, &dobDay, &status, &training, &personId)
		if err != nil {
			return nil, err
		}

		patient := &common.Patient{
			PatientId:         patientId,
			PaymentCustomerId: paymentServiceCustomerId.String,
			FirstName:         firstName,
			LastName:          lastName,
			Prefix:            prefix.String,
			Suffix:            suffix.String,
			MiddleName:        middleName.String,
			Email:             email.String,
			Status:            status,
			Gender:            gender,
			AccountId:         accountId,
			ZipCode:           zipCode.String,
			CityFromZipCode:   city.String,
			StateFromZipCode:  state.String,
			ERxPatientId:      erxPatientId,
			Training:          training,
			DOB:               encoding.DOB{Year: dobYear, Month: dobMonth, Day: dobDay},
			PhoneNumbers: []*common.PhoneNumber{
				&common.PhoneNumber{
					Phone: phone,
					Type:  phoneType.String,
				},
			},
			PersonId:   personId,
			IsUnlinked: status == PATIENT_UNLINKED,
		}

		patient.Pharmacy, err = d.getPatientPharmacySelection(patient.PatientId.Int64())
		if err != nil {
			return nil, err
		}

		patients = append(patients, patient)
	}

	return patients, rows.Err()
}

func (d *DataService) getOtherInfoForPatient(patient *common.Patient) error {
	var defaultPatientAddress common.Address

	// get default address information (if exists) for each patient
	err := d.db.QueryRow(`
		SELECT address.id, address_line_1, address_line_2, city, state, zip_code,
			country from patient_address_selection
		INNER JOIN address ON address_id = address.id
		WHERE patient_id = ? AND is_default = 1`,
		patient.PatientId.Int64(),
	).Scan(
		&defaultPatientAddress.Id, &defaultPatientAddress.AddressLine1,
		&defaultPatientAddress.AddressLine2, &defaultPatientAddress.City,
		&defaultPatientAddress.State, &defaultPatientAddress.ZipCode,
		&defaultPatientAddress.Country)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if defaultPatientAddress.AddressLine1 != "" {
		patient.PatientAddress = &defaultPatientAddress
	}

	// get prompt status
	patient.PromptStatus, err = d.GetPushPromptStatus(patient.AccountId.Int64())
	if err != nil {
		return err
	}

	rows, err := d.db.Query(`SELECT phone, phone_type FROM account_phone WHERE account_id = ? AND status = ?`,
		patient.AccountId.Int64(), STATUS_INACTIVE)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var phoneInformation common.PhoneNumber
		err = rows.Scan(&phoneInformation.Phone, &phoneInformation.Type)
		if err != nil {
			return err
		}
		patient.PhoneNumbers = append(patient.PhoneNumbers, &phoneInformation)
	}

	return rows.Err()
}

func (d *DataService) PatientState(patientID int64) (string, error) {
	var patientState string
	err := d.db.QueryRow(`SELECT state FROM patient_location WHERE patient_id = ?`, patientID).Scan(&patientState)
	if err == sql.ErrNoRows {
		return "", NoRowsError
	}
	return patientState, err
}
