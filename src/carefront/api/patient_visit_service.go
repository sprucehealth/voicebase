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
	err := d.db.QueryRow(`select patient_id, patient_case_id, health_condition_id, layout_version_id, 
		creation_date, submitted_date, closed_date, status from patient_visit where id = ?`, patientVisitId,
	).Scan(
		&patientVisit.PatientId,
		&patientVisit.PatientCaseId,
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

func (d *DataService) GetPatientCaseIdFromPatientVisitId(patientVisitId int64) (int64, error) {
	var patientCaseId int64
	if err := d.db.QueryRow(`select patient_case_id from patient_visit where id=?`, patientVisitId).Scan(&patientCaseId); err == sql.ErrNoRows {
		return 0, NoRowsError
	} else if err != nil {
		return 0, err
	}
	return patientCaseId, nil
}

func (d *DataService) CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	// implicitly create a new case when creating a new visit for now
	// for now treating the creation of every new case as an unclaimed case because we don't have a notion of a
	// new case for which the patient returns (and thus can be potentially claimed)
	res, err := tx.Exec(`insert into patient_case (patient_id, health_condition_id, status) values (?,?,?)`, patientId, healthConditionId, common.PCStatusUnclaimed)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	patientCaseId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	res, err = tx.Exec(`insert into patient_visit (patient_id, health_condition_id, layout_version_id, patient_case_id, status) 
								values (?, ?, ?, ?, ?)`, patientId, healthConditionId, layoutVersionId, patientCaseId, CASE_STATUS_OPEN)
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

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return lastId, nil
}

func (d *DataService) GetAbridgedTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error) {
	rows, err := d.db.Query(`select id, doctor_id, patient_id, patient_case_id, status, creation_date from treatment_plan where id = ?`, treatmentPlanId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	drTreatmentPlans, err := d.getAbridgedTreatmentPlanFromRows(rows, doctorId)
	if err != nil {
		return nil, err
	}

	switch l := len(drTreatmentPlans); {
	case l == 0:
		return nil, NoRowsError
	case l == 1:
		return drTreatmentPlans[0], nil
	}

	return nil, fmt.Errorf("Expected 1 drTreatmentPlan instead got %d", len(drTreatmentPlans))
}

func (d *DataService) GetTreatmentPlan(treatmentPlanId, doctorId int64) (*common.DoctorTreatmentPlan, error) {
	treatmentPlan, err := d.GetAbridgedTreatmentPlan(treatmentPlanId, doctorId)
	if err != nil {
		return nil, err
	}

	// get treatments
	treatmentPlan.TreatmentList = &common.TreatmentList{}
	treatmentPlan.TreatmentList.Treatments, err = d.GetTreatmentsBasedOnTreatmentPlanId(treatmentPlanId)
	if err != nil {
		return nil, err
	}

	// get advice
	treatmentPlan.Advice = &common.Advice{}
	treatmentPlan.Advice.SelectedAdvicePoints, err = d.GetAdvicePointsForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, err
	}

	// get regimen
	treatmentPlan.RegimenPlan = &common.RegimenPlan{}
	treatmentPlan.RegimenPlan, err = d.GetRegimenPlanForTreatmentPlan(treatmentPlanId)
	if err != nil {
		return nil, err
	}

	return treatmentPlan, nil
}

func (d *DataService) getAbridgedTreatmentPlanFromRows(rows *sql.Rows, doctorId int64) ([]*common.DoctorTreatmentPlan, error) {
	drTreatmentPlans := make([]*common.DoctorTreatmentPlan, 0)
	for rows.Next() {
		var drTreatmentPlan common.DoctorTreatmentPlan
		if err := rows.Scan(&drTreatmentPlan.Id, &drTreatmentPlan.DoctorId, &drTreatmentPlan.PatientId, &drTreatmentPlan.PatientCaseId, &drTreatmentPlan.Status, &drTreatmentPlan.CreationDate); err != nil {
			return nil, err
		}

		// parent information has to exist for every treatment plan; so if it doesn't that means something is logically inconsistent
		drTreatmentPlan.Parent = &common.TreatmentPlanParent{}
		err := d.db.QueryRow(`select parent_id, parent_type from treatment_plan_parent where treatment_plan_id = ?`, drTreatmentPlan.Id.Int64()).Scan(&drTreatmentPlan.Parent.ParentId, &drTreatmentPlan.Parent.ParentType)
		if err == sql.ErrNoRows {
			return nil, NoRowsError
		} else if err != nil {
			return nil, err
		}

		// get the creation date of the parent since this information is useful for the client
		var creationDate time.Time
		switch drTreatmentPlan.Parent.ParentType {
		case common.TPParentTypePatientVisit:
			if err := d.db.QueryRow(`select creation_date from patient_visit where id = ?`, drTreatmentPlan.Parent.ParentId.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, NoRowsError
			} else if err != nil {
				return nil, err
			}
		case common.TPParentTypeTreatmentPlan:
			if err := d.db.QueryRow(`select creation_date from treatment_plan where id = ?`, drTreatmentPlan.Parent.ParentId.Int64()).Scan(&creationDate); err == sql.ErrNoRows {
				return nil, NoRowsError
			} else if err != nil {
				return nil, err
			}
		}
		drTreatmentPlan.Parent.CreationDate = creationDate

		// only populate content source information if we are retrieving this information for the same doctor that created the treatment plan
		drTreatmentPlan.ContentSource = &common.TreatmentPlanContentSource{}
		err = d.db.QueryRow(`select content_source_id, content_source_type, has_deviated from treatment_plan_content_source where treatment_plan_id = ? and doctor_id = ?`, drTreatmentPlan.Id.Int64(), doctorId).Scan(&drTreatmentPlan.ContentSource.ContentSourceId, &drTreatmentPlan.ContentSource.ContentSourceType, &drTreatmentPlan.ContentSource.HasDeviated)
		if err == sql.ErrNoRows {
			// treat content source as empty if non specified
			drTreatmentPlan.ContentSource = nil
		} else if err != nil {
			return nil, err
		}

		drTreatmentPlans = append(drTreatmentPlans, &drTreatmentPlan)
	}
	return drTreatmentPlans, rows.Err()
}

