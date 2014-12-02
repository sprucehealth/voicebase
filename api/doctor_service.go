package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
)

func (d *DataService) RegisterDoctor(doctor *common.Doctor) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		insert into doctor (account_id, first_name, last_name, short_title, long_title, short_display_name, long_display_name, suffix, prefix, middle_name, gender, dob_year, dob_month, dob_day, status, clinician_id)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doctor.AccountId.Int64(), doctor.FirstName, doctor.LastName, doctor.ShortTitle, doctor.LongTitle, doctor.ShortDisplayName, doctor.LongDisplayName,
		doctor.MiddleName, doctor.Suffix, doctor.Prefix, doctor.Gender, doctor.DOB.Year, doctor.DOB.Month, doctor.DOB.Day,
		DOCTOR_REGISTERED, doctor.DoseSpotClinicianId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return 0, err
	}

	doctor.DoctorId = encoding.NewObjectId(lastId)
	doctor.DoctorAddress.Id, err = d.addAddress(tx, doctor.DoctorAddress)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = tx.Exec(`insert into doctor_address_selection (doctor_id, address_id) values (?,?)`, lastId, doctor.DoctorAddress.Id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if doctor.CellPhone != "" {
		_, err = tx.Exec(`INSERT INTO account_phone (phone, phone_type, account_id, status) VALUES (?,?,?,?) `,
			doctor.CellPhone.String(), PHONE_CELL, doctor.AccountId.Int64(), STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	res, err = tx.Exec(`INSERT INTO person (role_type_id, role_id) VALUES (?, ?)`, d.roleTypeMapping[DOCTOR_ROLE], lastId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	doctor.PersonId, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return lastId, tx.Commit()
}

func (d *DataService) GetDoctorFromId(doctorId int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		doctorId, PHONE_CELL)
}

func (d *DataService) Doctor(id int64, basicInfoOnly bool) (*common.Doctor, error) {
	if !basicInfoOnly {
		return d.GetDoctorFromId(id)
	}

	var doctor common.Doctor
	var dobMonth, dobDay, dobYear int
	var smallThumbnailID, largeThumbnailID sql.NullString
	var shortTitle, longTitle, shortDisplayName, longDisplayName sql.NullString
	var NPI, DEA sql.NullString
	var clinicianID sql.NullInt64
	err := d.db.QueryRow(`
		SELECT id, first_name, last_name, short_title, long_title, short_display_name, long_display_name, gender, 
				dob_year, dob_month, dob_day, status, clinician_id, small_thumbnail_id, large_thumbnail_id, npi_number, dea_number
		FROM doctor 
		WHERE id = ?`, id).Scan(
		&doctor.DoctorId,
		&doctor.FirstName,
		&doctor.LastName,
		&shortTitle,
		&longTitle,
		&shortDisplayName,
		&longDisplayName,
		&doctor.Gender,
		&dobYear, &dobMonth, &dobDay,
		&doctor.Status,
		&clinicianID,
		&smallThumbnailID,
		&largeThumbnailID,
		&NPI,
		&DEA)

	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	doctor.ShortTitle = shortTitle.String
	doctor.LongTitle = longTitle.String
	doctor.ShortDisplayName = shortDisplayName.String
	doctor.LongDisplayName = longDisplayName.String
	doctor.DOB = encoding.DOB{Year: dobYear, Month: dobMonth, Day: dobDay}
	doctor.SmallThumbnailID = smallThumbnailID.String
	doctor.DoseSpotClinicianId = clinicianID.Int64
	doctor.LargeThumbnailID = largeThumbnailID.String
	doctor.SmallThumbnailURL = app_url.SmallThumbnailURL(d.apiDomain, DOCTOR_ROLE, doctor.DoctorId.Int64())
	doctor.LargeThumbnailURL = app_url.LargeThumbnailURL(d.apiDomain, DOCTOR_ROLE, doctor.DoctorId.Int64())

	return &doctor, nil
}

func (d *DataService) GetDoctorFromAccountId(accountId int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.account_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		accountId, PHONE_CELL)
}

func (d *DataService) GetDoctorFromDoseSpotClinicianId(clinicianId int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.clinician_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		clinicianId, PHONE_CELL)
}

