package api

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/third_party/github.com/go-sql-driver/mysql"
)

func (d *DataService) RegisterDoctor(doctor *common.Doctor) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		insert into doctor (account_id, first_name, last_name, short_title, long_title, suffix, prefix, middle_name, gender, dob_year, dob_month, dob_day, status, clinician_id)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doctor.AccountId.Int64(), doctor.FirstName, doctor.LastName, doctor.ShortTitle, doctor.LongTitle,
		doctor.MiddleName, doctor.Suffix, doctor.Prefix, doctor.Gender, doctor.Dob.Year, doctor.Dob.Month, doctor.Dob.Day,
		DOCTOR_REGISTERED, doctor.DoseSpotClinicianId)
	if err != nil {
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
		_, err = tx.Exec(`INSERT INTO account_phone (phone, phone_type, account_id, status) VALUES (?,?,?) `,
			doctor.CellPhone, PHONE_CELL, doctor.AccountId.Int64(), STATUS_ACTIVE)
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

func (d *DataService) GetDoctorFromAccountId(accountId int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.account_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		accountId, PHONE_CELL)
}

func (d *DataService) GetDoctorFromDoseSpotClinicianId(clinicianId int64) (*common.Doctor, error) {
	return d.queryDoctor(`doctor.clinician_id = ? AND (account_phone.phone IS NULL OR account_phone.phone_type = ?)`,
		clinicianId, PHONE_CELL)
}

