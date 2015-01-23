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
	"github.com/sprucehealth/backend/libs/dbutil"
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

func (d *DataService) UpdatePatient(id int64, update *PatientUpdate, updateFromDoctor bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.updatePatient(tx, id, update, updateFromDoctor); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) updatePatient(tx *sql.Tx, id int64, update *PatientUpdate, updateFromDoctor bool) error {
	var cols []string
	var vals []interface{}

	if update.FirstName != nil {
		cols = append(cols, "first_name = ?")
		vals = append(vals, *update.FirstName)
	}
	if update.MiddleName != nil {
		cols = append(cols, "middle_name = ?")
		vals = append(vals, *update.MiddleName)
	}
	if update.LastName != nil {
		cols = append(cols, "last_name = ?")
		vals = append(vals, *update.LastName)
	}
	if update.Prefix != nil {
		cols = append(cols, "prefix = ?")
		vals = append(vals, *update.Prefix)
	}
	if update.Suffix != nil {
		cols = append(cols, "suffix = ?")
		vals = append(vals, *update.Suffix)
	}
	if update.DOB != nil {
		cols = append(cols, "dob_day = ?", "dob_month = ?", "dob_year = ?")
		vals = append(vals, update.DOB.Day, update.DOB.Month, update.DOB.Year)
	}
	if update.Gender != nil {
		cols = append(cols, "gender = ?")
		vals = append(vals, strings.ToLower(*update.Gender))
	}

	if len(cols) != 0 {
		vals = append(vals, id)
		_, err := tx.Exec(`
			UPDATE patient
			SET `+strings.Join(cols, ", ")+`
			WHERE id = ?`, vals...)
		if err != nil {
			return err
		}
	}

	if len(update.PhoneNumbers) != 0 {
		accountID, err := accountIDForPatient(tx, id)
		if err != nil {
			return err
		}
		if err := replaceAccountPhoneNumbers(tx, accountID, update.PhoneNumbers); err != nil {
			return err
		}
	}

	if update.Address != nil {
		if err := updatePatientAddress(tx, id, update.Address, updateFromDoctor); err != nil {
			return err
		}
	}

	return nil
}

func replaceAccountPhoneNumbers(tx *sql.Tx, accountID int64, numbers []*common.PhoneNumber) error {
	_, err := tx.Exec(`DELETE FROM account_phone WHERE account_id = ?`, accountID)
	if err != nil {
		return err
	}

	// Make sure there's at least one and only one active phone number
	hasActive := false
	for _, p := range numbers {
		if p.Status == STATUS_ACTIVE {
			if hasActive {
				p.Status = STATUS_INACTIVE
			} else {
				hasActive = true
			}
		} else if p.Status == "" {
			p.Status = STATUS_INACTIVE
		}
	}
	if !hasActive {
		numbers[0].Status = STATUS_ACTIVE
	}

	reps := make([]string, len(numbers))
	vals := make([]interface{}, 0, len(numbers)*5)
	for i, p := range numbers {
		reps[i] = "(?, ?, ?, ?, ?)"
		vals = append(vals, accountID, p.Phone.String(), p.Type, p.Status, p.Verified)
	}
	_, err = tx.Exec(`
			INSERT INTO account_phone (account_id, phone, phone_type, status, verified)
			VALUES `+strings.Join(reps, ", "), vals...)
	return err
}

func updatePatientAddress(tx *sql.Tx, patientID int64, address *common.Address, updateFromDoctor bool) error {
	addressID, err := addAddress(tx, address)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM patient_address_selection WHERE patient_id = ?`, patientID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
			INSERT INTO patient_address_selection
				(address_id, patient_id, is_default, is_updated_by_doctor)
			VALUES (?, ?, ?, ?)`, addressID, patientID, true, updateFromDoctor)
	return err
}