func (d *DataService) GetAccountIDFromDoctorID(doctorID int64) (int64, error) {
	var accountID int64
	err := d.db.QueryRow(`select account_id from doctor where id = ?`, doctorID).Scan(&accountID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return accountID, err
}

func (d *DataService) GetFirstDoctorWithAClinicianId() (*common.Doctor, error) {
	return d.queryDoctor(`doctor.clinician_id is not null AND (account_phone.phone IS NULL OR account_phone.phone_type = ?) LIMIT 1`, PHONE_CELL)
}

func (d *DataService) GetMAInClinic() (*common.Doctor, error) {
	return d.queryDoctor(`account.role_type_id = ? AND (account_phone.phone is NULL or account_phone.phone_type = ?)`, d.roleTypeMapping[MA_ROLE], PHONE_CELL)
}

func (d *DataService) queryDoctor(where string, queryParams ...interface{}) (*common.Doctor, error) {
	row := d.db.QueryRow(fmt.Sprintf(`
		SELECT doctor.id, doctor.account_id, phone, first_name, last_name, middle_name, suffix,
			prefix, short_title, long_title, short_display_name, long_display_name, account.email,
			gender, dob_year, dob_month, dob_day, doctor.status, clinician_id,
			address.address_line_1,	address.address_line_2, address.city, address.state,
			address.zip_code, person.id, npi_number, dea_number, account.role_type_id,
			doctor.small_thumbnail_id, doctor.large_thumbnail_id
		FROM doctor
		INNER JOIN account ON account.id = doctor.account_id
		INNER JOIN person ON person.role_type_id = account.role_type_id AND person.role_id = doctor.id
		LEFT OUTER JOIN account_phone ON account_phone.account_id = doctor.account_id
		LEFT OUTER JOIN doctor_address_selection ON doctor_address_selection.doctor_id = doctor.id
		LEFT OUTER JOIN address ON doctor_address_selection.address_id = address.id
		WHERE %s`, where),
		queryParams...)

	var firstName, lastName, status, gender, email string
	var addressLine1, addressLine2, city, state, zipCode sql.NullString
	var middleName, suffix, prefix, shortTitle, longTitle sql.NullString
	var smallThumbnailID, largeThumbnailID sql.NullString
	var cellPhoneNumber common.Phone
	var doctorId, accountId encoding.ObjectId
	var dobYear, dobMonth, dobDay int
	var personId, roleTypeId int64
	var clinicianId sql.NullInt64
	var NPI, DEA, shortDisplayName, longDisplayName sql.NullString

	err := row.Scan(
		&doctorId, &accountId, &cellPhoneNumber, &firstName, &lastName,
		&middleName, &suffix, &prefix, &shortTitle, &longTitle, &shortDisplayName,
		&longDisplayName, &email, &gender, &dobYear, &dobMonth,
		&dobDay, &status, &clinicianId, &addressLine1, &addressLine2,
		&city, &state, &zipCode, &personId, &NPI, &DEA, &roleTypeId,
		&smallThumbnailID, &largeThumbnailID)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	doctor := &common.Doctor{
		AccountId:           accountId,
		DoctorId:            doctorId,
		FirstName:           firstName,
		LastName:            lastName,
		MiddleName:          middleName.String,
		Suffix:              suffix.String,
		Prefix:              prefix.String,
		ShortTitle:          shortTitle.String,
		LongTitle:           longTitle.String,
		ShortDisplayName:    shortDisplayName.String,
		LongDisplayName:     longDisplayName.String,
		SmallThumbnailID:    smallThumbnailID.String,
		LargeThumbnailID:    largeThumbnailID.String,
		SmallThumbnailURL:   app_url.SmallThumbnailURL(d.apiDomain, DOCTOR_ROLE, doctorId.Int64()),
		LargeThumbnailURL:   app_url.LargeThumbnailURL(d.apiDomain, DOCTOR_ROLE, doctorId.Int64()),
		Status:              status,
		Gender:              gender,
		Email:               email,
		CellPhone:           cellPhoneNumber,
		DoseSpotClinicianId: clinicianId.Int64,
		DoctorAddress: &common.Address{
			AddressLine1: addressLine1.String,
			AddressLine2: addressLine2.String,
			City:         city.String,
			State:        state.String,
			ZipCode:      zipCode.String,
		},
		DOB:      encoding.DOB{Year: dobYear, Month: dobMonth, Day: dobDay},
		PersonId: personId,
		NPI:      NPI.String,
		DEA:      DEA.String,
		IsMA:     d.roleTypeMapping[MA_ROLE] == roleTypeId,
	}

	doctor.PromptStatus, err = d.GetPushPromptStatus(doctor.AccountId.Int64())
	if err != nil {
		return nil, err
	}

	return doctor, nil
}

func (d *DataService) GetDoctorIdFromAccountId(accountId int64) (int64, error) {
	var doctorId int64
	err := d.db.QueryRow("select id from doctor where account_id = ?", accountId).Scan(&doctorId)
	return doctorId, err
}

func (d *DataService) GetRegimenStepsForDoctor(doctorID int64) ([]*common.DoctorInstructionItem, error) {
	rows, err := d.db.Query(`
	SELECT id, text, status 
	FROM dr_regimen_step where doctor_id = ? AND status = ?`, doctorID, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*common.DoctorInstructionItem
	for rows.Next() {
		var step common.DoctorInstructionItem
		if err := rows.Scan(
			&step.ID,
			&step.Text,
			&step.Status); err != nil {
			return nil, err
		}
		steps = append(steps, &step)
	}

	return steps, rows.Err()
}

func (d *DataService) GetRegimenStepForDoctor(regimenStepID, doctorID int64) (*common.DoctorInstructionItem, error) {
	var regimenStep common.DoctorInstructionItem
	err := d.db.QueryRow(`
		SELECT id, text, status
		FROM dr_regimen_step
		WHERE id = ? AND doctor_id = ?`, regimenStepID, doctorID,
	).Scan(&regimenStep.ID, &regimenStep.Text, &regimenStep.Status)
	if err == sql.ErrNoRows {
		return &regimenStep, NoRowsError
	}

	return &regimenStep, err
}

func (d *DataService) AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	res, err := d.db.Exec(`insert into dr_regimen_step (text, doctor_id,status) values (?,?,?)`, regimenStep.Text, doctorId, STATUS_ACTIVE)
	if err != nil {
		return err
	}
	instructionId, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// assign an id given that its a new regimen step
	regimenStep.ID = encoding.NewObjectId(instructionId)
	return nil
}

func (d *DataService) UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// lookup the sourceId and status for the current regimen step if it exists
	var sourceId sql.NullInt64
	var status string
	if err := tx.QueryRow(`
		SELECT source_id, status
		FROM dr_regimen_step
		WHERE id = ? AND doctor_id = ?`,
		regimenStep.ID.Int64(), doctorID,
	).Scan(&sourceId, &status); err != nil {
		return err
	}

	// if the source id does not exist for the step, this means that
	// this step is the source itself. tracking the source id helps for
	// tracking revision from the beginning of time.
	sourceIdForUpdatedStep := regimenStep.ID.Int64()
	if sourceId.Valid {
		sourceIdForUpdatedStep = sourceId.Int64
	}

	// update the current regimen step to be inactive
	_, err = tx.Exec(`UPDATE dr_regimen_step SET status = ? WHERE id = ? AND doctor_id = ?`,
		STATUS_INACTIVE, regimenStep.ID.Int64(), doctorID)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert a new active regimen step in its place
	res, err := tx.Exec(`INSERT INTO dr_regimen_step (text, doctor_id, source_id, status) VALUES (?, ?, ?, ?)`,
		regimenStep.Text, doctorID, sourceIdForUpdatedStep, status)
	if err != nil {
		tx.Rollback()
		return err
	}

	instructionID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the regimenStep Id
	regimenStep.ID = encoding.NewObjectId(instructionID)
	return tx.Commit()
}

