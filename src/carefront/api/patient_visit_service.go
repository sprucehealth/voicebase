package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"time"
)

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

func (d *DataService) GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error) {
	var patientId int64
	err := d.DB.QueryRow("select patient_id from patient_visit where id = ?", patientVisitId).Scan(&patientId)
	return patientId, err
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

func (d *DataService) GetLatestClosedPatientVisitForPatient(patientId int64) (*common.PatientVisit, error) {
	var healthConditionId, layoutVersionId, patientVisitId int64
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string

	row := d.DB.QueryRow(`select id, health_condition_id, layout_version_id,
		creation_date, submitted_date, closed_date, status from patient_visit where status in ('CLOSED','TREATED') and patient_id = ? and closed_date is not null order by closed_date desc limit 1`, patientId)
	err := row.Scan(&patientVisitId, &healthConditionId, &layoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			err = NoRowsError
		}
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

	return patientVisit, nil
}

func (d *DataService) GetPatientVisitFromId(patientVisitId int64) (*common.PatientVisit, error) {
	patientVisit := common.PatientVisit{PatientVisitId: patientVisitId}
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	err := d.DB.QueryRow(`select patient_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status from patient_visit where id = ?`, patientVisitId,
	).Scan(
		&patientVisit.PatientId,
		&patientVisit.HealthConditionId,
		&patientVisit.LayoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &patientVisit.Status)
	if err != nil {
		return nil, err
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

	return &patientVisit, err
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

func (d *DataService) SubmitPatientVisitWithId(patientVisitId int64) error {
	_, err := d.DB.Exec("update patient_visit set status='SUBMITTED', submitted_date=now() where id = ? and STATUS in ('OPEN', 'PHOTOS_REJECTED')", patientVisitId)
	return err
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

func (d *DataService) GetFollowUpTimeForPatientVisit(patientVisitId int64) (*common.FollowUp, error) {
	var followupTime time.Time
	var followupValue int64
	var followupUnit string

	err := d.DB.QueryRow(`select follow_up_date, follow_up_value, follow_up_unit 
							from patient_visit_follow_up where patient_visit_id = ?`, patientVisitId).Scan(&followupTime, &followupValue, &followupUnit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	followUp := &common.FollowUp{}
	followUp.PatientVisitId = patientVisitId
	followUp.FollowUpValue = followupValue
	followUp.FollowUpUnit = followupUnit
	followUp.FollowUpTime = followupTime
	return followUp, nil
}

func (d *DataService) GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) ([]*common.AnswerIntake, error) {
	rows, err := d.DB.Query(`select info_intake.id, info_intake.question_id, info_intake.potential_answer_id, info_intake.answer_text, l2.ltext, l1.ltext
					from info_intake inner join question on question.id = question_id 
					inner join potential_answer on potential_answer_id = potential_answer.id
					inner join localized_text as l1 on answer_localized_text_id = l1.app_text_id
					left outer join localized_text as l2 on answer_summary_text_id = l2.app_text_id
					where info_intake.status='ACTIVE' and question_tag = ? and role_id = ? and role = 'DOCTOR' and info_intake.patient_visit_id = ? and l1.language_id = ?`, questionTag, doctorId, patientVisitId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerIntakes := make([]*common.AnswerIntake, 0)
	for rows.Next() {
		answerIntake := new(common.AnswerIntake)
		var potentialAnswerId sql.NullInt64
		var answerText, potentialAnswer, answerSummary sql.NullString

		rows.Scan(
			&answerIntake.AnswerIntakeId, &answerIntake.QuestionId,
			&potentialAnswerId, &answerText, &answerSummary, &potentialAnswer)

		if potentialAnswer.Valid {
			answerIntake.PotentialAnswer = potentialAnswer.String
		}
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

		answerIntakes = append(answerIntakes, answerIntake)
	}

	return answerIntakes, nil
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

func (d *DataService) DeactivatePreviousDiagnosisForPatientVisit(PatientVisitId int64, DoctorId int64) error {
	_, err := d.DB.Exec(`update info_intake set status='INACTIVE' where patient_visit_id = ? and status = 'ACTIVE' and role = 'DOCTOR' and role_id = ?`, PatientVisitId, DoctorId)
	return err
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

func (d *DataService) GetDoctorAssignedToPatientVisit(PatientVisitId int64) (*common.Doctor, error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var doctorId, accountId int64

	err := d.DB.QueryRow(`select doctor.id,account_id, first_name, last_name, gender, dob, doctor.status from doctor 
		inner join patient_visit_care_provider_assignment on provider_id = doctor.id
		inner join provider_role on provider_role_id = provider_role.id
		where provider_tag = 'DOCTOR' and patient_visit_id = ? and patient_visit_care_provider_assignment.status = 'ACTIVE'`, PatientVisitId).Scan(&doctorId, &accountId, &firstName, &lastName, &gender, &dob, &status)
	if err != nil {
		return nil, err
	}
	doctor := &common.Doctor{
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
	return doctor, nil
}

func (d *DataService) GetAdvicePointsForPatientVisit(patientVisitId int64) ([]*common.DoctorInstructionItem, error) {
	rows, err := d.DB.Query(`select dr_advice_point_id,text from advice inner join dr_advice_point on dr_advice_point_id = dr_advice_point.id where patient_visit_id = ?  and advice.status = ?`, patientVisitId, status_active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	advicePoints := make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}

		advicePoint := &common.DoctorInstructionItem{
			Id:   id,
			Text: text,
		}
		advicePoints = append(advicePoints, advicePoint)
	}
	return advicePoints, nil
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

func (d *DataService) GetRegimenPlanForPatientVisit(patientVisitId int64) (*common.RegimenPlan, error) {
	var regimenPlan common.RegimenPlan
	regimenPlan.PatientVisitId = patientVisitId

	rows, err := d.DB.Query(`select regimen_type, dr_regimen_step.id, dr_regimen_step.text 
								from regimen inner join dr_regimen_step on dr_regimen_step_id = dr_regimen_step.id 
									where patient_visit_id = ? and regimen.status = 'ACTIVE' order by regimen.id`, patientVisitId)
	if err != nil {
		return nil, err
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

	return &regimenPlan, nil
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
		treatment.TreatmentPlanId = treatmentPlanId
		err = d.addTreatment(treatment, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

	}

	tx.Commit()
	return nil
}

func (d *DataService) addTreatment(treatment *common.Treatment, tx *sql.Tx) error {
	substitutionsAllowedBit := 0
	if treatment.SubstitutionsAllowed == true {
		substitutionsAllowedBit = 1
	}

	treatmentType := treatment_rx
	if treatment.OTC == true {
		treatmentType = treatment_otc
	}

	var drugNameId, drugRouteId, drugFormId int64
	var err error

	drugNameIdStr := "NULL"
	if treatment.DrugName != "" {
		drugNameId, err = d.getOrInsertNameInTable(tx, drug_name_table, treatment.DrugName)
		if err != nil {
			return err
		}
		if drugNameId != 0 {
			drugNameIdStr = strconv.FormatInt(drugNameId, 10)
		}

	}

	drugFormIdStr := "NULL"
	if treatment.DrugForm != "" {
		drugFormId, err = d.getOrInsertNameInTable(tx, drug_form_table, treatment.DrugForm)
		if err != nil {
			return err
		}

		if drugFormId != 0 {
			drugFormIdStr = strconv.FormatInt(drugFormId, 10)
		}
	}

	drugRouteIdStr := "NULL"
	if treatment.DrugRoute != "" {
		drugRouteId, err = d.getOrInsertNameInTable(tx, drug_route_table, treatment.DrugRoute)
		if err != nil {
			return err
		}

		if drugRouteId != 0 {
			drugRouteIdStr = strconv.FormatInt(drugRouteId, 10)
		}
	}
	// add treatment for patient
	var treatmentId int64
	if treatment.TreatmentPlanId != 0 {
		insertTreatmentStr := fmt.Sprintf(`insert into treatment (treatment_plan_id, drug_internal_name, drug_name_id, drug_route_id, drug_form_id, dosage_strength, type, dispense_value, dispense_unit_id, refills, substitutions_allowed, days_supply, patient_instructions, pharmacy_notes, status) 
									values (?,?,%s,%s,%s,?,?,?,?,?,?,?,?,?,?)`, drugNameIdStr, drugRouteIdStr, drugFormIdStr)
		res, err := tx.Exec(insertTreatmentStr, treatment.TreatmentPlanId, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, substitutionsAllowedBit, treatment.DaysSupply, treatment.PatientInstructions, treatment.PharmacyNotes, status_created)
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
		insertTreatmentStr := fmt.Sprintf(`insert into treatment (drug_internal_name,drug_name_id, drug_route_id, drug_form_id, dosage_strength, type, dispense_value, dispense_unit_id, refills, substitutions_allowed, days_supply, patient_instructions, pharmacy_notes, status) 
									values (?,%s,%s,%s,?,?,?,?,?,?,?,?,?,?)`, drugNameIdStr, drugRouteIdStr, drugFormIdStr)
		res, err := tx.Exec(insertTreatmentStr, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, substitutionsAllowedBit, treatment.DaysSupply, treatment.PatientInstructions, treatment.PharmacyNotes, status_created)
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

	// add drug db ids to the table
	for drugDbTag, drugDbId := range treatment.DrugDBIds {
		_, err := tx.Exec(`insert into drug_db_id (drug_db_id_tag, drug_db_id, treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, treatment.Id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func (d *DataService) GetTreatmentPlanForPatientVisit(patientVisitId int64) (*common.TreatmentPlan, error) {
	var treatmentPlan common.TreatmentPlan
	treatmentPlan.PatientVisitId = patientVisitId

	// get treatment plan information
	var status string
	var treatmentPlanId int64
	var creationDate time.Time
	err := d.DB.QueryRow(`select id, status, creation_date from treatment_plan where patient_visit_id = ?`, patientVisitId).Scan(&treatmentPlanId, &status, &creationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}

	treatmentPlan.Id = treatmentPlanId
	treatmentPlan.Status = status
	treatmentPlan.CreationDate = creationDate
	treatmentPlan.Treatments = make([]*common.Treatment, 0)
	rows, err := d.DB.Query(`select treatment.id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, 
			treatment.status, drug_name.name, drug_route.name, drug_form.name from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where patient_visit_id=? and treatment.status=? and localized_text.language_id = ?`, patientVisitId, status_created, EN_LANGUAGE_ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return &treatmentPlan, nil
		} else {
			return nil, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		treatment, err := d.getTreatmentFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}
		treatment.TreatmentPlanId = treatmentPlan.Id
		treatment.PatientVisitId = patientVisitId
		treatmentPlan.Treatments = append(treatmentPlan.Treatments, treatment)
	}

	return &treatmentPlan, nil
}

func (d *DataService) getTreatmentFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentId, dispenseValue, dispenseUnitId, refills, daysSupply int64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
	var substitutionsAllowed bool
	var creationDate time.Time
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId, &dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyNotes, &patientInstructions, &creationDate, &status, &drugName, &drugRoute, &drugForm)
	if err != nil {
		return nil, err
	}

	treatment := &common.Treatment{}
	treatment.Id = treatmentId
	treatment.DrugInternalName = drugInternalName
	treatment.DosageStrength = dosageStrength
	treatment.DispenseValue = dispenseValue
	treatment.DispenseUnitId = dispenseUnitId
	treatment.DispenseUnitDescription = dispenseUnitDescription
	treatment.NumberRefills = refills
	treatment.SubstitutionsAllowed = substitutionsAllowed
	treatment.DaysSupply = daysSupply
	treatment.DrugName = drugName.String
	treatment.DrugForm = drugForm.String
	treatment.DrugRoute = drugRoute.String

	if treatmentType == treatment_otc {
		treatment.OTC = true
	}

	if pharmacyNotes.Valid {
		treatment.PharmacyNotes = pharmacyNotes.String
	}
	treatment.PatientInstructions = patientInstructions
	treatment.CreationDate = creationDate
	treatment.Status = status

	// for each of the drugs, populate the drug db ids
	drugDbIds := make(map[string]string)
	drugRows, err := d.DB.Query(`select drug_db_id_tag, drug_db_id from drug_db_id where treatment_id = ? `, treatmentId)
	if err != nil {
		return nil, err
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
	instructionsRows, err := d.DB.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions 
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id 
													where treatment_instructions.status=? and treatment_id=?`, status_active, treatmentId)
	if err != nil {
		return nil, err
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
	return treatment, nil
}