func (d *DataService) CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, pathwayTag string) error {
	tx, err := d.db.Begin()

	// create an account with no email and password for the unmatched patient
	lastID, err := tx.Exec(`insert into account (email, password, role_type_id) values (NULL,NULL, ?)`, d.roleTypeMapping[PATIENT_ROLE])
	if err != nil {
		tx.Rollback()
		return err
	}

	accountID, err := lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	patient.AccountID = encoding.NewObjectID(accountID)

	// create an account
	if err := d.createPatientWithStatus(patient, PATIENT_UNLINKED, tx); err != nil {
		tx.Rollback()
		return err
	}

	// create address for patient
	if patient.PatientAddress != nil {
		addressID, err := addAddress(tx, patient.PatientAddress)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_address_selection (address_id, patient_id, is_default, is_updated_by_doctor) values (?,?,1,0)`, addressID, patient.PatientID.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if patient.Pharmacy != nil {
		var existingPharmacyId int64
		err = tx.QueryRow(`select id from pharmacy_selection where pharmacy_id = ?`, patient.Pharmacy.SourceID).Scan(&existingPharmacyId)
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
			existingPharmacyId = patient.Pharmacy.LocalID
		}

		_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patient.PatientID.Int64(), existingPharmacyId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// create additional phone numbers for patient
	if len(patient.PhoneNumbers) > 1 {
		for _, phoneNumber := range patient.PhoneNumbers[1:] {
			_, err = tx.Exec(`INSERT INTO account_phone (account_id, phone, phone_type, status) VALUES (?,?,?,?)`,
				patient.AccountID.Int64(), phoneNumber.Phone.String(), phoneNumber.Type, STATUS_INACTIVE)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// assign the erx patient id to the patient
	_, err = tx.Exec(`update patient set erx_patient_id = ? where id = ?`, patient.ERxPatientID.Int64(), patient.PatientID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	patientCase := &common.PatientCase{
		PatientID:  patient.PatientID,
		PathwayTag: pathwayTag,
		Status:     common.PCStatusUnclaimed,
	}

	// create a case for the patient
	if err := d.createPatientCase(tx, patientCase); err != nil {
		tx.Rollback()
		return err
	}

	// assign the doctor to the case and patient
	if err := d.assignCareProviderToPatientFileAndCase(tx, doctor.DoctorID.Int64(), d.roleTypeMapping[DOCTOR_ROLE], patientCase); err != nil {
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
		patient.AccountID.Int64(),
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

	lastID, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return err
	}

	if len(patient.PhoneNumbers) > 0 {
		if err := replaceAccountPhoneNumbers(tx, patient.AccountID.Int64(), patient.PhoneNumbers); err != nil {
			return err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO patient_location (patient_id, zip_code, city, state, status)
		VALUES (?, ?, ?, ?, ?)`, lastID, patient.ZipCode, patient.CityFromZipCode,
		patient.StateFromZipCode, STATUS_ACTIVE)
	if err != nil {
		return err
	}

	res, err = tx.Exec(`INSERT INTO person (role_type_id, role_id) VALUES (?, ?)`, d.roleTypeMapping[PATIENT_ROLE], lastID)
	if err != nil {
		return err
	}
	patient.PersonID, err = res.LastInsertId()
	if err != nil {
		return err
	}

	patient.PatientID = encoding.NewObjectID(lastID)
	return nil
}

func (d *DataService) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	var patientID int64
	err := d.db.QueryRow("SELECT id FROM patient WHERE account_id = ?", accountID).Scan(&patientID)
	return patientID, err
}

func (d *DataService) IsEligibleToServePatientsInState(state, pathwayTag string) (bool, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return false, err
	}

	var id int64
	err = d.db.QueryRow(`
		SELECT 1
		FROM care_provider_state_elligibility
		INNER JOIN care_providing_state ON care_providing_state_id = care_providing_state.id
		WHERE (state = ? OR long_state = ?)
			AND clinical_pathway_id = ?
			AND role_type_id = ?
		LIMIT 1`,
		state, state, pathwayID, d.roleTypeMapping[DOCTOR_ROLE]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	return err == nil, err
}

func (d *DataService) UpdatePatientWithERxPatientID(patientID, erxPatientID int64) error {
	_, err := d.db.Exec(`UPDATE patient SET erx_patient_id = ? WHERE id = ? `, erxPatientID, patientID)
	return err
}

