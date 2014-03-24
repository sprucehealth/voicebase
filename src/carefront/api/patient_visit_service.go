package api

import (
	"carefront/common"
	pharmacyService "carefront/libs/pharmacy"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
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
		PatientVisitId:    common.NewObjectId(patientVisitId),
		PatientId:         common.NewObjectId(patientId),
		HealthConditionId: common.NewObjectId(healthConditionId),
		Status:            status,
		LayoutVersionId:   common.NewObjectId(layoutVersionId),
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
		PatientVisitId:    common.NewObjectId(patientVisitId),
		PatientId:         common.NewObjectId(patientId),
		HealthConditionId: common.NewObjectId(healthConditionId),
		Status:            status,
		LayoutVersionId:   common.NewObjectId(layoutVersionId),
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
	patientVisit := common.PatientVisit{PatientVisitId: common.NewObjectId(patientVisitId)}
	var patientId, healthConditionId, layoutVersionId int64
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	err := d.DB.QueryRow(`select patient_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status from patient_visit where id = ?`, patientVisitId,
	).Scan(
		&patientId,
		&healthConditionId,
		&layoutVersionId, &creationDateBytes, &submittedDateBytes, &closedDateBytes, &patientVisit.Status)
	if err != nil {
		return nil, err
	}

	patientVisit.PatientId = common.NewObjectId(patientId)
	patientVisit.HealthConditionId = common.NewObjectId(healthConditionId)
	patientVisit.LayoutVersionId = common.NewObjectId(layoutVersionId)

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
	err := d.DB.QueryRow(`select id from treatment_plan where patient_visit_id = ? and status = ?`, patientVisitId, STATUS_ACTIVE).Scan(&treatmentPlanId)
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
	_, err = tx.Exec(`update treatment_plan set status=? where patient_visit_id = ? and status = ?`, STATUS_INACTIVE, patientVisitId, STATUS_ACTIVE)
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

	lastId, err := tx.Exec(`insert into treatment_plan (patient_visit_id, doctor_id, status) values (?,?,?)`, patientVisitId, doctorId, STATUS_ACTIVE)
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
		_, err = tx.Exec(`update patient_visit_event set status=? where patient_visit_id = ? and status=?`, STATUS_INACTIVE, patientVisitId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_visit_event (patient_visit_id, status, event, message) values (?,?,?,?)`, patientVisitId, STATUS_ACTIVE, event, message)
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
	err = d.DB.QueryRow(`select message from patient_visit_event where patient_visit_id = ? and status = ?`, patientVisitId, STATUS_ACTIVE).Scan(&message)
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
		_, err = tx.Exec(`update patient_visit_event set status=? where patient_visit_id = ? and status=?`, STATUS_INACTIVE, patientVisitId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		_, err = tx.Exec(`insert into patient_visit_event (patient_visit_id, status, event, message) values (?,?,?,?)`, patientVisitId, STATUS_ACTIVE, event, message)
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
	followUp.TreatmentPlanId = common.NewObjectId(treatmentPlanId)
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
		var answerIntakeId, questionId int64

		rows.Scan(
			&answerIntakeId, &questionId,
			&potentialAnswerId, &answerText, &answerSummary, &potentialAnswer)

		answerIntake.AnswerIntakeId = common.NewObjectId(answerIntakeId)
		answerIntake.QuestionId = common.NewObjectId(questionId)

		if potentialAnswer.Valid {
			answerIntake.PotentialAnswer = potentialAnswer.String
		}
		if answerText.Valid {
			answerIntake.AnswerText = answerText.String
		}
		answerIntake.ContextId = common.NewObjectId(treatmentPlanId)
		if potentialAnswerId.Valid {
			answerIntake.PotentialAnswerId = common.NewObjectId(potentialAnswerId.Int64)
		}

		if answerSummary.Valid {
			answerIntake.AnswerSummary = answerSummary.String
		}

		answerIntakes = append(answerIntakes, answerIntake)
	}

	return answerIntakes, rows.Err()
}

func (d *DataService) AddDiagnosisSummaryForPatientVisit(summary string, treatmentPlanId, doctorId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	// inactivate any previous summaries for this patient visit
	_, err = tx.Exec(`update diagnosis_summary set status=? where doctor_id = ? and treatment_plan_id = ? and status = ?`, STATUS_INACTIVE, doctorId, treatmentPlanId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into diagnosis_summary (summary, treatment_plan_id, doctor_id, status) values (?, ?, ?, ?)`, summary, treatmentPlanId, doctorId, STATUS_ACTIVE)
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
	_, err = tx.Exec(`update patient_visit_care_provider_assignment set status=? where patient_visit_id=?`, STATUS_INACTIVE, patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert an assignment into table
	_, err = tx.Exec(`insert into patient_visit_care_provider_assignment (provider_role_id, provider_id, patient_visit_id, status) 
							values ((select id from provider_role where provider_tag = 'DOCTOR'), ?, ?, ?)`, doctorId, patientVisitId, STATUS_ACTIVE)
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
		AccountId: common.NewObjectId(accountId),
	}
	if dob.Valid {
		doctor.Dob = dob.Time
	}
	doctor.DoctorId = common.NewObjectId(doctorId)
	return doctor, nil
}

