package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	status_active                          = "ACTIVE"
	status_created                         = "CREATED"
	status_creating                        = "CREATING"
	status_deleted                         = "DELETED"
	status_inactive                        = "INACTIVE"
	treatment_otc                          = "OTC"
	treatment_rx                           = "RX"
	dr_drug_supplemental_instruction_table = "dr_drug_supplemental_instruction"
	dr_regimen_step_table                  = "dr_regimen_step"
	dr_advice_point_table                  = "dr_advice_point"
	drug_name_table                        = "drug_name"
	drug_form_table                        = "drug_form"
	drug_route_table                       = "drug_route"
	patient_phone_type                     = "MAIN"
	doctor_phone_type                      = "MAIN"
)

type DataService struct {
	DB *sql.DB
}

func (d *DataService) GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error) {
	rows, err := d.DB.Query(`select dispense_unit.id, ltext from dispense_unit inner join localized_text on app_text_id = dispense_unit_text_id where language_id=?`, languageId)
	if err != nil {
		return
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
	return
}

func (d *DataService) GetTreatmentPlanForPatientVisit(patientVisitId int64) (treatmentPlan *common.TreatmentPlan, err error) {
	treatmentPlan = &common.TreatmentPlan{}
	treatmentPlan.PatientVisitId = patientVisitId

	// get treatment plan information
	var status string
	var treatmentPlanId int64
	var creationDate time.Time
	err = d.DB.QueryRow(`select id, status, creation_date from treatment_plan where patient_visit_id = ?`, patientVisitId).Scan(&treatmentPlanId, &status, &creationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return
		}
	}

	treatmentPlan.Id = treatmentPlanId
	treatmentPlan.Status = status
	treatmentPlan.CreationDate = creationDate
	treatmentPlan.Treatments = make([]*common.Treatment, 0)
	rows, err := d.DB.Query(`select treatment.id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, 
			treatment.status from treatment inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id 
				where patient_visit_id=? and treatment.status=?`, patientVisitId, status_created)

	if err != nil {
		if err == sql.ErrNoRows {
			return treatmentPlan, nil
		} else {
			return
		}
	}

	defer rows.Close()

	for rows.Next() {
		var treatmentId, dispenseValue, dispenseUnitId, refills, daysSupply int64
		var drugInternalName, dosageStrength, patientInstructions, treatmentType string
		var substitutionsAllowed bool
		var creationDate time.Time
		var pharmacyNotes sql.NullString
		rows.Scan(&treatmentId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId, &refills, &substitutionsAllowed, &daysSupply, &pharmacyNotes, &patientInstructions, &creationDate, &status)

		treatment := &common.Treatment{}
		treatment.Id = treatmentId
		treatment.PatientVisitId = patientVisitId
		treatment.DrugInternalName = drugInternalName
		treatment.DosageStrength = dosageStrength
		treatment.TreatmentPlanId = treatmentPlan.Id
		treatment.DispenseValue = dispenseValue
		treatment.DispenseUnitId = dispenseUnitId
		treatment.NumberRefills = refills
		treatment.SubstitutionsAllowed = substitutionsAllowed
		treatment.DaysSupply = daysSupply

		if treatmentType == treatment_otc {
			treatment.OTC = true
		}

		if pharmacyNotes.Valid {
			treatment.PharmacyNotes = pharmacyNotes.String
		}
		treatment.PatientInstructions = patientInstructions
		treatment.CreationDate = creationDate
		treatment.Status = status
		treatmentPlan.Treatments = append(treatmentPlan.Treatments, treatment)

		// for each of the drugs, populate the drug db ids
		drugDbIds := make(map[string]string)
		drugRows, anotherErr := d.DB.Query(`select drug_db_id_tag, drug_db_id from drug_db_id where treatment_id = ? `, treatmentId)
		if anotherErr != nil {
			err = anotherErr
			return
		}
		defer drugRows.Close()

		for drugRows.Next() {
			var dbIdTag string
			var dbId int64
			drugRows.Scan(&dbIdTag, &dbId)
			drugDbIds[dbIdTag] = strconv.FormatInt(dbId, 10)
		}

		treatment.DrugDBIds = drugDbIds

		// get the supplemental instructions for this treatment
		instructionsRows, shadowedErr := d.DB.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions 
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id 
													where treatment_instructions.status=? and treatment_id=?`, status_active, treatmentId)
		if shadowedErr != nil {
			err = shadowedErr
			return
		}
		defer instructionsRows.Close()

		drugInstructions := make([]*common.DoctorInstructionItem, 0)
		for instructionsRows.Next() {
			var instructionId int64
			var text string
			instructionsRows.Scan(&instructionId, &text)
			drugInstruction := &common.DoctorInstructionItem{
				Id:       instructionId,
				Text:     text,
				Selected: true,
			}
			drugInstructions = append(drugInstructions, drugInstruction)
		}
		treatment.SupplementalInstructions = drugInstructions
	}

	return
}

func (d *DataService) AddTreatmentsForPatientVisit(treatments []*common.Treatment, PatientVisitId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// check if a treatment plan already exists
	var treatmentPlanId int64
	err = d.DB.QueryRow(`select id from treatment_plan where patient_visit_id = ? `, PatientVisitId).Scan(&treatmentPlanId)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}

	if treatmentPlanId == 0 {
		// if not treatment plan exists, create a treatment plan
		res, err := tx.Exec("insert into treatment_plan (patient_visit_id, status) values (?, ?)", PatientVisitId, status_created)
		if err != nil {
			tx.Rollback()
			return err
		}

		treatmentPlanId, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		// make sure to make inactive all previous treatments within this treatment plan given that new ones are being added
		_, err := tx.Exec("update treatment set status=? where treatment_plan_id = ?", status_inactive, treatmentPlanId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, treatment := range treatments {

		substitutionsAllowedBit := 0
		if treatment.SubstitutionsAllowed == true {
			substitutionsAllowedBit = 1
		}

		treatmentType := treatment_rx
		if treatment.OTC == true {
			treatmentType = treatment_otc
		}

		// add treatment for patient
		var treatmentId int64
		if treatment.PharmacyNotes != "" {
			insertTreatmentStr := `insert into treatment (treatment_plan_id, drug_internal_name, dosage_strength, type, dispense_value, dispense_unit_id, refills, substitutions_allowed, days_supply, patient_instructions, pharmacy_notes, status) 
									values (?,?,?,?,?,?,?,?,?,?,?,?)`
			res, err := tx.Exec(insertTreatmentStr, treatmentPlanId, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, substitutionsAllowedBit, treatment.DaysSupply, treatment.PatientInstructions, treatment.PharmacyNotes, status_created)
			if err != nil {
				tx.Rollback()
				return err
			}

			treatmentId, err = res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
		} else {
			insertTreatmentStr := `insert into treatment (treatment_plan_id, drug_internal_name, dosage_strength, type, dispense_value, dispense_unit_id, refills, substitutions_allowed, days_supply, patient_instructions, status) 
									values (?,?,?,?,?,?,?,?,?,?,?)`
			res, err := tx.Exec(insertTreatmentStr, treatmentPlanId, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, substitutionsAllowedBit, treatment.DaysSupply, treatment.PatientInstructions, status_created)
			if err != nil {
				tx.Rollback()
				return err
			}

			treatmentId, err = res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		// update the treatment object with the information
		treatment.Id = treatmentId
		treatment.TreatmentPlanId = treatmentPlanId

		// add drug db ids to the table
		for drugDbTag, drugDbId := range treatment.DrugDBIds {
			_, err = tx.Exec(`insert into drug_db_id (drug_db_id_tag, drug_db_id, treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, treatment.Id)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	tx.Commit()
	return nil
}

func (d *DataService) getIdForNameFromTable(tableName, drugComponentName string) (nullId sql.NullInt64, err error) {
	err = d.DB.QueryRow(fmt.Sprintf(`select id from %s where name=?`, tableName), drugComponentName).Scan(&nullId)
	return
}

func (d *DataService) getOrInsertNameInTable(tx *sql.Tx, tableName, drugComponentName string) (id int64, err error) {
	drugComponentNameNullId, err := d.getIdForNameFromTable(tableName, drugComponentName)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	if !drugComponentNameNullId.Valid {
		res, shadowedErr := tx.Exec(fmt.Sprintf(`insert into %s (name) values (?)`, tableName), drugComponentName)
		if shadowedErr != nil {
			err = shadowedErr
			return
		}

		id, err = res.LastInsertId()
		if err != nil {
			return
		}
	} else {
		id = drugComponentNameNullId.Int64
	}
	return
}

func (d *DataService) DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error {

	_, err := d.DB.Exec(`update dr_drug_supplemental_instruction set status=? where id = ? and doctor_id = ?`, status_deleted, drugInstructionToDelete.Id, doctorId)
	if err != nil {
		return err
	}

	return nil
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
		instructionIds = append(instructionIds, strconv.FormatInt(instructionItem.Id, 10))
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
	tx.Commit()
	return nil
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
	if drugInstructionToAdd.Id != 0 {
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

	tx.Commit()

	drugInstructionToAdd.Id = instructionId

	return nil
}

func (d *DataService) GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) (drugInstructions []*common.DoctorInstructionItem, err error) {
	// first, try and populate instructions belonging to the doctor based on just the drug name
	// if non exist, then check the predefined set of instructions, create a copy for the doctor and return this copy
	queryStr := `select drug_supplemental_instruction.id, text, drug_name_id, drug_form_id, drug_route_id from drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
										where name = ? and drug_form_id is null and drug_route_id is null and status='ACTIVE'`
	drugInstructions, err = d.queryAndInsertPredefinedInstructionsForDoctor(dr_drug_supplemental_instruction_table, queryStr, doctorId, getDoctorInstructionsBasedOnName, insertPredefinedInstructionsForDoctor, drugName)
	if err != nil {
		return
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
		return
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
		return
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
		return
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
		return
	}
	defer rows.Close()

	for rows.Next() {
		var instructionId int64
		rows.Scan(&instructionId)
		selectedInstructionIds[instructionId] = true
	}

	// go through the drug instructions to set the selected state
	for _, instructionItem := range drugInstructions {
		if selectedInstructionIds[instructionItem.Id] == true {
			instructionItem.Selected = true
		}
	}

	return
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

func (d *DataService) queryAndInsertPredefinedInstructionsForDoctor(drTableName string, queryStr string, doctorId int64, queryInstructionsFunc doctorInstructionQuery, insertInstructionsFunc insertDoctorInstructionFunc, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error) {
	drugInstructions, err = queryInstructionsFunc(d.DB, doctorId, drugComponents...)
	if err != nil {
		return
	}

	// nothing to do if the doctor already has instructions for the combination of the drug components
	if len(drugInstructions) > 0 {
		return
	}

	queryParams := make([]interface{}, 0)
	for _, drugComponent := range drugComponents {
		queryParams = append(queryParams, interface{}(drugComponent))
	}
	rows, err := d.DB.Query(queryStr, queryParams...)
	if err != nil {
		return
	}

	predefinedInstructions, err := getPredefinedInstructionsFromRows(rows)
	if err != nil {
		return
	}

	// nothing to do if no predefined instructions exist
	if len(predefinedInstructions) == 0 {
		return
	}

	err = insertInstructionsFunc(d.DB, predefinedInstructions, doctorId)
	if err != nil {
		return
	}

	drugInstructions, err = queryInstructionsFunc(d.DB, doctorId, drugComponents...)
	return
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
	tx.Commit()
	return nil
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
	tx.Commit()
	return nil
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
	tx.Commit()
	return nil
}

type doctorInstructionQuery func(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error)

func getAdvicePointsForDoctor(db *sql.DB, doctorId int64, drugComponents ...string) (advicePoints []*common.DoctorInstructionItem, err error) {
	rows, err := db.Query(`select id, text, status from dr_advice_point where doctor_id=?`, doctorId)
	if err != nil {
		return
	}

	advicePoints, err = getInstructionsFromRows(rows)
	return
}

func getRegimenStepsForDoctor(db *sql.DB, doctorId int64, drugComponents ...string) (regimenSteps []*common.DoctorInstructionItem, err error) {
	rows, err := db.Query(`select id, text, status from dr_regimen_step where doctor_id=?`, doctorId)
	if err != nil {
		return
	}

	regimenSteps, err = getInstructionsFromRows(rows)
	return
}

func getDoctorInstructionsBasedOnName(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error) {
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id, text,status from dr_drug_supplemental_instruction 
								inner join drug_name on drug_name_id=drug_name.id 
									where name=? and drug_form_id is null and drug_route_id is null and doctor_id=?`, drugComponents[0], doctorId)
	if err != nil {
		return
	}

	drugInstructions, err = getInstructionsFromRows(rows)
	return
}

func getDoctorInstructionsBasedOnNameAndForm(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error) {
	// then, get instructions belonging to doctor based on drug name and form
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id, text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_form on drug_form_id=drug_form.id 
										where drug_name.name=? and drug_form.name = ? and drug_route_id is null and doctor_id=?`, drugComponents[0], drugComponents[1], doctorId)
	if err != nil {
		return
	}

	drugInstructions, err = getInstructionsFromRows(rows)
	return
}

func getDoctorInstructionsBasedOnNameAndRoute(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error) {
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id,text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id 
										where drug_name.name=? and drug_route.name = ? and drug_form_id is null and doctor_id=?`, drugComponents[0], drugComponents[1], doctorId)
	if err != nil {
		return
	}

	drugInstructions, err = getInstructionsFromRows(rows)
	return
}

func getDoctorInstructionsBasedOnNameFormAndRoute(db *sql.DB, doctorId int64, drugComponents ...string) (drugInstructions []*common.DoctorInstructionItem, err error) {
	// then, get instructions belonging to doctor based on drug name, route and form
	rows, err := db.Query(`select dr_drug_supplemental_instruction.id,text,status from dr_drug_supplemental_instruction 
									inner join drug_name on drug_name_id=drug_name.id 
									inner join drug_route on drug_route_id=drug_route.id 
									inner join drug_form on drug_form_id = drug_form.id
										where drug_name.name=? and drug_form.name=? and drug_route.name=? and doctor_id=?`, drugComponents[0], drugComponents[1], drugComponents[2], doctorId)
	if err != nil {
		return
	}

	drugInstructions, err = getInstructionsFromRows(rows)
	return
}

type predefinedInstruction struct {
	id          int64
	drugFormId  int64
	drugNameId  int64
	drugRouteId int64
	text        string
}

func getPredefinedInstructionsFromRows(rows *sql.Rows) (predefinedInstructions []*predefinedInstruction, err error) {
	defer rows.Close()
	predefinedInstructions = make([]*predefinedInstruction, 0)
	for rows.Next() {
		var id int64
		var drugFormId, drugNameId, drugRouteId sql.NullInt64
		var text string
		err = rows.Scan(&id, &text, &drugNameId, &drugFormId, &drugRouteId)
		if err != nil {
			return
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
	return
}

func getInstructionsFromRows(rows *sql.Rows) (drugInstructions []*common.DoctorInstructionItem, err error) {
	defer rows.Close()
	drugInstructions = make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id int64
		var text, status string
		err = rows.Scan(&id, &text, &status)
		if err != nil {
			return
		}
		supplementalInstruction := &common.DoctorInstructionItem{}
		supplementalInstruction.Id = id
		supplementalInstruction.Text = text
		supplementalInstruction.Status = status
		drugInstructions = append(drugInstructions, supplementalInstruction)
	}
	return
}

func (d *DataService) GetAdvicePointsForPatientVisit(patientVisitId int64) (advicePoints []*common.DoctorInstructionItem, err error) {
	rows, err := d.DB.Query(`select dr_advice_point_id,text from advice inner join dr_advice_point on dr_advice_point_id = dr_advice_point.id where patient_visit_id = ?  and advice.status = ?`, patientVisitId, status_active)
	if err != nil {
		return
	}
	defer rows.Close()

	advicePoints = make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id int64
		var text string
		err = rows.Scan(&id, &text)
		if err != nil {
			return
		}

		advicePoint := &common.DoctorInstructionItem{
			Id:   id,
			Text: text,
		}
		advicePoints = append(advicePoints, advicePoint)
	}
	return
}

func (d *DataService) CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, patientVisitId int64) error {
	// begin tx
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update advice set status=? where patient_visit_id=?`, status_inactive, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`insert into advice (patient_visit_id, dr_advice_point_id, status) values (?, ?, ?)`, patientVisitId, advicePoint.Id, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

func (d *DataService) GetAdvicePointsForDoctor(doctorId int64) (advicePoints []*common.DoctorInstructionItem, err error) {
	queryStr := `select id, text from advice_point where status='ACTIVE'`

	advicePoints, err = d.queryAndInsertPredefinedInstructionsForDoctor(dr_advice_point_table, queryStr, doctorId, getAdvicePointsForDoctor, insertPredefinedAdvicePointsForDoctor)
	if err != nil {
		return
	}

	advicePoints = getActiveInstructions(advicePoints)
	return
}

func (d *DataService) AddOrUpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	if advicePoint.Id != 0 {
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
	advicePoint.Id = instructionId
	tx.Commit()
	return nil
}

func (d *DataService) MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error {
	// mark the advice point to be deleted
	_, err := d.DB.Exec(`update dr_advice_point set status='DELETED' where id = ? and doctor_id = ?`, advicePoint.Id, doctorId)
	if err != nil {
		return err
	}
	return nil
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
	regimenStep.Id = instructionId
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
	regimenStep.Id = instructionId
	tx.Commit()
	return nil
}

func (d *DataService) MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error {
	// mark the regimen step to be deleted
	_, err := d.DB.Exec(`update dr_regimen_step set status='DELETED' where id = ? and doctor_id = ?`, regimenStep.Id, doctorId)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataService) CreateRegimenPlanForPatientVisit(regimenPlan *common.RegimenPlan) error {
	// begin tx
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// mark any previous regimen steps for this patient visit and regimen type as inactive
	_, err = tx.Exec(`update regimen set status=? where patient_visit_id = ?`, status_inactive, regimenPlan.PatientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// create new regimen steps within each section
	for _, regimenSection := range regimenPlan.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			_, err = tx.Exec(`insert into regimen (patient_visit_id, regimen_type, dr_regimen_step_id, status) values (?,?,?,?)`, regimenPlan.PatientVisitId, regimenSection.RegimenName, regimenStep.Id, status_active)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// commit tx
	tx.Commit()
	return nil
}

func (d *DataService) GetRegimenPlanForPatientVisit(patientVisitId int64) (regimenPlan *common.RegimenPlan, err error) {
	regimenPlan = &common.RegimenPlan{}
	regimenPlan.PatientVisitId = patientVisitId

	rows, err := d.DB.Query(`select regimen_type, dr_regimen_step.id, dr_regimen_step.text 
								from regimen inner join dr_regimen_step on dr_regimen_step_id = dr_regimen_step.id 
									where patient_visit_id = ? and regimen.status = 'ACTIVE' order by regimen.id`, patientVisitId)
	if err != nil {
		return
	}
	defer rows.Close()

	regimenSections := make(map[string][]*common.DoctorInstructionItem)
	for rows.Next() {
		var regimenType, regimenText string
		var regimenStepId int64
		err = rows.Scan(&regimenType, &regimenStepId, &regimenText)
		regimenStep := &common.DoctorInstructionItem{
			Id:   regimenStepId,
			Text: regimenText,
		}

		regimenSteps := regimenSections[regimenType]
		if regimenSteps == nil {
			regimenSteps = make([]*common.DoctorInstructionItem, 0)
		}
		regimenSteps = append(regimenSteps, regimenStep)
		regimenSections[regimenType] = regimenSteps
	}

	// if there are no regimen steps to return, error out indicating so
	if len(regimenSections) == 0 {
		return nil, NoRegimenPlanForPatientVisit
	}

	regimenSectionsArray := make([]*common.RegimenSection, 0)
	// create the regimen sections
	for regimenSectionName, regimenSteps := range regimenSections {
		regimenSection := &common.RegimenSection{
			RegimenName:  regimenSectionName,
			RegimenSteps: regimenSteps,
		}
		regimenSectionsArray = append(regimenSectionsArray, regimenSection)
	}
	regimenPlan.RegimenSections = regimenSectionsArray
	return
}

func (d *DataService) GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) (answerIntake *common.AnswerIntake, err error) {
	var id, questionId int64
	var potentialAnswerId sql.NullInt64
	var answerText, potentialAnswer, answerSummary sql.NullString
	err = d.DB.QueryRow(`select info_intake.id, info_intake.question_id, info_intake.potential_answer_id, info_intake.answer_text, l2.ltext, l1.ltext
					from info_intake inner join question on question.id = question_id 
					inner join potential_answer on potential_answer_id = potential_answer.id
					inner join localized_text as l1 on answer_localized_text_id = l1.app_text_id
					left outer join localized_text as l2 on answer_summary_text_id = l2.app_text_id
					where info_intake.status='ACTIVE' and question_tag = ? and role_id = ? and role = 'DOCTOR' and info_intake.patient_visit_id = ? and l1.language_id = ?`, questionTag, doctorId, patientVisitId, EN_LANGUAGE_ID).Scan(&id, &questionId, &potentialAnswerId, &answerText, &answerSummary, &potentialAnswer)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoDiagnosisResponseErr
		}
		return
	}

	answerIntake = &common.AnswerIntake{}
	answerIntake.QuestionId = questionId
	if potentialAnswer.Valid {
		answerIntake.PotentialAnswer = potentialAnswer.String
	}
	answerIntake.AnswerIntakeId = id
	if answerText.Valid {
		answerIntake.AnswerText = answerText.String
	}
	answerIntake.PatientVisitId = patientVisitId
	if potentialAnswerId.Valid {
		answerIntake.PotentialAnswerId = potentialAnswerId.Int64
	}

	if answerSummary.Valid {
		answerIntake.AnswerSummary = answerSummary.String
	}

	return
}

