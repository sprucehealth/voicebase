package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func (d *DataService) RegisterDoctor(accountId int64, firstName, lastName, gender string, dob time.Time, clinicianId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into doctor (account_id, first_name, last_name, gender, dob, status, clinician_id) 
								values (?, ?, ?, ?, ? , 'REGISTERED', ?)`, accountId, firstName, lastName, gender, dob, clinicianId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return 0, err
	}
	return lastId, err
}

func (d *DataService) GetDoctorFromId(doctorId int64) (*common.Doctor, error) {
	row := d.DB.QueryRow(`select doctor.id, account_id, phone, first_name, last_name, gender, dob, status, clinician_id from doctor 
							left outer join doctor_phone on doctor_phone.doctor_id = doctor.id
								where doctor.id = ? and (doctor_phone.phone is null or doctor_phone.phone_type = ?)`, doctorId, doctor_phone_type)
	return getDoctorFromRow(row)
}

func (d *DataService) GetDoctorFromAccountId(accountId int64) (*common.Doctor, error) {
	row := d.DB.QueryRow(`select doctor.id, account_id, phone, first_name, last_name, gender, dob, status, clinician_id from doctor 
							left outer join doctor_phone on doctor_phone.doctor_id = doctor.id
								where doctor.account_id = ? and (doctor_phone.phone is null or doctor_phone.phone_type = ?)`, accountId, doctor_phone_type)
	return getDoctorFromRow(row)
}

func (d *DataService) GetDoctorFromDoseSpotClinicianId(clinicianId int64) (*common.Doctor, error) {
	row := d.DB.QueryRow(`select doctor.id, account_id, phone, first_name, last_name, gender, dob, status, clinician_id from doctor 
							left outer join doctor_phone on doctor_phone.doctor_id = doctor.id
								where doctor.clinician_id = ? and (doctor_phone.phone is null or doctor_phone.phone_type = ?)`, clinicianId, doctor_phone_type)
	return getDoctorFromRow(row)
}

func getDoctorFromRow(row *sql.Row) (*common.Doctor, error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var cellPhoneNumber sql.NullString
	var doctorId, accountId int64
	var clinicianId sql.NullInt64
	err := row.Scan(&doctorId, &accountId, &cellPhoneNumber, &firstName, &lastName, &gender, &dob, &status, &clinicianId)
	if err != nil {
		return nil, err
	}
	doctor := &common.Doctor{
		AccountId:           common.NewObjectId(accountId),
		DoctorId:            common.NewObjectId(doctorId),
		FirstName:           firstName,
		LastName:            lastName,
		Status:              status,
		Gender:              gender,
		CellPhone:           cellPhoneNumber.String,
		DoseSpotClinicianId: clinicianId.Int64,
	}
	if dob.Valid {
		doctor.Dob = dob.Time
	}

	return doctor, nil
}

func (d *DataService) GetDoctorIdFromAccountId(accountId int64) (int64, error) {
	var doctorId int64
	err := d.DB.QueryRow("select id from doctor where account_id = ?", accountId).Scan(&doctorId)
	return doctorId, err
}

func (d *DataService) GetRegimenStepsForDoctor(doctorId int64) (regimenSteps []*common.DoctorInstructionItem, err error) {
	// attempt to get regimen steps for doctor
	queryStr := fmt.Sprintf(`select regimen_step.id, text, drug_name_id, drug_form_id, drug_route_id from regimen_step 
										where status='ACTIVE'`)
	regimenSteps, err = d.queryAndInsertPredefinedInstructionsForDoctor(dr_regimen_step_table, queryStr, doctorId, getRegimenStepsForDoctor, insertPredefinedRegimenStepsForDoctor)
	if err != nil {
		return
	}

	regimenSteps = getActiveInstructions(regimenSteps)
	return
}

func (d *DataService) AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	res, err := d.DB.Exec(`insert into dr_regimen_step (text, doctor_id,status) values (?,?,?)`, regimenStep.Text, doctorId, status_active)
	if err != nil {
		return err
	}
	instructionId, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// assign an id given that its a new regimen step
	regimenStep.Id = common.NewObjectId(instructionId)
	return nil
}

func (d *DataService) UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// update the current regimen step to be inactive
	_, err = tx.Exec(`update dr_regimen_step set status=? where id = ? and doctor_id = ?`, status_inactive, regimenStep.Id, doctorId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert a new active regimen step in its place
	res, err := tx.Exec(`insert into dr_regimen_step (text, doctor_id, status) values (?, ?, ?)`, regimenStep.Text, doctorId, status_active)
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
	regimenStep.Id = common.NewObjectId(instructionId)
	return tx.Commit()
}

func (d *DataService) MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	// mark the regimen step to be deleted
	_, err := d.DB.Exec(`update dr_regimen_step set status='DELETED' where id = ? and doctor_id = ?`, regimenStep.Id, doctorId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, regimenStep := range regimenSteps {
		_, err = tx.Exec(`update dr_regimen_step set status='DELETED' where id = ? and doctor_id=?`, regimenStep.Id, doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (d *DataService) GetAdvicePointsForDoctor(doctorId int64) ([]*common.DoctorInstructionItem, error) {
	queryStr := `select id, text from advice_point where status='ACTIVE'`

	advicePoints, err := d.queryAndInsertPredefinedInstructionsForDoctor(dr_advice_point_table, queryStr, doctorId, getAdvicePointsForDoctor, insertPredefinedAdvicePointsForDoctor)
	if err != nil {
		return nil, err
	}

	return getActiveInstructions(advicePoints), nil
}

func (d *DataService) AddOrUpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	if advicePoint.Id.Int64() != 0 {
		// update the current advice point to be inactive
		_, err = tx.Exec(`update dr_advice_point set status=? where id = ? and doctor_id = ?`, status_inactive, advicePoint.Id, doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	res, err := tx.Exec(`insert into dr_advice_point (text, doctor_id,status) values (?,?,?)`, advicePoint.Text, doctorId, status_active)
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
	advicePoint.Id = common.NewObjectId(instructionId)
	return tx.Commit()
}

func (d *DataService) MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	// mark the advice point to be deleted
	_, err := d.DB.Exec(`update dr_advice_point set status='DELETED' where id = ? and doctor_id = ?`, advicePoint.Id, doctorId)
	return err
}

func (d *DataService) MarkAdvicePointsToBeDeleted(advicePoints []*common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`update dr_advice_point set status='DELETED' where id = ? and doctor_id = ?`, advicePoint.Id, doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) AssignPatientVisitToDoctor(doctorId, patientVisitId int64) error {
	_, err := d.DB.Exec("insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, ?, ?, ?)", doctorId, status_pending, event_type_patient_visit, patientVisitId)
	return err
}

func (d *DataService) MarkPatientVisitAsOngoingInDoctorQueue(doctorId, patientVisitId int64) error {
	_, err := d.DB.Exec(`update doctor_queue set status=? where event_type=? and item_id=? and doctor_id=?`, status_ongoing, event_type_patient_visit, patientVisitId, doctorId)
	return err
}

func (d *DataService) MarkGenerationOfTreatmentPlanInVisitQueue(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from doctor_queue where status = ? and doctor_id = ? and event_type = ? and item_id = ?`, currentState, doctorId, event_type_patient_visit, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, ?, ?, ?)`, doctorId, updatedState, event_type_treatment_plan, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) GetPendingItemsInDoctorQueue(doctorId int64) ([]*DoctorQueueItem, error) {
	params := []interface{}{doctorId}
	params = appendStringsToInterfaceSlice(params, []string{status_pending, status_ongoing})
	rows, err := d.DB.Query(fmt.Sprintf(`select id, event_type, item_id, enqueue_date, completed_date, status from doctor_queue where doctor_id = ? and status in (%s) order by enqueue_date`, nReplacements(2)), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func (d *DataService) GetCompletedItemsInDoctorQueue(doctorId int64) ([]*DoctorQueueItem, error) {

	params := []interface{}{doctorId}
	params = appendStringsToInterfaceSlice(params, []string{status_pending, status_ongoing})
	rows, err := d.DB.Query(fmt.Sprintf(`select id, event_type, item_id, enqueue_date, completed_date, status from doctor_queue where doctor_id = ? and status not in (%s) order by enqueue_date desc`, nReplacements(2)), params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return populateDoctorQueueFromRows(rows)
}

func populateDoctorQueueFromRows(rows *sql.Rows) ([]*DoctorQueueItem, error) {
	doctorQueue := make([]*DoctorQueueItem, 0)
	for rows.Next() {
		var id, itemId int64
		var eventType, status string
		var completedDate mysql.NullTime
		var enqueueDate time.Time
		err := rows.Scan(&id, &eventType, &itemId, &enqueueDate, &completedDate, &status)
		if err != nil {
			return nil, err
		}

		queueItem := &DoctorQueueItem{}
		queueItem.Id = id
		queueItem.ItemId = itemId
		queueItem.EventType = eventType
		queueItem.Status = status
		queueItem.EnqueueDate = enqueueDate
		if completedDate.Valid {
			queueItem.CompletedDate = completedDate.Time
		}
		doctorQueue = append(doctorQueue, queueItem)
	}
	return doctorQueue, nil
}

func (d *DataService) GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error) {
	rows, err := d.DB.Query(`select dispense_unit.id, ltext from dispense_unit inner join localized_text on app_text_id = dispense_unit_text_id where language_id=?`, languageId)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	dispenseUnitIds = make([]int64, 0)
	dispenseUnits = make([]string, 0)
	for rows.Next() {
		var dipenseUnitId int64
		var dispenseUnit string
		rows.Scan(&dipenseUnitId, &dispenseUnit)
		dispenseUnits = append(dispenseUnits, dispenseUnit)
		dispenseUnitIds = append(dispenseUnitIds, dipenseUnitId)
	}
	return dispenseUnitIds, dispenseUnits, nil
}

func (d *DataService) GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) ([]*common.DoctorInstructionItem, error) {
	// first, try and populate instructions belonging to the doctor based on just the drug name
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr := `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
										where name = ? and drug_form_id is null and drug_route_id is null and status='ACTIVE'`
	drugInstructions, err := d.queryAndInsertPredefinedInstructionsForDoctor(dr_drug_supplemental_instruction_table, queryStr, doctorId, getDoctorInstructionsBasedOnName, insertPredefinedInstructionsForDoctor, drugName)
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
	moreInstructions, err := d.queryAndInsertPredefinedInstructionsForDoctor(dr_drug_supplemental_instruction_table, queryStr, doctorId, getDoctorInstructionsBasedOnNameAndForm, insertPredefinedInstructionsForDoctor, drugName, drugForm)
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
	moreInstructions, err = d.queryAndInsertPredefinedInstructionsForDoctor(dr_drug_supplemental_instruction_table, queryStr, doctorId, getDoctorInstructionsBasedOnNameAndRoute, insertPredefinedInstructionsForDoctor, drugName, drugRoute)
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
	moreInstructions, err = d.queryAndInsertPredefinedInstructionsForDoctor(dr_drug_supplemental_instruction_table, queryStr, doctorId, getDoctorInstructionsBasedOnNameFormAndRoute, insertPredefinedInstructionsForDoctor, drugName, drugForm, drugRoute)
	if err != nil {
		return nil, err
	}
	drugInstructions = append(drugInstructions, getActiveInstructions(moreInstructions)...)

	// get the selected state for this drug
	selectedInstructionIds := make(map[int64]bool, 0)
	rows, err := d.DB.Query(`select dr_drug_supplemental_instruction_id from dr_drug_supplemental_instruction_selected_state 
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
		rows.Scan(&instructionId)
		selectedInstructionIds[instructionId] = true
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
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	drugNameId, err := d.getOrInsertNameInTable(tx, drug_name_table, drugName)
	if err != nil {
		tx.Rollback()
		return err
	}

	drugFormId, err := d.getOrInsertNameInTable(tx, drug_form_table, drugForm)
	if err != nil {
		tx.Rollback()
		return err
	}

	drugRouteId, err := d.getOrInsertNameInTable(tx, drug_route_table, drugRoute)
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

		_, shadowedErr := tx.Exec(`update dr_drug_supplemental_instruction set status=? where id=? and doctor_id = ?`, status_inactive, drugInstructionToAdd.Id, doctorId)
		if shadowedErr != nil {
			tx.Rollback()
			return shadowedErr
		}
	}

	// insert instruction for doctor
	res, err := tx.Exec(`insert into dr_drug_supplemental_instruction (drug_name_id, drug_form_id, drug_route_id, text, doctor_id,status) values (?,?,?,?,?,?)`, drugNameIdStr, drugFormIdStr, drugRouteIdStr, drugInstructionToAdd.Text, doctorId, status_active)
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

	drugInstructionToAdd.Id = common.NewObjectId(instructionId)

	return err
}