func (d *DataService) GetAdvicePointsForPatientVisit(patientVisitId, treatmentPlanId int64) ([]*common.DoctorInstructionItem, error) {
	rows, err := d.DB.Query(`select dr_advice_point_id,text from advice inner join dr_advice_point on dr_advice_point_id = dr_advice_point.id where (treatment_plan_id = ? or patient_visit_id = ?)  and advice.status = ?`, treatmentPlanId, patientVisitId, STATUS_ACTIVE)
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
			Id:   common.NewObjectId(id),
			Text: text,
		}
		advicePoints = append(advicePoints, advicePoint)
	}
	return advicePoints, rows.Err()
}

func (d *DataService) CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error {
	// begin tx
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update advice set status=? where treatment_plan_id=?`, STATUS_INACTIVE, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`insert into advice (treatment_plan_id, dr_advice_point_id, status) values (?, ?, ?)`, treatmentPlanId, advicePoint.Id, STATUS_ACTIVE)
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
	_, err = tx.Exec(`update regimen set status=? where treatment_plan_id = ?`, STATUS_INACTIVE, regimenPlan.TreatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// create new regimen steps within each section
	for _, regimenSection := range regimenPlan.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			_, err = tx.Exec(`insert into regimen (treatment_plan_id, regimen_type, dr_regimen_step_id, status) values (?,?,?,?)`, regimenPlan.TreatmentPlanId, regimenSection.RegimenName, regimenStep.Id, STATUS_ACTIVE)
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
	regimenPlan.TreatmentPlanId = common.NewObjectId(treatmentPlanId)

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
			Id:   common.NewObjectId(regimenStepId),
			Text: regimenText,
		}

		regimenSteps := regimenSections[regimenType]
		if regimenSteps == nil {
			regimenSteps = make([]*common.DoctorInstructionItem, 0)
		}
		regimenSteps = append(regimenSteps, regimenStep)
		regimenSections[regimenType] = regimenSteps
	}
	if rows.Err() != nil {
		return nil, rows.Err()
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

	_, err = tx.Exec("update treatment set status=? where treatment_plan_id = ?", STATUS_INACTIVE, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, treatment := range treatments {
		treatment.TreatmentPlanId = common.NewObjectId(treatmentPlanId)
		err = d.addTreatment(treatment, with_link_to_treatment_plan, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

		if treatment.DoctorTreatmentTemplateId.Int64() != 0 {
			_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatment.Id, treatment.DoctorTreatmentTemplateId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	}

	return tx.Commit()
}

func (d *DataService) addTreatment(treatment *common.Treatment, withoutLinkToTreatmentPlan bool, tx *sql.Tx) error {
	treatmentType := treatment_rx
	if treatment.OTC {
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
	if treatment.TreatmentPlanId.Int64() != 0 && !withoutLinkToTreatmentPlan {
		insertTreatmentStr := fmt.Sprintf(`insert into treatment (treatment_plan_id, drug_internal_name, drug_name_id, drug_route_id, drug_form_id, dosage_strength, type, dispense_value, dispense_unit_id, refills, substitutions_allowed, days_supply, patient_instructions, pharmacy_notes, status) 
									values (?,?,%s,%s,%s,?,?,?,?,?,?,?,?,?,?)`, drugNameIdStr, drugRouteIdStr, drugFormIdStr)
		res, err := tx.Exec(insertTreatmentStr, treatment.TreatmentPlanId, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, treatment.SubstitutionsAllowed, treatment.DaysSupply, treatment.PatientInstructions, treatment.PharmacyNotes, STATUS_CREATED)
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
		res, err := tx.Exec(insertTreatmentStr, treatment.DrugInternalName, treatment.DosageStrength, treatmentType, treatment.DispenseValue, treatment.DispenseUnitId, treatment.NumberRefills, treatment.SubstitutionsAllowed, treatment.DaysSupply, treatment.PatientInstructions, treatment.PharmacyNotes, STATUS_CREATED)
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
	treatment.Id = common.NewObjectId(treatmentId)

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
	rows, err := d.DB.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.erx_last_filled_date, treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where (treatment_plan.patient_visit_id = ? or treatment_plan_id=?) and treatment.status=? and localized_text.language_id = ?`, patientVisitId, treatmentPlanId, STATUS_CREATED, EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatmentIds := make([]int64, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatment.TreatmentPlanId = common.NewObjectId(treatmentPlanId)
		treatments = append(treatments, treatment)
		treatmentIds = append(treatmentIds, treatment.Id.Int64())
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return treatments, nil
	}

	favoriteRows, err := d.DB.Query(fmt.Sprintf(`select dr_treatment_template_id , treatment_dr_template_selection.treatment_id from treatment_dr_template_selection 
													inner join dr_treatment_template on dr_treatment_template.id = dr_treatment_template_id
														where treatment_dr_template_selection.treatment_id in (%s) and dr_treatment_template.status = ?`, enumerateItemsIntoString(treatmentIds)), STATUS_ACTIVE)
	treatmentIdToFavoriteIdMapping := make(map[int64]int64)
	if err != nil {
		return nil, err
	}
	defer favoriteRows.Close()

	for favoriteRows.Next() {
		var drFavoriteTreatmentId, treatmentId int64
		err = favoriteRows.Scan(&drFavoriteTreatmentId, &treatmentId)
		if err != nil {
			return nil, err
		}
		treatmentIdToFavoriteIdMapping[treatmentId] = drFavoriteTreatmentId
	}

	// assign the treatments the doctor favorite id if one exists
	for _, treatment := range treatments {
		if treatmentIdToFavoriteIdMapping[treatment.Id.Int64()] != 0 {
			treatment.DoctorTreatmentTemplateId = common.NewObjectId(treatmentIdToFavoriteIdMapping[treatment.Id.Int64()])
		}
	}

	return treatments, nil
}