func (d *DataService) AddDiagnosisSummaryForPatientVisit(summary string, patientVisitId, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// inactivate any previous summaries for this patient visit
	_, err = tx.Exec(`update diagnosis_summary set status=? where doctor_id = ? and patient_visit_id = ? and status = ?`, status_inactive, doctorId, patientVisitId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into diagnosis_summary (summary, patient_visit_id, doctor_id, status) values (?, ?, ?, ?)`, summary, patientVisitId, doctorId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return err
}

func (d *DataService) GetDiagnosisSummaryForPatientVisit(patientVisitId int64) (summary string, err error) {
	err = d.DB.QueryRow(`select summary from diagnosis_summary where patient_visit_id = ? and status='ACTIVE'`, patientVisitId).Scan(&summary)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
	}
	return
}

func (d *DataService) RecordDoctorAssignmentToPatientVisit(PatientVisitId, DoctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// update any previous assignment to be inactive
	_, err = tx.Exec(`update patient_visit_care_provider_assignment set status=? where patient_visit_id=?`, status_inactive, PatientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert an assignment into table
	_, err = tx.Exec(`insert into patient_visit_care_provider_assignment (provider_role_id, provider_id, patient_visit_id, status) 
							values ((select id from provider_role where provider_tag = 'DOCTOR'), ?, ?, ?)`, DoctorId, PatientVisitId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (d *DataService) GetDoctorAssignedToPatientVisit(PatientVisitId int64) (doctor *common.Doctor, err error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var doctorId, accountId int64

	err = d.DB.QueryRow(`select doctor.id,account_id, first_name, last_name, gender, dob, doctor.status from doctor 
		inner join patient_visit_care_provider_assignment on provider_id = doctor.id
		inner join provider_role on provider_role_id = provider_role.id
		where provider_tag = 'DOCTOR' and patient_visit_id = ? and patient_visit_care_provider_assignment.status = 'ACTIVE'`, PatientVisitId).Scan(&doctorId, &accountId, &firstName, &lastName, &gender, &dob, &status)
	if err != nil {
		return
	}
	doctor = &common.Doctor{
		FirstName: firstName,
		LastName:  lastName,
		Status:    status,
		Gender:    gender,
		AccountId: accountId,
	}
	if dob.Valid {
		doctor.Dob = dob.Time
	}
	doctor.DoctorId = doctorId
	return
}

func (d *DataService) CheckCareProvidingElligibility(shortState string, healthConditionId int64) (isElligible bool, err error) {
	rows, err := d.DB.Query(`select provider_id from care_provider_state_elligibility 
								inner join care_providing_state on care_providing_state_id = care_providing_state.id 
								inner join provider_role on provider_role_id = provider_role.id 
									where state = ? and health_condition_id = ? and provider_tag='DOCTOR'`, shortState, healthConditionId)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	doctorIds := make([]int64, 0)
	for rows.Next() {
		var doctorId int64
		rows.Scan(&doctorId)
		doctorIds = append(doctorIds, doctorId)
	}

	if len(doctorIds) == 0 {
		return false, nil
	}

	return true, nil
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

func (d *DataService) UpdatePatientPharmacy(patientId int64, pharmacyId, pharmacySourceType string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update patient_pharmacy_selection set status=? where patient_id = ? and source = ?`, status_inactive, patientId, pharmacySourceType)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into patient_pharmacy_selection (patient_id, pharmacy_id, source, status) values (?,?,?,?)`, patientId, pharmacyId, pharmacySourceType, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (d *DataService) GetPatientPharmacySelection(patientId int64) (pharmacyId, pharmacySourceType string, err error) {
	err = d.DB.QueryRow(`select pharmacy_id, source from patient_pharmacy_selection where patient_id = ? and status=?`, patientId, status_active).Scan(&pharmacyId, &pharmacySourceType)
	if err == sql.ErrNoRows {
		err = NoRowsError
		return
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

func (d *DataService) RegisterPatient(accountId int64, firstName, lastName, gender, zipCode, city, state, phone string, dob time.Time) (patient *common.Patient, err error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return
	}

	res, err := tx.Exec(`insert into patient (account_id, first_name, last_name, gender, dob, status) 
								values (?, ?, ?,  ?, ?, 'REGISTERED')`, accountId, firstName, lastName, gender, dob)
	if err != nil {
		tx.Rollback()
		return
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Fatal("Unable to return id of inserted item as error was returned when trying to return id", err)
		return
	}

	_, err = tx.Exec(`insert into patient_phone (patient_id, phone, phone_type, status) values (?,?,?, 'ACTIVE')`, lastId, phone, patient_phone_type)
	if err != nil {
		tx.Rollback()
		return
	}

	_, err = tx.Exec(`insert into patient_location (patient_id, zip_code, city, state, status) 
									values (?, ?, ?, ?, 'ACTIVE')`, lastId, zipCode, city, state)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	patient, err = d.GetPatientFromId(lastId)
	return
}

func (d *DataService) RegisterDoctor(accountId int64, firstName, lastName, gender string, dob time.Time) (int64, error) {
	res, err := d.DB.Exec(`insert into doctor (account_id, first_name, last_name, gender, dob, status) 
								values (?, ?, ?, ?, ? , 'REGISTERED')`, accountId, firstName, lastName, gender, dob)
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

func (d *DataService) GetDoctorFromId(doctorId int64) (doctor *common.Doctor, err error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var cellPhoneNumber sql.NullString
	var accountId int64
	err = d.DB.QueryRow(`select account_id, phone, first_name, last_name, gender, dob, status from doctor 
							left outer join doctor_phone on doctor_phone.doctor_id = doctor.id
								where doctor.id = ? and (doctor_phone.phone is null or doctor_phone.phone_type = ?)`, doctorId, doctor_phone_type).Scan(&accountId, &cellPhoneNumber, &firstName, &lastName, &gender, &dob, &status)
	if err != nil {
		return
	}
	doctor = &common.Doctor{
		FirstName: firstName,
		LastName:  lastName,
		Status:    status,
		Gender:    gender,
		AccountId: accountId,
	}
	if dob.Valid {
		doctor.Dob = dob.Time
	}
	if cellPhoneNumber.Valid {
		doctor.CellPhone = cellPhoneNumber.String
	}
	doctor.DoctorId = doctorId
	return
}

func (d *DataService) GetDoctorIdFromAccountId(accountId int64) (int64, error) {
	var doctorId int64
	err := d.DB.QueryRow("select id from doctor where account_id = ?", accountId).Scan(&doctorId)
	return doctorId, err
}

func (d *DataService) GetPatientFromId(patientId int64) (patient *common.Patient, err error) {
	patient, err = d.getPatientBasedOnQuery(`select patient.id, account_id, first_name, last_name, zip_code, city, state, phone, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.id = ? and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
								and (zip_code is null or patient_location.status='ACTIVE') `, patientId, patient_phone_type)
	return
}

func (d *DataService) getPatientBasedOnQuery(queryStr string, queryParams ...interface{}) (patient *common.Patient, err error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var phone, zipCode, city, state sql.NullString
	var patientId, accountId int64
	err = d.DB.QueryRow(queryStr, queryParams...).Scan(&patientId, &accountId, &firstName, &lastName, &zipCode, &city, &state, &phone, &gender, &dob, &status)
	if err != nil {
		return
	}
	patient = &common.Patient{
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
	return
}

func (d *DataService) GetPatientFromAccountId(accountId int64) (patient *common.Patient, err error) {
	patient, err = d.getPatientBasedOnQuery(`select patient.id, account_id, first_name, last_name, zip_code,city,state, phone, gender, dob, patient.status from patient 
							left outer join patient_phone on patient_phone.patient_id = patient.id
							left outer join patient_location on patient_location.patient_id = patient.id
							where patient.account_id = ? and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
								and (zip_code is null or patient_location.status='ACTIVE')`, accountId, patient_phone_type)
	return
}

func (d *DataService) GetPatientIdFromAccountId(accountId int64) (int64, error) {
	var patientId int64
	err := d.DB.QueryRow("select id from patient where account_id = ?", accountId).Scan(&patientId)
	return patientId, err
}
func (d *DataService) GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error) {
	var patientId int64
	err := d.DB.QueryRow("select patient_id from patient_visit where id = ?", patientVisitId).Scan(&patientId)
	return patientId, err
}

func (d *DataService) SubmitPatientVisitWithId(patientVisitId int64) error {
	_, err := d.DB.Exec("update patient_visit set status='SUBMITTED', submitted_date=now() where id = ? and STATUS in ('OPEN', 'PHOTOS_REJECTED')", patientVisitId)
	return err
}

func (d *DataService) AssignPatientVisitToDoctor(DoctorId, PatientVisitId int64) error {
	_, err := d.DB.Exec("insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, 'PENDING', 'PATIENT_VISIT', ?)", DoctorId, PatientVisitId)
	return err
}

func (d *DataService) UpdateStateForPatientVisitInDoctorQueue(DoctorId, PatientVisitId int64, currentState, updatedState string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from doctor_queue where status = ? and doctor_id = ? and event_type = 'PATIENT_VISIT' and item_id = ?`, currentState, DoctorId, PatientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`insert into doctor_queue (doctor_id, status, event_type, item_id) values (?, ?, 'PATIENT_VISIT', ?)`, DoctorId, updatedState, PatientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return err
}

