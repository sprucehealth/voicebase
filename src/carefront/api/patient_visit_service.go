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

func (d *DataService) GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error) {
	var patientVisitId int64
	err := d.DB.QueryRow(`select patient_visit_id from treatment_plan where id = ?`, treatmentPlanId).Scan(&patientVisitId)
	return patientVisitId, err
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

func (d *DataService) GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId int64) (int64, error) {
	var treatmentPlanId int64
	err := d.DB.QueryRow(`select id from treatment_plan where patient_visit_id = ? and status = ?`, patientVisitId, status_active).Scan(&treatmentPlanId)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return treatmentPlanId, err
}

func (d *DataService) StartNewTreatmentPlanForPatientVisit(patientId, patientVisitId, doctorId int64) (int64, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return 0, err
	}

	// when starting a new treatment plan, ensure to inactive any old treatment plans
	_, err = tx.Exec(`update treatment_plan set status=? where patient_visit_id = ? and status = ?`, status_inactive, patientVisitId, status_active)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// also ensure to update the status of the visit to reviewing
	_, err = tx.Exec(`update patient_visit set status=? where id=?`, CASE_STATUS_REVIEWING, patientVisitId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastId, err := tx.Exec(`insert into treatment_plan (patient_visit_id, doctor_id, status) values (?,?,?)`, patientVisitId, doctorId, status_active)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	treatmentPlanId, err := lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	err = tx.Commit()
	return treatmentPlanId, err
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
	return tx.Commit()
}

func (d *DataService) GetMessageForPatientVisitStatus(patientVisitId int64) (message string, err error) {
	err = d.DB.QueryRow(`select message from patient_visit_event where patient_visit_id = ? and status = ?`, patientVisitId, status_active).Scan(&message)
	if err != nil && err == sql.ErrNoRows {
		return "", nil
	}
	return
}