func (d *DataService) GetTreatmentsForPatient(patientId int64) ([]*common.Treatment, error) {
	rows, err := d.DB.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.erx_last_filled_date, treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where patient_visit.patient_id = ? and treatment.status=? and localized_text.language_id = ?`, patientId, STATUS_CREATED, EN_LANGUAGE_ID)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// get treatment plan information
	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}
		treatments = append(treatments, treatment)
	}

	return treatments, rows.Err()
}

func (d *DataService) GetTreatmentBasedOnPrescriptionId(erxId int64) (*common.Treatment, error) {
	rows, err := d.DB.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.erx_last_filled_date, treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment

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
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatments = append(treatments, treatment)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return nil, nil
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) GetTreatmentFromId(treatmentId int64) (*common.Treatment, error) {
	rows, err := d.DB.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.erx_last_filled_date, treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment

				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment.id=? and localized_text.language_id = ?`, treatmentId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		treatment, err := d.getTreatmentAndMetadataFromCurrentRow(rows)
		if err != nil {
			return nil, err
		}

		treatments = append(treatments, treatment)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return nil, nil
	}

	if len(treatments) > 1 {
		return nil, fmt.Errorf("Expected just 1 treatment to be returned based on the prescription id, instead got %d", len(treatments))
	}

	return treatments[0], nil
}