func (d *DataService) GetDoctorQueue(DoctorId int64) (doctorQueue []*DoctorQueueItem, err error) {
	rows, err := d.DB.Query(`select id, event_type, item_id, enqueue_date, completed_date, status from doctor_queue where doctor_id = ? order by enqueue_date desc limit 20`, DoctorId)
	if err != nil {
		return
	}
	defer rows.Close()

	doctorQueue = make([]*DoctorQueueItem, 0)
	for rows.Next() {
		var id, itemId int64
		var eventType, status string
		var completedDate mysql.NullTime
		var enqueueDate time.Time
		err = rows.Scan(&id, &eventType, &itemId, &enqueueDate, &completedDate, &status)
		if err != nil {
			return
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
	return
}

func (d *DataService) getPatientAnswersForQuestionsBasedOnQuery(query string, args ...interface{}) (patientAnswers map[int64][]*common.AnswerIntake, err error) {
	rows, err := d.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	patientAnswers = make(map[int64][]*common.AnswerIntake)
	queriedAnswers := make([]*common.AnswerIntake, 0)
	for rows.Next() {
		var answerId, questionId, layoutVersionId int64
		var potentialAnswerId sql.NullInt64
		var answerText, answerSummaryText, storageBucket, storageKey, storageRegion, potentialAnswer sql.NullString
		var parentQuestionId, parentInfoIntakeId sql.NullInt64
		err = rows.Scan(&answerId, &questionId, &potentialAnswerId, &potentialAnswer, &answerSummaryText, &answerText, &storageBucket, &storageKey, &storageRegion, &layoutVersionId, &parentQuestionId, &parentInfoIntakeId)
		if err != nil {
			return
		}
		patientAnswerToQuestion := &common.AnswerIntake{AnswerIntakeId: answerId,
			QuestionId:      questionId,
			LayoutVersionId: layoutVersionId,
		}

		if potentialAnswerId.Valid {
			patientAnswerToQuestion.PotentialAnswerId = potentialAnswerId.Int64
		}

		if potentialAnswer.Valid {
			patientAnswerToQuestion.PotentialAnswer = potentialAnswer.String
		}
		if answerText.Valid {
			patientAnswerToQuestion.AnswerText = answerText.String
		}
		if answerSummaryText.Valid {
			patientAnswerToQuestion.AnswerSummary = answerSummaryText.String
		}
		if storageBucket.Valid {
			patientAnswerToQuestion.StorageBucket = storageBucket.String
		}
		if storageRegion.Valid {
			patientAnswerToQuestion.StorageRegion = storageRegion.String
		}
		if storageKey.Valid {
			patientAnswerToQuestion.StorageKey = storageKey.String
		}
		if parentQuestionId.Valid {
			patientAnswerToQuestion.ParentQuestionId = parentQuestionId.Int64
		}
		if parentInfoIntakeId.Valid {
			patientAnswerToQuestion.ParentAnswerId = parentInfoIntakeId.Int64
		}
		queriedAnswers = append(queriedAnswers, patientAnswerToQuestion)
	}

	// populate all top-level answers into the map
	patientAnswers = make(map[int64][]*common.AnswerIntake)
	for _, patientAnswerToQuestion := range queriedAnswers {
		if patientAnswerToQuestion.ParentQuestionId == 0 {
			questionId := patientAnswerToQuestion.QuestionId
			if patientAnswers[questionId] == nil {
				patientAnswers[questionId] = make([]*common.AnswerIntake, 0)
			}
			patientAnswers[questionId] = append(patientAnswers[questionId], patientAnswerToQuestion)
		}
	}

	// add all subanswers to the top-level answers by iterating through the queried answers
	// to identify any sub answers
	for _, patientAnswerToQuestion := range queriedAnswers {
		if patientAnswerToQuestion.ParentQuestionId != 0 {
			questionId := patientAnswerToQuestion.ParentQuestionId
			// go through the list of answers to identify the particular answer we care about
			for _, patientAnswer := range patientAnswers[questionId] {
				if patientAnswer.AnswerIntakeId == patientAnswerToQuestion.ParentAnswerId {
					// this is the top level answer to
					if patientAnswer.SubAnswers == nil {
						patientAnswer.SubAnswers = make([]*common.AnswerIntake, 0)
					}
					patientAnswer.SubAnswers = append(patientAnswer.SubAnswers, patientAnswerToQuestion)
				}
			}
		}
	}
	return
}

func (d *DataService) GetFollowUpTimeForPatientVisit(patientVisitId int64) (followUp *common.FollowUp, err error) {
	var followupTime time.Time
	var followupValue int64
	var followupUnit string

	err = d.DB.QueryRow(`select follow_up_date, follow_up_value, follow_up_unit 
							from patient_visit_follow_up where patient_visit_id = ?`, patientVisitId).Scan(&followupTime, &followupValue, &followupUnit)
	if err == sql.ErrNoRows {
		err = nil
		return
	}

	followUp = &common.FollowUp{}
	followUp.PatientVisitId = patientVisitId
	followUp.FollowUpValue = followupValue
	followUp.FollowUpUnit = followupUnit
	followUp.FollowUpTime = followupTime
	return
}

func (d *DataService) UpdateFollowUpTimeForPatientVisit(patientVisitId, currentTimeSinceEpoch, doctorId, followUpValue int64, followUpUnit string) error {
	// check if a follow up time already exists that we can update
	var followupId int64
	err := d.DB.QueryRow(`select id from patient_visit_follow_up where patient_visit_id = ?`, patientVisitId).Scan(&followupId)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	followUpTime := time.Unix(currentTimeSinceEpoch, 0)
	switch followUpUnit {
	case FOLLOW_UP_DAY:
		followUpTime = followUpTime.Add(time.Duration(followUpValue) * 24 * 60 * time.Minute)
	case FOLLOW_UP_MONTH:
		followUpTime = followUpTime.Add(time.Duration(followUpValue) * 30 * 24 * 60 * time.Minute)
	case FOLLOW_UP_WEEK:
		followUpTime = followUpTime.Add(time.Duration(followUpValue) * 7 * 24 * 60 * time.Minute)
	}

	if followupId == 0 {
		_, err = d.DB.Exec(`insert into patient_visit_follow_up (patient_visit_id, doctor_id, follow_up_date, follow_up_value, follow_up_unit, status) 
				values (?,?,?,?,?, 'ADDED')`, patientVisitId, doctorId, followUpTime, followUpValue, followUpUnit)
		if err != nil {
			return err
		}
	} else {
		_, err = d.DB.Exec(`update patient_visit_follow_up set follow_up_date=?, follow_up_value=?, follow_up_unit=?, doctor_id=?, status='UPDATED' where patient_visit_id = ?`, followUpTime, followUpValue, followUpUnit, doctorId, patientVisitId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataService) GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (patientAnswers map[int64][]*common.AnswerIntake, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, object_storage.bucket, object_storage.storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (info_intake.question_id in (%s) or parent_question_id in (%s)) and role_id = ? and info_intake.status='ACTIVE' and role='PATIENT'`, enumeratedStrings, enumeratedStrings)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, patientId)
}

func (d *DataService) GetAnswersForQuestionsInPatientVisit(role string, questionIds []int64, roleId int64, patientVisitId int64) (answerIntakes map[int64][]*common.AnswerIntake, err error) {
	enumeratedStrings := enumerateItemsIntoString(questionIds)
	queryStr := fmt.Sprintf(`select info_intake.id, info_intake.question_id, potential_answer_id, l1.ltext, l2.ltext, answer_text, bucket, storage_key, region_tag,
								layout_version_id, parent_question_id, parent_info_intake_id from info_intake  
								left outer join object_storage on object_storage_id = object_storage.id 
								left outer join region on region_id=region.id 
								left outer join potential_answer on potential_answer_id = potential_answer.id
								left outer join localized_text as l1 on potential_answer.answer_localized_text_id = l1.app_text_id
								left outer join localized_text as l2 on potential_answer.answer_summary_text_id = l2.app_text_id
								where (info_intake.question_id in (%s) or parent_question_id in (%s)) and role_id = ? and patient_visit_id = ? and info_intake.status='ACTIVE' and role='%s'`, enumeratedStrings, enumeratedStrings, role)
	return d.getPatientAnswersForQuestionsBasedOnQuery(queryStr, roleId, patientVisitId)
}

func (d *DataService) GetGlobalSectionIds() (globalSectionIds []int64, err error) {
	rows, err := d.DB.Query(`select id from section where health_condition_id is null`)
	if err != nil {
		return nil, err
	}

	globalSectionIds = make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		globalSectionIds = append(globalSectionIds, sectionId)
	}
	return
}

func (d *DataService) GetSectionIdsForHealthCondition(healthConditionId int64) (sectionIds []int64, err error) {
	rows, err := d.DB.Query(`select id from section where health_condition_id = ?`, healthConditionId)
	if err != nil {
		return nil, err
	}

	sectionIds = make([]int64, 0)
	for rows.Next() {
		var sectionId int64
		rows.Scan(&sectionId)
		sectionIds = append(sectionIds, sectionId)
	}
	return
}

func (d *DataService) GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error) {
	var patientVisitId int64
	err := d.DB.QueryRow("select id from patient_visit where patient_id = ? and health_condition_id = ? and status='OPEN'", patientId, healthConditionId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, err
}

func (d *DataService) GetLastCreatedPatientVisitIdForPatient(patientId int64) (int64, error) {
	var patientVisitId int64
	err := d.DB.QueryRow(`select id from patient_visit where patient_id = ? and creation_date is not null order by creation_date desc limit 1`, patientId).Scan(&patientVisitId)
	if err != nil && err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, nil
}

func (d *DataService) GetCareTeamForPatient(patientId int64) (careTeam *common.PatientCareProviderGroup, err error) {
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

	careTeam = nil
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

func (d *DataService) CreateCareTeamForPatient(patientId int64) (careTeam *common.PatientCareProviderGroup, err error) {
	// identify providers in the state required. Assuming for now that we can only have one provider in the
	// state of CA. The reason for this assumption is that we have not yet figured out how best to deal with
	// multiple active doctors in how they will be assigned to the patient.
	// TODO : Update care team formation when we have more than 1 doctor that we can have as active in our system
	var providerId, providerRoleId int64
	err = d.DB.QueryRow(`select provider_id, provider_role_id from care_provider_state_elligibility 
					inner join care_providing_state on care_providing_state_id = care_providing_state.id
					where state = 'CA'`).Scan(&providerId, &providerRoleId)

	if err == sql.ErrNoRows {
		return nil, NoElligibileProviderInState
	} else if err != nil {
		return
	}

	// create new group assignment for patient visit
	tx, err := d.DB.Begin()
	if err != nil {
		return
	}

	res, err := tx.Exec(`insert into patient_care_provider_group (patient_id, status) values (?, ?)`, patientId, status_creating)
	if err != nil {
		tx.Rollback()
		return
	}

	lastInsertId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return
	}

	// create new assignment for patient
	_, err = tx.Exec("insert into patient_care_provider_assignment (patient_id, provider_role_id, provider_id, assignment_group_id, status) values (?, ?, ?, ?, 'PRIMARY')", patientId, providerRoleId, providerId, lastInsertId)
	if err != nil {
		tx.Rollback()
		return
	}

	// update group assignment to be the active group assignment for this patient visit
	_, err = tx.Exec(`update patient_care_provider_group set status='ACTIVE' where id=?`, lastInsertId)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	careTeam, err = d.GetCareTeamForPatient(patientId)
	return
}

// Adding this only to link the patient and the doctor app so as to show the doctor
// the patient visit review of the latest submitted patient visit
func (d *DataService) GetLatestSubmittedPatientVisit() (*common.PatientVisit, error) {
	var patientId, healthConditionId, layoutVersionId, patientVisitId int64
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string

	row := d.DB.QueryRow(`select id,patient_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status from patient_visit where status in ('SUBMITTED', 'REVIEWING') order by submitted_date desc limit 1`)
	err := row.Scan(&patientVisitId, &patientId, &healthConditionId, &layoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &status)
	if err != nil {
		return nil, err
	}

	patientVisit := &common.PatientVisit{
		PatientVisitId:    patientVisitId,
		PatientId:         patientId,
		HealthConditionId: healthConditionId,
		Status:            status,
		LayoutVersionId:   layoutVersionId,
	}

	if creationDateBytes.Valid {
		patientVisit.CreationDate = creationDateBytes.Time
	}

	if submittedDateBytes.Valid {
		patientVisit.SubmittedDate = submittedDateBytes.Time
	}

	if closedDateBytes.Valid {
		patientVisit.ClosedDate = closedDateBytes.Time
	}

	return patientVisit, err
}

func (d *DataService) GetLatestClosedPatientVisitForPatient(patientId int64) (patientVisit *common.PatientVisit, err error) {
	var healthConditionId, layoutVersionId, patientVisitId int64
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string

	row := d.DB.QueryRow(`select id, health_condition_id, layout_version_id,
		creation_date, submitted_date, closed_date, status from patient_visit where status = 'CLOSED' and patient_id = ? and closed_date is not null order by closed_date desc limit 1`, patientId)
	err = row.Scan(&patientVisitId, &healthConditionId, &layoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoRowsError
		}
		return
	}

	patientVisit = &common.PatientVisit{
		PatientVisitId:    patientVisitId,
		PatientId:         patientId,
		HealthConditionId: healthConditionId,
		Status:            status,
		LayoutVersionId:   layoutVersionId,
	}

	if creationDateBytes.Valid {
		patientVisit.CreationDate = creationDateBytes.Time
	}

	if submittedDateBytes.Valid {
		patientVisit.SubmittedDate = submittedDateBytes.Time
	}

	if closedDateBytes.Valid {
		patientVisit.ClosedDate = closedDateBytes.Time
	}

	return
}

func (d *DataService) GetPatientVisitFromId(patientVisitId int64) (patientVisit *common.PatientVisit, err error) {
	var patientId, healthConditionId, layoutVersionId int64
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string
	row := d.DB.QueryRow(`select patient_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status from patient_visit where id = ?`, patientVisitId)
	err = row.Scan(&patientId, &healthConditionId, &layoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &status)
	if err != nil {
		return nil, err
	}
	patientVisit = &common.PatientVisit{
		PatientVisitId:    patientVisitId,
		PatientId:         patientId,
		HealthConditionId: healthConditionId,
		Status:            status,
		LayoutVersionId:   layoutVersionId,
	}

	if creationDateBytes.Valid {
		patientVisit.CreationDate = creationDateBytes.Time
	}

	if submittedDateBytes.Valid {
		patientVisit.SubmittedDate = submittedDateBytes.Time
	}

	if closedDateBytes.Valid {
		patientVisit.ClosedDate = closedDateBytes.Time
	}

	return patientVisit, err
}

func (d *DataService) GetPatientFromPatientVisitId(patientVisitId int64) (patient *common.Patient, err error) {
	var patientId, accountId int64
	var firstName, lastName, status, gender string
	var phone, zipCode sql.NullString
	var dob mysql.NullTime
	err = d.DB.QueryRow(`select patient.id, account_id, first_name, last_name, zip_code, phone, gender, dob, patient.status from patient_visit
							left outer join patient_phone on patient_phone.patient_id = patient_visit.patient_id
							left outer join patient_location on patient_location.patient_id = patient_visit.patient_id
							inner join patient on patient_visit.patient_id = patient.id where patient_visit.id = ? 
							and (phone is null or (patient_phone.status='ACTIVE' and patient_phone.phone_type=?))
							and (zip_code is null or patient_location.status = 'ACTIVE')`, patientVisitId, patient_phone_type).Scan(&patientId, &accountId, &firstName, &lastName, &zipCode, &phone, &gender, &dob, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return
	}

	patient = &common.Patient{}
	patient.PatientId = patientId
	patient.AccountId = accountId
	patient.FirstName = firstName
	patient.LastName = lastName
	if phone.Valid {
		patient.Phone = phone.String
	}
	if dob.Valid {
		patient.Dob = dob.Time
	}

	patient.Gender = gender
	patient.Status = status
	if zipCode.Valid {
		patient.ZipCode = zipCode.String
	}

	return
}

func (d *DataService) CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_visit (patient_id, health_condition_id, layout_version_id, status) 
								values (?, ?, ?, 'OPEN')`, patientId, healthConditionId, layoutVersionId)
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

func (d *DataService) UpdatePatientVisitStatus(patientVisitId int64, message, event string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	if message != "" {
		// inactivate any existing message given that there is a new message for the patient
		_, err = tx.Exec(`update patient_visit_event set status=? where patient_visit_id = ? and status=?`, status_inactive, patientVisitId, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_visit_event (patient_visit_id, status, event, message) values (?,?,?,?)`, patientVisitId, status_active, event, message)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`update patient_visit set status=? where id = ?`, event, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (d *DataService) GetMessageForPatientVisitStatus(patientVisitId int64) (message string, err error) {
	err = d.DB.QueryRow(`select message from patient_visit_event where patient_visit_id = ? and status = ?`, patientVisitId, status_active).Scan(&message)
	if err != nil && err == sql.ErrNoRows {
		return "", nil
	}
	return
}

func (d *DataService) ClosePatientVisit(patientVisitId int64, event, message string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	if message != "" {
		// inactivate any existing message given that there is a new message for the patient
		_, err = tx.Exec(`update patient_visit_event set status=? where patient_visit_id = ? and status=?`, status_inactive, patientVisitId, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_visit_event (patient_visit_id, status, event, message) values (?,?,?,?)`, patientVisitId, status_active, event, message)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`update patient_visit set status=?, closed_date=now() where id = ?`, event, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (d *DataService) GetQuestionType(questionId int64) (string, error) {
	var questionType string
	err := d.DB.QueryRow(`select qtype from question
						inner join question_type on question_type.id = qtype_id
						where question.id = ?`, questionId).Scan(&questionType)
	return questionType, err
}

func (d *DataService) GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where layout_version_id = ? and language_id = ?`, layoutVersionId, languageId).Scan(&bucket, &key, &region)
	return
}

func (d *DataService) GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from patient_layout_version 
							inner join object_storage on object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where patient_layout_version.status='ACTIVE' and health_condition_id = ? and language_id = ?`, healthConditionId, languageId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetStorageInfoOfCurrentActiveDoctorLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from dr_layout_version 
							inner join layout_version on layout_version_id=layout_version.id 
							inner join object_storage on dr_layout_version.object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where dr_layout_version.status='ACTIVE' and layout_purpose='REVIEW' and role='DOCTOR' and dr_layout_version.health_condition_id = ?`, healthConditionId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error) {
	row := d.DB.QueryRow(`select bucket, storage_key, region_tag, layout_version_id from dr_layout_version
							inner join layout_version on layout_version_id=layout_version.id 
							inner join object_storage on dr_layout_version.object_storage_id=object_storage.id 
							inner join region on region_id=region.id 
								where dr_layout_version.status='ACTIVE' and 
								layout_purpose='DIAGNOSE' and role = 'DOCTOR' and dr_layout_version.health_condition_id = ?`, healthConditionId)
	err = row.Scan(&bucket, &storage, &region, &layoutVersionId)
	return
}

func (d *DataService) GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error) {
	err = d.DB.QueryRow("select layout_version_id from patient_visit where id = ?", patientVisitId).Scan(&layoutVersionId)
	return
}

func (d *DataService) updatePatientInfoIntakesWithStatus(role string, questionIds []int64, roleId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {
	updateStr := fmt.Sprintf(`update info_intake set status=? 
						where role_id = ? and question_id in (%s)
						and patient_visit_id = ? and layout_version_id = ? and status=? and role=?`, enumerateItemsIntoString(questionIds))
	_, err = tx.Exec(updateStr, status, roleId, patientVisitId, layoutVersionId, previousStatus, role)
	return err
}

// This private helper method is to make it possible to update the status of sub answers
// only in combination with the top-level answer to the question. This method makes it possible
// to change the status of the entire set in an atomic fashion.
func (d *DataService) updateSubAnswersToPatientInfoIntakesWithStatus(role string, questionIds []int64, roleId, patientVisitId, layoutVersionId int64, status string, previousStatus string, tx *sql.Tx) (err error) {

	if len(questionIds) == 0 {
		return
	}

	parentInfoIntakeIds := make([]int64, 0)
	queryStr := fmt.Sprintf(`select id from info_intake where role_id = ? and question_id in (%s) and patient_visit_id = ? and layout_version_id = ? and status=? and role=?`, enumerateItemsIntoString(questionIds))
	rows, err := tx.Query(queryStr, roleId, patientVisitId, layoutVersionId, previousStatus, role)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		rows.Scan(&id)
		parentInfoIntakeIds = append(parentInfoIntakeIds, id)
	}

	if len(parentInfoIntakeIds) == 0 {
		return
	}

	updateStr := fmt.Sprintf(`update info_intake set status=? 
						where parent_info_intake_id in (%s) and role=?`, enumerateItemsIntoString(parentInfoIntakeIds))
	_, err = tx.Exec(updateStr, status, role)
	return err
}

func (d *DataService) deleteAnswersWithId(role string, answerIds []int64) error {
	// delete all ids that were in CREATING state since they were committed in that state
	query := fmt.Sprintf("delete from info_intake where id in (%s) and role=?", enumerateItemsIntoString(answerIds))
	_, err := d.DB.Exec(query, role)
	return err
}

func insertAnswersForSubQuestions(tx *sql.Tx, answersToStore []*common.AnswerIntake, parentInfoIntakeId string, parentQuestionId string, status string) (res sql.Result, err error) {

	for _, answerToStore := range answersToStore {

		if answerToStore.PotentialAnswerId == 0 {
			res, err = tx.Exec(`insert into info_intake (role_id, patient_visit_id, parent_info_intake_id, parent_question_id, 
			question_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId, answerToStore.PatientVisitId, parentInfoIntakeId, parentQuestionId,
				answerToStore.QuestionId, answerToStore.AnswerText, answerToStore.LayoutVersionId, answerToStore.Role, status)
		} else {
			res, err = tx.Exec(`insert into info_intake (role_id, patient_visit_id, parent_info_intake_id, parent_question_id, 
			question_id, potential_answer_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId, answerToStore.PatientVisitId, parentInfoIntakeId, parentQuestionId,
				answerToStore.QuestionId, answerToStore.PotentialAnswerId, answerToStore.AnswerText, answerToStore.LayoutVersionId, answerToStore.Role, status)
		}

		if err != nil {
			return
		}
	}

	return
}

func insertAnswers(tx *sql.Tx, answersToStore []*common.AnswerIntake, status string) (res sql.Result, err error) {

	for _, answerToStore := range answersToStore {

		if answerToStore.PotentialAnswerId == 0 {
			res, err = tx.Exec(`insert into info_intake (role_id, patient_visit_id, 
			question_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId, answerToStore.PatientVisitId,
				answerToStore.QuestionId, answerToStore.AnswerText, answerToStore.LayoutVersionId, answerToStore.Role, status)
		} else {
			res, err = tx.Exec(`insert into info_intake (role_id, patient_visit_id,  
			question_id, potential_answer_id, answer_text, layout_version_id, role, status) values
			(?, ?, ?, ?, ?, ?, ?, ?)`, answerToStore.RoleId, answerToStore.PatientVisitId,
				answerToStore.QuestionId, answerToStore.PotentialAnswerId, answerToStore.AnswerText, answerToStore.LayoutVersionId, answerToStore.Role, status)
		}

		if err != nil {
			return
		}
	}

	return
}
func (d *DataService) StoreAnswersForQuestion(role string, roleId, patientVisitId, layoutVersionId int64, answersToStorePerQuestion map[int64][]*common.AnswerIntake) error {

	if len(answersToStorePerQuestion) == 0 {
		return nil
	}

	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for questionId, answersToStore := range answersToStorePerQuestion {
		// keep track of all question ids for which we are storing answers.
		questionIds := make(map[int64]bool)
		questionIds[questionId] = true

		infoIdToAnswersWithSubAnswers := make(map[int64]*common.AnswerIntake)
		subAnswersFound := false
		for _, answerToStore := range answersToStore {
			res, err := insertAnswers(tx, []*common.AnswerIntake{answerToStore}, status_creating)
			if err != nil {
				tx.Rollback()
				return err
			}

			if answerToStore.SubAnswers != nil {
				subAnswersFound = true

				lastInsertId, err := res.LastInsertId()
				if err != nil {
					tx.Rollback()
					return err
				}
				infoIdToAnswersWithSubAnswers[lastInsertId] = answerToStore
			}
		}

		// if there are no subanswers found, then we are pretty much done with the insertion of the
		// answers into the database.
		if !subAnswersFound {
			// ensure to update the status of any prior subquestions linked to the responses
			// of the top level questions that need to be inactivated, along with the answers
			// to the top level question itself.
			d.updateSubAnswersToPatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, status_inactive, status_active, tx)
			d.updatePatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, status_inactive, status_active, tx)

			// if there are no subanswers to store, our job is done with just the top level answers
			d.updatePatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
				patientVisitId, layoutVersionId, status_active, status_creating, tx)
			// tx.Commit()
			continue
		}

		// tx.Commit()
		// create a query to batch insert all subanswers
		for infoIntakeId, answerToStore := range infoIdToAnswersWithSubAnswers {
			_, err = insertAnswersForSubQuestions(tx, answerToStore.SubAnswers, strconv.FormatInt(infoIntakeId, 10), strconv.FormatInt(answerToStore.QuestionId, 10), status_creating)
			if err != nil {
				tx.Rollback()
				return err
			}
			// keep track of all questions for which we are storing answers
			for _, subAnswer := range answerToStore.SubAnswers {
				questionIds[subAnswer.QuestionId] = true
			}
		}

		// deactivate all answers to top level questions as well as their sub-questions
		// as we make the new answers the most current 	up-to-date patient info intake
		err = d.updateSubAnswersToPatientInfoIntakesWithStatus(role, []int64{questionId}, roleId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}

		err = d.updatePatientInfoIntakesWithStatus(role, createKeysArrayFromMap(questionIds), roleId,
			patientVisitId, layoutVersionId, status_inactive, status_active, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}

		// make all answers pertanining to the questionIds collected the new active set of answers for the
		// questions traversed
		err = d.updatePatientInfoIntakesWithStatus(role, createKeysArrayFromMap(questionIds), roleId,
			patientVisitId, layoutVersionId, status_active, status_creating, tx)
		if err != nil {
			tx.Rollback()
			// d.deleteAnswersWithId(role, infoIdsFromMap(infoIdToAnswersWithSubAnswers))
			return err
		}
	}

	tx.Commit()
	return nil
}

func (d *DataService) CreatePhotoAnswerForQuestionRecord(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error) {
	res, err := d.DB.Exec(`insert into info_intake (role, role_id, patient_visit_id, question_id, potential_answer_id, layout_version_id, status) 
							values (?, ?, ?, ?, ?, ?, 'PENDING_UPLOAD')`, role, roleId, patientVisitId, questionId, potentialAnswerId, layoutVersionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return lastId, nil
}

func (d *DataService) UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error {
	_, err := d.DB.Exec(`update info_intake set object_storage_id = ?, status='ACTIVE' where id = ?`, objectStorageId, patientInfoIntakeId)
	return err
}

func (d *DataService) MakeCurrentPhotoAnswerInactive(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) error {
	_, err := d.DB.Exec(`update info_intake set status='INACTIVE' where role_id = ? and question_id = ? 
							and patient_visit_id = ? and potential_answer_id = ? 
							and layout_version_id = ? and role=?`, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId, role)
	return err
}

func (d *DataService) RejectPatientVisitPhotos(patientVisitId int64) error {
	_, err := d.DB.Exec(`update info_intake 
		inner join question on info_intake.question_id = question.id 
		inner join question_type on question_type.id = question.qtype_id 
		set info_intake.status='REJECTED' 
			where info_intake.patient_visit_id = ? and qtype='q_type_photo' and status='ACTIVE'`, patientVisitId)
	return err
}

func (d *DataService) DeactivatePreviousDiagnosisForPatientVisit(PatientVisitId int64, DoctorId int64) error {
	_, err := d.DB.Exec(`update info_intake set status='INACTIVE' where patient_visit_id = ? and status = 'ACTIVE' and role = 'DOCTOR' and role_id = ?`, PatientVisitId, DoctorId)
	return err
}

func (d *DataService) GetHealthConditionInfo(healthConditionTag string) (int64, error) {
	var id int64
	err := d.DB.QueryRow("select id from health_condition where comment = ? ", healthConditionTag).Scan(&id)
	return id, err
}

func (d *DataService) GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error) {
	err = d.DB.QueryRow(`select section.id, ltext from section 
					inner join app_text on section_title_app_text_id = app_text.id 
					inner join localized_text on app_text_id = app_text.id 
						where language_id = ? and section_tag = ?`, languageId, sectionTag).Scan(&id, &title)
	return
}

func (d *DataService) GetQuestionInfo(questionTag string, languageId int64) (id int64, questionTitle string, questionType string, questionSummary string, questionSubText string, parentQuestionId int64, additionalFields map[string]string, formattedFieldTags string, required bool, err error) {
	var byteQuestionTitle, byteQuestionType, byteQuestionSummary, byteQuestionSubtext []byte
	var formattedFieldTagsNull sql.NullString
	var nullParentQuestionId, requiredBit sql.NullInt64
	err = d.DB.QueryRow(
		`select question.id, l1.ltext, qtype, parent_question_id, l2.ltext, l3.ltext, formatted_field_tags, required from question 
			left outer join localized_text as l1 on l1.app_text_id=qtext_app_text_id
			left outer join question_type on qtype_id=question_type.id
			left outer join localized_text as l2 on qtext_short_text_id = l2.app_text_id
			left outer join localized_text as l3 on subtext_app_text_id = l3.app_text_id
				where question_tag = ? and (l1.ltext is NULL or l1.language_id = ?) and (l3.ltext is NULL or l3.language_id=?)`,
		questionTag, languageId, languageId).Scan(&id, &byteQuestionTitle, &byteQuestionType, &nullParentQuestionId, &byteQuestionSummary, &byteQuestionSubtext, &formattedFieldTagsNull, &requiredBit)
	if nullParentQuestionId.Valid {
		parentQuestionId = nullParentQuestionId.Int64
	}
	questionTitle = string(byteQuestionTitle)
	questionType = string(byteQuestionType)
	questionSummary = string(byteQuestionSummary)
	questionSubText = string(byteQuestionSubtext)
	if formattedFieldTagsNull.Valid {
		formattedFieldTags = formattedFieldTagsNull.String
	}
	if requiredBit.Valid && requiredBit.Int64 == 1 {
		required = true
	}

	// get any additional fields pertaining to the question from the database
	rows, err := d.DB.Query(`select question_field, ltext from question_fields
								inner join localized_text on question_fields.app_text_id = localized_text.app_text_id
								where question_id = ? and language_id = ?`, id, languageId)
	if err != nil {
		return
	}
	for rows.Next() {
		var questionField, fieldText string
		err = rows.Scan(&questionField, &fieldText)
		if err != nil {
			return
		}
		if additionalFields == nil {
			additionalFields = make(map[string]string)
		}
		additionalFields[questionField] = fieldText
	}

	return
}

func (d *DataService) GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error) {
	rows, err := d.DB.Query(`select potential_answer.id, l1.ltext, l2.ltext, atype, potential_answer_tag, ordering from potential_answer 
								left outer join localized_text as l1 on answer_localized_text_id=l1.app_text_id 
								left outer join answer_type on atype_id=answer_type.id 
								left outer join localized_text as l2 on answer_summary_text_id=l2.app_text_id
									where question_id = ? and (l1.language_id = ? or l1.ltext is null) and (l2.language_id = ? or l2.ltext is null) and status='ACTIVE'`, questionId, languageId, languageId)
	if err != nil {
		return
	}
	defer rows.Close()
	answerInfos = make([]PotentialAnswerInfo, 0)
	for rows.Next() {
		var id, ordering int64
		var answerType, answerTag string
		var answer, answerSummary sql.NullString
		err = rows.Scan(&id, &answer, &answerSummary, &answerType, &answerTag, &ordering)
		potentialAnswerInfo := PotentialAnswerInfo{}
		if answer.Valid {
			potentialAnswerInfo.Answer = answer.String
		}
		if answerSummary.Valid {
			potentialAnswerInfo.AnswerSummary = answerSummary.String
		}
		potentialAnswerInfo.PotentialAnswerId = id
		potentialAnswerInfo.AnswerTag = answerTag
		potentialAnswerInfo.Ordering = ordering
		potentialAnswerInfo.AnswerType = answerType
		answerInfos = append(answerInfos, potentialAnswerInfo)
		if err != nil {
			return
		}
	}
	return
}

