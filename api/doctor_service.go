package api

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/golog"
)

func (d *DataService) RegisterDoctor(doctor *common.Doctor) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		insert into doctor (account_id, first_name, last_name, short_title, long_title, short_display_name, long_display_name, suffix, prefix, middle_name, gender, dob_year, dob_month, dob_day, status, clinician_id)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doctor.AccountID.Int64(), doctor.FirstName, doctor.LastName, doctor.ShortTitle, doctor.LongTitle, doctor.ShortDisplayName, doctor.LongDisplayName,
		doctor.MiddleName, doctor.Suffix, doctor.Prefix, doctor.Gender, doctor.DOB.Year, doctor.DOB.Month, doctor.DOB.Day,
		DOCTOR_REGISTERED, doctor.DoseSpotClinicianID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return 0, err
	}

	doctor.DoctorID = encoding.NewObjectID(lastID)
	doctor.DoctorAddress.ID, err = addAddress(tx, doctor.DoctorAddress)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = tx.Exec(`insert into doctor_address_selection (doctor_id, address_id) values (?,?)`, lastID, doctor.DoctorAddress.ID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if doctor.CellPhone != "" {
		_, err = tx.Exec(`INSERT INTO account_phone (phone, phone_type, account_id, status) VALUES (?,?,?,?) `,
			doctor.CellPhone.String(), PHONE_CELL, doctor.AccountID.Int64(), STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	res, err = tx.Exec(`INSERT INTO person (role_type_id, role_id) VALUES (?, ?)`, d.roleTypeMapping[DOCTOR_ROLE], lastID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	doctor.PersonID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return lastID, tx.Commit()
}

func (d *DataService) GetDoctorFromID(doctorID int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		doctorID, PHONE_CELL)
}

func (d *DataService) Doctor(id int64, basicInfoOnly bool) (*common.Doctor, error) {
	if !basicInfoOnly {
		return d.GetDoctorFromID(id)
	}

	return scanDoctor(d.db.QueryRow(`
		SELECT id, first_name, last_name, short_title, long_title, short_display_name, long_display_name, gender,
			dob_year, dob_month, dob_day, status, clinician_id, small_thumbnail_id, large_thumbnail_id, hero_image_id, npi_number, dea_number
		FROM doctor
		WHERE id = ?`, id))
}

func (d *DataService) Doctors(ids []int64) ([]*common.Doctor, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := d.db.Query(`
		SELECT id, first_name, last_name, short_title, long_title, short_display_name, long_display_name, gender,
			dob_year, dob_month, dob_day, status, clinician_id, small_thumbnail_id, large_thumbnail_id, hero_image_id, npi_number, dea_number
		FROM doctor
		WHERE id in (`+dbutil.MySQLArgs(len(ids))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, ids)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	doctorMap := make(map[int64]*common.Doctor)
	for rows.Next() {
		doctor, err := scanDoctor(rows)
		if err != nil {
			return nil, err
		}
		doctorMap[doctor.DoctorID.Int64()] = doctor
	}

	doctors := make([]*common.Doctor, len(ids))
	for i, doctorID := range ids {
		doctors[i] = doctorMap[doctorID]
	}

	return doctors, rows.Err()
}

func scanDoctor(s scannable) (*common.Doctor, error) {
	var doctor common.Doctor
	var smallThumbnailID, largeThumbnailID, heroImageID sql.NullString
	var shortTitle, longTitle, shortDisplayName, longDisplayName sql.NullString
	var NPI, DEA sql.NullString
	var clinicianID sql.NullInt64
	err := s.Scan(
		&doctor.DoctorID,
		&doctor.FirstName,
		&doctor.LastName,
		&shortTitle,
		&longTitle,
		&shortDisplayName,
		&longDisplayName,
		&doctor.Gender,
		&doctor.DOB.Year, &doctor.DOB.Month, &doctor.DOB.Day,
		&doctor.Status,
		&clinicianID,
		&smallThumbnailID,
		&largeThumbnailID,
		&heroImageID,
		&NPI,
		&DEA)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound("doctor")
	} else if err != nil {
		return nil, err
	}

	doctor.ShortTitle = shortTitle.String
	doctor.LongTitle = longTitle.String
	doctor.ShortDisplayName = shortDisplayName.String
	doctor.LongDisplayName = longDisplayName.String
	doctor.SmallThumbnailID = smallThumbnailID.String
	doctor.DoseSpotClinicianID = clinicianID.Int64
	doctor.LargeThumbnailID = largeThumbnailID.String
	doctor.HeroImageID = heroImageID.String

	return &doctor, nil
}

func (d *DataService) GetDoctorFromAccountID(accountID int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.account_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		accountID, PHONE_CELL)
}