func (d *DataService) ClosePatientVisit(patientVisitId, treatmentPlanId int64, event, message string) error {
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

	_, err = tx.Exec(`update treatment_plan set sent_date=now() where id = ?`, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update patient_visit set status=?, closed_date=now() where id = ?`, event, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DataService) SubmitPatientVisitWithId(patientVisitId int64) error {
	_, err := d.DB.Exec("update patient_visit set status='SUBMITTED', submitted_date=now() where id = ? and STATUS in ('OPEN', 'PHOTOS_REJECTED')", patientVisitId)
	return err
}

func (d *DataService) UpdateFollowUpTimeForPatientVisit(treatmentPlanId, currentTimeSinceEpoch, doctorId, followUpValue int64, followUpUnit string) error {
	// check if a follow up time already exists that we can update
	var followupId int64
	err := d.DB.QueryRow(`select id from patient_visit_follow_up where treatment_plan_id = ?`, treatmentPlanId).Scan(&followupId)
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
		_, err = d.DB.Exec(`insert into patient_visit_follow_up (treatment_plan_id, doctor_id, follow_up_date, follow_up_value, follow_up_unit, status) 
				values (?,?,?,?,?, 'ADDED')`, treatmentPlanId, doctorId, followUpTime, followUpValue, followUpUnit)
		if err != nil {
			return err
		}
	} else {
		_, err = d.DB.Exec(`update patient_visit_follow_up set follow_up_date=?, follow_up_value=?, follow_up_unit=?, doctor_id=?, status='UPDATED' where treatment_plan_id = ?`, followUpTime, followUpValue, followUpUnit, doctorId, treatmentPlanId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataService) GetFollowUpTimeForPatientVisit(patientVisitId, treatmentPlanId int64) (*common.FollowUp, error) {
	var followupTime time.Time
	var followupValue int64
	var followupUnit string

	err := d.DB.QueryRow(`select follow_up_date, follow_up_value, follow_up_unit 
							from patient_visit_follow_up where (patient_visit_id = ? or treatment_plan_id = ?)`, patientVisitId, treatmentPlanId).Scan(&followupTime, &followupValue, &followupUnit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	followUp := &common.FollowUp{}
	followUp.TreatmentPlanId = treatmentPlanId
	followUp.FollowUpValue = followupValue
	followUp.FollowUpUnit = followupUnit
	followUp.FollowUpTime = followupTime
	return followUp, nil
}

func (d *DataService) GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, treatmentPlanId int64) ([]*common.AnswerIntake, error) {
	rows, err := d.DB.Query(`select info_intake.id, info_intake.question_id, info_intake.potential_answer_id, info_intake.answer_text, l2.ltext, l1.ltext
					from info_intake inner join question on question.id = question_id 
					inner join potential_answer on potential_answer_id = potential_answer.id
					inner join localized_text as l1 on answer_localized_text_id = l1.app_text_id
					left outer join localized_text as l2 on answer_summary_text_id = l2.app_text_id
					where info_intake.status='ACTIVE' and question_tag = ? and role_id = ? and role = 'DOCTOR' and info_intake.context_id = ? and l1.language_id = ?`, questionTag, doctorId, treatmentPlanId, EN_LANGUAGE_ID)
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
		answerIntake.ContextId = treatmentPlanId
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

func (d *DataService) AddDiagnosisSummaryForPatientVisit(summary string, treatmentPlanId, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// inactivate any previous summaries for this patient visit
	_, err = tx.Exec(`update diagnosis_summary set status=? where doctor_id = ? and treatment_plan_id = ? and status = ?`, status_inactive, doctorId, treatmentPlanId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into diagnosis_summary (summary, treatment_plan_id, doctor_id, status) values (?, ?, ?, ?)`, summary, treatmentPlanId, doctorId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetDiagnosisSummaryForPatientVisit(patientVisitId, treatmentPlanId int64) (summary string, err error) {
	err = d.DB.QueryRow(`select summary from diagnosis_summary where (patient_visit_id = ? or treatment_plan_id = ?) and status='ACTIVE'`, patientVisitId, treatmentPlanId).Scan(&summary)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
	}
	return
}

func (d *DataService) DeactivatePreviousDiagnosisForPatientVisit(treatmentPlanId int64, doctorId int64) error {
	_, err := d.DB.Exec(`update info_intake set status='INACTIVE' where context_id = ? and status = 'ACTIVE' and role = 'DOCTOR' and role_id = ?`, treatmentPlanId, doctorId)
	return err
}

func (d *DataService) RecordDoctorAssignmentToPatientVisit(patientVisitId, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// update any previous assignment to be inactive
	_, err = tx.Exec(`update patient_visit_care_provider_assignment set status=? where patient_visit_id=?`, status_inactive, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert an assignment into table
	_, err = tx.Exec(`insert into patient_visit_care_provider_assignment (provider_role_id, provider_id, patient_visit_id, status) 
							values ((select id from provider_role where provider_tag = 'DOCTOR'), ?, ?, ?)`, doctorId, patientVisitId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetDoctorAssignedToPatientVisit(patientVisitId int64) (*common.Doctor, error) {
	var firstName, lastName, status, gender string
	var dob mysql.NullTime
	var doctorId, accountId int64

	err := d.DB.QueryRow(`select doctor.id,account_id, first_name, last_name, gender, dob, doctor.status from doctor 
		inner join patient_visit_care_provider_assignment on provider_id = doctor.id
		inner join provider_role on provider_role_id = provider_role.id
		where provider_tag = 'DOCTOR' and patient_visit_id = ? and patient_visit_care_provider_assignment.status = 'ACTIVE'`, patientVisitId).Scan(&doctorId, &accountId, &firstName, &lastName, &gender, &dob, &status)
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

func (d *DataService) GetAdvicePointsForPatientVisit(patientVisitId, treatmentPlanId int64) ([]*common.DoctorInstructionItem, error) {
	rows, err := d.DB.Query(`select dr_advice_point_id,text from advice inner join dr_advice_point on dr_advice_point_id = dr_advice_point.id where (treatment_plan_id = ? or patient_visit_id = ?)  and advice.status = ?`, treatmentPlanId, patientVisitId, status_active)
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

func (d *DataService) CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error {
	// begin tx
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update advice set status=? where treatment_plan_id=?`, status_inactive, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`insert into advice (treatment_plan_id, dr_advice_point_id, status) values (?, ?, ?)`, treatmentPlanId, advicePoint.Id, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) CreateRegimenPlanForPatientVisit(regimenPlan *common.RegimenPlan) error {
	// begin tx
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// mark any previous regimen steps for this patient visit and regimen type as inactive
	_, err = tx.Exec(`update regimen set status=? where treatment_plan_id = ?`, status_inactive, regimenPlan.TreatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// create new regimen steps within each section
	for _, regimenSection := range regimenPlan.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			_, err = tx.Exec(`insert into regimen (treatment_plan_id, regimen_type, dr_regimen_step_id, status) values (?,?,?,?)`, regimenPlan.TreatmentPlanId, regimenSection.RegimenName, regimenStep.Id, status_active)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) GetRegimenPlanForPatientVisit(patientVisitId, treatmentPlanId int64) (*common.RegimenPlan, error) {
	var regimenPlan common.RegimenPlan
	regimenPlan.TreatmentPlanId = treatmentPlanId

	rows, err := d.DB.Query(`select regimen_type, dr_regimen_step.id, dr_regimen_step.text 
								from regimen inner join dr_regimen_step on dr_regimen_step_id = dr_regimen_step.id 
									where (treatment_plan_id = ? or patient_visit_id=?) and regimen.status = 'ACTIVE' order by regimen.id`, treatmentPlanId, patientVisitId)
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

func (d *DataService) AddTreatmentsForPatientVisit(treatments []*common.Treatment, doctorId, treatmentPlanId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("update treatment set status=? where treatment_plan_id = ?", status_inactive, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, treatment := range treatments {
		treatment.TreatmentPlanId = treatmentPlanId
		err = d.addTreatment(treatment, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

	}

	return tx.Commit()
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

func (d *DataService) GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, treatmentPlanId int64) ([]*common.Treatment, error) {

	// get treatment plan information
	treatments := make([]*common.Treatment, 0)
	rows, err := d.DB.Query(`select treatment.id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where (treatment_plan.patient_visit_id = ? or treatment_plan_id=?) and treatment.status=? and localized_text.language_id = ?`, patientVisitId, treatmentPlanId, status_created, EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		treatment, err := d.getTreatmentFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}
		treatment.TreatmentPlanId = treatmentPlanId
		treatments = append(treatments, treatment)
	}

	return treatments, nil
}

func (d *DataService) GetTreatmentBasedOnPrescriptionId(erxId int64) (*common.Treatment, error) {
	rows, err := d.DB.Query(`select treatment.id,treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id from treatment

				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where erx_id=? and localized_text.language_id = ?`, erxId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatments = append(treatments, treatment)
	}

	if len(treatments) == 0 {
		return nil, nil
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) MarkTreatmentsAsPrescriptionsSent(treatments []*common.Treatment, DoctorId, PatientVisitId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		if treatment.PrescriptionId != 0 {
			_, err = tx.Exec(`update treatment set erx_id = ?, erx_sent_date=now() where id = ? and treatment_plan_id = ?`, treatment.PrescriptionId, treatment.Id, treatment.TreatmentPlanId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (d *DataService) AddErxStatusEvent(treatments []*common.Treatment, statusEvent string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, status_inactive, treatment.Id, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into erx_status_events (treatment_id, erx_status, status) values (?,?,?)`, treatment.Id, statusEvent, status_active)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) AddErxErrorEventWithMessage(treatment *common.Treatment, statusEvent, errorDetails string, errorTimeStamp time.Time) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, status_inactive, treatment.Id, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into erx_status_events (treatment_id, erx_status, event_details, creation_date, status) values (?,?,?,?,?)`, treatment.Id, statusEvent, errorDetails, errorTimeStamp, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPrescriptionStatusEventsForPatient(patientId int64) ([]*PrescriptionStatus, error) {
	rows, err := d.DB.Query(`select erx_status_events.treatment_id, treatment.erx_id, erx_status_events.erx_status, erx_status_events.creation_date from treatment 
								inner join treatment_plan on treatment_plan_id = treatment_plan.id 
								inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id 
								left outer join erx_status_events on erx_status_events.treatment_id = treatment.id 
								inner join patient on patient.id = patient_visit.patient_id 
									where patient.erx_patient_id = ? and erx_status_events.status = ? order by erx_status_events.creation_date desc`, patientId, status_active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptionStatuses := make([]*PrescriptionStatus, 0)
	for rows.Next() {
		var treatmentId int64
		var prescriptionId sql.NullInt64
		var status string
		var creationDate time.Time
		err = rows.Scan(&treatmentId, &prescriptionId, &status, &creationDate)
		if err != nil {
			return nil, err
		}

		prescriptionStatus := &PrescriptionStatus{
			PrescriptionStatus: status,
			TreatmentId:        treatmentId,
			StatusTimeStamp:    creationDate,
		}

		if prescriptionId.Valid {
			prescriptionStatus.PrescriptionId = prescriptionId.Int64
		}

		prescriptionStatuses = append(prescriptionStatuses, prescriptionStatus)
	}

	return prescriptionStatuses, nil
}

func (d *DataService) getTreatmentFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentId, treatmentPlanId, dispenseValue, dispenseUnitId, refills, daysSupply, patientId, patientVisitId int64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
	var substitutionsAllowed bool
	var creationDate time.Time
	var erxSentDate mysql.NullTime
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentId, &treatmentPlanId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId, &dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &status, &drugName, &drugRoute, &drugForm, &patientId, &patientVisitId)
	if err != nil {
		return nil, err
	}

	treatment := &common.Treatment{
		Id:                      treatmentId,
		TreatmentPlanId:         treatmentPlanId,
		PatientId:               patientId,
		PatientVisitId:          patientVisitId,
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
	}

	if treatmentType == treatment_otc {
		treatment.OTC = true
	}

	if erxSentDate.Valid {
		treatment.ErxSentDate = &erxSentDate.Time
	}

	err = d.fillInDrugDBIdsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	err = d.fillInSupplementalInstructionsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	return treatment, nil
}

func (d *DataService) fillInDrugDBIdsForTreatment(treatment *common.Treatment) error {
	// for each of the drugs, populate the drug db ids
	drugDbIds := make(map[string]string)
	drugRows, err := d.DB.Query(`select drug_db_id_tag, drug_db_id from drug_db_id where treatment_id = ? `, treatment.Id)
	if err != nil {
		return err
	}
	defer drugRows.Close()

	for drugRows.Next() {
		var dbIdTag string
		var dbId int64
		drugRows.Scan(&dbIdTag, &dbId)
		drugDbIds[dbIdTag] = strconv.FormatInt(dbId, 10)
	}

	treatment.DrugDBIds = drugDbIds
	return nil
}

func (d *DataService) fillInSupplementalInstructionsForTreatment(treatment *common.Treatment) error {
	// get the supplemental instructions for this treatment
	instructionsRows, err := d.DB.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions 
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id 
													where treatment_instructions.status=? and treatment_id=?`, status_active, treatment.Id)
	if err != nil {
		return err
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
	return nil
}