func (d *DataService) GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error) {
	err = d.DB.QueryRow(`select tips.id, ltext from tips
								inner join localized_text on app_text_id=tips_text_id 
									where tips_tag = ? and language_id = ?`, tipTag, languageId).Scan(&id, &tip)
	return
}

func (d *DataService) GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error) {
	err = d.DB.QueryRow(`select tips_section.id, ltext1.ltext, ltext2.ltext from tips_section 
								inner join localized_text as ltext1 on tips_title_text_id=ltext1.app_text_id 
								inner join localized_text as ltext2 on tips_subtext_text_id=ltext2.app_text_id 
									where ltext1.language_id = ? and tips_section_tag = ?`, languageId, tipSectionTag).Scan(&id, &tipSectionTitle, &tipSectionSubtext)
	return
}

func (d *DataService) GetActiveLayoutInfoForHealthCondition(healthConditionTag, role, purpose string) (bucket, key, region string, err error) {
	err = d.DB.QueryRow(`select bucket, storage_key, region_tag from layout_version 
								inner join object_storage on object_storage_id = object_storage.id 
								inner join region on region_id=region.id 
								inner join health_condition on health_condition_id = health_condition.id 
									where layout_version.status='ACTIVE' and role = ? and layout_purpose = ? and health_condition.health_condition_tag = ?`, role, purpose, healthConditionTag).Scan(&bucket, &key, &region)
	return
}

