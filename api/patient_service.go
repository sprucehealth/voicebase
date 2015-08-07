package api

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/pharmacy"
)

func (d *dataService) RegisterPatient(patient *common.Patient) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	if err := d.createPatientWithStatus(patient, PatientRegistered, tx); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

func (d *dataService) UpdatePatient(id int64, update *PatientUpdate, updateFromDoctor bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	if err := d.updatePatient(tx, id, update, updateFromDoctor); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

func (d *dataService) updatePatient(tx *sql.Tx, id int64, update *PatientUpdate, updateFromDoctor bool) error {
	args := dbutil.MySQLVarArgs()

	if update.FirstName != nil {
		args.Append("first_name", *update.FirstName)
	}
	if update.MiddleName != nil {
		args.Append("middle_name", *update.MiddleName)
	}
	if update.LastName != nil {
		args.Append("last_name", *update.LastName)
	}
	if update.Prefix != nil {
		args.Append("prefix", *update.Prefix)
	}
	if update.Suffix != nil {
		args.Append("suffix", *update.Suffix)
	}
	if update.DOB != nil {
		args.Append("dob_day", update.DOB.Day)
		args.Append("dob_month", update.DOB.Month)
		args.Append("dob_year", update.DOB.Year)
	}
	if update.Gender != nil {
		args.Append("gender", strings.ToLower(*update.Gender))
	}
	if update.ERxID != nil {
		args.Append("erx_patient_id", *update.ERxID)
	}
	if update.StripeCustomerID != nil {
		args.Append("payment_service_customer_id", *update.StripeCustomerID)
	}
	if update.HasParentalConsent != nil {
		args.Append("has_parental_consent", *update.HasParentalConsent)
	}

	if !args.IsEmpty() {
		_, err := tx.Exec(`UPDATE patient SET `+args.Columns()+` WHERE id = ?`, append(args.Values(), id)...)
		if err != nil {
			return errors.Trace(err)
		}
	}

	if len(update.PhoneNumbers) != 0 {
		accountID, err := accountIDForPatient(tx, id)
		if err != nil {
			return errors.Trace(err)
		}
		if err := replaceAccountPhoneNumbers(tx, accountID, update.PhoneNumbers); err != nil {
			return errors.Trace(err)
		}
	}

	if update.Address != nil {
		if err := updatePatientAddress(tx, id, update.Address, updateFromDoctor); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func replaceAccountPhoneNumbers(tx *sql.Tx, accountID int64, numbers []*common.PhoneNumber) error {
	_, err := tx.Exec(`DELETE FROM account_phone WHERE account_id = ?`, accountID)
	if err != nil {
		return errors.Trace(err)
	}

	// Make sure there's at least one and only one active phone number
	hasActive := false
	for _, p := range numbers {
		if p.Status == StatusActive {
			if hasActive {
				p.Status = StatusInactive
			} else {
				hasActive = true
			}
		} else if p.Status == "" {
			p.Status = StatusInactive
		}
	}
	if !hasActive {
		numbers[0].Status = StatusActive
	}

	inserts := dbutil.MySQLMultiInsert(len(numbers))
	for _, p := range numbers {
		inserts.Append(accountID, p.Phone.String(), p.Type.String(), p.Status, p.Verified)
	}
	_, err = tx.Exec(`
			INSERT INTO account_phone (account_id, phone, phone_type, status, verified)
			VALUES `+inserts.Query(), inserts.Values()...)
	return errors.Trace(err)
}

func updatePatientAddress(tx *sql.Tx, patientID int64, address *common.Address, updateFromDoctor bool) error {
	addressID, err := addAddress(tx, address)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = tx.Exec(`DELETE FROM patient_address_selection WHERE patient_id = ?`, patientID)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = tx.Exec(`
		INSERT INTO patient_address_selection
			(address_id, patient_id, is_default, is_updated_by_doctor)
		VALUES (?, ?, ?, ?)`, addressID, patientID, true, updateFromDoctor)
	return errors.Trace(err)
}

func (d *dataService) CreateUnlinkedPatientFromRefillRequest(patient *common.Patient, doctor *common.Doctor, pathwayTag string) error {
	tx, err := d.db.Begin()

	// create an account with no email and password for the unmatched patient
	lastID, err := tx.Exec(`insert into account (email, password, role_type_id) values (NULL,NULL, ?)`, d.roleTypeMapping[RolePatient])
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	accountID, err := lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	patient.AccountID = encoding.NewObjectID(accountID)

	// create an account
	if err := d.createPatientWithStatus(patient, PatientUnlinked, tx); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	// create address for patient
	if patient.PatientAddress != nil {
		addressID, err := addAddress(tx, patient.PatientAddress)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}

		_, err = tx.Exec(
			`INSERT INTO patient_address_selection (address_id, patient_id, is_default, is_updated_by_doctor) VALUES (?,?,1,0)`,
			addressID, patient.ID.Int64())
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	if patient.Pharmacy != nil {
		var existingPharmacyID int64
		err = tx.QueryRow(`SELECT id FROM pharmacy_selection WHERE pharmacy_id = ?`, patient.Pharmacy.SourceID).Scan(&existingPharmacyID)
		if err != nil && err != sql.ErrNoRows {
			tx.Rollback()
			return errors.Trace(err)
		}

		if existingPharmacyID == 0 {
			err = addPharmacy(patient.Pharmacy, tx)
			if err != nil {
				tx.Rollback()
				return errors.Trace(err)
			}
			existingPharmacyID = patient.Pharmacy.LocalID
		}

		_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) values (?,?,?)`, patient.ID.Int64(), existingPharmacyID, StatusActive)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	// create additional phone numbers for patient
	if len(patient.PhoneNumbers) > 1 {
		for _, phoneNumber := range patient.PhoneNumbers[1:] {
			_, err = tx.Exec(`INSERT INTO account_phone (account_id, phone, phone_type, status) VALUES (?,?,?,?)`,
				patient.AccountID.Int64(), phoneNumber.Phone.String(), phoneNumber.Type, StatusInactive)
			if err != nil {
				tx.Rollback()
				return errors.Trace(err)
			}
		}
	}

	// assign the erx patient id to the patient
	_, err = tx.Exec(`UPDATE patient SET erx_patient_id = ? WHERE id = ?`, patient.ERxPatientID.Int64(), patient.ID.Int64())
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	patientCase := &common.PatientCase{
		PatientID:  patient.ID,
		PathwayTag: pathwayTag,
		Status:     common.PCStatusInactive,
	}

	// create a case for the patient
	if err := d.createPatientCase(tx, patientCase); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	// assign the doctor to the case and patient
	if err := d.assignCareProviderToPatientFileAndCase(tx, doctor.ID.Int64(), d.roleTypeMapping[RoleDoctor], patientCase); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

func (d *dataService) createPatientWithStatus(patient *common.Patient, status string, tx *sql.Tx) error {
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
		return errors.Trace(err)
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return errors.Trace(err)
	}

	if len(patient.PhoneNumbers) > 0 {
		if err := replaceAccountPhoneNumbers(tx, patient.AccountID.Int64(), patient.PhoneNumbers); err != nil {
			return errors.Trace(err)
		}
	}

	_, err = tx.Exec(`
		INSERT INTO patient_location (patient_id, zip_code, city, state, status)
		VALUES (?, ?, ?, ?, ?)`, lastID, patient.ZipCode, patient.CityFromZipCode,
		patient.StateFromZipCode, StatusActive)
	if err != nil {
		return errors.Trace(err)
	}

	res, err = tx.Exec(`INSERT INTO person (role_type_id, role_id) VALUES (?, ?)`, d.roleTypeMapping[RolePatient], lastID)
	if err != nil {
		return errors.Trace(err)
	}
	patient.PersonID, err = res.LastInsertId()
	if err != nil {
		return errors.Trace(err)
	}

	patient.ID = encoding.NewObjectID(lastID)
	return nil
}

func (d *dataService) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	var patientID int64
	err := d.db.QueryRow("SELECT id FROM patient WHERE account_id = ?", accountID).Scan(&patientID)
	return patientID, err
}

func (d *dataService) AddDoctorToCareTeamForPatient(patientID, doctorID int64, pathwayTag string) error {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`
		INSERT INTO patient_care_provider_assignment
			(patient_id, clinical_pathway_id, provider_id, role_type_id, status)
		VALUES (?,?,?,?,?)`,
		patientID, pathwayID, doctorID, d.roleTypeMapping[RoleDoctor], StatusActive)
	return err
}

func (d *dataService) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
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

func (d *dataService) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	if !basicInfoOnly {
		return d.GetPatientFromID(id)
	}

	row := d.db.QueryRow(`
		SELECT id, account_id, first_name, last_name, gender, status, account_id,
			dob_month, dob_year, dob_day, payment_service_customer_id, erx_patient_id,
			has_parental_consent
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

func (d *dataService) Patients(ids []int64) (map[int64]*common.Patient, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, account_id, first_name, last_name, gender, status, account_id,
			dob_month, dob_year, dob_day, payment_service_customer_id, erx_patient_id,
			has_parental_consent
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
		patients[patient.ID.Int64()] = patient
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return patients, nil
}

func scanRowForPatient(scanner dbutil.Scanner) (*common.Patient, error) {
	var patient common.Patient
	var dobMonth, dobDay, dobYear int
	var stripeID sql.NullString
	err := scanner.Scan(
		&patient.ID,
		&patient.AccountID,
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
		&patient.HasParentalConsent,
	)
	if err != nil {
		return nil, err
	}

	patient.PaymentCustomerID = stripeID.String
	patient.DOB = encoding.Date{
		Month: dobMonth,
		Day:   dobDay,
		Year:  dobYear,
	}
	return &patient, nil
}

func (d *dataService) GetPatientFromID(patientID int64) (*common.Patient, error) {
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

func (d *dataService) GetPatientsForIDs(patientIDs []int64) ([]*common.Patient, error) {
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

func (d *dataService) GetPatientFromTreatmentPlanID(treatmentPlanID int64) (*common.Patient, error) {
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

func (d *dataService) GetPatientFromPatientVisitID(patientVisitID int64) (*common.Patient, error) {
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

func (d *dataService) GetPatientFromErxPatientID(erxPatientID int64) (*common.Patient, error) {
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

func (d *dataService) AnyVisitSubmitted(patientID int64) (bool, error) {
	var count int64
	if err := d.db.QueryRow(`
		SELECT count(*)
		FROM patient_visit
		WHERE patient_visit.status NOT IN (?,?,?) AND patient_id = ? LIMIT 1`,
		common.PVStatusOpen, common.PVStatusDeleted, common.PVStatusPreSubmissionTriage, patientID).Scan(&count); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (d *dataService) UpdatePatientPharmacy(patientID int64, pharmacyDetails *pharmacy.PharmacyData) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE patient_pharmacy_selection SET status = ? WHERE patient_id = ?`, StatusInactive, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = addPharmacy(pharmacyDetails, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	existingPharmacyID := pharmacyDetails.LocalID

	_, err = tx.Exec(`INSERT INTO patient_pharmacy_selection (patient_id, pharmacy_selection_id, status) VALUES (?,?,?)`,
		patientID, existingPharmacyID, StatusActive)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *dataService) getPatientPharmacySelection(patientID int64) (*pharmacy.PharmacyData, error) {
	row := d.db.QueryRow(`
		SELECT pharmacy_selection.id, patient_id, pharmacy_selection.pharmacy_id, source,
			name, address_line_1, address_line_2, city, state, zip_code, phone, lat, lng
		FROM patient_pharmacy_selection
		INNER JOIN pharmacy_selection ON pharmacy_selection.id = pharmacy_selection_id
		WHERE patient_id = ? AND status = ?
		LIMIT 1`, patientID, StatusActive)
	return scanPharmacy(row)
}

func (d *dataService) GetPharmacyBasedOnReferenceIDAndSource(pharmacyID int64, pharmacySource string) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var id int64
	err := d.db.QueryRow(`
		SELECT id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat, lng
		FROM pharmacy_selection
		WHERE pharmacy_id = ? AND source = ?`, pharmacyID, pharmacySource).
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

func (d *dataService) GetPharmacyFromID(pharmacyLocalID int64) (*pharmacy.PharmacyData, error) {
	var addressLine1, addressLine2, city, state, country, phone, zipCode, lat, lng, name sql.NullString
	var source string
	var pharmacyReferenceID int64
	err := d.db.QueryRow(`
		SELECT source, pharmacy_id, address_line_1, address_line_2, city, state, country, phone, zip_code, name, lat, lng
		FROM pharmacy_selection
		WHERE id = ?`, pharmacyLocalID).
		Scan(&source, &pharmacyReferenceID, &addressLine1, &addressLine2, &city, &state, &country, &phone, &zipCode, &name, &lat, &lng)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	pharmacyToReturn := &pharmacy.PharmacyData{
		LocalID:      pharmacyLocalID,
		SourceID:     pharmacyReferenceID,
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

func (d *dataService) AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error {
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

	lastID, err := tx.Exec(fmt.Sprintf("INSERT INTO pharmacy_selection (%s) VALUES (%s)", strings.Join(columns, ","),
		dbutil.MySQLArgs(len(columns))), dataForColumns...)

	if err != nil {
		return err
	}

	lastInsertID, err := lastID.LastInsertId()
	if err != nil {
		return err
	}

	pharmacyDetails.LocalID = lastInsertID
	return nil
}

func scanPharmacy(row scannable) (*pharmacy.PharmacyData, error) {
	var localID, patientID int64
	var sourceType, name, addressLine1, addressLine2, phone, city, state, zipCode, lat, lng sql.NullString
	var id sql.NullInt64
	err := row.Scan(
		&localID, &patientID, &id, &sourceType, &name, &addressLine1,
		&addressLine2, &city, &state, &zipCode, &phone, &lat, &lng)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("pharmacy_selection")
	} else if err != nil {
		return nil, err
	}

	pharmacySelection := &pharmacy.PharmacyData{
		LocalID:      localID,
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

func (d *dataService) TrackPatientAgreements(patientID int64, agreements map[string]bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for agreementType, agreed := range agreements {
		_, err = tx.Exec(`UPDATE patient_agreement SET status = ? WHERE patient_id = ? AND agreement_type = ?`,
			StatusInactive, patientID, agreementType)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`INSERT INTO patient_agreement (patient_id, agreement_type, agreed, status) VALUES (?,?,?,?)`,
			patientID, agreementType, agreed, StatusActive)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *dataService) PatientAgreements(patientID int64) (map[string]time.Time, error) {
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

func (d *dataService) AddCardForPatient(patientID int64, card *common.Card) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// add a new address to db
	var addressID *int64
	if card.BillingAddress != nil {
		aID, err := addAddress(tx, card.BillingAddress)
		if err != nil {
			tx.Rollback()
			return err
		}
		card.BillingAddress.ID = aID
		addressID = &aID
	}

	if card.IsDefault {
		// undo all previous default cards for the patient
		_, err = tx.Exec(`UPDATE credit_card SET is_default = 0 WHERE patient_id = ?`, patientID)
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
		addressID, card.IsDefault, card.Label, StatusActive, card.ApplePay)
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

func (d *dataService) MakeCardDefaultForPatient(patientID int64, card *common.Card) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE credit_card SET is_default = 0 WHERE patient_id = ?`, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`UPDATE credit_card SET is_default = 1 WHERE id = ?`, card.ID.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *dataService) MarkCardInactiveForPatient(patientID int64, card *common.Card) error {
	_, err := d.db.Exec(`UPDATE credit_card SET status = ? WHERE patient_id = ? AND id = ?`, StatusDeleted, patientID, card.ID.Int64())
	return err
}

func (d *dataService) DeleteCardForPatient(patientID int64, card *common.Card) error {
	_, err := d.db.Exec(`DELETE FROM credit_card WHERE patient_id = ? AND id = ?`, patientID, card.ID.Int64())
	return err
}

func (d *dataService) MakeLatestCardDefaultForPatient(patientID int64) (*common.Card, error) {
	var latestCardID int64
	err := d.db.QueryRow(`
		SELECT id
		FROM credit_card
		WHERE patient_id = ? AND status = ? AND apple_pay = false
		ORDER BY creation_date DESC
		LIMIT 1`, patientID, StatusActive).Scan(&latestCardID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	_, err = d.db.Exec(`UPDATE credit_card SET is_default = 1 WHERE patient_id = ? AND id = ?`, patientID, latestCardID)
	if err != nil {
		return nil, err
	}

	card, err := d.GetCardFromID(latestCardID)
	if err != nil {
		return nil, err
	}
	return card, err
}

func addAddress(tx *sql.Tx, address *common.Address) (int64, error) {
	lastID, err := tx.Exec(`INSERT INTO address (address_line_1, address_line_2, city, state, zip_code, country) VALUES (?,?,?,?,?,?)`,
		strings.TrimSpace(address.AddressLine1), strings.TrimSpace(address.AddressLine2),
		strings.TrimSpace(address.City), strings.TrimSpace(address.State),
		strings.TrimSpace(address.ZipCode), addressUSA)
	if err != nil {
		return 0, err
	}

	addressID, err := lastID.LastInsertId()
	if err != nil {
		return 0, err
	}

	return addressID, nil
}

func (d *dataService) GetCardsForPatient(patientID int64) ([]*common.Card, error) {
	rows, err := d.db.Query(`
		SELECT id, third_party_card_id, fingerprint, type, is_default, creation_date, apple_pay
		FROM credit_card
		WHERE patient_id = ? AND status = ?
		ORDER BY id`, patientID, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []*common.Card
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

func (d *dataService) GetDefaultCardForPatient(patientID int64) (*common.Card, error) {
	row := d.db.QueryRow(`
		SELECT id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay
		FROM credit_card
		WHERE patient_id = ? AND is_default = 1`,
		patientID)
	return d.getCardFromRow(row)
}

func (d *dataService) GetCardFromID(cardID int64) (*common.Card, error) {
	row := d.db.QueryRow(`
		SELECT id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay
		FROM credit_card
		WHERE id = ?`,
		cardID)
	return d.getCardFromRow(row)
}

func (d *dataService) GetCardFromThirdPartyID(thirdPartyID string) (*common.Card, error) {
	row := d.db.QueryRow(`
		SELECT id, third_party_card_id, fingerprint, type, address_id, is_default, creation_date, apple_pay
		FROM credit_card
		WHERE third_party_card_id = ?`,
		thirdPartyID)
	return d.getCardFromRow(row)
}

func (d *dataService) getCardFromRow(row *sql.Row) (*common.Card, error) {
	var card common.Card
	var addressID sql.NullInt64
	err := row.Scan(
		&card.ID, &card.ThirdPartyID, &card.Fingerprint, &card.Type,
		&addressID, &card.IsDefault, &card.CreationDate, &card.ApplePay)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("credit_card")
	} else if err != nil {
		return nil, err
	}

	if addressID.Valid {
		var addressLine1, addressLine2, city, state, country, zipCode sql.NullString
		err = d.db.QueryRow(`
			SELECT address_line_1, address_line_2, city, state, zip_code, country
			FROM address
			WHERE id = ?`, addressID.Int64).Scan(&addressLine1, &addressLine2, &city, &state, &zipCode, &country)
		if err != nil {
			if err == sql.ErrNoRows {
				return &card, nil
			}
			return nil, err
		}

		card.BillingAddress = &common.Address{
			ID:           addressID.Int64,
			AddressLine1: addressLine1.String,
			AddressLine2: addressLine2.String,
			City:         city.String,
			State:        state.String,
			ZipCode:      zipCode.String,
			Country:      country.String,
		}
	}

	return &card, nil
}

func (d *dataService) UpdateDefaultAddressForPatient(patientID int64, address *common.Address) error {
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

	_, err = tx.Exec(`DELETE FROM patient_address_selection WHERE patient_id = ?`, patientID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`INSERT INTO patient_address_selection (patient_id, address_id, is_default, is_updated_by_doctor) VALUES (?,?,1,0)`,
		patientID, address.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *dataService) DeleteAddress(addressID int64) error {
	_, err := d.db.Exec(`delete from address where id = ?`, addressID)
	return err
}

func (d *dataService) CreatePendingTask(workType, status string, itemID int64) (int64, error) {
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

func (d *dataService) DeletePendingTask(pendingTaskID int64) error {
	_, err := d.db.Exec(`delete from pending_task where id = ?`, pendingTaskID)
	return err
}

func (d *dataService) UpdatePatientPCP(pcp *common.PCP) error {
	_, err := d.db.Exec(`replace into patient_pcp (patient_id, physician_name, phone_number, practice_name, email, fax_number) values (?,?,?,?,?,?)`, pcp.PatientID, pcp.PhysicianName, pcp.PhoneNumber,
		pcp.PracticeName, pcp.Email, pcp.FaxNumber)
	return err
}

func (d *dataService) GetPatientPCP(patientID int64) (*common.PCP, error) {
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

func (d *dataService) DeletePatientPCP(patientID int64) error {
	_, err := d.db.Exec(`DELETE FROM patient_pcp WHERE patient_id = ?`, patientID)
	return err
}

func (d *dataService) UpdatePatientEmergencyContacts(patientID int64, emergencyContacts []*common.EmergencyContact) error {
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

func (d *dataService) GetPatientEmergencyContacts(patientID int64) ([]*common.EmergencyContact, error) {
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

func (d *dataService) GetActiveMembersOfCareTeamForPatient(patientID int64, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	rows, err := d.db.Query(`select provider_id, role_type_tag, status, creation_date from patient_care_provider_assignment
		inner join role_type on role_type_id = role_type.id
		where status = ? and patient_id = ?`, StatusActive, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getMembersOfCareTeam(rows, fillInDetails)
}

func (d *dataService) getMembersOfCareTeam(rows *sql.Rows, fillInDetails bool) ([]*common.CareProviderAssignment, error) {
	var assignments []*common.CareProviderAssignment
	for rows.Next() {
		var assignment common.CareProviderAssignment
		if err := rows.Scan(&assignment.ProviderID, &assignment.ProviderRole, &assignment.Status, &assignment.CreationDate); err != nil {
			return nil, err
		}

		if fillInDetails {
			switch assignment.ProviderRole {
			case RoleDoctor, RoleCC:
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
			}
		}

		assignments = append(assignments, &assignment)

	}

	// sort by role so that the doctors are shown first in the care team
	sort.Sort(ByCareProviderRole(assignments))
	return assignments, rows.Err()
}

func (d *dataService) getPatientBasedOnQuery(table, joins, where string, queryParams ...interface{}) ([]*common.Patient, error) {
	queryStr := fmt.Sprintf(`
		SELECT patient.id, patient.erx_patient_id, patient.payment_service_customer_id, patient.account_id,
			COALESCE(a.email, ''), first_name, COALESCE(middle_name, ''), last_name, COALESCE(suffix, ''), COALESCE(prefix, ''),
			zip_code, city, state, phone, phone_type, gender, dob_year, dob_month, dob_day, patient.status,
			patient.training, person.id, patient.has_parental_consent
		FROM %s
		%s
		INNER JOIN person ON role_type_id = %d AND role_id = patient.id
		LEFT OUTER JOIN account_phone ON account_phone.account_id = patient.account_id
		LEFT OUTER JOIN patient_location ON patient_location.patient_id = patient.id
		INNER JOIN account a ON a.id = patient.account_id
		WHERE %s`, table, joins, d.roleTypeMapping[RolePatient], where)
	rows, err := d.db.Query(queryStr, queryParams...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var patients []*common.Patient
	for rows.Next() {
		p := &common.Patient{}
		var phoneType, zipCode, city, state, paymentServiceCustomerID sql.NullString
		var phone common.Phone
		var dobMonth, dobYear, dobDay int
		err = rows.Scan(&p.ID, &p.ERxPatientID, &paymentServiceCustomerID, &p.AccountID, &p.Email,
			&p.FirstName, &p.MiddleName, &p.LastName, &p.Suffix, &p.Prefix, &zipCode, &city, &state, &phone,
			&phoneType, &p.Gender, &dobYear, &dobMonth, &dobDay, &p.Status, &p.Training, &p.PersonID,
			&p.HasParentalConsent)
		if err != nil {
			return nil, errors.Trace(err)
		}
		p.IsUnlinked = p.Status == PatientUnlinked
		p.DOB = encoding.Date{Year: dobYear, Month: dobMonth, Day: dobDay}
		p.PaymentCustomerID = paymentServiceCustomerID.String
		p.ZipCode = zipCode.String
		p.CityFromZipCode = city.String
		p.StateFromZipCode = state.String

		if phone.String() != "" {
			phoneNumberType, err := common.ParsePhoneNumberType(phoneType.String)
			if err != nil {
				return nil, errors.Trace(err)
			}
			p.PhoneNumbers = []*common.PhoneNumber{
				&common.PhoneNumber{
					Phone: phone,
					Type:  phoneNumberType,
				},
			}
		}

		p.Pharmacy, err = d.getPatientPharmacySelection(p.ID.Int64())
		if err != nil && !IsErrNotFound(err) {
			return nil, errors.Trace(err)
		}

		patients = append(patients, p)
	}

	return patients, errors.Trace(rows.Err())
}

func (d *dataService) getOtherInfoForPatient(patient *common.Patient) error {
	var defaultPatientAddress common.Address

	// get default address information (if exists) for each patient
	err := d.db.QueryRow(`
		SELECT address.id, address_line_1, address_line_2, city, state, zip_code,
			country from patient_address_selection
		INNER JOIN address ON address_id = address.id
		WHERE patient_id = ? AND is_default = 1`,
		patient.ID.Int64(),
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

	return nil
}

func (d *dataService) PatientLocation(patientID int64) (zipcode string, state string, err error) {
	err = d.db.QueryRow(`SELECT zip_code, state FROM patient_location WHERE patient_id = ?`, patientID).Scan(&zipcode, &state)
	if err == sql.ErrNoRows {
		return "", "", ErrNotFound("patient_location")
	}
	return zipcode, state, err
}