func (d *DataService) DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error {
	_, err := d.DB.Exec(`update dr_drug_supplemental_instruction set status=? where id = ? and doctor_id = ?`, status_deleted, drugInstructionToDelete.Id, doctorId)
	return err
}

func (d *DataService) AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute string, drugInstructions []*common.DoctorInstructionItem, treatmentId int64, doctorId int64) error {

	drugNameNullId, err := d.getIdForNameFromTable(drug_name_table, drugName)
	if err != nil {
		return err
	}

	drugFormNullId, err := d.getIdForNameFromTable(drug_form_table, drugForm)
	if err != nil {
		return err
	}

	drugRouteNullId, err := d.getIdForNameFromTable(drug_route_table, drugRoute)
	if err != nil {
		return err
	}

	// start a transaction
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// mark the current set of active instructions as inactive
	_, err = tx.Exec(`update treatment_instructions set status=? where treatment_id = ?`, status_inactive, treatmentId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert the new set of instructions into the treatment instructions
	instructionIds := make([]string, 0)

	for _, instructionItem := range drugInstructions {
		_, err = tx.Exec(`insert into treatment_instructions (treatment_id, dr_drug_instruction_id, status) values (?, ?, ?)`, treatmentId, instructionItem.Id, status_active)
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

func (d *DataService) AddTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {

		var treatmentIdInPatientTreatmentPlan int64
		if doctorTreatmentTemplate.Treatment.TreatmentPlanId.Int64() != 0 {
			treatmentIdInPatientTreatmentPlan = doctorTreatmentTemplate.Treatment.Id.Int64()
		}

		err = d.addTreatment(doctorTreatmentTemplate.Treatment, without_link_to_treatment_plan, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

		lastInsertId, err := tx.Exec(`insert into dr_treatment_template (doctor_id, treatment_id, name, status) values (?,?,?,?)`, doctorId, doctorTreatmentTemplate.Treatment.Id, doctorTreatmentTemplate.Name, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}

		// mark the fact that the treatment was added as a favorite from a patient's treatment
		// and so the selection needs to be maintained
		if treatmentIdInPatientTreatmentPlan != 0 {

			drFavoriteTreatmentId, err := lastInsertId.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}

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
				_, err = tx.Exec(`update dr_treatment_template set status = ? where id = ?`, status_deleted, preExistingDoctorFavoriteTreatmentId)
				if err != nil {
					tx.Rollback()
					return err
				}
			}

			_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatmentIdInPatientTreatmentPlan, drFavoriteTreatmentId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	}

	return tx.Commit()
}

func (d *DataService) DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	for _, doctorTreatmentTemplate := range doctorTreatmentTemplates {
		_, err = tx.Exec(`update dr_treatment_template set status='DELETED' where id = ? and doctor_id = ?`, doctorTreatmentTemplate.Id, doctorId)
		if err != nil {
			tx.Rollback()
			return err
		}

		// delete all previous selections for this favorited treatment
		_, err = tx.Exec(`delete from treatment_dr_template_selection where dr_treatment_template_id = ?`, doctorTreatmentTemplate.Id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentTemplates(doctorId int64) ([]*common.DoctorTreatmentTemplate, error) {
	rows, err := d.DB.Query(`select id, name, treatment_id from dr_treatment_template where status='ACTIVE' and doctor_id = ?`, doctorId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatmentIds := make([]int64, 0)
	treatmentTemplateMapping := make(map[int64]*common.DoctorTreatmentTemplate)
	for rows.Next() {
		var name string
		var treatmentTemplateId, treatmentId int64
		err = rows.Scan(&treatmentTemplateId, &name, &treatmentId)
		if err != nil {
			return nil, err
		}
		treatmentIds = append(treatmentIds, treatmentId)
		treatmentTemplateMapping[treatmentId] = &common.DoctorTreatmentTemplate{
			Id:   common.NewObjectId(treatmentTemplateId),
			Name: name,
		}
	}

	// there are no favorited items to return
	if len(treatmentIds) == 0 {
		return []*common.DoctorTreatmentTemplate{}, nil
	}

	treatmentIdsString := make([]string, 0)
	for _, treatmentId := range treatmentIds {
		treatmentIdsString = append(treatmentIdsString, strconv.FormatInt(treatmentId, 10))
	}

	// get the treatments from the database
	rows, err = d.DB.Query(fmt.Sprintf(`select treatment.id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name from treatment 
				
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment.id in (%s) and localized_text.language_id = ?`, strings.Join(treatmentIdsString, ",")), EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatmentTemplates := make([]*common.DoctorTreatmentTemplate, 0)
	for rows.Next() {
		var treatmentId, dispenseValue, dispenseUnitId, refills, daysSupply int64
		var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
		var substitutionsAllowed bool
		var creationDate time.Time
		var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
		err = rows.Scan(&treatmentId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId, &dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyNotes, &patientInstructions, &creationDate, &status, &drugName, &drugRoute, &drugForm)
		if err != nil {
			return nil, err
		}

		treatment := &common.Treatment{
			Id:                      common.NewObjectId(treatmentId),
			DrugInternalName:        drugInternalName,
			DosageStrength:          dosageStrength,
			DispenseValue:           dispenseValue,
			DispenseUnitId:          common.NewObjectId(dispenseUnitId),
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
		}

		if treatmentType == treatment_otc {
			treatment.OTC = true
		}

		err = d.fillInDrugDBIdsForTreatment(treatment)
		if err != nil {
			return nil, err
		}

		err = d.fillInSupplementalInstructionsForTreatment(treatment)
		if err != nil {
			return nil, err
		}

		treatmentTemplate := treatmentTemplateMapping[treatmentId]
		treatmentTemplate.Treatment = treatment

		// removing the treatmentId for doctorFavorites for the treatment since it does not make sense
		// to have the doctorFavoritetreatmentId and the treatmentId (can be confusing to the developer)
		treatment.Id = nil
		treatmentTemplates = append(treatmentTemplates, treatmentTemplate)

	}
	return treatmentTemplates, nil
}

func (d *DataService) GetCompletedPrescriptionsForDoctor(from, to time.Time, doctorId int64) ([]*common.TreatmentPlan, error) {
	treatmentPlanIdToPlanMapping := make(map[int64]*common.TreatmentPlan)
	treatmentPlans := make([]*common.TreatmentPlan, 0)
	rows, err := d.DB.Query(`select treatment.id, treatment.treatment_plan_id, patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.creation_date, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, 
			treatment.status, drug_name.name, drug_route.name, drug_form.name, erx_status_events.erx_status, erx_status_events.event_details, treatment_plan.sent_date from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on patient_visit.id = treatment_plan.patient_visit_id
				left outer join erx_status_events on erx_status_events.treatment_id = treatment.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
					where localized_text.language_id = ? and treatment_plan.doctor_id = ? and treatment.status = ?
						and (treatment_plan.sent_date is not null and treatment_plan.sent_date >= ? and treatment_plan.sent_date < ?)
						and (erx_status_events.erx_status is null or erx_status_events.status = ?)`, EN_LANGUAGE_ID, doctorId, status_created, from, to, status_active)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var treatmentId, treatmentPlanId, patientId, patientVisitId, dispenseValue, dispenseUnitId, refills, daysSupply int64
		var drugInternalName, dosageStrength, treatmentType, dispenseUnitDescription, patientInstructions, status string
		var creationDate, sentDate, treatmentPlanCreationDate time.Time
		var substituionsAllowed bool
		var erxStatus, pharmacyNotes, drugName, drugForm, drugRoute, eventDetails sql.NullString

		err = rows.Scan(&treatmentId, &treatmentPlanId, &patientId, &patientVisitId, &treatmentPlanCreationDate, &drugInternalName, &dosageStrength, &treatmentType,
			&dispenseValue, &dispenseUnitId, &dispenseUnitDescription, &refills, &substituionsAllowed,
			&daysSupply, &pharmacyNotes, &patientInstructions, &creationDate, &status, &drugName, &drugRoute, &drugForm, &erxStatus, &eventDetails, &sentDate)
		if err != nil {
			return nil, err
		}

		var treatmentPlan *common.TreatmentPlan
		if treatmentPlanIdToPlanMapping[treatmentPlanId] != nil {
			treatmentPlan = treatmentPlanIdToPlanMapping[treatmentPlanId]
		} else {
			treatmentPlan = &common.TreatmentPlan{
				Id:             common.NewObjectId(treatmentPlanId),
				PatientId:      common.NewObjectId(patientId),
				PatientVisitId: common.NewObjectId(patientVisitId),
				CreationDate:   &creationDate,
				SentDate:       &sentDate,
			}
			treatmentPlanIdToPlanMapping[treatmentPlanId] = treatmentPlan
			treatmentPlans = append(treatmentPlans, treatmentPlan)
		}

		treatment := &common.Treatment{
			Id:                      common.NewObjectId(treatmentId),
			TreatmentPlanId:         common.NewObjectId(treatmentPlanId),
			DrugInternalName:        drugInternalName,
			DosageStrength:          dosageStrength,
			DispenseValue:           dispenseValue,
			DispenseUnitId:          common.NewObjectId(dispenseUnitId),
			DispenseUnitDescription: dispenseUnitDescription,
			NumberRefills:           refills,
			SubstitutionsAllowed:    substituionsAllowed,
			DaysSupply:              daysSupply,
			DrugName:                drugName.String,
			DrugForm:                drugForm.String,
			DrugRoute:               drugRoute.String,
			CreationDate:            &creationDate,
			Status:                  status,
			PatientInstructions:     patientInstructions,
			PrescriptionStatus:      erxStatus.String,
			StatusDetails:           eventDetails.String,
			PharmacyNotes:           pharmacyNotes.String,
			OTC:                     treatmentType == treatment_otc,
		}

		err = d.fillInDrugDBIdsForTreatment(treatment)
		if err != nil {
			return nil, err
		}

		err = d.fillInSupplementalInstructionsForTreatment(treatment)
		if err != nil {
			return nil, err
		}

		if treatmentPlan.Treatments == nil {
			treatmentPlan.Treatments = make([]*common.Treatment, 0)
		}
		treatmentPlan.Treatments = append(treatmentPlan.Treatments, treatment)
	}

	return treatmentPlans, nil
}

func (d *DataService) GetPendingRefillRequestStatusEventsForClinic() ([]RefillRequestStatus, error) {
	rows, err := d.DB.Query(`select rx_refill_request_id, rx_refill_request.erx_request_queue_item_id, rx_refill_status, rx_refill_status_date   
								from rx_refill_status_events 
									inner join rx_refill_request on rx_refill_request_id = rx_refill_request.id
									where rx_refill_status_events.status = ? and rx_refill_status = ?`, status_active, RX_REFILL_STATUS_QUEUED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	refillRequestStatuses := make([]RefillRequestStatus, 0)
	for rows.Next() {
		var refillRequestStatus RefillRequestStatus
		err = rows.Scan(&refillRequestStatus.ErxRefillRequestId, &refillRequestStatus.RxRequestQueueItemId, &refillRequestStatus.Status, &refillRequestStatus.StatusTimeStamp)
		if err != nil {
			return nil, err
		}
		refillRequestStatuses = append(refillRequestStatuses, refillRequestStatus)
	}
	return refillRequestStatuses, nil
}

func (d *DataService) AddUnlinkedTreatmentFromPharmacy(unlinkedTreatment *common.Treatment) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	err = d.addUnlinkedTreatmentFromPharmacy(unlinkedTreatment, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CreateRefillRequest(refillRequest *common.RefillRequestItem) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err

	}

	err = d.addPharmacyDispensedTreatment(refillRequest.DispensedPrescription, refillRequest.RequestedPrescription, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	columnsAndData := map[string]interface{}{
		"erx_request_queue_item_id":  refillRequest.RxRequestQueueItemId,
		"requested_drug_description": refillRequest.RequestedDrugDescription,
		"requested_refill_amount":    refillRequest.RequestedRefillAmount,
		"requested_dispense":         refillRequest.RequestedDispense,
		"patient_id":                 refillRequest.Patient.PatientId.Int64(),
		"request_date":               refillRequest.RequestDateStamp,
		"doctor_id":                  refillRequest.Doctor.DoctorId.Int64(),
		"dispensed_treatment_id":     refillRequest.DispensedPrescription.Id.Int64(),
	}

	// only have a link to the unlinked treatment if it so exists
	if refillRequest.RequestedPrescription.Status == common.TREATMENT_STATUS_UNLINKED {
		columnsAndData["unlinked_requested_treatment_id"] = refillRequest.RequestedPrescription.Id.Int64()
	} else {
		columnsAndData["requested_treatment_id"] = refillRequest.RequestedPrescription.Id.Int64()

	}

	if refillRequest.ReferenceNumber != "" {
		columnsAndData["reference_number"] = refillRequest.ReferenceNumber
	}

	if refillRequest.PharmacyRxReferenceNumber != "" {
		columnsAndData["pharmacy_rx_reference_number"] = refillRequest.PharmacyRxReferenceNumber
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)

	lastId, err := tx.Exec(fmt.Sprintf(`insert into rx_refill_request (%s) values (%s)`,
		strings.Join(columns, ","), nReplacements(len(columns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	refillRequest.Id, err = lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (d *DataService) AddRefillRequestStatusEvent(rxRefillRequestId int64, status string, statusDate time.Time) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where status = ? and rx_refill_request_id = ?`, status_inactive, status_active, rxRefillRequestId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, rx_refill_status_date, status) values (?,?,?,?)`, rxRefillRequestId, status, statusDate, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) InsertNewRefillRequestIntoDoctorQueue(refillRequestId int64, doctorId int64) error {
	_, err := d.DB.Exec(`insert into doctor_queue (doctor_id, item_id, event_type, status) values (?,?,?,?) `,
		doctorId, refillRequestId, event_type_refill_request, status_pending)
	return err
}

func (d *DataService) getIdForNameFromTable(tableName, drugComponentName string) (nullId sql.NullInt64, err error) {
	err = d.DB.QueryRow(fmt.Sprintf(`select id from %s where name=?`, tableName), drugComponentName).Scan(&nullId)
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

func (d *DataService) GetRefillRequestFromId(refillRequestId int64) (*common.RefillRequestItem, error) {
	var refillRequest common.RefillRequestItem
	var patientId, doctorId, pharmacyDispensedTreatmentId int64
	var unlinkedRequestedTreatmentId, requestedTreatmentId sql.NullInt64

	// get the refill request
	err := d.DB.QueryRow(`select id, requested_drug_description, requested_refill_amount, 
		requested_dispense, patient_id, request_date, doctor_id, requested_treatment_id, 
		dispensed_treatment_id, unlinked_requested_treatment_id from rx_refill_request 
		where id = ?`, refillRequestId).Scan(&refillRequest.Id, &refillRequest.RequestedDrugDescription, &refillRequest.RequestedRefillAmount,
		&refillRequest.RequestedDispense, &patientId, &refillRequest.RequestDateStamp, &doctorId, &requestedTreatmentId,
		&pharmacyDispensedTreatmentId, &unlinkedRequestedTreatmentId)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// get the patient associated with the refill request
	refillRequest.Patient, err = d.GetPatientFromId(patientId)
	if err != nil {
		return nil, err
	}

	// get the doctor associated with the refill request
	refillRequest.Doctor, err = d.GetDoctorFromId(doctorId)
	if err != nil {
		return nil, err
	}

	// get the pharmacy dispensed treatment
	refillRequest.DispensedPrescription, err = d.getTreatmentForRefillRequest(table_name_pharmacy_dispensed_treatment, pharmacyDispensedTreatmentId)
	if err != nil {
		return nil, err
	}

	if requestedTreatmentId.Valid {
		// get the requested treatment
		rows, err := d.DB.Query(`select treatment.id, treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatmeid = ?`, requestedTreatmentId.Int64)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		if rows.Next() {
			refillRequest.RequestedPrescription, err = d.getTreatmentFromCurrentRow(rows)
			if err != nil {
				return nil, err
			}
			refillRequest.RequestedPrescription.Status = common.TREATMENT_STATUS_LINKED
		}
	}

	// get the unlinked requested treatment
	if unlinkedRequestedTreatmentId.Valid {
		refillRequest.RequestedPrescription, err = d.getTreatmentForRefillRequest(table_name_unlinked_requested_treatment, unlinkedRequestedTreatmentId.Int64)
		if err != nil {
			return nil, err
		}
		refillRequest.RequestedPrescription.Status = common.TREATMENT_STATUS_UNLINKED
	}

	// get the pharmacy
	var pharmacyLocalId int64
	if refillRequest.DispensedPrescription.PharmacyLocalId != nil {
		pharmacyLocalId = refillRequest.DispensedPrescription.PharmacyLocalId.Int64()
	} else if refillRequest.RequestedPrescription != nil && refillRequest.RequestedPrescription.PharmacyLocalId != nil {
		pharmacyLocalId = refillRequest.RequestedPrescription.PharmacyLocalId.Int64()
	}

	refillRequest.Pharmacy, err = d.GetPharmacyFromId(pharmacyLocalId)
	if err != nil {
		return nil, err
	}

	return &refillRequest, nil
}

func (d *DataService) getTreatmentForRefillRequest(tableName string, treatmentId int64) (*common.Treatment, error) {
	var treatment common.Treatment
	var erxId, pharmacyLocalId int64
	var treatmentType string
	var drugName, drugForm, drugRoute sql.NullString

	err := d.DB.QueryRow(fmt.Sprintf(`select erx_id, drug_internal_name, 
							dosage_strength, type, dispense_value, 
							dispense_unit, refills, substitutions_allowed, 
							pharmacy_id, days_supply, pharmacy_notes, 
							patient_instructions, erx_sent_date,
							erx_last_filled_date,  status, drug_name.name, drug_route.name, drug_form.name from %s 
								left outer join drug_name on drug_name_id = drug_name.id
								left outer join drug_route on drug_route_id = drug_route.id
								left outer join drug_form on drug_form_id = drug_form.id
									where %s.id = ?`, tableName, tableName), treatmentId).Scan(&erxId, &treatment.DrugInternalName,
		&treatment.DosageStrength, &treatmentType, &treatment.DispenseValue,
		&treatment.DispenseUnitDescription, &treatment.NumberRefills,
		&treatment.SubstitutionsAllowed, &pharmacyLocalId,
		&treatment.DaysSupply, &treatment.PharmacyNotes,
		&treatment.PatientInstructions, &treatment.ErxSentDate,
		&treatment.ErxLastDateFilled, &treatment.Status,
		&drugName, &drugForm, &drugRoute)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	treatment.PrescriptionId = common.NewObjectId(erxId)
	treatment.DrugName = drugName.String
	treatment.DrugForm = drugForm.String
	treatment.DrugRoute = drugRoute.String
	treatment.OTC = treatmentType == treatment_otc
	treatment.PharmacyLocalId = common.NewObjectId(pharmacyLocalId)

	return &treatment, nil
}

func getActiveInstructions(drugInstructions []*common.DoctorInstructionItem) []*common.DoctorInstructionItem {
	activeInstructions := make([]*common.DoctorInstructionItem, 0)
	for _, instruction := range drugInstructions {
		if instruction.Status == status_active {
			activeInstructions = append(activeInstructions, instruction)
		}
	}
	return activeInstructions
}

func (d *DataService) queryAndInsertPredefinedInstructionsForDoctor(drTableName string, queryStr string, doctorId int64, queryInstructionsFunc doctorInstructionQuery, insertInstructionsFunc insertDoctorInstructionFunc, drugComponents ...string) ([]*common.DoctorInstructionItem, error) {
	drugInstructions, err := queryInstructionsFunc(d.DB, doctorId, drugComponents...)
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
	rows, err := d.DB.Query(queryStr, queryParams...)
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

	if err := insertInstructionsFunc(d.DB, predefinedInstructions, doctorId); err != nil {
		return nil, err
	}

	drugInstructions, err = queryInstructionsFunc(d.DB, doctorId, drugComponents...)

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
		_, err = tx.Exec(`insert into dr_advice_point (doctor_id, text, status) values (?, ?, ?)`, doctorId, instruction.text, status_active)
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
		_, err = tx.Exec(`insert into dr_regimen_step (doctor_id, text, status) values (?, ?, ?) `, doctorId, instruction.text, status_active)
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
							(doctor_id, text, drug_name_id, drug_form_id, drug_route_id, status, drug_supplemental_instruction_id) values (?, ?, ?, ?, ?, ?, ?)`, doctorId, instruction.text, drugNameIdStr, drugFormIdStr, drugRouteIdStr, status_active, instruction.id)
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
		instruction := &predefinedInstruction{}
		instruction.id = id
		if drugFormId.Valid {
			instruction.drugFormId = drugFormId.Int64
		}

		if drugNameId.Valid {
			instruction.drugNameId = drugNameId.Int64
		}

		if drugRouteId.Valid {
			instruction.drugRouteId = drugRouteId.Int64
		}

		instruction.text = text
		predefinedInstructions = append(predefinedInstructions, instruction)
	}
	return predefinedInstructions, nil
}

func getInstructionsFromRows(rows *sql.Rows) ([]*common.DoctorInstructionItem, error) {
	defer rows.Close()
	drugInstructions := make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id int64
		var text, status string
		if err := rows.Scan(&id, &text, &status); err != nil {
			return nil, err
		}
		supplementalInstruction := &common.DoctorInstructionItem{}
		supplementalInstruction.Id = common.NewObjectId(id)
		supplementalInstruction.Text = text
		supplementalInstruction.Status = status
		drugInstructions = append(drugInstructions, supplementalInstruction)
	}
	return drugInstructions, nil
}

// this method is used to add treatments that come in from dosespot (either pharmacy dispensed medication or treatments that don't exist but
// are the basis of a refill request)
func (d *DataService) addUnlinkedTreatmentFromPharmacy(treatment *common.Treatment, tx *sql.Tx) error {
	substitutionsAllowedBit := 0
	if treatment.SubstitutionsAllowed {
		substitutionsAllowedBit = 1
	}

	treatmentType := treatment_rx
	if treatment.OTC {
		treatmentType = treatment_otc
	}

	columnsAndData := map[string]interface{}{
		"drug_internal_name":    treatment.DrugInternalName,
		"dosage_strength":       treatment.DosageStrength,
		"type":                  treatmentType,
		"dispense_value":        treatment.DispenseValue,
		"dispense_unit":         treatment.DispenseUnitDescription,
		"refills":               treatment.NumberRefills,
		"substitutions_allowed": substitutionsAllowedBit,
		"days_supply":           treatment.DaysSupply,
		"patient_instructions":  treatment.PatientInstructions,
		"pharmacy_notes":        treatment.PharmacyNotes,
		"status":                treatment.Status,
		"erx_id":                treatment.PrescriptionId.Int64(),
		"erx_sent_date":         treatment.ErxSentDate,
		"erx_last_filled_date":  treatment.ErxLastDateFilled,
		"pharmacy_id":           treatment.PharmacyLocalId,
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugName, drug_name_table, "drug_name_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugForm, drug_form_table, "drug_form_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugRoute, drug_route_table, "drug_route_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)
	res, err := tx.Exec(fmt.Sprintf(`insert into unlinked_requested_treatment (%s) values (%s)`, strings.Join(columns, ","), nReplacements(len(dataForColumns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	treatmentId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	treatment.Id = common.NewObjectId(treatmentId)
	// add drug db ids to the table
	for drugDbTag, drugDbId := range treatment.DrugDBIds {
		_, err := tx.Exec(`insert into unlinked_requested_treatment_drug_db_id (drug_db_id_tag, drug_db_id, unlinked_requested_treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, treatment.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func (d *DataService) addPharmacyDispensedTreatment(dispensedTreatment, requestedTreatment *common.Treatment, tx *sql.Tx) error {
	substitutionsAllowedBit := 0
	if dispensedTreatment.SubstitutionsAllowed {
		substitutionsAllowedBit = 1
	}

	treatmentType := treatment_rx
	if dispensedTreatment.OTC {
		treatmentType = treatment_otc
	}

	columnsAndData := map[string]interface{}{
		"drug_internal_name":    dispensedTreatment.DrugInternalName,
		"dosage_strength":       dispensedTreatment.DosageStrength,
		"type":                  treatmentType,
		"dispense_value":        dispensedTreatment.DispenseValue,
		"dispense_unit":         dispensedTreatment.DispenseUnitDescription,
		"refills":               dispensedTreatment.NumberRefills,
		"substitutions_allowed": substitutionsAllowedBit,
		"days_supply":           dispensedTreatment.DaysSupply,
		"patient_instructions":  dispensedTreatment.PatientInstructions,
		"pharmacy_notes":        dispensedTreatment.PharmacyNotes,
		"status":                dispensedTreatment.Status,
		"erx_id":                dispensedTreatment.PrescriptionId.Int64(),
		"erx_sent_date":         dispensedTreatment.ErxSentDate,
		"erx_last_filled_date":  dispensedTreatment.ErxLastDateFilled,
		"pharmacy_id":           dispensedTreatment.PharmacyLocalId,
	}

	if requestedTreatment.Status == common.TREATMENT_STATUS_LINKED {
		columnsAndData["treatment_id"] = requestedTreatment.Id.Int64()
	} else if requestedTreatment.Status == common.TREATMENT_STATUS_UNLINKED {
		columnsAndData["unlinked_requested_treatment_id"] = requestedTreatment.Id.Int64()
	}

	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugName, drug_name_table, "drug_name_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugForm, drug_form_table, "drug_form_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugRoute, drug_route_table, "drug_route_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)
	res, err := tx.Exec(fmt.Sprintf(`insert into pharmacy_dispensed_treatment (%s) values (%s)`, strings.Join(columns, ","), nReplacements(len(dataForColumns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	treatmentId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	dispensedTreatment.Id = common.NewObjectId(treatmentId)
	// add drug db ids to the table
	for drugDbTag, drugDbId := range dispensedTreatment.DrugDBIds {
		_, err := tx.Exec(`insert into pharmacy_dispensed_treatment_drug_db_id (drug_db_id_tag, drug_db_id, pharmacy_dispensed_treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, dispensedTreatment.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func getKeysAndValuesFromMap(m map[string]interface{}) ([]string, []interface{}) {
	values := make([]interface{}, 0)
	keys := make([]string, 0)
	for key, value := range m {
		keys = append(keys, key)
		values = append(values, value)
	}
	return keys, values
}

func (d *DataService) includeDrugNameComponentIfNonZero(drugNameComponent, tableName, columnName string, columnsAndData map[string]interface{}, tx *sql.Tx) error {
	if drugNameComponent != "" {
		componentId, err := d.getOrInsertNameInTable(tx, tableName, drugNameComponent)
		if err != nil {
			return err
		}
		columnsAndData[columnName] = componentId
	}
	return nil
}