func (d *DataService) MarkTreatmentsAsPrescriptionsSent(treatments []*common.Treatment, pharmacySentTo *pharmacyService.PharmacyData, doctorId, patientVisitId int64) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		if treatment.ERx.PrescriptionId.Int64() != 0 {
			_, err = tx.Exec(`update treatment set erx_id = ?, pharmacy_id = ?, erx_sent_date=now() where id = ? and treatment_plan_id = ?`, treatment.ERx.PrescriptionId, pharmacySentTo.LocalId, treatment.Id, treatment.TreatmentPlanId)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (d *DataService) AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {

		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, STATUS_INACTIVE, treatment.Id, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		columnsAndData := make(map[string]interface{}, 0)
		columnsAndData["treatment_id"] = treatment.Id
		columnsAndData["erx_status"] = prescriptionStatus.Status
		columnsAndData["status"] = STATUS_ACTIVE
		if !prescriptionStatus.ReportedTimestamp.IsZero() {
			columnsAndData["reported_timestamp"] = prescriptionStatus.ReportedTimestamp
		}
		if prescriptionStatus.StatusDetails != "" {
			columnsAndData["event_details"] = prescriptionStatus.StatusDetails
		}

		keys, values := getKeysAndValuesFromMap(columnsAndData)

		_, err = tx.Exec(fmt.Sprintf(`insert into erx_status_events (%s) values (%s)`, strings.Join(keys, ","), nReplacements(len(values))), values...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()

}

func (d *DataService) GetPrescriptionStatusEventsForPatient(patientId int64) ([]common.StatusEvent, error) {
	rows, err := d.DB.Query(`select erx_status_events.treatment_id, treatment.erx_id, erx_status_events.erx_status, erx_status_events.creation_date from treatment 
								inner join treatment_plan on treatment_plan_id = treatment_plan.id 
								inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id 
								left outer join erx_status_events on erx_status_events.treatment_id = treatment.id 
								inner join patient on patient.id = patient_visit.patient_id 
									where patient.erx_patient_id = ? and erx_status_events.status = ? order by erx_status_events.creation_date desc`, patientId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptionStatuses := make([]common.StatusEvent, 0)
	for rows.Next() {
		var treatmentId int64
		var prescriptionId sql.NullInt64
		var status string
		var creationDate time.Time
		err = rows.Scan(&treatmentId, &prescriptionId, &status, &creationDate)
		if err != nil {
			return nil, err
		}

		prescriptionStatus := common.StatusEvent{
			Status:          status,
			ItemId:          treatmentId,
			StatusTimestamp: creationDate,
		}

		if prescriptionId.Valid {
			prescriptionStatus.PrescriptionId = prescriptionId.Int64
		}

		prescriptionStatuses = append(prescriptionStatuses, prescriptionStatus)
	}

	return prescriptionStatuses, rows.Err()
}

func (d *DataService) GetPrescriptionStatusEventsForTreatment(treatmentId int64) ([]common.StatusEvent, error) {
	rows, err := d.DB.Query(`select erx_status_events.treatment_id, erx_status_events.erx_status, erx_status_events.event_details, erx_status_events.creation_date
									  from erx_status_events where treatment_id = ? order by erx_status_events.creation_date desc`, treatmentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prescriptionStatuses := make([]common.StatusEvent, 0)
	for rows.Next() {
		var statusDetails sql.NullString
		var prescriptionStatus common.StatusEvent
		err = rows.Scan(&prescriptionStatus.ItemId, &prescriptionStatus.Status, &statusDetails, &prescriptionStatus.StatusTimestamp)

		if err != nil {
			return nil, err
		}
		prescriptionStatus.StatusDetails = statusDetails.String

		prescriptionStatuses = append(prescriptionStatuses, prescriptionStatus)
	}

	return prescriptionStatuses, rows.Err()
}

func (d *DataService) UpdateDateInfoForTreatmentId(treatmentId int64, erxSentDate time.Time) error {
	_, err := d.DB.Exec(`update treatment set erx_sent_date = ? where treatment_id = ?`, erxSentDate, treatmentId)
	return err
}

func (d *DataService) getTreatmentAndMetadataFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentId, treatmentPlanId, dispenseValue, dispenseUnitId, refills, daysSupply, patientId, patientVisitId, prescriberId int64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
	var prescriptionId, pharmacyId sql.NullInt64
	var substitutionsAllowed bool
	var creationDate time.Time
	var erxSentDate, erxLastFilledDate mysql.NullTime
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentId, &prescriptionId, &treatmentPlanId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId,
		&dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyId,
		&pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &erxLastFilledDate, &status, &drugName, &drugRoute, &drugForm, &patientId, &patientVisitId, &prescriberId)
	if err != nil {
		return nil, err
	}

	treatment := &common.Treatment{
		Id:                      common.NewObjectId(treatmentId),
		TreatmentPlanId:         common.NewObjectId(treatmentPlanId),
		PatientId:               patientId,
		PatientVisitId:          common.NewObjectId(patientVisitId),
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
		DoctorId:                prescriberId,
		ERx: &common.ERxData{
			ErxLastDateFilled: &erxLastFilledDate.Time,
		},
	}

	if pharmacyId.Valid {
		treatment.ERx.PharmacyLocalId = common.NewObjectId(pharmacyId.Int64)
	}

	if prescriptionId.Valid {
		treatment.ERx.PrescriptionId = common.NewObjectId(prescriptionId.Int64)
	}

	if treatmentType == treatment_otc {
		treatment.OTC = true
	}

	if erxSentDate.Valid {
		treatment.ERx.ErxSentDate = &erxSentDate.Time
	}

	err = d.fillInDrugDBIdsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	err = d.fillInSupplementalInstructionsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	treatment.ERx.RxHistory, err = d.GetPrescriptionStatusEventsForTreatment(treatment.Id.Int64())
	if err != nil {
		return nil, err
	}

	treatment.ERx.Pharmacy, err = d.GetPharmacyFromId(treatment.ERx.PharmacyLocalId.Int64())
	if err != nil {
		return nil, err
	}

	treatment.Doctor, err = d.GetDoctorFromId(treatment.DoctorId)
	if err != nil {
		return nil, err
	}

	treatment.Patient, err = d.GetPatientFromId(treatment.PatientId)
	if err != nil {
		return nil, err
	}

	treatment.Patient, err = d.GetPatientFromId(treatment.PatientId)
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
													where treatment_instructions.status=? and treatment_id=?`, STATUS_ACTIVE, treatment.Id)
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
			Id:       common.NewObjectId(instructionId),
			Text:     text,
			Selected: true,
		}
		drugInstructions = append(drugInstructions, drugInstruction)
	}
	treatment.SupplementalInstructions = drugInstructions
	return nil
}
