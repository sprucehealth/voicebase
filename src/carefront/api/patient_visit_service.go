package api

import (
	"carefront/app_url"
	"carefront/common"
	"carefront/encoding"
	pharmacyService "carefront/libs/pharmacy"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func (d *DataService) GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error) {
	var patientVisitId int64
	err := d.db.QueryRow("select id from patient_visit where patient_id = ? and health_condition_id = ? and status='OPEN'", patientId, healthConditionId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, err
}

func (d *DataService) GetLastCreatedPatientVisitIdForPatient(patientId int64) (int64, error) {
	var patientVisitId int64
	err := d.db.QueryRow(`select id from patient_visit where patient_id = ? and creation_date is not null order by creation_date desc limit 1`, patientId).Scan(&patientVisitId)
	if err != nil && err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, nil
}

func (d *DataService) GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error) {
	var patientId int64
	err := d.db.QueryRow("select patient_id from patient_visit where id = ?", patientVisitId).Scan(&patientId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientId, err
}

// Adding this only to link the patient and the doctor app so as to show the doctor
// the patient visit review of the latest submitted patient visit
func (d *DataService) GetLatestSubmittedPatientVisit() (*common.PatientVisit, error) {
	var patientId, healthConditionId, layoutVersionId, patientVisitId encoding.ObjectId
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string

	row := d.db.QueryRow(`select id,patient_id, health_condition_id, layout_version_id, 
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
	err := d.db.QueryRow(`select patient_visit_id from treatment_plan where id = ?`, treatmentPlanId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}
	return patientVisitId, err
}

func (d *DataService) GetLatestClosedPatientVisitForPatient(patientId int64) (*common.PatientVisit, error) {
	var healthConditionId, layoutVersionId, patientVisitId encoding.ObjectId
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	var status string

	row := d.db.QueryRow(`select id, health_condition_id, layout_version_id,
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
		PatientId:         encoding.NewObjectId(patientId),
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
	patientVisit := common.PatientVisit{PatientVisitId: encoding.NewObjectId(patientVisitId)}
	var creationDateBytes, submittedDateBytes, closedDateBytes mysql.NullTime
	err := d.db.QueryRow(`select patient_id, health_condition_id, layout_version_id, 
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
	res, err := d.db.Exec(`insert into patient_visit (patient_id, health_condition_id, layout_version_id, status) 
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

func (d *DataService) GetAbbreviatedTreatmentPlanForPatientVisit(doctorId, patientVisitId int64) (*common.DoctorTreatmentPlan, error) {
	drTreatmentPlan := common.DoctorTreatmentPlan{
		PatientVisitId: encoding.NewObjectId(patientVisitId),
	}
	err := d.db.QueryRow(`select id from treatment_plan where patient_visit_id = ? and status =?`, patientVisitId, STATUS_ACTIVE).Scan(&drTreatmentPlan.Id)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	// return the favorite treatment plan info as well if it exists
	err = d.db.QueryRow(`select dr_favorite_treatment_plan_id, name from treatment_plan_favorite_mapping 
							inner join dr_favorite_treatment_plan on dr_favorite_treatment_plan.id = dr_favorite_treatment_plan_id
								where treatment_plan_id = ?`, drTreatmentPlan.Id.Int64()).Scan(
		&drTreatmentPlan.DoctorFavoriteTreatmentPlanId,
		&drTreatmentPlan.DoctorFavoriteTreatmentPlanName)
	if err == sql.ErrNoRows {
		return &drTreatmentPlan, nil
	} else if err != nil {
		return nil, err
	}

	return &drTreatmentPlan, nil
}

func (d *DataService) GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId int64) (int64, error) {
	var treatmentPlanId int64
	err := d.db.QueryRow(`select id from treatment_plan where patient_visit_id = ? and status = ?`, patientVisitId, STATUS_ACTIVE).Scan(&treatmentPlanId)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return treatmentPlanId, err
}

func (d *DataService) StartNewTreatmentPlanForPatientVisit(patientId, patientVisitId, doctorId, favoriteTreatmentPlanId int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	// when starting a new treatment plan, ensure to delete any old treatment plan
	// this will probably have to be handled more gracefully when we have versioning of treatment plans
	_, err = tx.Exec(`delete from treatment_plan where patient_visit_id = ? and status = ?`, patientVisitId, STATUS_ACTIVE)
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

	// include the mapping between the favorite treatment plan and the treatment plan if non-zero
	if favoriteTreatmentPlanId != 0 {
		_, err := tx.Exec(`insert into treatment_plan_favorite_mapping (treatment_plan_id, dr_favorite_treatment_plan_id) values (?,?)`, treatmentPlanId, favoriteTreatmentPlanId)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	err = tx.Commit()
	return treatmentPlanId, err
}

func (d *DataService) UpdatePatientVisitStatus(patientVisitId int64, message, event string) error {
	tx, err := d.db.Begin()
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
	err = d.db.QueryRow(`select message from patient_visit_event where patient_visit_id = ? and status = ?`, patientVisitId, STATUS_ACTIVE).Scan(&message)
	if err != nil && err == sql.ErrNoRows {
		return "", nil
	}
	return
}

func (d *DataService) ClosePatientVisit(patientVisitId, treatmentPlanId int64, event, message string) error {
	tx, err := d.db.Begin()
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
	_, err := d.db.Exec("update patient_visit set status='SUBMITTED', submitted_date=now() where id = ? and STATUS in ('OPEN', 'PHOTOS_REJECTED')", patientVisitId)
	return err
}

func (d *DataService) UpdateFollowUpTimeForPatientVisit(treatmentPlanId, currentTimeSinceEpoch, doctorId, followUpValue int64, followUpUnit string) error {
	// check if a follow up time already exists that we can update
	var followupId int64
	err := d.db.QueryRow(`select id from patient_visit_follow_up where treatment_plan_id = ?`, treatmentPlanId).Scan(&followupId)
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
		_, err = d.db.Exec(`insert into patient_visit_follow_up (treatment_plan_id, doctor_id, follow_up_date, follow_up_value, follow_up_unit, status) 
				values (?,?,?,?,?, 'ADDED')`, treatmentPlanId, doctorId, followUpTime, followUpValue, followUpUnit)
		if err != nil {
			return err
		}
	} else {
		_, err = d.db.Exec(`update patient_visit_follow_up set follow_up_date=?, follow_up_value=?, follow_up_unit=?, doctor_id=?, status='UPDATED' where treatment_plan_id = ?`, followUpTime, followUpValue, followUpUnit, doctorId, treatmentPlanId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataService) GetFollowUpTimeForTreatmentPlan(treatmentPlanId int64) (*common.FollowUp, error) {
	var followupTime time.Time
	var followupValue int64
	var followupUnit string

	err := d.db.QueryRow(`select follow_up_date, follow_up_value, follow_up_unit 
							from patient_visit_follow_up where treatment_plan_id = ?`, treatmentPlanId).Scan(&followupTime, &followupValue, &followupUnit)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	followUp := &common.FollowUp{}
	followUp.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)
	followUp.FollowUpValue = followupValue
	followUp.FollowUpUnit = followupUnit
	followUp.FollowUpTime = followupTime
	return followUp, nil
}

func (d *DataService) GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) ([]*common.AnswerIntake, error) {
	rows, err := d.db.Query(`select info_intake.id, info_intake.question_id, info_intake.potential_answer_id, info_intake.answer_text, l2.ltext, l1.ltext
					from info_intake inner join question on question.id = question_id 
					inner join potential_answer on potential_answer_id = potential_answer.id
					inner join localized_text as l1 on answer_localized_text_id = l1.app_text_id
					left outer join localized_text as l2 on answer_summary_text_id = l2.app_text_id
					where info_intake.status='ACTIVE' and question_tag = ? and role_id = ? and role = 'DOCTOR' and info_intake.context_id = ? and l1.language_id = ?`, questionTag, doctorId, patientVisitId, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	answerIntakes := make([]*common.AnswerIntake, 0)
	for rows.Next() {
		answerIntake := new(common.AnswerIntake)
		var answerText, potentialAnswer, answerSummary sql.NullString

		err := rows.Scan(
			&answerIntake.AnswerIntakeId, &answerIntake.QuestionId,
			&answerIntake.PotentialAnswerId, &answerText, &answerSummary, &potentialAnswer)
		if err != nil {
			return nil, err
		}

		if potentialAnswer.Valid {
			answerIntake.PotentialAnswer = potentialAnswer.String
		}
		if answerText.Valid {
			answerIntake.AnswerText = answerText.String
		}
		answerIntake.ContextId = encoding.NewObjectId(patientVisitId)

		if answerSummary.Valid {
			answerIntake.AnswerSummary = answerSummary.String
		}

		answerIntakes = append(answerIntakes, answerIntake)
	}

	return answerIntakes, rows.Err()
}

func (d *DataService) AddDiagnosisSummaryForTreatmentPlan(summary string, treatmentPlanId, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// inactivate any previous summaries for this patient visit
	_, err = tx.Exec(`delete from diagnosis_summary where doctor_id = ? and treatment_plan_id = ? and status = ?`, doctorId, treatmentPlanId, STATUS_ACTIVE)
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

func (d *DataService) GetDiagnosisSummaryForTreatmentPlan(treatmentPlanId int64) (*common.DiagnosisSummary, error) {
	var diagnosisSummary common.DiagnosisSummary
	err := d.db.QueryRow(`select summary, updated_by_doctor from diagnosis_summary where treatment_plan_id = ? and status='ACTIVE'`, treatmentPlanId).Scan(&diagnosisSummary.Summary, &diagnosisSummary.UpdatedByDoctor)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}

	return &diagnosisSummary, nil
}

func (d *DataService) AddOrUpdateDiagnosisSummaryForTreatmentPlan(summary string, treatmentPlanId, doctorId int64, isUpdatedByDoctor bool) error {
	_, err := d.db.Exec(`replace into diagnosis_summary (summary, treatment_plan_id, doctor_id, updated_by_doctor, status) values (?,?,?,?,?)`, summary, treatmentPlanId, doctorId, isUpdatedByDoctor, STATUS_ACTIVE)
	return err
}

func (d *DataService) DeactivatePreviousDiagnosisForPatientVisit(treatmentPlanId int64, doctorId int64) error {
	_, err := d.db.Exec(`update info_intake set status='INACTIVE' where context_id = ? and status = 'ACTIVE' and role = 'DOCTOR' and role_id = ?`, treatmentPlanId, doctorId)
	return err
}

func (d *DataService) RecordDoctorAssignmentToPatientVisit(patientVisitId, doctorId int64) error {
	tx, err := d.db.Begin()
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
	_, err = tx.Exec(`insert into patient_visit_care_provider_assignment (role_type_id, provider_id, patient_visit_id, status) 
							values (?, ?, ?, ?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientVisitId, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetDoctorAssignedToPatientVisit(patientVisitId int64) (*common.Doctor, error) {
	var firstName, lastName, status, gender string
	var doctorId, accountId encoding.ObjectId
	var dobYear, dobMonth, dobDay int

	err := d.db.QueryRow(`select doctor.id,account_id, first_name, last_name, gender, dob_year, dob_month, dob_day, doctor.status from doctor 
		inner join patient_visit_care_provider_assignment on provider_id = doctor.id
		where role_type_id = ? and patient_visit_id = ? and patient_visit_care_provider_assignment.status = 'ACTIVE'`, d.roleTypeMapping[DOCTOR_ROLE], patientVisitId).Scan(&doctorId, &accountId, &firstName, &lastName, &gender, &dobYear, &dobMonth, &dobDay, &status)
	if err != nil {
		return nil, err
	}
	doctor := &common.Doctor{
		DoctorId:  doctorId,
		FirstName: firstName,
		LastName:  lastName,
		Status:    status,
		Gender:    gender,
		Dob:       encoding.Dob{Year: dobYear, Month: dobMonth, Day: dobDay},
		AccountId: accountId,
	}

	doctor.LargeThumbnailUrl = app_url.GetLargeThumbnail(DOCTOR_ROLE, doctor.DoctorId.Int64())
	doctor.SmallThumbnailUrl = app_url.GetSmallThumbnail(DOCTOR_ROLE, doctor.DoctorId.Int64())

	return doctor, nil
}

func (d *DataService) GetAdvicePointsForTreatmentPlan(treatmentPlanId int64) ([]*common.DoctorInstructionItem, error) {
	rows, err := d.db.Query(`select id, dr_advice_point_id, advice.text from advice 
			where treatment_plan_id = ?  and advice.status = ?`, treatmentPlanId, STATUS_ACTIVE)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getAdvicePointsFromRows(rows)
}

func (d *DataService) CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error {
	// begin tx
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`delete from advice where treatment_plan_id=?`, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, advicePoint := range advicePoints {
		_, err = tx.Exec(`insert into advice (treatment_plan_id, dr_advice_point_id, text, status) values (?, ?, ?, ?)`, treatmentPlanId, advicePoint.ParentId.Int64Ptr(), advicePoint.Text, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (d *DataService) CreateRegimenPlanForPatientVisit(regimenPlan *common.RegimenPlan) error {
	// begin tx
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// delete any previous steps given that we have new ones coming in
	_, err = tx.Exec(`delete from regimen where treatment_plan_id = ?`, regimenPlan.TreatmentPlanId.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	// create new regimen steps within each section
	for _, regimenSection := range regimenPlan.RegimenSections {
		for _, regimenStep := range regimenSection.RegimenSteps {
			_, err = tx.Exec(`insert into regimen (treatment_plan_id, regimen_type, dr_regimen_step_id, text, status) values (?,?,?,?,?)`, regimenPlan.TreatmentPlanId.Int64(), regimenSection.RegimenName, regimenStep.ParentId.Int64Ptr(), regimenStep.Text, STATUS_ACTIVE)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d *DataService) GetRegimenPlanForTreatmentPlan(treatmentPlanId int64) (*common.RegimenPlan, error) {

	rows, err := d.db.Query(`select id, regimen_type, dr_regimen_step_id, regimen.text 
								from regimen where treatment_plan_id = ? and regimen.status = 'ACTIVE' order by regimen.id`, treatmentPlanId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regimenPlan, err := getRegimenPlanFromRows(rows)
	if err != nil {
		return nil, err
	}
	regimenPlan.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)

	return regimenPlan, nil
}

func (d *DataService) AddTreatmentsForPatientVisit(treatments []*common.Treatment, doctorId, treatmentPlanId, patientId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("update treatment set status=? where treatment_plan_id = ?", STATUS_INACTIVE, treatmentPlanId)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, treatment := range treatments {
		treatment.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)
		err = d.addTreatment(treatmentForPatientType, treatment, nil, tx)
		if err != nil {
			tx.Rollback()
			return err
		}

		if treatment.DoctorTreatmentTemplateId.Int64() != 0 {
			_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatment.Id.Int64(), treatment.DoctorTreatmentTemplateId.Int64())
			if err != nil {
				tx.Rollback()
				return err
			}
		}

	}

	return tx.Commit()
}

func (d *DataService) GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId int64) ([]*common.Treatment, error) {

	// get treatment plan information
	treatments := make([]*common.Treatment, 0)
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date, 
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment_plan_id=? and treatment.status=? and localized_text.language_id = ?`, treatmentPlanId, STATUS_CREATED, EN_LANGUAGE_ID)

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

		treatment.TreatmentPlanId = encoding.NewObjectId(treatmentPlanId)
		treatments = append(treatments, treatment)
		treatmentIds = append(treatmentIds, treatment.Id.Int64())
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(treatments) == 0 {
		return treatments, nil
	}

	favoriteRows, err := d.db.Query(fmt.Sprintf(`select dr_treatment_template_id , treatment_dr_template_selection.treatment_id from treatment_dr_template_selection 
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
			treatment.DoctorTreatmentTemplateId = encoding.NewObjectId(treatmentIdToFavoriteIdMapping[treatment.Id.Int64()])
		}
	}

	return treatments, nil
}

func (d *DataService) GetTreatmentsForPatient(patientId int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where patient_id = ? and treatment.status=? and localized_text.language_id = ?`, patientId, STATUS_CREATED, EN_LANGUAGE_ID)

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
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
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
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id, treatment_plan.doctor_id from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
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

func (d *DataService) UpdateTreatmentWithPharmacyAndErxId(treatments []*common.Treatment, pharmacySentTo *pharmacyService.PharmacyData, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {
		if treatment.ERx != nil && treatment.ERx.PrescriptionId.Int64() != 0 {
			_, err = tx.Exec(`update treatment set erx_id = ?, pharmacy_id = ?, erx_sent_date=now() where id = ?`, treatment.ERx.PrescriptionId.Int64(), pharmacySentTo.LocalId, treatment.Id.Int64())
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

func (d *DataService) AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, treatment := range treatments {

		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, STATUS_INACTIVE, treatment.Id.Int64(), STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		columnsAndData := make(map[string]interface{}, 0)
		columnsAndData["treatment_id"] = treatment.Id.Int64()
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
	rows, err := d.db.Query(`select erx_status_events.treatment_id, treatment.erx_id, erx_status_events.erx_status, erx_status_events.creation_date from treatment 
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
	rows, err := d.db.Query(`select erx_status_events.treatment_id, erx_status_events.erx_status, erx_status_events.event_details, erx_status_events.creation_date
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
	_, err := d.db.Exec(`update treatment set erx_sent_date = ? where treatment_id = ?`, erxSentDate, treatmentId)
	return err
}

func (d *DataService) getTreatmentAndMetadataFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentId, treatmentPlanId, dispenseUnitId, patientId, prescriberId, patientVisitId, prescriptionId, pharmacyId encoding.ObjectId
	var dispenseValue encoding.HighPrecisionFloat64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
	var substitutionsAllowed bool
	var refills, daysSupply encoding.NullInt64
	var creationDate time.Time
	var erxSentDate mysql.NullTime
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentId, &prescriptionId, &treatmentPlanId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId,
		&dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyId,
		&pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &status, &drugName, &drugRoute, &drugForm, &patientId, &patientVisitId, &prescriberId)
	if err != nil {
		return nil, err
	}

	treatment := &common.Treatment{
		Id:                      treatmentId,
		PatientId:               patientId,
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
		DoctorId:                prescriberId,
		TreatmentPlanId:         treatmentPlanId,
		PatientVisitId:          patientVisitId,
	}
	if treatmentType == treatmentOTC {
		treatment.OTC = true
	}

	if pharmacyId.IsValid || prescriptionId.IsValid || erxSentDate.Valid {
		treatment.ERx = &common.ERxData{}
		treatment.ERx.PharmacyLocalId = pharmacyId
		treatment.ERx.PrescriptionId = prescriptionId
	}

	if erxSentDate.Valid {
		treatment.ERx.ErxSentDate = &erxSentDate.Time
	}

	err = d.fillInDrugDBIdsForTreatment(treatment, treatment.Id.Int64(), possibleTreatmentTables[treatmentForPatientType])
	if err != nil {
		return nil, err
	}

	err = d.fillInSupplementalInstructionsForTreatment(treatment)
	if err != nil {
		return nil, err
	}

	// if its null that means that there isn't any erx related information
	if treatment.ERx != nil {
		treatment.ERx.RxHistory, err = d.GetPrescriptionStatusEventsForTreatment(treatment.Id.Int64())
		if err != nil {
			return nil, err
		}

		treatment.ERx.Pharmacy, err = d.GetPharmacyFromId(treatment.ERx.PharmacyLocalId.Int64())
		if err != nil {
			return nil, err
		}

	}

	treatment.Doctor, err = d.GetDoctorFromId(treatment.DoctorId.Int64())
	if err != nil {
		return nil, err
	}

	treatment.Patient, err = d.GetPatientFromId(treatment.PatientId.Int64())
	if err != nil {
		return nil, err
	}
	return treatment, nil
}

func (d *DataService) fillInDrugDBIdsForTreatment(treatment *common.Treatment, id int64, tableName string) error {
	// for each of the drugs, populate the drug db ids
	drugDbIds := make(map[string]string)
	drugRows, err := d.db.Query(fmt.Sprintf(`select drug_db_id_tag, drug_db_id from %s_drug_db_id where %s_id = ? `, tableName, tableName), id)
	if err != nil {
		return err
	}
	defer drugRows.Close()

	for drugRows.Next() {
		var dbIdTag string
		var dbId string
		if err := drugRows.Scan(&dbIdTag, &dbId); err != nil {
			return err
		}
		drugDbIds[dbIdTag] = dbId
	}

	treatment.DrugDBIds = drugDbIds
	return nil
}

func (d *DataService) fillInSupplementalInstructionsForTreatment(treatment *common.Treatment) error {
	// get the supplemental instructions for this treatment
	instructionsRows, err := d.db.Query(`select dr_drug_supplemental_instruction.id, dr_drug_supplemental_instruction.text from treatment_instructions 
												inner join dr_drug_supplemental_instruction on dr_drug_instruction_id = dr_drug_supplemental_instruction.id 
													where treatment_instructions.status=? and treatment_id=?`, STATUS_ACTIVE, treatment.Id.Int64())
	if err != nil {
		return err
	}
	defer instructionsRows.Close()

	drugInstructions := make([]*common.DoctorInstructionItem, 0)
	for instructionsRows.Next() {
		var instructionId encoding.ObjectId
		var text string
		if err := instructionsRows.Scan(&instructionId, &text); err != nil {
			return err
		}
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
func getRegimenPlanFromRows(rows *sql.Rows) (*common.RegimenPlan, error) {
	var regimenPlan common.RegimenPlan
	regimenSections := make(map[string][]*common.DoctorInstructionItem)
	for rows.Next() {
		var regimenType, regimenText string
		var regimenId, parentId encoding.ObjectId
		err := rows.Scan(&regimenId, &regimenType, &parentId, &regimenText)
		if err != nil {
			return nil, err
		}
		regimenStep := &common.DoctorInstructionItem{
			Id:       regimenId,
			Text:     regimenText,
			ParentId: parentId,
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

func getAdvicePointsFromRows(rows *sql.Rows) ([]*common.DoctorInstructionItem, error) {
	advicePoints := make([]*common.DoctorInstructionItem, 0)
	for rows.Next() {
		var id, parentId encoding.ObjectId
		var text string
		if err := rows.Scan(&id, &parentId, &text); err != nil {
			return nil, err
		}

		advicePoint := &common.DoctorInstructionItem{
			Id:       id,
			ParentId: parentId,
			Text:     text,
		}
		advicePoints = append(advicePoints, advicePoint)
	}
	return advicePoints, rows.Err()
}