func (d *DataService) GetAbridgedTreatmentPlanList(doctorId, patientId int64, status string) ([]*common.DoctorTreatmentPlan, error) {
	rows, err := d.db.Query(`select id, doctor_id, patient_id, patient_case_id, status, creation_date from treatment_plan where patient_id = ? AND status = ?`, patientId, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getAbridgedTreatmentPlanFromRows(rows, doctorId)
}

func (d *DataService) GetAbridgedTreatmentPlanListInDraftForDoctor(doctorId, patientId int64) ([]*common.DoctorTreatmentPlan, error) {
	rows, err := d.db.Query(`select id, doctor_id, patient_id, patient_case_id, status, creation_date from treatment_plan where doctor_id = ?  and patient_id = ? and status = ?`, doctorId, patientId, STATUS_DRAFT)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return d.getAbridgedTreatmentPlanFromRows(rows, doctorId)
}

func (d *DataService) DeleteTreatmentPlan(treatmentPlanId int64) error {
	_, err := d.db.Exec(`delete from treatment_plan where id = ?`, treatmentPlanId)
	return err
}

func (d *DataService) GetPatientIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error) {
	var patientId int64
	err := d.db.QueryRow(`select patient_id from treatment_plan where id = ?`, treatmentPlanId).Scan(&patientId)

	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}

	return patientId, err
}

func (d *DataService) GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error) {
	var patientVisitId int64
	err := d.db.QueryRow(`select patient_visit_id from treatment_plan_patient_visit_mapping where treatment_plan_id = ?`, treatmentPlanId).Scan(&patientVisitId)
	if err == sql.ErrNoRows {
		return 0, NoRowsError
	}

	return patientVisitId, nil
}