// Utility function for populating assignment refernces with their provider's data
func (d *DataService) populateAssignmentInfoFromProviderID(assignment *common.CareProviderAssignment, providerID int64) error {
	doctor, err := d.Doctor(assignment.ProviderID, true)
	if err != nil {
		return err
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
	return nil
}

// GetCareTeamsForPatientByCase returns all care teams for a given patient mapped by CaseID. This includes all care teams across all given conditions.
// The caller is expected to filter this list down to the desired subset.
// TODO:REFACTOR: There is likely a clever functional way to merge this and GetCareTeamsForPatientByHealthCondition
func (d *DataService) GetCareTeamsForPatientByCase(patientID int64) (map[int64]*common.PatientCareTeam, error) {
	rows, err := d.db.Query(`
			SELECT role_type_tag, pccpa.creation_date, expires, provider_id, pccpa.status, patient_id, patient_case_id
			FROM patient_case_care_provider_assignment AS pccpa 
			INNER JOIN role_type ON role_type.id = role_type_id
			INNER JOIN patient_case ON patient_case.id = patient_case_id
			WHERE patient_id=?`, patientID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patientCaseID int64
	careTeams := make(map[int64]*common.PatientCareTeam)
	for rows.Next() {
		var assignment common.CareProviderAssignment
		err := rows.Scan(&assignment.ProviderRole,
			&assignment.CreationDate,
			&assignment.Expires,
			&assignment.ProviderID,
			&assignment.Status,
			&assignment.PatientID,
			&patientCaseID)
		if err != nil {
			return nil, err
		}

		d.populateAssignmentInfoFromProviderID(&assignment, assignment.ProviderID)

		if _, ok := careTeams[patientCaseID]; !ok {
			careTeams[patientCaseID] = &common.PatientCareTeam{}
			careTeams[patientCaseID].Assignments = make([]*common.CareProviderAssignment, 0)
		}

		careTeam := careTeams[patientCaseID]
		careTeam.Assignments = append(careTeam.Assignments, &assignment)
	}

	return careTeams, rows.Err()
}

func (d *DataService) GetCareTeamForPatient(patientID int64) (*common.PatientCareTeam, error) {
	rows, err := d.db.Query(`
		SELECT role_type_tag, creation_date, expires, provider_id, status, patient_id, clinical_pathway_id
		FROM patient_care_provider_assignment
		INNER JOIN role_type ON role_type.id = role_type_id
		WHERE patient_id = ?`, patientID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var careTeam common.PatientCareTeam
	careTeam.Assignments = make([]*common.CareProviderAssignment, 0)
	for rows.Next() {
		var assignment common.CareProviderAssignment
		var pathwayID int64
		err := rows.Scan(&assignment.ProviderRole,
			&assignment.CreationDate,
			&assignment.Expires,
			&assignment.ProviderID,
			&assignment.Status,
			&assignment.PatientID,
			&pathwayID)
		if err != nil {
			return nil, err
		}
		assignment.PathwayTag, err = d.pathwayTagFromID(pathwayID)
		if err != nil {
			return nil, err
		}
		careTeam.Assignments = append(careTeam.Assignments, &assignment)
	}

	return &careTeam, rows.Err()
}

func (d *DataService) CreateCareTeamForPatientWithPrimaryDoctor(patientID, doctorID int64, pathwayTag string) (*common.PatientCareTeam, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return nil, err
	}

	// create new assignment for patient
	_, err = d.db.Exec(`
		REPLACE INTO patient_care_provider_assignment
		(patient_id, clinical_pathway_id, role_type_id, provider_id, status)
		VALUES (?, ?, ?, ?, ?)`, patientID, pathwayID, d.roleTypeMapping[DOCTOR_ROLE], doctorID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}

	return d.GetCareTeamForPatient(patientID)
}

func (d *DataService) AddDoctorToCareTeamForPatient(patientID, doctorID int64, pathwayTag string) error {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO patient_care_provider_assignment
			(patient_id, clinical_pathway_id, provider_id, role_type_id, status)
		VALUES (?,?,?,?,?)`,
		patientID, pathwayID, doctorID, d.roleTypeMapping[DOCTOR_ROLE], STATUS_ACTIVE)
	return err
}

func (d *DataService) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "", `
		patient.account_id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (patient_location.zip_code IS NULL OR patient_location.status = 'ACTIVE')`, accountID)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		return patients[0], d.getOtherInfoForPatient(patients[0])
	}
	return nil, ErrNotFound("patient")
}

func (d *DataService) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	if !basicInfoOnly {
		return d.GetPatientFromID(id)
	}

	row := d.db.QueryRow(`
		SELECT id, first_name, last_name, gender, status, account_id, dob_month, dob_year, dob_day, payment_service_customer_id, erx_patient_id 
		FROM patient
		WHERE id = ?`, id)

	patient, err := scanRowForPatient(row)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("patient")
	} else if err != nil {
		return nil, err
	}

	return patient, nil
}

func (d *DataService) Patients(ids []int64) (map[int64]*common.Patient, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, first_name, last_name, gender, status, account_id, dob_month, dob_year, dob_day, 
		payment_service_customer_id, erx_patient_id 
		FROM patient
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patients := make(map[int64]*common.Patient)
	for rows.Next() {
		patient, err := scanRowForPatient(rows)
		if err != nil {
			return nil, err
		}
		patients[patient.PatientID.Int64()] = patient
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patients, nil
}

type rowScanner interface {
	Scan(vals ...interface{}) error
}

func scanRowForPatient(scanner rowScanner) (*common.Patient, error) {
	var patient common.Patient
	var dobMonth, dobDay, dobYear int
	var stripeID sql.NullString
	err := scanner.Scan(
		&patient.PatientID,
		&patient.FirstName,
		&patient.LastName,
		&patient.Gender,
		&patient.Status,
		&patient.AccountID,
		&dobMonth,
		&dobYear,
		&dobDay,
		&stripeID,
		&patient.ERxPatientID,
	)
	if err != nil {
		return nil, err
	}

	patient.PaymentCustomerID = stripeID.String
	patient.DOB = encoding.DOB{
		Month: dobMonth,
		Day:   dobDay,
		Year:  dobYear,
	}
	return &patient, nil
}

func (d *DataService) GetPatientFromID(patientID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "", `
		patient.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (patient_location.zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientID)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, ErrNotFound("patient")
	}
	return nil, errors.New("Got more than 1 patient when expected just 1")
}

func (d *DataService) GetPatientsForIDs(patientIDs []int64) ([]*common.Patient, error) {
	if len(patientIDs) == 0 {
		return nil, nil
	}
	return d.getPatientBasedOnQuery("patient", "",
		fmt.Sprintf(`
			patient.id IN (%s)
				AND (phone IS NULL OR (account_phone.status='ACTIVE'))
				AND (patient_location.zip_code IS NULL OR patient_location.status='ACTIVE')`,
			enumerateItemsIntoString(patientIDs)))
}

func (d *DataService) GetPatientFromTreatmentPlanID(treatmentPlanID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("treatment_plan",
		`INNER JOIN patient ON patient.id = treatment_plan.patient_id`,
		`treatment_plan.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, treatmentPlanID)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromPatientVisitID(patientVisitID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient_visit",
		`INNER JOIN patient ON patient_visit.patient_id = patient.id`,
		`patient_visit.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientVisitID)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromErxPatientID(erxPatientID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient", "",
		`patient.erx_patient_id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, erxPatientID)
	if err != nil {
		return nil, err
	}
	if len(patients) > 0 {
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	}
	return nil, err
}

func (d *DataService) GetPatientFromRefillRequestID(refillRequestID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("rx_refill_request",
		`INNER JOIN patient ON rx_refill_request.patient_id = patient.id`,
		`rx_refill_request.id = ?
			AND (phone IS NULL OR (account_phone.status='ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, refillRequestID)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, ErrNotFound("patient")
	}

	return nil, errors.New("Got more than 1 patient for refill request when expected just 1")
}

func (d *DataService) GetPatientFromTreatmentID(treatmentID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("treatment",
		`INNER JOIN treatment_plan ON treatment.treatment_plan_id = treatment_plan.id
		INNER JOIN patient ON treatment_plan.patient_id = patient.id`,
		`treatment.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, treatmentID)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, ErrNotFound("patient")
	}

	return nil, errors.New("Got more than 1 patient for treatment when expected just 1")
}

func (d *DataService) GetPatientFromCaseID(patientCaseID int64) (*common.Patient, error) {
	patients, err := d.getPatientBasedOnQuery("patient_case",
		`INNER JOIN patient ON patient_case.patient_id = patient.id`,
		`patient_case.id = ?
			AND (phone IS NULL OR (account_phone.status = 'ACTIVE'))
			AND (zip_code IS NULL OR patient_location.status = 'ACTIVE')`, patientCaseID)
	if err != nil {
		return nil, err
	}
	switch l := len(patients); {
	case l == 1:
		err = d.getOtherInfoForPatient(patients[0])
		return patients[0], err
	case l == 0:
		return nil, ErrNotFound("patient")
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
		return nil, ErrNotFound("patient")
	}

	return nil, errors.New("Got more than 1 patient for treatment when expected just 1")
}

func (d *DataService) GetPatientVisitsForPatient(patientID int64) ([]*common.PatientVisit, error) {
	rows, err := d.db.Query(`
	SELECT id, patient_id, patient_case_id, clinical_pathway_id, layout_version_id, 
	creation_date, submitted_date, closed_date, status, sku_id, followup
	FROM patient_visit 
	WHERE patient_id = ?`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getPatientVisitFromRows(rows)
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

func (d *DataService) UpdatePatientAddress(patientID int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// update any existing address for the address type as inactive
	_, err = tx.Exec(`update patient_address set status=? where patient_id = ? and address_type = ?`, STATUS_INACTIVE, addressType, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert new address
	if addressLine2 != "" {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, address_line_2, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?, ?)`, patientID, addressLine1, addressLine2, city, state, zipCode, addressType, STATUS_ACTIVE)
		if err != nil {
			return err
		}
	} else {
		_, err = tx.Exec(`insert into patient_address (patient_id, address_line_1, city, state, zip_code, address_type, status) values 
							(?, ?, ?, ?, ?, ?, ?)`, patientID, addressLine1, city, state, zipCode, addressType, STATUS_ACTIVE)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) UpdatePatientPharmacy(patientID int64, pharmacyDetails *pharmacy.PharmacyData) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update patient_pharmacy_selection set status=? where patient_id = ?`, STATUS_INACTIVE, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = addPharmacy(pharmacyDetails, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	existingPharmacyId := pharmacyDetails.LocalID

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patientID, existingPharmacyId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) getPatientPharmacySelection(patientID int64) (pharmacySelection *pharmacy.PharmacyData, err error) {
	rows, err := d.db.Query(`select pharmacy_selection.id, patient_id, pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
		from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id 
				where patient_id = ? and status=?`, patientID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		pharmacySelection, err = getPharmacyFromCurrentRow(rows)
	}

	return
}

func (d *DataService) GetPharmacySelectionForPatients(patientIDs []int64) ([]*pharmacy.PharmacyData, error) {
	if len(patientIDs) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(fmt.Sprintf(`select pharmacy_selection.id, patient_id,  pharmacy_selection.pharmacy_id, source, name, address_line_1, address_line_2, city, state, zip_code, phone,lat,lng 
			from patient_pharmacy_selection 
			inner join pharmacy_selection on pharmacy_selection.id = pharmacy_selection_id where patient_id in (%s) and status=?`, enumerateItemsIntoString(patientIDs)), STATUS_ACTIVE)
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

func (d *DataService) GetPharmacyBasedOnReferenceIdAndSource(pharmacyID int64, pharmacySource string) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var id int64
	err := d.db.QueryRow(`select id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat,lng
		from pharmacy_selection where pharmacy_id = ? and source = ?`, pharmacyID, pharmacySource).
		Scan(&id, &addressLine1, &addressLine2, &city, &state, &country, &phone, &zipCode, &name, &lat, &lng)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	pharmacyToReturn := &pharmacy.PharmacyData{
		LocalID:      id,
		SourceID:     pharmacyID,
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

func (d *DataService) GetPharmacyFromID(pharmacyLocalId int64) (*pharmacy.PharmacyData, error) {

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
		LocalID:      pharmacyLocalId,
		SourceID:     pharmacyReferenceId,
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
		"pharmacy_id":    pharmacyDetails.SourceID,
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

	lastID, err := tx.Exec(fmt.Sprintf("insert into pharmacy_selection (%s) values (%s)", strings.Join(columns, ","),
		dbutil.MySQLArgs(len(columns))), dataForColumns...)

	if err != nil {
		return err
	}

	lastInsertId, err := lastID.LastInsertId()
	if err != nil {
		return err
	}

	pharmacyDetails.LocalID = lastInsertId
	return nil
}

func getPharmacyFromCurrentRow(rows *sql.Rows) (*pharmacy.PharmacyData, error) {
	var localId, patientID int64
	var sourceType, name, addressLine1, addressLine2, phone, city, state, zipCode, lat, lng sql.NullString
	var id sql.NullInt64
	err := rows.Scan(&localId, &patientID, &id, &sourceType, &name, &addressLine1, &addressLine2, &city, &state, &zipCode, &phone, &lat, &lng)
	if err != nil {
		return nil, err
	}

	pharmacySelection := &pharmacy.PharmacyData{
		LocalID:      localId,
		PatientID:    patientID,
		SourceID:     id.Int64,
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

func (d *DataService) TrackPatientAgreements(patientID int64, agreements map[string]bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for agreementType, agreed := range agreements {
		_, err = tx.Exec(`update patient_agreement set status=? where patient_id = ? and agreement_type = ?`, STATUS_INACTIVE, patientID, agreementType)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_agreement (patient_id, agreement_type, agreed, status) values (?,?,?,?)`, patientID, agreementType, agreed, STATUS_ACTIVE)
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

func (d *DataService) UpdatePatientWithPaymentCustomerId(patientID int64, paymentCustomerID string) error {
	_, err := d.db.Exec("update patient set payment_service_customer_id = ? where id = ?", paymentCustomerID, patientID)
	return err
}

func (d *DataService) AddCardForPatient(patientID int64, card *common.Card) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// add a new address to db
	addressID, err := addAddress(tx, card.BillingAddress)
	if err != nil {
		tx.Rollback()
		return err
	}

	card.BillingAddress.ID = addressID

	if card.IsDefault {
		// undo all previous default cards for the patient
		_, err = tx.Exec(`update credit_card set is_default = 0 where patient_id = ?`, patientID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// add new card as the default card
	lastID, err := tx.Exec(`
		INSERT INTO credit_card (
			third_party_card_id, fingerprint, type, patient_id,
			address_id, is_default, label, status, apple_pay
		) VALUES (?,?,?,?,?,?,?,?,?)`,
		card.ThirdPartyID, card.Fingerprint, card.Type, patientID,
		addressID, card.IsDefault, card.Label, STATUS_ACTIVE, card.ApplePay)
	if err != nil {
		tx.Rollback()
		return err
	}

	cardID, err := lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	card.ID = encoding.NewObjectID(cardID)
	return tx.Commit()
}

func (d *DataService) MakeCardDefaultForPatient(patientID int64, card *common.Card) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update credit_card set is_default = 0 where patient_id = ?`, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update credit_card set is_default = 1 where id = ?`, card.ID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) MarkCardInactiveForPatient(patientID int64, card *common.Card) error {
	_, err := d.db.Exec(`update credit_card set status = ? where patient_id = ? and id = ?`, STATUS_DELETED, patientID, card.ID.Int64())
	return err
}

func (d *DataService) DeleteCardForPatient(patientID int64, card *common.Card) error {
	_, err := d.db.Exec(`delete from credit_card where patient_id = ? and id = ?`, patientID, card.ID.Int64())
	return err
}

func (d *DataService) MakeLatestCardDefaultForPatient(patientID int64) (*common.Card, error) {
	var latestCardId int64
	err := d.db.QueryRow(`select id from credit_card where patient_id = ? and status = ? AND apple_pay = false order by creation_date desc limit 1`, patientID, STATUS_ACTIVE).Scan(&latestCardId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_, err = d.db.Exec(`update credit_card set is_default = 1 where patient_id = ? and id = ?`, patientID, latestCardId)
	if err != nil {
		return nil, err
	}

	card, err := d.GetCardFromID(latestCardId)
	if err != nil {
		return nil, err
	}
	return card, err
}

func addAddress(tx *sql.Tx, address *common.Address) (int64, error) {
	lastID, err := tx.Exec(`insert into address (address_line_1, address_line_2, city, state, zip_code, country) values (?,?,?,?,?,?)`,
		address.AddressLine1, address.AddressLine2, address.City, address.State, address.ZipCode, addressUsa)
	if err != nil {
		return 0, err
	}

	addressID, err := lastID.LastInsertId()
	if err != nil {
		return 0, err
	}

	return addressID, nil
}

func (d *DataService) GetCardsForPatient(patientID int64) ([]*common.Card, error) {
	cards := make([]*common.Card, 0)

	rows, err := d.db.Query(`
		SELECT id, third_party_card_id, fingerprint, type, is_default, creation_date, apple_pay
		FROM credit_card
		WHERE patient_id = ? AND status = ?
		ORDER BY id`, patientID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var card common.Card
		if err := rows.Scan(
			&card.ID, &card.ThirdPartyID, &card.Fingerprint, &card.Type,
			&card.IsDefault, &card.CreationDate, &card.ApplePay,
		); err != nil {
			return nil, err
		}
		cards = append(cards, &card)
	}

	return cards, rows.Err()
}

func (d *DataService) GetDefaultCardForPatient(patientID int64) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay from credit_card where patient_id = ? and is_default = 1`,
		patientID)
	return d.getCardFromRow(row)
}

func (d *DataService) GetCardFromID(cardID int64) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay from credit_card where id = ?`,
		cardID)
	return d.getCardFromRow(row)
}

func (d *DataService) GetCardFromThirdPartyID(thirdPartyID string) (*common.Card, error) {
	row := d.db.QueryRow(`select id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay from credit_card where third_party_card_id = ?`,
		thirdPartyID)
	return d.getCardFromRow(row)
}

func (d *DataService) getCardFromRow(row *sql.Row) (*common.Card, error) {
	var card common.Card
	var addressID int64
	err := row.Scan(
		&card.ID, &card.ThirdPartyID, &card.Fingerprint, &card.Type,
		&addressID, &card.IsDefault, &card.CreationDate, &card.ApplePay)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("credit_card")
	} else if err != nil {
		return nil, err
	}
	var addressLine1, addressLine2, city, state, country, zipCode sql.NullString
	err = d.db.QueryRow(`select address_line_1, address_line_2, city, state, zip_code, country from address where id = ?`, addressID).Scan(&addressLine1, &addressLine2, &city, &state, &zipCode, &country)
	if err != nil {
		if err == sql.ErrNoRows {
			return &card, nil
		}
		return nil, err
	}
	card.BillingAddress = &common.Address{
		ID:           addressID,
		AddressLine1: addressLine1.String,
		AddressLine2: addressLine2.String,
		City:         city.String,
		State:        state.String,
		ZipCode:      zipCode.String,
		Country:      country.String,
	}
	return &card, nil
}