func (d *DataService) GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error) {
	rows, err := d.DB.Query(`select id,language from languages_supported`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	languagesSupported = make([]string, 0)
	languagesSupportedIds = make([]int64, 0)
	for rows.Next() {
		var languageId int64
		var language string
		err := rows.Scan(&languageId, &language)
		if err != nil {
			return nil, nil, err
		}
		languagesSupported = append(languagesSupported, language)
		languagesSupportedIds = append(languagesSupportedIds, languageId)
	}
	return languagesSupported, languagesSupportedIds, nil
}

func (d *DataService) CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error) {
	res, err := d.DB.Exec(`insert into object_storage (bucket, storage_key, status, region_id) 
								values (?, ?, 'CREATING', (select id from region where region_tag = ?))`, bucket, key, region)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) UpdateCloudObjectRecordToSayCompleted(id int64) error {
	_, err := d.DB.Exec("update object_storage set status='ACTIVE' where id = ?", id)
	if err != nil {
		return err
	}

	return nil
}

func (d *DataService) MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, purpose, comment string) (int64, error) {
	res, err := d.DB.Exec(`insert into layout_version (object_storage_id, syntax_version, health_condition_id,role, layout_purpose, comment, status) 
							values (?, ?, ?, ?, ?, ?, 'CREATING')`, objectId, syntaxVersion, healthConditionId, role, purpose, comment)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into dr_layout_version (object_storage_id, layout_version_id, health_condition_id, status) 
							values (?, ?, ?, 'CREATING')`, objectId, layoutVersionId, healthConditionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error) {
	res, err := d.DB.Exec(`insert into patient_layout_version (object_storage_id, language_id, layout_version_id, health_condition_id, status) 
								values (?, ?, ?, ?, 'CREATING')`, objectId, languageId, layoutVersionId, healthConditionId)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, err
}

func (d *DataService) UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error {
	tx, _ := d.DB.Begin()
	// update the current active layouts to DEPRECATED
	_, err := tx.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and role = 'PATIENT' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the current client active layouts to DEPRECATED
	_, err = tx.Exec(`update patient_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ?`, healthConditionId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = tx.Exec(`update layout_version set status='ACTIVE' where id = ?`, layoutId)
	if err != nil {
		tx.Rollback()
		return err
	}

	updateStr := fmt.Sprintf(`update patient_layout_version set status='ACTIVE' where id in (%s)`, enumerateItemsIntoString(clientLayoutIds))
	_, err = tx.Exec(updateStr)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (d *DataService) UpdateDoctorActiveLayouts(layoutId int64, doctorLayoutId int64, healthConditionId int64, purpose string) error {
	tx, _ := d.DB.Begin()

	// update the current client active layouts to DEPRECATED
	_, err := tx.Exec(`update dr_layout_version set status='DEPCRECATED' where status='ACTIVE' and health_condition_id = ? and layout_version_id in (select id from layout_version where role = 'DOCTOR' and layout_purpose = ?)`, healthConditionId, purpose)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return err
	}

	// update the current active layouts to DEPRECATED
	_, err = tx.Exec(`update layout_version set status='DEPCRECATED' where status='ACTIVE' and role = 'DOCTOR' and layout_purpose = ? and health_condition_id = ?`, purpose, healthConditionId)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return err
	}

	// update the new layout as ACTIVE
	_, err = tx.Exec(`update layout_version set status='ACTIVE' where id = ?`, layoutId)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update dr_layout_version set status='ACTIVE' where id = ?`, doctorLayoutId)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func infoIdsFromMap(m map[int64]*common.AnswerIntake) []int64 {
	infoIds := make([]int64, 0)
	for key, _ := range m {
		infoIds = append(infoIds, key)
	}
	return infoIds
}

func createKeysArrayFromMap(m map[int64]bool) []int64 {
	keys := make([]int64, 0)
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

func createValuesArrayFromMap(m map[int64]int64) []int64 {
	values := make([]int64, 0)
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

func enumerateItemsIntoString(ids []int64) string {
	if ids == nil || len(ids) == 0 {
		return ""
	}
	idsStr := make([]string, 0)
	for _, id := range ids {
		idsStr = append(idsStr, strconv.FormatInt(id, 10))
	}
	return strings.Join(idsStr, ",")
}