func (d *DataService) StartNewTreatmentPlan(patientId, patientVisitId, doctorId int64, parent *common.TreatmentPlanParent, contentSource *common.TreatmentPlanContentSource) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`delete from treatment_plan where id = (select treatment_plan_id from treatment_plan_parent where parent_id = ? and parent_type = ?) and status = ? and doctor_id = ?`, parent.ParentId.Int64(), parent.ParentType, STATUS_DRAFT, doctorId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// get the case the treatment plan belongs to from the patient visit
	patientCaseId, err := d.GetPatientCaseIdFromPatientVisitId(patientVisitId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	lastId, err := tx.Exec(`insert into treatment_plan (patient_id, doctor_id, patient_case_id, status) values (?,?,?,?)`, patientId, doctorId, patientCaseId, STATUS_DRAFT)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	treatmentPlanId, err := lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the patient visit that is the reason for which the treatment plan is being created
	_, err = tx.Exec(`insert into treatment_plan_patient_visit_mapping (treatment_plan_id, patient_visit_id) values (?,?)`, treatmentPlanId, patientVisitId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the parent information for treatment plan
	_, err = tx.Exec(`insert into treatment_plan_parent (treatment_plan_id,parent_id, parent_type) values (?,?,?)`, treatmentPlanId, parent.ParentId.Int64(), parent.ParentType)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// track the original content source for the treatment plan
	if contentSource != nil {
		_, err := tx.Exec(`insert into treatment_plan_content_source (treatment_plan_id, doctor_id, content_source_id, content_source_type) values (?,?,?,?)`, treatmentPlanId, doctorId, contentSource.ContentSourceId.Int64(), contentSource.ContentSourceType)
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

func (d *DataService) ClosePatientVisit(patientVisitId int64, event string) error {
	_, err := d.db.Exec(`update patient_visit set status=?, closed_date=now() where id = ?`, event, patientVisitId)
	return err
}

func (d *DataService) ActivateTreatmentPlan(treatmentPlanId, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	treatmentPlan, err := d.GetAbridgedTreatmentPlan(treatmentPlanId, doctorId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Based on the parent of the treatment plan, ensure to "close" the patient visit
	// or to inactivate previous treatment plan before marking the new one as ACTIVE
	switch treatmentPlan.Parent.ParentType {
	case common.TPParentTypePatientVisit:
		// mark the patient visit as TREATED
		_, err = tx.Exec(`update patient_visit set status=?, closed_date=now(6) where id = ?`, CASE_STATUS_TREATED, treatmentPlan.Parent.ParentId.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}

	case common.TPParentTypeTreatmentPlan:
		// mark the previous treatment plan as INACTIVE
		_, err = tx.Exec(`update treatment_plan set status = ? where id = ?`, STATUS_INACTIVE, treatmentPlan.Parent.ParentId.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// mark the treatment plan as ACTIVE
	_, err = tx.Exec(`update treatment_plan set status = ? where id = ?`, STATUS_ACTIVE, treatmentPlan.Id.Int64())
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

	// currently assign case to the same doctor until we have a better way to manage when a case is assigned to a doctor
	patientCaseId, err := d.GetPatientCaseIdFromPatientVisitId(patientVisitId)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`insert into patient_case_care_provider_assignment (patient_case_id, provider_id, role_type_id, status) values (?,?,?,?)`, patientCaseId, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_ACTIVE)
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

func (d *DataService) CreateAdviceForTreatmentPlan(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error {
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

func (d *DataService) CreateRegimenPlanForTreatmentPlan(regimenPlan *common.RegimenPlan) error {
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

func (d *DataService) AddTreatmentsForTreatmentPlan(treatments []*common.Treatment, doctorId, treatmentPlanId, patientId int64) error {
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
			treatment_plan.patient_id, treatment_plan.doctor_id from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
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
			treatment_plan.patient_id, treatment_plan.doctor_id from treatment 
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment_plan.patient_id = ? and treatment.status=? and localized_text.language_id = ?`, patientId, STATUS_CREATED, EN_LANGUAGE_ID)

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

func (d *DataService) GetActiveTreatmentPlanIdForPatient(patientId int64) (int64, error) {
	var treatmentPlanIds []int64
	rows, err := d.db.Query(`select id from treatment_plan where patient_id = ? and status = ?`, patientId, STATUS_ACTIVE)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var treatmentPlanId int64
		if err := rows.Scan(&treatmentPlanId); err != nil {
			return 0, err
		}
		treatmentPlanIds = append(treatmentPlanIds, treatmentPlanId)
	}
	if rows.Err() != nil {
		return 0, rows.Err()
	}

	switch l := len(treatmentPlanIds); {
	case l == 0:
		return 0, NoRowsError
	case l == 1:
		return treatmentPlanIds[0], nil
	}

	return 0, fmt.Errorf("Expected 1 active treatment plan id instead got %d", len(treatmentPlanIds))
}

func (d *DataService) GetTreatmentBasedOnPrescriptionId(erxId int64) (*common.Treatment, error) {
	rows, err := d.db.Query(`select treatment.id,treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			treatment_plan.patient_id, treatment_plan.doctor_id from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
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
			treatment_plan.patient_id, treatment_plan.doctor_id from treatment
				inner join treatment_plan on treatment.treatment_plan_id = treatment_plan.id
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

func (d *DataService) AddErxStatusEvent(treatmentIds []int64, prescriptionStatus common.StatusEvent) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	for _, treatmentId := range treatmentIds {

		_, err = tx.Exec(`update erx_status_events set status = ? where treatment_id = ? and status = ?`, STATUS_INACTIVE, treatmentId, STATUS_ACTIVE)
		if err != nil {
			tx.Rollback()
			return err
		}

		columnsAndData := make(map[string]interface{}, 0)
		columnsAndData["treatment_id"] = treatmentId
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
								left outer join erx_status_events on erx_status_events.treatment_id = treatment.id 
								inner join patient on patient.id = treatment_plan.patient_id 
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

func (d *DataService) MarkTPDeviatedFromContentSource(treatmentPlanId int64) error {
	_, err := d.db.Exec(`update treatment_plan_content_source set has_deviated = 1, deviated_date = now(6) where treatment_plan_id = ?`, treatmentPlanId)
	return err
}

func (d *DataService) getTreatmentAndMetadataFromCurrentRow(rows *sql.Rows) (*common.Treatment, error) {
	var treatmentId, treatmentPlanId, dispenseUnitId, patientId, prescriberId, prescriptionId, pharmacyId encoding.ObjectId
	var dispenseValue encoding.HighPrecisionFloat64
	var drugInternalName, dosageStrength, patientInstructions, treatmentType, dispenseUnitDescription, status string
	var substitutionsAllowed bool
	var refills, daysSupply encoding.NullInt64
	var creationDate time.Time
	var erxSentDate mysql.NullTime
	var pharmacyNotes, drugName, drugForm, drugRoute sql.NullString
	err := rows.Scan(&treatmentId, &prescriptionId, &treatmentPlanId, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId,
		&dispenseUnitDescription, &refills, &substitutionsAllowed, &daysSupply, &pharmacyId,
		&pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &status, &drugName, &drugRoute, &drugForm, &patientId, &prescriberId)
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