func (d *DataService) UpdateDefaultAddressForPatient(patientID int64, address *common.Address) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if address.ID == 0 {
		address.ID, err = addAddress(tx, address)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`delete from patient_address_selection where patient_id = ?`, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into patient_address_selection (patient_id, address_id, is_default, is_updated_by_doctor) values (?,?,1,0)`, patientID, address.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) DeleteAddress(addressID int64) error {
	_, err := d.db.Exec(`delete from address where id = ?`, addressID)
	return err
}

func (d *DataService) CreatePendingTask(workType, status string, itemID int64) (int64, error) {
	lastID, err := d.db.Exec(`insert into pending_task (type, item_id, status) values (?,?,?)`, workType, itemID, status)
	if err != nil {
		return 0, err
	}

	pendingTaskID, err := lastID.LastInsertId()
	if err != nil {
		return 0, err
	}

	return pendingTaskID, nil
}

func (d *DataService) DeletePendingTask(pendingTaskID int64) error {
	_, err := d.db.Exec(`delete from pending_task where id = ?`, pendingTaskID)
	return err
}

func (d *DataService) AddAlertsForPatient(patientID int64, source string, alerts []*common.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// inactivate any alerts for the patient that have updated alerts from the same source
	alertSourceIds := make([]int64, len(alerts))
	for i, alert := range alerts {
		alertSourceIds[i] = alert.SourceID
	}

	vals := make([]interface{}, 0, len(alerts)+3)
	vals = append(vals, common.PAStatusInactive, patientID, source)
	vals = dbutil.AppendInt64sToInterfaceSlice(vals, alertSourceIds)

	_, err = tx.Exec(`
		UPDATE patient_alerts SET status = ?
		WHERE patient_id = ? AND source = ?
		AND source_id in (`+dbutil.MySQLArgs(len(alertSourceIds))+`)`, vals...)
	if err != nil {
		tx.Rollback()
		return err
	}

	fields := make([]string, 0, len(alerts))
	values := make([]interface{}, 0, 5*len(alerts))
	for _, alert := range alerts {
		values = append(values, alert.PatientID, alert.Message, alert.Source, alert.SourceID, alert.Status)
		fields = append(fields, "(?,?,?,?,?)")
	}

	_, err = tx.Exec(`INSERT INTO patient_alerts (patient_id, alert, source, source_id, status) VALUES `+strings.Join(fields, ","), values...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetAlertsForPatient(patientID int64) ([]*common.Alert, error) {
	rows, err := d.db.Query(`
		SELECT id, patient_id, creation_date, alert, source, source_id, status
		FROM patient_alerts WHERE patient_id = ? AND status = ?`, patientID, common.PAStatusActive)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	var alerts []*common.Alert
	for rows.Next() {
		alert := &common.Alert{}
		if err := rows.Scan(&alert.ID, &alert.PatientID, &alert.CreationDate, &alert.Message, &alert.Source, &alert.SourceID, &alert.Status); err != nil {
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

func (d *DataService) GetPatientPCP(patientID int64) (*common.PCP, error) {
	var pcp common.PCP
	err := d.db.QueryRow(`select patient_id, physician_name, phone_number, practice_name, email, fax_number from patient_pcp where patient_id = ?`, patientID).Scan(
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

func (d *DataService) DeletePatientPCP(patientID int64) error {
	_, err := d.db.Exec(`delete from patient_pcp where patient_id = ?`, patientID)
	return err
}

func (d *DataService) UpdatePatientEmergencyContacts(patientID int64, emergencyContacts []*common.EmergencyContact) error {
	tx, err := d.db.Begin()
	if err != nil {
		return nil
	}

	// delete any existing emergency contacts for the patient
	_, err = tx.Exec(`delete from patient_emergency_contact where patient_id = ?`, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// add all emergency contacts
	for _, eContact := range emergencyContacts {
		res, err := tx.Exec(`insert into patient_emergency_contact (patient_id, full_name, phone_number, relationship) values (?,?,?,?)`, patientID, eContact.FullName, eContact.PhoneNumber, eContact.Relationship)
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

func (d *DataService) GetPatientEmergencyContacts(patientID int64) ([]*common.EmergencyContact, error) {
	rows, err := d.db.Query(`select id, patient_id, full_name, phone_number, relationship from patient_emergency_contact where patient_id = ?`, patientID)
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

func (d *DataService) GetActiveMembersOfCareTeamForPatient(patientID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_care_provider_assignment 
		inner join role_type on role_type_id = role_type.id
		where status = ? and patient_id = ?`, STATUS_ACTIVE, patientID)
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
				doctor, err := d.Doctor(assignment.ProviderID, true)
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
		var patientID, accountID, erxPatientID encoding.ObjectID
		var dobMonth, dobYear, dobDay int
		var personID int64
		var training bool
		err = rows.Scan(&patientID, &erxPatientID, &paymentServiceCustomerId, &accountID, &email, &firstName, &middleName, &lastName, &suffix, &prefix,
			&zipCode, &city, &state, &phone, &phoneType, &gender, &dobYear, &dobMonth, &dobDay, &status, &training, &personID)
		if err != nil {
			return nil, err
		}

		patient := &common.Patient{
			PatientID:         patientID,
			PaymentCustomerID: paymentServiceCustomerId.String,
			FirstName:         firstName,
			LastName:          lastName,
			Prefix:            prefix.String,
			Suffix:            suffix.String,
			MiddleName:        middleName.String,
			Email:             email.String,
			Status:            status,
			Gender:            gender,
			AccountID:         accountID,
			ZipCode:           zipCode.String,
			CityFromZipCode:   city.String,
			StateFromZipCode:  state.String,
			ERxPatientID:      erxPatientID,
			Training:          training,
			DOB:               encoding.DOB{Year: dobYear, Month: dobMonth, Day: dobDay},
			PhoneNumbers: []*common.PhoneNumber{
				&common.PhoneNumber{
					Phone: phone,
					Type:  phoneType.String,
				},
			},
			PersonID:   personID,
			IsUnlinked: status == PATIENT_UNLINKED,
		}

		patient.Pharmacy, err = d.getPatientPharmacySelection(patient.PatientID.Int64())
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
		patient.PatientID.Int64(),
	).Scan(
		&defaultPatientAddress.ID, &defaultPatientAddress.AddressLine1,
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
	patient.PromptStatus, err = d.GetPushPromptStatus(patient.AccountID.Int64())
	if err != nil {
		return err
	}

	rows, err := d.db.Query(`SELECT phone, phone_type FROM account_phone WHERE account_id = ? AND status = ?`,
		patient.AccountID.Int64(), STATUS_INACTIVE)
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
		return "", ErrNotFound("patient_location")
	}
	return patientState, err
}