func (d *DataService) queryDoctor(where string, queryParams ...interface{}) (*common.Doctor, error) {
	row := d.db.QueryRow(fmt.Sprintf(`
		SELECT doctor.id, account_id, phone, first_name, last_name, middle_name, suffix,
			prefix, short_title, long_title, gender, dob_year, dob_month, dob_day, status, clinician_id,
			address.address_line_1,	address.address_line_2, address.city, address.state,
			address.zip_code, person.id
		FROM doctor
		INNER JOIN person ON person.role_type_id = %d AND person.role_id = doctor.id
		LEFT OUTER JOIN account_phone ON account_phone.account_id = doctor.account_id
		LEFT OUTER JOIN doctor_address_selection ON doctor_address_selection.doctor_id = doctor.id
		LEFT OUTER JOIN address ON doctor_address_selection.address_id = address.id
		WHERE %s`, d.roleTypeMapping[DOCTOR_ROLE], where),
		queryParams...)

	var firstName, lastName, status, gender string
	var cellPhoneNumber, addressLine1, addressLine2, city, state, zipCode, middleName, suffix, prefix, shortTitle, longTitle sql.NullString
	var doctorId, accountId encoding.ObjectId
	var dobYear, dobMonth, dobDay int
	var personId int64
	var clinicianId sql.NullInt64
	err := row.Scan(
		&doctorId, &accountId, &cellPhoneNumber, &firstName, &lastName,
		&middleName, &suffix, &prefix, &shortTitle, &longTitle, &gender, &dobYear, &dobMonth,
		&dobDay, &status, &clinicianId, &addressLine1, &addressLine2,
		&city, &state, &zipCode, &personId)
	if err != nil {
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
		Status:              status,
		Gender:              gender,
		CellPhone:           cellPhoneNumber.String,
		DoseSpotClinicianId: clinicianId.Int64,
		DoctorAddress: &common.Address{
			AddressLine1: addressLine1.String,
			AddressLine2: addressLine2.String,
			City:         city.String,
			State:        state.String,
			ZipCode:      zipCode.String,
		},
		Dob:      encoding.Dob{Year: dobYear, Month: dobMonth, Day: dobDay},
		PersonId: personId,
	}

	// populate the doctor url
	doctor.LargeThumbnailUrl = app_url.GetLargeThumbnail(DOCTOR_ROLE, doctor.DoctorId.Int64())
	doctor.SmallThumbnailUrl = app_url.GetSmallThumbnail(DOCTOR_ROLE, doctor.DoctorId.Int64())

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

func (d *DataService) GetRegimenStepsForDoctor(doctorId int64) (regimenSteps []*common.DoctorInstructionItem, err error) {
	// attempt to get regimen steps for doctor
	queryStr := fmt.Sprintf(`select regimen_step.id, text, drug_name_id, drug_form_id, drug_route_id from regimen_step 
										where status='ACTIVE'`)
	regimenSteps, err = d.queryAndInsertPredefinedInstructionsForDoctor(drRegimenStepTable, queryStr, doctorId, getRegimenStepsForDoctor, insertPredefinedRegimenStepsForDoctor)
	if err != nil {
		return
	}

	regimenSteps = getActiveInstructions(regimenSteps)
	return
}

func (d *DataService) GetRegimenStepForDoctor(regimenStepId, doctorId int64) (*common.DoctorInstructionItem, error) {
	var regimenStep common.DoctorInstructionItem
	err := d.db.QueryRow(`select id, text, status from dr_regimen_step where id=? and doctor_id=?`, regimenStepId, doctorId).Scan(&regimenStep.Id, &regimenStep.Text, &regimenStep.Status)
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
	regimenStep.Id = encoding.NewObjectId(instructionId)
	return nil
}

func (d *DataService) UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// lookup the sourceId and status for the current regimen step if it exists
	var sourceId sql.NullInt64
	var status string
	if err := tx.QueryRow(`select source_id, status from dr_regimen_step where id=? and doctor_id=?`, regimenStep.Id.Int64(), doctorId).Scan(&sourceId, &status); err != nil {
		return err
	}

	// if the source id does not exist for the step, this means that
	// this step is the source itself. tracking the source id helps for
	// tracking revision from the beginning of time.
	sourceIdForUpdatedStep := regimenStep.Id.Int64()
	if sourceId.Valid {
		sourceIdForUpdatedStep = sourceId.Int64
	}

	// update the current regimen step to be inactive
	_, err = tx.Exec(`update dr_regimen_step set status=? where id = ? and doctor_id = ?`, STATUS_INACTIVE, regimenStep.Id.Int64(), doctorId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert a new active regimen step in its place
	res, err := tx.Exec(`insert into dr_regimen_step (text, doctor_id, source_id, status) values (?, ?, ?, ?)`, regimenStep.Text, doctorId, sourceIdForUpdatedStep, status)
	if err != nil {
		tx.Rollback()
		return err
	}

	instructionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the regimenStep Id
	regimenStep.Id = encoding.NewObjectId(instructionId)
	return tx.Commit()
}

func (d *DataService) MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	// mark the regimen step to be deleted
	_, err := d.db.Exec(`update dr_regimen_step set status='DELETED' where id = ? and doctor_id = ?`, regimenStep.Id.Int64(), doctorId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, regimenStep := range regimenSteps {
		_, err = tx.Exec(`update dr_regimen_step set status='DELETED' where id = ? and doctor_id=?`, regimenStep.Id.Int64(), doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (d *DataService) GetAdvicePointsForDoctor(doctorId int64) ([]*common.DoctorInstructionItem, error) {
	queryStr := `select id, text from advice_point where status='ACTIVE'`

	advicePoints, err := d.queryAndInsertPredefinedInstructionsForDoctor(drAdvicePointTable, queryStr, doctorId, getAdvicePointsForDoctor, insertPredefinedAdvicePointsForDoctor)
	if err != nil {
		return nil, err
	}

	return getActiveInstructions(advicePoints), nil
}

func (d *DataService) GetAdvicePointForDoctor(advicePointId, doctorId int64) (*common.DoctorInstructionItem, error) {
	var advicePoint common.DoctorInstructionItem
	err := d.db.QueryRow(`select id, text, status from dr_advice_point where id=? and doctor_id=?`, advicePointId, doctorId).Scan(&advicePoint.Id, &advicePoint.Text, &advicePoint.Status)
	if err == sql.ErrNoRows {
		return &advicePoint, NoRowsError
	}
	return &advicePoint, err
}

func (d *DataService) UpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	var sourceId sql.NullInt64
	var status string
	if err := tx.QueryRow(`select source_id, status from dr_advice_point where id=? and doctor_id=?`, advicePoint.Id.Int64(), doctorId).Scan(&sourceId, &status); err != nil {
		return err
	}

	// If a sourceId does not exist for the current advice point, this means that this point
	// is being updated for the first time. In this case, the advice point itself is the source id.
	// Storing the sourceId helps tracking revision on a particular step.
	sourceIdForUpdatedAdvicePoint := advicePoint.Id.Int64()
	if sourceId.Valid {
		sourceIdForUpdatedAdvicePoint = sourceId.Int64
	}

	// update the current advice point to be inactive
	_, err = tx.Exec(`update dr_advice_point set status=? where id = ? and doctor_id = ?`, STATUS_INACTIVE, advicePoint.Id.Int64(), doctorId)
	if err != nil {
		tx.Rollback()
		return err
	}

	res, err := tx.Exec(`insert into dr_advice_point (text, doctor_id, source_id, status) values (?,?,?,?)`, advicePoint.Text, doctorId, sourceIdForUpdatedAdvicePoint, status)
	if err != nil {
		tx.Rollback()
		return err
	}

	instructionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// assign an id given that its a new advice point
	advicePoint.Id = encoding.NewObjectId(instructionId)
	return tx.Commit()
}

func (d *DataService) AddAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec(`insert into dr_advice_point (text, doctor_id, status) values (?,?,?)`, advicePoint.Text, doctorId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}
	instructionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	// assign an id given that its a new advice point
	advicePoint.Id = encoding.NewObjectId(instructionId)
	return tx.Commit()
}

func (d *DataService) MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	// mark the advice point to be deleted
	_, err := d.db.Exec(`update dr_advice_point set status='DELETED' where id = ? and doctor_id = ?`, advicePoint.Id.Int64(), doctorId)
	return err
}

func (d *DataService) MarkAdvicePointsToBeDeleted(advicePoints []*common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`update dr_advice_point set status='DELETED' where id = ? and doctor_id = ?`, advicePoint.Id.Int64(), doctorId)
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
	_, err := d.Exec(`insert into doctor_queue (doctor_id, item_id, event_type, status) values (?,?,?,?)`, doctorQueueItem.DoctorId, doctorQueueItem.ItemId, doctorQueueItem.EventType, doctorQueueItem.Status)
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

func (d *DataService) MarkGenerationOfTreatmentPlanInVisitQueue(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from doctor_queue where status = ? and doctor_id = ? and event_type = ? and item_id = ?`, currentState, doctorId, DQEventTypePatientVisit, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
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

func (d *DataService) GetPendingItemCountForDoctorQueue(doctorId int64) (int64, error) {
	var count int64
	params := [3]interface{}{doctorId, STATUS_PENDING, STATUS_ONGOING}
	err := d.db.QueryRow(fmt.Sprintf(`select count(*) from doctor_queue where doctor_id = ? and status in (%s)`, nReplacements(2)), params).Scan(&count)
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

func (d *DataService) GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) ([]*common.DoctorInstructionItem, error) {
	// first, try and populate instructions belonging to the doctor based on just the drug name
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr := `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
										where name = ? and drug_form_id is null and drug_route_id is null and status='ACTIVE'`
	drugInstructions, err := d.queryAndInsertPredefinedInstructionsForDoctor(drDrugSupplementalInstructionTable, queryStr, doctorId, getDoctorInstructionsBasedOnName, insertPredefinedInstructionsForDoctor, drugName)
	if err != nil {
		return nil, err
	}

	drugInstructions = getActiveInstructions(drugInstructions)

	// second, try and populate instructions belonging to the doctor based on the drug name and the form
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr = `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_form on drug_form_id=drug_form.id 
										where drug_name.name=? and drug_form.name =? and drug_route_id is null and status='ACTIVE'`
	moreInstructions, err := d.queryAndInsertPredefinedInstructionsForDoctor(drDrugSupplementalInstructionTable, queryStr, doctorId, getDoctorInstructionsBasedOnNameAndForm, insertPredefinedInstructionsForDoctor, drugName, drugForm)
	if err != nil {
		return nil, err
	}
	drugInstructions = append(drugInstructions, getActiveInstructions(moreInstructions)...)

	// third, try and populate instructions belonging to the doctor based on the drug name and route
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr = `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id 
										where drug_name.name = ? and drug_route.name = ? and drug_form_id is null and status='ACTIVE'`
	moreInstructions, err = d.queryAndInsertPredefinedInstructionsForDoctor(drDrugSupplementalInstructionTable, queryStr, doctorId, getDoctorInstructionsBasedOnNameAndRoute, insertPredefinedInstructionsForDoctor, drugName, drugRoute)
	if err != nil {
		return nil, err
	}
	drugInstructions = append(drugInstructions, getActiveInstructions(moreInstructions)...)

	// fourth, try and populate instructions belonging to the doctor based on the drug name, form and route
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr = `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id
									inner join drug_form on drug_form_id=drug_form.id
										where drug_name.name=? and drug_route.name = ? and drug_form.name = ? and status='ACTIVE'`
	moreInstructions, err = d.queryAndInsertPredefinedInstructionsForDoctor(drDrugSupplementalInstructionTable, queryStr, doctorId, getDoctorInstructionsBasedOnNameFormAndRoute, insertPredefinedInstructionsForDoctor, drugName, drugForm, drugRoute)
	if err != nil {
		return nil, err
	}
	drugInstructions = append(drugInstructions, getActiveInstructions(moreInstructions)...)

	// get the selected state for this drug
	selectedInstructionIds := make(map[int64]bool, 0)
	rows, err := d.db.Query(`select dr_drug_supplemental_instruction_id from dr_drug_supplemental_instruction_selected_state 
								inner join drug_name on drug_name_id = drug_name.id
								inner join drug_form on drug_form_id = drug_form.id
								inner join drug_route on drug_route_id = drug_route.id
									where drug_name.name = ? and drug_form.name = ? and drug_route.name = ? and doctor_id = ? `, drugName, drugForm, drugRoute, doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var instructionId int64
		if err := rows.Scan(&instructionId); err != nil {
			return nil, err
		}
		selectedInstructionIds[instructionId] = true
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// go through the drug instructions to set the selected state
	for _, instructionItem := range drugInstructions {
		if selectedInstructionIds[instructionItem.Id.Int64()] == true {
			instructionItem.Selected = true
		}
	}

	return drugInstructions, nil
}

func (d *DataService) AddOrUpdateDrugInstructionForDoctor(drugName, drugForm, drugRoute string, drugInstructionToAdd *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	drugNameId, err := d.getOrInsertNameInTable(tx, drugNameTable, drugName)
	if err != nil {
		tx.Rollback()
		return err
	}

	drugFormId, err := d.getOrInsertNameInTable(tx, drugFormTable, drugForm)
	if err != nil {
		tx.Rollback()
		return err
	}

	drugRouteId, err := d.getOrInsertNameInTable(tx, drugRouteTable, drugRoute)
	if err != nil {
		tx.Rollback()
		return err
	}

	drugNameIdStr := strconv.FormatInt(drugNameId, 10)
	drugFormIdStr := strconv.FormatInt(drugFormId, 10)
	drugRouteIdStr := strconv.FormatInt(drugRouteId, 10)

	// check if this is an update to an existing instruction, in which case, retire the existing instruction
	if drugInstructionToAdd.Id.Int64() != 0 {
		// get the heirarcy at which this particular instruction exists so that it can be modified at the same level
		var drugNameNullId, drugFormNullId, drugRouteNullId sql.NullInt64
		err = tx.QueryRow(`select drug_name_id, drug_form_id, drug_route_id from dr_drug_supplemental_instruction where id=? and doctor_id=?`,
			drugInstructionToAdd.Id, doctorId).Scan(&drugNameNullId, &drugFormNullId, &drugRouteNullId)
		if err != nil {
			tx.Rollback()
			return err
		}

		if drugNameNullId.Valid {
			drugNameIdStr = strconv.FormatInt(drugNameNullId.Int64, 10)
		} else {
			drugNameIdStr = "NULL"
		}

		if drugFormNullId.Valid {
			drugFormIdStr = strconv.FormatInt(drugFormNullId.Int64, 10)
		} else {
			drugFormIdStr = "NULL"
		}

		if drugRouteNullId.Valid {
			drugRouteIdStr = strconv.FormatInt(drugRouteNullId.Int64, 10)
		} else {
			drugRouteIdStr = "NULL"
		}

		_, shadowedErr := tx.Exec(`update dr_drug_supplemental_instruction set status=? where id=? and doctor_id = ?`, STATUS_INACTIVE, drugInstructionToAdd.Id, doctorId)
		if shadowedErr != nil {
			tx.Rollback()
			return shadowedErr
		}
	}

	// insert instruction for doctor
	res, err := tx.Exec(`insert into dr_drug_supplemental_instruction (drug_name_id, drug_form_id, drug_route_id, text, doctor_id,status) values (?,?,?,?,?,?)`, drugNameIdStr, drugFormIdStr, drugRouteIdStr, drugInstructionToAdd.Text, doctorId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	instructionId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()

	drugInstructionToAdd.Id = encoding.NewObjectId(instructionId)

	return err
}

func (d *DataService) DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error {
	_, err := d.db.Exec(`update dr_drug_supplemental_instruction set status=? where id = ? and doctor_id = ?`, STATUS_DELETED, drugInstructionToDelete.Id, doctorId)
	return err
}

func (d *DataService) AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute string, drugInstructions []*common.DoctorInstructionItem, treatmentId int64, doctorId int64) error {

	drugNameNullId, err := d.getIdForNameFromTable(drugNameTable, drugName)
	if err != nil {
		return err
	}

	drugFormNullId, err := d.getIdForNameFromTable(drugFormTable, drugForm)
	if err != nil {
		return err
	}

	drugRouteNullId, err := d.getIdForNameFromTable(drugRouteTable, drugRoute)
	if err != nil {
		return err
	}

	// start a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// mark the current set of active instructions as inactive
	_, err = tx.Exec(`update treatment_instructions set status=? where treatment_id = ?`, STATUS_INACTIVE, treatmentId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert the new set of instructions into the treatment instructions
	instructionIds := make([]string, 0)

	for _, instructionItem := range drugInstructions {
		_, err = tx.Exec(`insert into treatment_instructions (treatment_id, dr_drug_instruction_id, status) values (?, ?, ?)`, treatmentId, instructionItem.Id, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
		instructionIds = append(instructionIds, strconv.FormatInt(instructionItem.Id.Int64(), 10))
	}

	// remove the selected state of drug instructions for the drug
	_, err = tx.Exec(`delete from dr_drug_supplemental_instruction_selected_state 
						where drug_name_id = ? and drug_form_id = ? and drug_route_id = ? and doctor_id = ?`,
		drugNameNullId.Int64, drugFormNullId.Int64, drugRouteNullId.Int64, doctorId)

	if err != nil {
		tx.Rollback()
		return err
	}

	//  insert the selected state of drug instructions for the drug
	for _, instructionItem := range drugInstructions {
		_, err := tx.Exec(`insert into dr_drug_supplemental_instruction_selected_state 
										 (drug_name_id, drug_form_id, drug_route_id, dr_drug_supplemental_instruction_id, doctor_id) values (?, ?, ?, ?, ?)`,
			drugNameNullId.Int64, drugFormNullId.Int64, drugRouteNullId.Int64, instructionItem.Id, doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	// commit transaction
	return tx.Commit()
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
			"status":                STATUS_ACTIVE,
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
				_, err = tx.Exec(`update dr_treatment_template set status = ? where id = ?`, STATUS_DELETED, preExistingDoctorFavoriteTreatmentId)
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
		_, err = tx.Exec(`update dr_treatment_template set status='DELETED' where id = ? and doctor_id = ?`, doctorTreatmentTemplate.Id.Int64(), doctorId)
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
			 					where status='ACTIVE' and doctor_id = ? and localized_text.language_id=?`, doctorId, EN_LANGUAGE_ID)
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
		var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
		var substitutionsAllowed bool
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

func (d *DataService) GetSavedMessageForDoctor(doctorID int64) (string, error) {
	var message string
	row := d.db.QueryRow(`SELECT message FROM doctor_saved_case_message WHERE doctor_id = ?`, doctorID)
	if err := row.Scan(&message); err == sql.ErrNoRows {
		return "", NoRowsError
	} else if err != nil {
		return "", err
	}
	return message, nil
}

func (d *DataService) SetSavedMessageForDoctor(doctorID int64, message string) error {
	_, err := d.db.Exec(`REPLACE INTO doctor_saved_case_message (doctor_id, message) VALUES (?, ?)`, doctorID, message)
	return err
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

func getActiveInstructions(drugInstructions []*common.DoctorInstructionItem) []*common.DoctorInstructionItem {
	activeInstructions := make([]*common.DoctorInstructionItem, 0)
	for _, instruction := range drugInstructions {
		if instruction.Status == STATUS_ACTIVE {
			activeInstructions = append(activeInstructions, instruction)
		}
	}
	return activeInstructions
}

func (d *DataService) queryAndInsertPredefinedInstructionsForDoctor(drTableName string, queryStr string, doctorId int64, queryInstructionsFunc doctorInstructionQuery, insertInstructionsFunc insertDoctorInstructionFunc, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	drugInstructions, err := queryInstructionsFunc(d.db, doctorId, drugComponents...)
	if err != nil {
		return nil, err
	}

	// nothing to do if the doctor already has instructions for the combination of the drug components
	if len(drugInstructions) > 0 {
		return drugInstructions, nil
	}

	queryParams := make([]interface{}, 0)
	for _, drugComponent := range drugComponents {
		queryParams = append(queryParams, interface{}(drugComponent))
	}
	rows, err := d.db.Query(queryStr, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	predefinedInstructions, err := getPredefinedInstructionsFromRows(rows)
	if err != nil {
		return nil, err
	}

	// nothing to do if no predefined instructions exist
	if len(predefinedInstructions) == 0 {
		return drugInstructions, nil
	}

	if err := insertInstructionsFunc(d.db, predefinedInstructions, doctorId); err != nil {
		return nil, err
	}

	drugInstructions, err = queryInstructionsFunc(d.db, doctorId, drugComponents...)

	return drugInstructions, nil
}

type insertDoctorInstructionFunc func(db *sql.DB, predefinedInstructions []*predefinedInstruction, doctorId int64) error

func insertPredefinedAdvicePointsForDoctor(db *sql.DB, predefinedAdvicePoints []*predefinedInstruction, doctorId int64) error {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, instruction := range predefinedAdvicePoints {
		_, err = tx.Exec(`insert into dr_advice_point (doctor_id, text, status) values (?, ?, ?)`, doctorId, instruction.text, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func insertPredefinedRegimenStepsForDoctor(db *sql.DB, predefinedInstructions []*predefinedInstruction, doctorId int64) error {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, instruction := range predefinedInstructions {
		_, err = tx.Exec(`insert into dr_regimen_step (doctor_id, text, status) values (?, ?, ?) `, doctorId, instruction.text, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
func insertPredefinedInstructionsForDoctor(db *sql.DB, predefinedInstructions []*predefinedInstruction, doctorId int64) error {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, instruction := range predefinedInstructions {

		drugNameIdStr := "NULL"
		if instruction.drugNameId != 0 {
			drugNameIdStr = strconv.FormatInt(instruction.drugNameId, 10)
		}

		drugFormIdStr := "NULL"
		if instruction.drugFormId != 0 {
			drugFormIdStr = strconv.FormatInt(instruction.drugFormId, 10)
		}

		drugRouteIdStr := "NULL"
		if instruction.drugRouteId != 0 {
			drugRouteIdStr = strconv.FormatInt(instruction.drugRouteId, 10)
		}

		_, err = tx.Exec(`insert into dr_drug_supplemental_instruction 
							(doctor_id, text, drug_name_id, drug_form_id, drug_route_id, status, drug_supplemental_instruction_id) values (?, ?, ?, ?, ?, ?, ?)`, doctorId, instruction.text, drugNameIdStr, drugFormIdStr, drugRouteIdStr, STATUS_ACTIVE, instruction.id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type doctorInstructionQuery func(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error)

func getAdvicePointsForDoctor(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	rows, err := db.Query(`select id, text, status from dr_advice_point where doctor_id=?`, doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

func getRegimenStepsForDoctor(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	rows, err := db.Query(`select id, text, status from dr_regimen_step where doctor_id=?`, doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

func getDoctorInstructionsBasedOnName(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id, text,status from dr_drug_supplemental_instruction 
								inner join drug_name on drug_name_id=drug_name.id 
									where name=? and drug_form_id is null and drug_route_id is null and doctor_id=?`, drugComponents[0], doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

func getDoctorInstructionsBasedOnNameAndForm(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	// then, get instructions belonging to doctor based on drug name and form
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id, text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_form on drug_form_id=drug_form.id 
										where drug_name.name=? and drug_form.name = ? and drug_route_id is null and doctor_id=?`, drugComponents[0], drugComponents[1], doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

func getDoctorInstructionsBasedOnNameAndRoute(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id,text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id 
										where drug_name.name=? and drug_route.name = ? and drug_form_id is null and doctor_id=?`, drugComponents[0], drugComponents[1], doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

func getDoctorInstructionsBasedOnNameFormAndRoute(db *sql.DB, doctorId int64, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	// then, get instructions belonging to doctor based on drug name, route and form
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id,text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id 
									inner join drug_form on drug_form_id = drug_form.id
										where drug_name.name=? and drug_form.name=? and drug_route.name=? and doctor_id=?`, drugComponents[0], drugComponents[1], drugComponents[2], doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getInstructionsFromRows(rows)
}

type predefinedInstruction struct {
	id          int64
	drugFormId  int64
	drugNameId  int64
	drugRouteId int64
	text        string
}

func getPredefinedInstructionsFromRows(rows *sql.Rows) ([]*predefinedInstruction, error) {
	defer rows.Close()
	predefinedInstructions := make([]*predefinedInstruction, 0)
	for rows.Next() {
		var id int64
		var drugFormId, drugNameId, drugRouteId sql.NullInt64
		var text string
		if err := rows.Scan(&id, &text, &drugNameId, &drugFormId, &drugRouteId); err != nil {
			return nil, err
		}
		instruction := &predefinedInstruction{
			id:          id,
			drugFormId:  drugFormId.Int64,
			drugNameId:  drugNameId.Int64,
			drugRouteId: drugRouteId.Int64,
			text:        text,
		}
		predefinedInstructions = append(predefinedInstructions, instruction)
	}
	return predefinedInstructions, rows.Err()
}

func getInstructionsFromRows(rows *sql.Rows) ([]*common.DoctorInstructionItem, error) {
	defer rows.Close()
	drugInstructions := make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id encoding.ObjectId
		var text, status string
		if err := rows.Scan(&id, &text, &status); err != nil {
			return nil, err
		}
		supplementalInstruction := &common.DoctorInstructionItem{}
		supplementalInstruction.Id = id
		supplementalInstruction.Text = text
		supplementalInstruction.Status = status
		drugInstructions = append(drugInstructions, supplementalInstruction)
	}
	return drugInstructions, rows.Err()
}