func (d *DataService) MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorID int64) error {
	// mark the regimen step to be deleted
	_, err := d.db.Exec(`UPDATE dr_regimen_step SET status = ? WHERE id = ? AND doctor_id = ?`,
		STATUS_DELETED, regimenStep.ID.Int64(), doctorID)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, regimenStep := range regimenSteps {
		_, err = tx.Exec(`UPDATE dr_regimen_step SET status = ? WHERE id = ? AND doctor_id=?`,
			STATUS_DELETED, regimenStep.ID.Int64(), doctorID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (d *DataService) InsertItemIntoDoctorQueue(doctorQueueItem DoctorQueueItem) error {
	return insertItemIntoDoctorQueue(d.db, &doctorQueueItem)
}

func insertItemIntoDoctorQueue(d db, doctorQueueItem *DoctorQueueItem) error {
	// only insert if the item doesn't already exist
	var id int64
	err := d.QueryRow(`select id from doctor_queue where doctor_id = ? and item_id = ? and event_type = ? and status = ? LIMIT 1`,
		doctorQueueItem.DoctorId, doctorQueueItem.ItemId, doctorQueueItem.EventType, doctorQueueItem.Status).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == nil {
		// nothing to do if the item already exists in the queuereturn nil
		return nil
	}

	_, err = d.Exec(`insert into doctor_queue (doctor_id, item_id, event_type, status) values (?,?,?,?)`, doctorQueueItem.DoctorId, doctorQueueItem.ItemId, doctorQueueItem.EventType, doctorQueueItem.Status)
	return err
}

func (d *DataService) ReplaceItemInDoctorQueue(doctorQueueItem DoctorQueueItem, currentState string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from doctor_queue where status = ? and doctor_id = ? and event_type = ? and item_id = ?`,
		currentState, doctorQueueItem.DoctorId, doctorQueueItem.EventType, doctorQueueItem.ItemId)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, ?, ?, ?)`,
		doctorQueueItem.DoctorId, doctorQueueItem.Status, doctorQueueItem.EventType, doctorQueueItem.ItemId)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error {
	_, err := d.db.Exec(`delete from doctor_queue where doctor_id = ? and item_id = ? and event_type = ? and status = ?`, doctorQueueItem.DoctorId, doctorQueueItem.ItemId, doctorQueueItem.EventType, doctorQueueItem.Status)
	return err
}

func (d *DataService) MarkPatientVisitAsOngoingInDoctorQueue(doctorId, patientVisitId int64) error {
	_, err := d.db.Exec(`update doctor_queue set status=? where event_type=? and item_id=? and doctor_id=?`, STATUS_ONGOING, DQEventTypePatientVisit, patientVisitId, doctorId)
	return err
}

// CompleteVisitOnTreatmentPlanGeneration updates the doctor queue upon the generation of a treatment plan to create a completed item as well as
// clear out any submitted visit by the patient pertaining to the case.
func (d *DataService) CompleteVisitOnTreatmentPlanGeneration(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// get list of possible patient visits that could be in the doctor's queue in this case
	openStates := common.OpenPatientVisitStates()
	vals := []interface{}{treatmentPlanId}
	vals = appendStringsToInterfaceSlice(vals, openStates)
	rows, err := tx.Query(`
		SELECT patient_visit.id
		FROM patient_visit
		INNER JOIN treatment_plan on treatment_plan.patient_case_id = patient_visit.patient_case_id
		WHERE treatment_plan.id = ?
		AND patient_visit.status not in (`+nReplacements(len(openStates))+`)`, vals...)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer rows.Close()

	var visitIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			tx.Rollback()
			return err
		}

		visitIDs = append(visitIDs, id)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}

	if len(visitIDs) > 0 {
		vals := []interface{}{currentState, doctorId, DQEventTypePatientVisit}
		vals = appendInt64sToInterfaceSlice(vals, visitIDs)

		_, err = tx.Exec(`
		DELETE FROM doctor_queue
		WHERE status = ? AND doctor_id = ? AND event_type = ?
		AND item_id in (`+nReplacements(len(visitIDs))+`)`, vals...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, ?, ?, ?)`, doctorId, updatedState, DQEventTypeTreatmentPlan, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) GetPendingItemsInDoctorQueue(doctorId int64) ([]*DoctorQueueItem, error) {
	params := []interface{}{doctorId}
	params = appendStringsToInterfaceSlice(params, []string{STATUS_PENDING, STATUS_ONGOING})
	rows, err := d.db.Query(fmt.Sprintf(`select id, event_type, item_id, enqueue_date, completed_date, status, doctor_id from doctor_queue where doctor_id = ? and status in (%s) order by enqueue_date`, nReplacements(2)), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetCompletedItemsInDoctorQueue(doctorId int64) ([]*DoctorQueueItem, error) {
	params := []interface{}{doctorId}
	params = appendStringsToInterfaceSlice(params, []string{STATUS_PENDING, STATUS_ONGOING})
	rows, err := d.db.Query(fmt.Sprintf(`select id, event_type, item_id, enqueue_date, completed_date, status, doctor_id from doctor_queue where doctor_id = ? and status not in (%s) order by enqueue_date desc`, nReplacements(2)), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetPendingItemsForClinic() ([]*DoctorQueueItem, error) {
	// get all the items in in the unassigned queue
	unclaimedQueueItems, err := d.GetAllItemsInUnclaimedQueue()
	if err != nil {
		return nil, err
	}

	// now get all pending items in the doctor queue
	rows, err := d.db.Query(`SELECT id, event_type, item_id, enqueue_date, completed_date, status, doctor_id FROM doctor_queue WHERE status IN (`+nReplacements(2)+`) ORDER BY enqueue_date`, STATUS_PENDING, STATUS_ONGOING)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queueItems, err := populateDoctorQueueFromRows(rows)
	if err != nil {
		return nil, err
	}

	queueItems = append(queueItems, unclaimedQueueItems...)

	// sort the items in ascending order so as to return a wholistic view of the items that are pending
	sort.Sort(ByTimestamp(queueItems))

	return queueItems, nil
}

func (d *DataService) GetCompletedItemsForClinic() ([]*DoctorQueueItem, error) {
	rows, err := d.db.Query(`SELECT id, event_type, item_id, enqueue_date, completed_date, status, doctor_id FROM doctor_queue WHERE status NOT IN (`+nReplacements(2)+`) ORDER BY enqueue_date desc`, STATUS_ONGOING, STATUS_PENDING)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetPendingItemCountForDoctorQueue(doctorId int64) (int64, error) {
	var count int64
	err := d.db.QueryRow(fmt.Sprintf(`select count(*) from doctor_queue where doctor_id = ? and status in (%s)`, nReplacements(2)), doctorId, STATUS_PENDING, STATUS_ONGOING).Scan(&count)
	return count, err
}

func populateDoctorQueueFromRows(rows *sql.Rows) ([]*DoctorQueueItem, error) {
	doctorQueue := make([]*DoctorQueueItem, 0)
	for rows.Next() {
		var queueItem DoctorQueueItem
		var completedDate mysql.NullTime
		err := rows.Scan(&queueItem.Id, &queueItem.EventType, &queueItem.ItemId, &queueItem.EnqueueDate, &completedDate, &queueItem.Status, &queueItem.DoctorId)
		if err != nil {
			return nil, err
		}
		queueItem.CompletedDate = completedDate.Time
		doctorQueue = append(doctorQueue, &queueItem)
	}
	return doctorQueue, rows.Err()
}

func (d *DataService) GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error) {
	rows, err := d.db.Query(`select dispense_unit.id, ltext from dispense_unit inner join localized_text on app_text_id = dispense_unit_text_id where language_id=?`, languageId)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	dispenseUnitIds = make([]int64, 0)
	dispenseUnits = make([]string, 0)
	for rows.Next() {
		var dipenseUnitId int64
		var dispenseUnit string
		if err := rows.Scan(&dipenseUnitId, &dispenseUnit); err != nil {
			return nil, nil, err
		}
		dispenseUnits = append(dispenseUnits, dispenseUnit)
		dispenseUnitIds = append(dispenseUnitIds, dipenseUnitId)
	}
	return dispenseUnitIds, dispenseUnits, rows.Err()
}

func (d *DataService) AddTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId, treatmentPlanId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {

		var treatmentIdInPatientTreatmentPlan int64
		if treatmentPlanId != 0 {
			treatmentIdInPatientTreatmentPlan = doctorTreatmentTemplate.Treatment.Id.Int64()
		}

		treatmentType := treatmentRX
		if doctorTreatmentTemplate.Treatment.OTC {
			treatmentType = treatmentOTC
		}

		columnsAndData := map[string]interface{}{
			"drug_internal_name":    doctorTreatmentTemplate.Treatment.DrugInternalName,
			"dosage_strength":       doctorTreatmentTemplate.Treatment.DosageStrength,
			"type":                  treatmentType,
			"dispense_value":        doctorTreatmentTemplate.Treatment.DispenseValue,
			"dispense_unit_id":      doctorTreatmentTemplate.Treatment.DispenseUnitId.Int64(),
			"refills":               doctorTreatmentTemplate.Treatment.NumberRefills.Int64Value,
			"substitutions_allowed": doctorTreatmentTemplate.Treatment.SubstitutionsAllowed,
			"patient_instructions":  doctorTreatmentTemplate.Treatment.PatientInstructions,
			"pharmacy_notes":        doctorTreatmentTemplate.Treatment.PharmacyNotes,
			"status":                common.TStatusCreated.String(),
			"doctor_id":             doctorId,
			"name":                  doctorTreatmentTemplate.Name,
		}

		if doctorTreatmentTemplate.Treatment.DaysSupply.IsValid {
			columnsAndData["days_supply"] = doctorTreatmentTemplate.Treatment.DaysSupply.Int64Value
		}

		if err := d.includeDrugNameComponentIfNonZero(doctorTreatmentTemplate.Treatment.DrugName, drugNameTable, "drug_name_id", columnsAndData, tx); err != nil {
			tx.Rollback()
			return err
		}

		if err := d.includeDrugNameComponentIfNonZero(doctorTreatmentTemplate.Treatment.DrugForm, drugFormTable, "drug_form_id", columnsAndData, tx); err != nil {
			tx.Rollback()
			return err
		}

		if err := d.includeDrugNameComponentIfNonZero(doctorTreatmentTemplate.Treatment.DrugRoute, drugRouteTable, "drug_route_id", columnsAndData, tx); err != nil {
			tx.Rollback()
			return err
		}

		columns, values := getKeysAndValuesFromMap(columnsAndData)
		res, err := tx.Exec(fmt.Sprintf(`insert into dr_treatment_template (%s) values (%s)`, strings.Join(columns, ","), nReplacements(len(values))), values...)
		if err != nil {
			tx.Rollback()
			return err
		}

		drTreatmentTemplateId, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}

		// update the treatment object with the information
		doctorTreatmentTemplate.Id = encoding.NewObjectId(drTreatmentTemplateId)

		// add drug db ids to the table
		for drugDbTag, drugDbId := range doctorTreatmentTemplate.Treatment.DrugDBIds {
			_, err := tx.Exec(`insert into dr_treatment_template_drug_db_id (drug_db_id_tag, drug_db_id, dr_treatment_template_id) values (?, ?, ?)`, drugDbTag, drugDbId, drTreatmentTemplateId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		// mark the fact that the treatment was added as a favorite from a patient's treatment
		// and so the selection needs to be maintained
		if treatmentIdInPatientTreatmentPlan != 0 {

			// delete any pre-existing favorite treatment that is already linked against this treatment in the patient visit,
			// because that means that the client has an out-of-sync list for some reason, and we should treat
			// what the client has as the source of truth. Otherwise, we will have two favorite treatments that are craeted
			// both of which are mapped against the exist same treatment_id
			// this should rarely happen; but what this will do is help ensure that a treatment within a patient visit can only be favorited
			// once and only once.
			var preExistingDoctorFavoriteTreatmentId int64
			err = tx.QueryRow(`select dr_treatment_template_id from treatment_dr_template_selection where treatment_id = ? `, treatmentIdInPatientTreatmentPlan).Scan(&preExistingDoctorFavoriteTreatmentId)
			if err != nil && err != sql.ErrNoRows {
				tx.Rollback()
				return err
			}

			if preExistingDoctorFavoriteTreatmentId != 0 {
				// go ahead and delete the selection
				_, err = tx.Exec(`delete from treatment_dr_template_selection where treatment_id = ?`, treatmentIdInPatientTreatmentPlan)
				if err != nil {
					tx.Rollback()
					return err
				}

				// also, go ahead and mark this particular favorited treatment as deleted
				_, err = tx.Exec(`update dr_treatment_template set status = ? where id = ?`, common.TStatusDeleted.String(), preExistingDoctorFavoriteTreatmentId)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatmentIdInPatientTreatmentPlan, drTreatmentTemplateId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	}

	return tx.Commit()
}

func (d *DataService) DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {
		_, err = tx.Exec(`update dr_treatment_template set status=? where id = ? and doctor_id = ?`, common.TStatusDeleted.String(), doctorTreatmentTemplate.Id.Int64(), doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}

		// delete all previous selections for this favorited treatment
		_, err = tx.Exec(`delete from treatment_dr_template_selection where dr_treatment_template_id = ?`, doctorTreatmentTemplate.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentTemplates(doctorId int64) ([]*common.DoctorTreatmentTemplate, error) {
	rows, err := d.db.Query(`select dr_treatment_template.id, dr_treatment_template.name, drug_internal_name, dosage_strength, type, 
				dispense_value, dispense_unit_id, ltext, refills, substitutions_allowed,
				days_supply, pharmacy_notes, patient_instructions, creation_date, status,
				 drug_name.name, drug_route.name, drug_form.name
			 		from dr_treatment_template 
						inner join dispense_unit on dr_treatment_template.dispense_unit_id = dispense_unit.id
						inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
						left outer join drug_name on drug_name_id = drug_name.id
						left outer join drug_route on drug_route_id = drug_route.id
						left outer join drug_form on drug_form_id = drug_form.id
			 					where status=? and doctor_id = ? and localized_text.language_id=?`, common.TStatusCreated.String(), doctorId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatmentTemplates := make([]*common.DoctorTreatmentTemplate, 0)
	for rows.Next() {
		var drTreatmentTemplateId, dispenseUnitId encoding.ObjectId
		var name string
		var daysSupply, refills encoding.NullInt64
		var dispenseValue encoding.HighPrecisionFloat64
		var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription string
		var substitutionsAllowed bool
		var status common.TreatmentStatus
		var creationDate time.Time
		var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
		err = rows.Scan(&drTreatmentTemplateId, &name, &drugInternalName, &dosageStrength, &treatmentType,
			&dispenseValue, &dispenseUnitId, &dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyNotes,
			&patientInstructions, &creationDate, &status, &drugName, &drugRoute, &drugForm)
		if err != nil {
			return nil, err
		}

		drTreatmenTemplate := &common.DoctorTreatmentTemplate{
			Id:   drTreatmentTemplateId,
			Name: name,
			Treatment: &common.Treatment{
				DrugInternalName:        drugInternalName,
				DosageStrength:          dosageStrength,
				DispenseValue:           dispenseValue,
				DispenseUnitId:          dispenseUnitId,
				DispenseUnitDescription: dispenseUnitDescription,
				NumberRefills:           refills,
				SubstitutionsAllowed:    substitutionsAllowed,
				DaysSupply:              daysSupply,
				DrugName:                drugName.String,
				DrugForm:                drugForm.String,
				DrugRoute:               drugRoute.String,
				PatientInstructions:     patientInstructions,
				CreationDate:            &creationDate,
				Status:                  status,
				PharmacyNotes:           pharmacyNotes.String,
			},
		}

		if treatmentType == treatmentOTC {
			drTreatmenTemplate.Treatment.OTC = true
		}

		err = d.fillInDrugDBIdsForTreatment(drTreatmenTemplate.Treatment, drTreatmenTemplate.Id.Int64(), "dr_treatment_template")
		if err != nil {
			return nil, err
		}

		treatmentTemplates = append(treatmentTemplates, drTreatmenTemplate)
	}
	return treatmentTemplates, rows.Err()
}

func (d *DataService) SetTreatmentPlanNote(doctorID, treatmentPlanID int64, note string) error {
	// Use NULL for empty note
	msg := sql.NullString{
		String: note,
		Valid:  note != "",
	}
	_, err := d.db.Exec(`UPDATE treatment_plan SET note = ? WHERE id = ? AND doctor_id = ?`,
		msg, treatmentPlanID, doctorID)
	return err
}

func (d *DataService) GetTreatmentPlanNote(treatmentPlanID int64) (string, error) {
	var note sql.NullString
	row := d.db.QueryRow(`SELECT note FROM treatment_plan WHERE id = ?`, treatmentPlanID)
	err := row.Scan(&note)
	if err == sql.ErrNoRows {
		err = NoRowsError
	}
	return note.String, err
}

func (d *DataService) getIdForNameFromTable(tableName, drugComponentName string) (nullId sql.NullInt64, err error) {
	err = d.db.QueryRow(fmt.Sprintf(`select id from %s where name=?`, tableName), drugComponentName).Scan(&nullId)
	return
}

func (d *DataService) getOrInsertNameInTable(tx *sql.Tx, tableName, drugComponentName string) (int64, error) {
	drugComponentNameNullId, err := d.getIdForNameFromTable(tableName, drugComponentName)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	if !drugComponentNameNullId.Valid {
		res, err := tx.Exec(fmt.Sprintf(`insert into %s (name) values (?)`, tableName), drugComponentName)
		if err != nil {
			return 0, err
		}

		return res.LastInsertId()
	}
	return drugComponentNameNullId.Int64, nil
}

type DoctorUpdate struct {
	ShortTitle          *string
	LongTitle           *string
	ShortDisplayName    *string
	LongDisplayName     *string
	NPI                 *string
	DEA                 *string
	SmallThumbnailID    *string
	LargeThumbnailID    *string
	DosespotClinicianID *int64
}

func (d *DataService) UpdateDoctor(doctorID int64, update *DoctorUpdate) error {
	var cols []string
	var vals []interface{}

	if update.ShortTitle != nil {
		cols = append(cols, "short_title = ?")
		vals = append(vals, *update.ShortTitle)
	}
	if update.LongTitle != nil {
		cols = append(cols, "long_title = ?")
		vals = append(vals, *update.LongTitle)
	}
	if update.ShortDisplayName != nil {
		cols = append(cols, "short_display_name = ?")
		vals = append(vals, *update.ShortDisplayName)
	}
	if update.LongDisplayName != nil {
		cols = append(cols, "long_display_name = ?")
		vals = append(vals, *update.LongDisplayName)
	}
	if update.NPI != nil {
		cols = append(cols, "npi_number = ?")
		vals = append(vals, *update.NPI)
	}
	if update.DEA != nil {
		cols = append(cols, "dea_number = ?")
		vals = append(vals, *update.DEA)
	}
	if update.SmallThumbnailID != nil {
		cols = append(cols, "small_thumbnail_id = ?")
		vals = append(vals, *update.SmallThumbnailID)
	}
	if update.LargeThumbnailID != nil {
		cols = append(cols, "large_thumbnail_id = ?")
		vals = append(vals, *update.LargeThumbnailID)
	}

	if update.DosespotClinicianID != nil {
		cols = append(cols, "clinician_id = ?")
		vals = append(vals, *update.DosespotClinicianID)
	}

	if len(cols) == 0 {
		return nil
	}
	vals = append(vals, doctorID)

	colStr := strings.Join(cols, ", ")
	_, err := d.db.Exec(`UPDATE doctor SET `+colStr+` WHERE id = ?`, vals...)
	return err
}

func (d *DataService) DoctorAttributes(doctorID int64, names []string) (map[string]string, error) {
	var rows *sql.Rows
	var err error
	if len(names) == 0 {
		rows, err = d.db.Query(`SELECT name, value FROM doctor_attribute WHERE doctor_id = ?`, doctorID)
	} else {
		rows, err = d.db.Query(`SELECT name, value FROM doctor_attribute WHERE doctor_id = ? AND name IN (`+nReplacements(len(names))+`)`,
			appendStringsToInterfaceSlice([]interface{}{doctorID}, names)...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	attr := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		attr[name] = value
	}
	return attr, rows.Err()
}

func (d *DataService) UpdateDoctorAttributes(doctorID int64, attributes map[string]string) error {
	if len(attributes) == 0 {
		return nil
	}
	var toDelete []interface{}
	var replacements []string
	var values []interface{}
	for name, value := range attributes {
		if value == "" {
			toDelete = append(toDelete, name)
		} else {
			replacements = append(replacements, "(?,?,?)")
			values = append(values, doctorID, name, value)
		}
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	if len(toDelete) != 0 {
		_, err := tx.Exec(`DELETE FROM doctor_attribute WHERE name IN (`+nReplacements(len(toDelete))+`) AND doctor_id = ?`,
			append(toDelete, doctorID)...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	if len(replacements) != 0 {
		_, err := tx.Exec(`REPLACE INTO doctor_attribute (doctor_id, name, value) VALUES `+strings.Join(replacements, ","), values...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (d *DataService) AddMedicalLicenses(licenses []*common.MedicalLicense) error {
	if len(licenses) == 0 {
		return nil
	}
	replacements := make([]string, len(licenses))
	values := make([]interface{}, 0, 4*len(licenses))
	for i, l := range licenses {
		if l.State == "" || l.Number == "" || l.Status == "" {
			return errors.New("api: license is missing state, number, or status")
		}
		replacements[i] = "(?,?,?,?,?)"
		values = append(values, l.DoctorID, l.State, l.Number, l.Status.String(), l.Expiration)
	}
	_, err := d.db.Exec(`REPLACE INTO doctor_medical_license (doctor_id, state, license_number, status, expiration_date) VALUES `+strings.Join(replacements, ","),
		values...)
	return err
}

func (d *DataService) MedicalLicenses(doctorID int64) ([]*common.MedicalLicense, error) {
	rows, err := d.db.Query(`
		SELECT id, state, license_number, status, expiration_date
		FROM doctor_medical_license
		WHERE doctor_id = ?
		ORDER BY state`, doctorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []*common.MedicalLicense
	for rows.Next() {
		l := &common.MedicalLicense{DoctorID: doctorID}
		if err := rows.Scan(&l.ID, &l.State, &l.Number, &l.Status, &l.Expiration); err != nil {
			return nil, err
		}
		licenses = append(licenses, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return licenses, nil
}

func (d *DataService) CareProviderProfile(accountID int64) (*common.CareProviderProfile, error) {
	row := d.db.QueryRow(`
		SELECT full_name, why_spruce, qualifications, undergraduate_school, graduate_school,
			medical_school, residency, fellowship, experience, creation_date, modified_date
		FROM care_provider_profile
		WHERE account_id = ?`, accountID)

	profile := common.CareProviderProfile{
		AccountID: accountID,
	}
	// If there's no profile then return an empty struct. There's no need for the
	// caller to care if the profile is empty or doesn't exist.
	if err := row.Scan(
		&profile.FullName, &profile.WhySpruce, &profile.Qualifications, &profile.UndergraduateSchool,
		&profile.GraduateSchool, &profile.MedicalSchool, &profile.Residency, &profile.Fellowship,
		&profile.Experience, &profile.Created, &profile.Modified,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &profile, nil
}

func (d *DataService) UpdateCareProviderProfile(accountID int64, profile *common.CareProviderProfile) error {
	_, err := d.db.Exec(`
		REPLACE INTO care_provider_profile (
			account_id, full_name, why_spruce, qualifications, undergraduate_school,
			graduate_school, medical_school, residency, fellowship, experience
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, profile.FullName, profile.WhySpruce, profile.Qualifications,
		profile.UndergraduateSchool, profile.GraduateSchool, profile.MedicalSchool,
		profile.Residency, profile.Fellowship, profile.Experience)
	return err
}

func (d *DataService) GetOldestTreatmentPlanInStatuses(max int, statuses []common.TreatmentPlanStatus) ([]*TreatmentPlanAge, error) {
	var whereClause string
	var params []interface{}

	if len(statuses) > 0 {
		whereClause = `WHERE status in (` + nReplacements(len(statuses)) + `)`
		for _, tpStatus := range statuses {
			params = append(params, tpStatus.String())
		}
	}
	params = append(params, max)

	rows, err := d.db.Query(`
		SELECT id, last_modified_date
		FROM treatment_plan
		`+whereClause+`
		ORDER BY last_modified_date LIMIT ?`, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tpAges []*TreatmentPlanAge
	for rows.Next() {
		var tpAge TreatmentPlanAge
		var lastModifiedDate time.Time
		if err := rows.Scan(
			&tpAge.ID,
			&lastModifiedDate); err != nil {
			return nil, err
		}
		tpAge.Age = time.Since(lastModifiedDate)
		tpAges = append(tpAges, &tpAge)
	}

	return tpAges, rows.Err()
}

func (d *DataService) DoctorEligibleToTreatInState(state string, doctorID, healthConditionID int64) (bool, error) {
	var id int64
	err := d.db.QueryRow(`
		SELECT care_provider_state_elligibility.id
				FROM care_provider_state_elligibility 
				INNER JOIN care_providing_state on care_providing_state.id = care_providing_state_id
				WHERE health_condition_id = ? AND care_providing_state.state = ? AND provider_id = ?
				AND role_type_id = ?`, healthConditionID, state, doctorID, d.roleTypeMapping[DOCTOR_ROLE]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return (err == nil), err
}