func (d *DataService) GetDoctorFromDoseSpotClinicianID(clinicianID int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.clinician_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		clinicianID, PHONE_CELL)
}

func (d *DataService) GetAccountIDFromDoctorID(doctorID int64) (int64, error) {
	var accountID int64
	err := d.db.QueryRow(`select account_id from doctor where id = ?`, doctorID).Scan(&accountID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return accountID, err
}

func (d *DataService) GetFirstDoctorWithAClinicianID() (*common.Doctor, error) {
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
			doctor.small_thumbnail_id, doctor.large_thumbnail_id, doctor.hero_image_id
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
	var smallThumbnailID, largeThumbnailID, heroImageID sql.NullString
	var cellPhoneNumber common.Phone
	var doctorID, accountID encoding.ObjectID
	var dobYear, dobMonth, dobDay int
	var personID, roleTypeId int64
	var clinicianID sql.NullInt64
	var NPI, DEA, shortDisplayName, longDisplayName sql.NullString

	err := row.Scan(
		&doctorID, &accountID, &cellPhoneNumber, &firstName, &lastName,
		&middleName, &suffix, &prefix, &shortTitle, &longTitle, &shortDisplayName,
		&longDisplayName, &email, &gender, &dobYear, &dobMonth,
		&dobDay, &status, &clinicianID, &addressLine1, &addressLine2,
		&city, &state, &zipCode, &personID, &NPI, &DEA, &roleTypeId,
		&smallThumbnailID, &largeThumbnailID, &heroImageID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound("doctor")
	} else if err != nil {
		return nil, err
	}

	doctor := &common.Doctor{
		AccountID:           accountID,
		DoctorID:            doctorID,
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
		HeroImageID:         heroImageID.String,
		Status:              status,
		Gender:              gender,
		Email:               email,
		CellPhone:           cellPhoneNumber,
		DoseSpotClinicianID: clinicianID.Int64,
		DoctorAddress: &common.Address{
			AddressLine1: addressLine1.String,
			AddressLine2: addressLine2.String,
			City:         city.String,
			State:        state.String,
			ZipCode:      zipCode.String,
		},
		DOB:      encoding.Date{Year: dobYear, Month: dobMonth, Day: dobDay},
		PersonID: personID,
		NPI:      NPI.String,
		DEA:      DEA.String,
		IsMA:     d.roleTypeMapping[MA_ROLE] == roleTypeId,
	}

	doctor.PromptStatus, err = d.GetPushPromptStatus(doctor.AccountID.Int64())
	if err != nil {
		return nil, err
	}

	return doctor, nil
}

func (d *DataService) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	var doctorID int64
	err := d.db.QueryRow("select id from doctor where account_id = ?", accountID).Scan(&doctorID)
	return doctorID, err
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
		return nil, ErrNotFound("dr_regimen_step")
	}

	return &regimenStep, err
}

func (d *DataService) AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorID int64) error {
	res, err := d.db.Exec(`insert into dr_regimen_step (text, doctor_id,status) values (?,?,?)`, regimenStep.Text, doctorID, STATUS_ACTIVE)
	if err != nil {
		return err
	}
	instructionId, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// assign an id given that its a new regimen step
	regimenStep.ID = encoding.NewObjectID(instructionId)
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
	regimenStep.ID = encoding.NewObjectID(instructionID)
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

func (d *DataService) InsertItemIntoDoctorQueue(dqi DoctorQueueItem) error {
	return insertItemIntoDoctorQueue(d.db, &dqi)
}

func insertItemIntoDoctorQueue(d db, dqi *DoctorQueueItem) error {
	if err := dqi.Validate(); err != nil {
		return err
	}

	// only insert if the item doesn't already exist
	var id int64
	err := d.QueryRow(`
		SELECT id
		FROM doctor_queue
		WHERE doctor_id = ?
			AND item_id = ?
			AND event_type = ?
			AND status = ?
		LIMIT 1`,
		dqi.DoctorID, dqi.ItemID, dqi.EventType, dqi.Status).Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return err
	} else if err == nil {
		// nothing to do if the item already exists in the queuereturn nil
		return nil
	}

	_, err = d.Exec(`
		INSERT INTO doctor_queue (
			doctor_id, patient_id, item_id, event_type, status,
			description, short_description, action_url, tags)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		dqi.DoctorID,
		dqi.PatientID,
		dqi.ItemID,
		dqi.EventType,
		dqi.Status,
		dqi.Description,
		dqi.ShortDescription,
		dqi.ActionURL.String(),
		strings.Join(dqi.Tags, tagSeparator))
	return err
}

func (d *DataService) ReplaceItemInDoctorQueue(dqi DoctorQueueItem, currentState string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		DELETE FROM doctor_queue 
		WHERE status = ? AND doctor_id = ? AND event_type = ? AND item_id = ?`,
		currentState, dqi.DoctorID, dqi.EventType, dqi.ItemID)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := insertItemIntoDoctorQueue(tx, &dqi); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error {
	_, err := d.db.Exec(`
	DELETE FROM doctor_queue
	WHERE doctor_id = ? AND item_id = ? AND event_type = ? AND status = ?`,
		doctorQueueItem.DoctorID,
		doctorQueueItem.ItemID,
		doctorQueueItem.EventType,
		doctorQueueItem.Status)
	return err
}

func (d *DataService) MarkPatientVisitAsOngoingInDoctorQueue(doctorID, patientVisitID int64) error {
	_, err := d.db.Exec(`
		UPDATE doctor_queue SET status=? WHERE event_type=? AND item_id=? AND doctor_id=?`,
		STATUS_ONGOING,
		DQEventTypePatientVisit,
		patientVisitID,
		doctorID)
	return err
}

// CompleteVisitOnTreatmentPlanGeneration updates the doctor queue upon the generation of a treatment plan to create a completed item as well as
// clear out any submitted visit by the patient pertaining to the case.
func (d *DataService) CompleteVisitOnTreatmentPlanGeneration(
	doctorID, patientVisitID, treatmentPlanID int64,
	currentState string,
	queueItem *DoctorQueueItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// get list of possible patient visits that could be in the doctor's queue in this case
	openStates := common.OpenPatientVisitStates()
	vals := []interface{}{treatmentPlanID}
	vals = dbutil.AppendStringsToInterfaceSlice(vals, openStates)
	rows, err := tx.Query(`
		SELECT patient_visit.id
		FROM patient_visit
		INNER JOIN treatment_plan on treatment_plan.patient_case_id = patient_visit.patient_case_id
		WHERE treatment_plan.id = ?
		AND patient_visit.status not in (`+dbutil.MySQLArgs(len(openStates))+`)`, vals...)
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
		vals := []interface{}{currentState, doctorID, DQEventTypePatientVisit}
		vals = dbutil.AppendInt64sToInterfaceSlice(vals, visitIDs)

		_, err = tx.Exec(`
		DELETE FROM doctor_queue
		WHERE status = ? AND doctor_id = ? AND event_type = ?
		AND item_id in (`+dbutil.MySQLArgs(len(visitIDs))+`)`, vals...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := insertItemIntoDoctorQueue(tx, queueItem); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPendingItemsInDoctorQueue(doctorID int64) ([]*DoctorQueueItem, error) {
	params := []interface{}{doctorID}
	params = dbutil.AppendStringsToInterfaceSlice(params, []string{STATUS_PENDING, STATUS_ONGOING})
	rows, err := d.db.Query(fmt.Sprintf(`
		SELECT id, event_type, item_id, enqueue_date, status, doctor_id, patient_id, description, short_description, action_url, tags
		FROM doctor_queue
		WHERE doctor_id = ? AND status IN (%s)
		ORDER BY enqueue_date`, dbutil.MySQLArgs(2)), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetNDQItemsWithoutDescription(n int) ([]*DoctorQueueItem, error) {
	rows, err := d.db.Query(`
		SELECT id, event_type, item_id, enqueue_date, status, doctor_id, patient_id, description, short_description, action_url, tags
		FROM doctor_queue
		WHERE description = '' OR short_description = ''
		LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) UpdateDoctorQueueItems(dqItems []*DoctorQueueItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	updateStatement, err := tx.Prepare(`
		UPDATE doctor_queue
		SET description = ?, short_description = ?, action_url = ?, patient_id = ?
		WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer updateStatement.Close()

	for _, dqItem := range dqItems {

		if dqItem.ID == 0 {
			tx.Rollback()
			return errors.New("id required")
		}

		if dqItem.ActionURL != nil && dqItem.Description != "" {
			if _, err := updateStatement.Exec(
				dqItem.Description,
				dqItem.ShortDescription,
				dqItem.ActionURL.String(),
				dqItem.PatientID,
				dqItem.ID); err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) GetTotalNumberOfDoctorQueueItemsWithoutDescription() (int, error) {
	var count int
	err := d.db.QueryRow(`
		SELECT count(*) FROM doctor_queue
		WHERE description = '' OR short_description = ''`).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (d *DataService) GetCompletedItemsInDoctorQueue(doctorID int64) ([]*DoctorQueueItem, error) {
	params := []interface{}{doctorID}
	params = dbutil.AppendStringsToInterfaceSlice(params, []string{STATUS_PENDING, STATUS_ONGOING})
	rows, err := d.db.Query(fmt.Sprintf(`
		SELECT id, event_type, item_id, enqueue_date, status, doctor_id, patient_id, description, short_description, action_url, tags
		FROM doctor_queue
		WHERE doctor_id = ? AND status NOT IN (%s)
		ORDER BY enqueue_date DESC`, dbutil.MySQLArgs(2)), params...)
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
	rows, err := d.db.Query(`
		SELECT id, event_type, item_id, enqueue_date, status, doctor_id, patient_id, description, short_description, action_url, tags
		FROM doctor_queue
		WHERE status IN (`+dbutil.MySQLArgs(2)+`)
		ORDER BY enqueue_date`, STATUS_PENDING, STATUS_ONGOING)
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
	rows, err := d.db.Query(`
		SELECT id, event_type, item_id, enqueue_date, status, doctor_id, patient_id, description, short_description, action_url, tags
		FROM doctor_queue
		WHERE status NOT IN (`+dbutil.MySQLArgs(2)+`)
		ORDER BY enqueue_date desc`, STATUS_ONGOING, STATUS_PENDING)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetPendingItemCountForDoctorQueue(doctorID int64) (int64, error) {
	var count int64
	err := d.db.QueryRow(fmt.Sprintf(`
		SELECT count(*)
		FROM doctor_queue
		WHERE doctor_id = ? AND status IN (%s)`,
		dbutil.MySQLArgs(2)),
		doctorID, STATUS_PENDING, STATUS_ONGOING).Scan(&count)
	return count, err
}

func populateDoctorQueueFromRows(rows *sql.Rows) ([]*DoctorQueueItem, error) {
	doctorQueue := make([]*DoctorQueueItem, 0)
	for rows.Next() {
		var queueItem DoctorQueueItem
		var actionURL string
		var tags sql.NullString
		err := rows.Scan(
			&queueItem.ID,
			&queueItem.EventType,
			&queueItem.ItemID,
			&queueItem.EnqueueDate,
			&queueItem.Status,
			&queueItem.DoctorID,
			&queueItem.PatientID,
			&queueItem.Description,
			&queueItem.ShortDescription,
			&actionURL,
			&tags)
		if err != nil {
			return nil, err
		}

		if actionURL != "" {
			aURL, err := app_url.ParseSpruceAction(actionURL)
			if err != nil {
				golog.Errorf("Unable to parse actionURL: %s", err.Error())
			} else {
				queueItem.ActionURL = &aURL
			}
		}
		if tags.String != "" {
			queueItem.Tags = strings.Split(tags.String, tagSeparator)
		} else {
			queueItem.Tags = make([]string, 0)
		}

		doctorQueue = append(doctorQueue, &queueItem)
	}
	return doctorQueue, rows.Err()
}

func (d *DataService) GetMedicationDispenseUnits(languageID int64) (dispenseUnitIDs []int64, dispenseUnits []string, err error) {
	rows, err := d.db.Query(`select dispense_unit.id, ltext from dispense_unit inner join localized_text on app_text_id = dispense_unit_text_id where language_id=?`, languageID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	dispenseUnitIDs = make([]int64, 0)
	dispenseUnits = make([]string, 0)
	for rows.Next() {
		var dipenseUnitId int64
		var dispenseUnit string
		if err := rows.Scan(&dipenseUnitId, &dispenseUnit); err != nil {
			return nil, nil, err
		}
		dispenseUnits = append(dispenseUnits, dispenseUnit)
		dispenseUnitIDs = append(dispenseUnitIDs, dipenseUnitId)
	}
	return dispenseUnitIDs, dispenseUnits, rows.Err()
}

func (d *DataService) AddTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorID, treatmentPlanID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {

		var treatmentIdInPatientTreatmentPlan int64
		if treatmentPlanID != 0 {
			treatmentIdInPatientTreatmentPlan = doctorTreatmentTemplate.Treatment.ID.Int64()
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
			"dispense_unit_id":      doctorTreatmentTemplate.Treatment.DispenseUnitID.Int64(),
			"refills":               doctorTreatmentTemplate.Treatment.NumberRefills.Int64Value,
			"substitutions_allowed": doctorTreatmentTemplate.Treatment.SubstitutionsAllowed,
			"patient_instructions":  doctorTreatmentTemplate.Treatment.PatientInstructions,
			"pharmacy_notes":        doctorTreatmentTemplate.Treatment.PharmacyNotes,
			"status":                common.TStatusCreated.String(),
			"doctor_id":             doctorID,
			"name":                  doctorTreatmentTemplate.Name,
		}

		if doctorTreatmentTemplate.Treatment.DaysSupply.IsValid {
			columnsAndData["days_supply"] = doctorTreatmentTemplate.Treatment.DaysSupply.Int64Value
		}

		if err := d.includeDrugNameComponentIfNonZero(doctorTreatmentTemplate.Treatment.GenericDrugName, drugNameTable, "generic_drug_name_id", columnsAndData, tx); err != nil {
			tx.Rollback()
			return err
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
		for i, c := range columns {
			columns[i] = dbutil.EscapeMySQLName(c)
		}
		res, err := tx.Exec(fmt.Sprintf(`INSERT INTO dr_treatment_template (%s) VALUES (%s)`,
			strings.Join(columns, ","), dbutil.MySQLArgs(len(values))), values...)
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
		doctorTreatmentTemplate.ID = encoding.NewObjectID(drTreatmentTemplateId)

		// add drug db ids to the table
		for drugDbTag, drugDbId := range doctorTreatmentTemplate.Treatment.DrugDBIDs {
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

func (d *DataService) DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {
		_, err = tx.Exec(`update dr_treatment_template set status=? where id = ? and doctor_id = ?`, common.TStatusDeleted.String(), doctorTreatmentTemplate.ID.Int64(), doctorID)
		if err != nil {
			tx.Rollback()
			return err
		}

		// delete all previous selections for this favorited treatment
		_, err = tx.Exec(`delete from treatment_dr_template_selection where dr_treatment_template_id = ?`, doctorTreatmentTemplate.ID.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentTemplates(doctorID int64) ([]*common.DoctorTreatmentTemplate, error) {
	rows, err := d.db.Query(`
		SELECT dtt.id, dtt.name, drug_internal_name, dosage_strength, type,
			dispense_value, dispense_unit_id, ltext, refills, substitutions_allowed,
			days_supply, COALESCE(pharmacy_notes, ''), patient_instructions, creation_date,
			status, COALESCE(dn.name, ''), COALESCE(dr.name, ''), COALESCE(df.name, ''),
			COALESCE(dgn.name, '')
		FROM dr_treatment_template dtt
		INNER JOIN dispense_unit ON dtt.dispense_unit_id = dispense_unit.id
		INNER JOIN localized_text ON localized_text.app_text_id = dispense_unit.dispense_unit_text_id
		LEFT JOIN drug_name dn ON dn.id = drug_name_id
		LEFT JOIN drug_route dr ON dr.id = drug_route_id
		LEFT JOIN drug_form df ON df.id = drug_form_id
		LEFT JOIN drug_name dgn ON dgn.id = generic_drug_name_id
		WHERE status = ? AND doctor_id = ? AND localized_text.language_id = ?`,
		common.TStatusCreated.String(), doctorID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatmentTemplates := make([]*common.DoctorTreatmentTemplate, 0)
	for rows.Next() {
		dtt := &common.DoctorTreatmentTemplate{
			Treatment: &common.Treatment{},
		}
		var treatmentType string
		err = rows.Scan(
			&dtt.ID, &dtt.Name, &dtt.Treatment.DrugInternalName, &dtt.Treatment.DosageStrength, &treatmentType,
			&dtt.Treatment.DispenseValue, &dtt.Treatment.DispenseUnitID, &dtt.Treatment.DispenseUnitDescription,
			&dtt.Treatment.NumberRefills, &dtt.Treatment.SubstitutionsAllowed, &dtt.Treatment.DaysSupply,
			&dtt.Treatment.PharmacyNotes, &dtt.Treatment.PatientInstructions, &dtt.Treatment.CreationDate,
			&dtt.Treatment.Status, &dtt.Treatment.DrugName, &dtt.Treatment.DrugRoute, &dtt.Treatment.DrugForm,
			&dtt.Treatment.GenericDrugName)
		if err != nil {
			return nil, err
		}

		dtt.Treatment.OTC = treatmentType == treatmentOTC

		err = d.fillInDrugDBIdsForTreatment(dtt.Treatment, dtt.ID.Int64(), "dr_treatment_template")
		if err != nil {
			return nil, err
		}

		treatmentTemplates = append(treatmentTemplates, dtt)
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
		err = ErrNotFound("note")
	}
	return note.String, err
}

func (d *DataService) getIdForNameFromTable(tableName, drugComponentName string) (int64, error) {
	var id int64
	err := d.db.QueryRow(`SELECT id FROM `+dbutil.EscapeMySQLName(tableName)+` WHERE name = ?`, drugComponentName).Scan(&id)
	return id, err
}

func (d *DataService) getOrInsertNameInTable(db db, tableName, drugComponentName string) (int64, error) {
	id, err := d.getIdForNameFromTable(tableName, drugComponentName)
	if err == nil {
		return id, nil
	} else if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO `+dbutil.EscapeMySQLName(tableName)+` (name) VALUES (?)`, drugComponentName)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

type DoctorUpdate struct {
	ShortTitle          *string
	LongTitle           *string
	ShortDisplayName    *string
	LongDisplayName     *string
	NPI                 *string
	DEA                 *string
	LargeThumbnailID    *string
	HeroImageID         *string
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
	if update.HeroImageID != nil {
		cols = append(cols, "hero_image_id = ?")
		vals = append(vals, *update.HeroImageID)
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
		rows, err = d.db.Query(`SELECT name, value FROM doctor_attribute WHERE doctor_id = ? AND name IN (`+dbutil.MySQLArgs(len(names))+`)`,
			dbutil.AppendStringsToInterfaceSlice([]interface{}{doctorID}, names)...)
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
		_, err := tx.Exec(`DELETE FROM doctor_attribute WHERE name IN (`+dbutil.MySQLArgs(len(toDelete))+`) AND doctor_id = ?`,
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
	return d.addMedicalLicenses(d.db, licenses)
}

func (d *DataService) addMedicalLicenses(db db, licenses []*common.MedicalLicense) error {
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
	_, err := db.Exec(`
		REPLACE INTO doctor_medical_license
			(doctor_id, state, license_number, status, expiration_date)
		VALUES `+strings.Join(replacements, ","), values...)
	return err
}

func (d *DataService) UpdateMedicalLicenses(doctorID int64, licenses []*common.MedicalLicense) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM doctor_medical_license WHERE doctor_id = ?`, doctorID); err != nil {
		tx.Rollback()
		return err
	}

	if err := d.addMedicalLicenses(tx, licenses); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
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
		whereClause = `WHERE status in (` + dbutil.MySQLArgs(len(statuses)) + `)`
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

func (d *DataService) DoctorEligibleToTreatInState(state string, doctorID int64, pathwayTag string) (bool, error) {
	pathwayID, err := d.pathwayIDFromTag(pathwayTag)
	if err != nil {
		return false, err
	}

	var id int64
	err = d.db.QueryRow(`
		SELECT 1
		FROM care_provider_state_elligibility
		INNER JOIN care_providing_state on care_providing_state.id = care_providing_state_id
		WHERE clinical_pathway_id = ? AND care_providing_state.state = ? AND provider_id = ?
			AND role_type_id = ?`, pathwayID, state, doctorID, d.roleTypeMapping[DOCTOR_ROLE]).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return (err == nil), err
}

// DEPRECATED: remove after Buzz Lightyear release
func (d *DataService) GetSavedDoctorNote(doctorID int64) (string, error) {
	var note sql.NullString
	if err := d.db.QueryRow(
		`SELECT note FROM dr_favorite_treatment_plan ftp
		 INNER JOIN dr_favorite_treatment_plan_membership ftpm ON ftpm.dr_favorite_treatment_plan_id = ftp.id 
		 WHERE ftpm.doctor_id = ? ORDER BY ftp.id LIMIT 1`, doctorID,
	).Scan(&note); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return note.String, nil
}

func (d *DataService) ListTreatmentPlanResourceGuides(tpID int64) ([]*common.ResourceGuide, error) {
	rows, err := d.db.Query(`
		SELECT id, section_id, ordinal, title, photo_url
		FROM treatment_plan_resource_guide
		INNER JOIN resource_guide rg ON rg.id = resource_guide_id
		WHERE treatment_plan_id = ?`,
		tpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guides []*common.ResourceGuide
	for rows.Next() {
		g := &common.ResourceGuide{}
		if err := rows.Scan(&g.ID, &g.SectionID, &g.Ordinal, &g.Title, &g.PhotoURL); err != nil {
			return nil, err
		}
		guides = append(guides, g)
	}

	return guides, rows.Err()
}

func (d *DataService) AddResourceGuidesToTreatmentPlan(tpID int64, guideIDs []int64) error {
	if len(guideIDs) == 0 {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := addResourceGuidesToTreatmentPlan(tx, tpID, guideIDs); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func addResourceGuidesToTreatmentPlan(tx *sql.Tx, tpID int64, guideIDs []int64) error {
	// TODO: optimize this into a single query. not critical though since
	// the number of queries should be very low (1 or 2 maybe)
	stmt, err := tx.Prepare(`
		REPLACE INTO treatment_plan_resource_guide
			(treatment_plan_id, resource_guide_id)
		VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, id := range guideIDs {
		if _, err := stmt.Exec(tpID, id); err != nil {
			return err
		}
	}

	return err
}

func (d *DataService) RemoveResourceGuidesFromTreatmentPlan(tpID int64, guideIDs []int64) error {
	if len(guideIDs) == 0 {
		return nil
	}
	// Optimize for the common case (and currently only case)
	if len(guideIDs) == 1 {
		_, err := d.db.Exec(`
			DELETE FROM treatment_plan_resource_guide
			WHERE treatment_plan_id = ?
				AND resource_guide_id = ?`, tpID, guideIDs[0])
		return err
	}
	vals := make([]interface{}, 1, len(guideIDs)+1)
	vals[0] = tpID
	vals = dbutil.AppendInt64sToInterfaceSlice(vals, guideIDs)
	_, err := d.db.Exec(`
		DELETE FROM treatment_plan_resource_guide
		WHERE treatment_plan_id = ?
			AND resource_guide_id IN (`+dbutil.MySQLArgs(len(guideIDs))+`)`,
		vals...)
	return err
}
