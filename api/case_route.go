package api

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/third_party/github.com/go-sql-driver/mysql"
)

type JBCQItemClaimForbidden string

func (j JBCQItemClaimForbidden) Error() string {
	return string(j)
}

type CaseClaimForbidden string

func (c CaseClaimForbidden) Error() string {
	return string(c)
}

// InsertUnclaimedItemIntoQueue inserts an unclaimed case into the queue for eligible doctors to consume
func (d *DataService) InsertUnclaimedItemIntoQueue(queueItem *DoctorQueueItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`insert into unclaimed_case_queue (care_providing_state_id, item_id, patient_case_id, event_type, status) values (?,?,?,?,?)`, queueItem.CareProvidingStateId, queueItem.ItemId, queueItem.PatientCaseId, queueItem.EventType, queueItem.Status)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update the status of the patient visit to indicate that it was routed if we are dealing with a visit
	if queueItem.EventType == DQEventTypePatientVisit {
		pvStatus := common.PVStatusRouted
		if err := updatePatientVisit(tx, queueItem.ItemId, &PatientVisitUpdate{Status: &pvStatus}); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient does as the name says - it temporarily assigns a case and the patient file to an eligible doctor such
// that the doctor has exclusive access to the patient case. Note that its possible that the doctor already has access to the patient file, in which case
// the existing access to the patient file is maintained, while temporary access is added for the patient case.
func (d *DataService) TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorId int64, patientCase *common.PatientCase, duration time.Duration) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// mark the case as temporarily claimed
	_, err = tx.Exec(`update patient_case set status = ? where id = ?`, common.PCStatusTempClaimed, patientCase.Id.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	expiresTime := time.Now().Add(duration)

	// lock the visit in the unclaimed item queue
	_, err = tx.Exec(`update unclaimed_case_queue set locked = 1, doctor_id = ?, expires = ? where patient_case_id = ?`, doctorId, expiresTime, patientCase.Id.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	// temporarily assign the doctor to the patient
	var count int64
	if err := tx.QueryRow(`select count(*) from patient_care_provider_assignment where provider_id = ?  and role_type_id = ? and patient_id=?`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.PatientId.Int64()).Scan(&count); err != nil {
		tx.Rollback()
		return err
	}

	if count == 0 {
		// give temp access for the doctor to the patient file only if the doctor does not already have access to the patient file
		_, err = tx.Exec(`insert into patient_care_provider_assignment (role_type_id, provider_id, patient_id, health_condition_id, status, expires) values (?,?,?,?,?,?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientCase.PatientId.Int64(), patientCase.HealthConditionId.Int64(), STATUS_TEMP, expiresTime)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// temporarily assign the doctor to the patient_case
	_, err = tx.Exec(`replace into patient_case_care_provider_assignment (role_type_id, provider_id, patient_case_id, status, expires) values (?,?,?,?,?)`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientCase.Id.Int64(), STATUS_TEMP, expiresTime)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ExtendClaimForDoctor extends an existing claim on a case for a doctor. The method ensures to check that the current owner of the case is indeed the doctor
// before extending the claim. Note that the claim on the patient file as well as the case is atomically extended given that the access to the global information
// should go hand in hand with access to the patient case in this situation.
func (d *DataService) ExtendClaimForDoctor(doctorId, patientId, patientCaseId int64, duration time.Duration) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// ensure that the current doctor is the one holding on to the lock in the queue
	var currentLockHolder int64
	if err := tx.QueryRow(`select doctor_id from unclaimed_case_queue where patient_case_id = ? and locked = ?`, patientCaseId, true).Scan(&currentLockHolder); err == sql.ErrNoRows {
		tx.Rollback()
		return JBCQItemClaimForbidden("Doctor no longer listed as current claimer of case")
	} else if err != nil {
		tx.Rollback()
		return err
	}

	if currentLockHolder != doctorId {
		tx.Rollback()
		return JBCQItemClaimForbidden("Current lock holder is not the same as the doctor id provided")
	}

	expires := time.Now().Add(duration)

	// extend the claim of the doctor on the case and the patient file
	_, err = tx.Exec(`update unclaimed_case_queue set expires = ? where doctor_id = ? and patient_case_id = ?`, expires, doctorId, patientCaseId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update patient_care_provider_assignment set expires = ? where provider_id = ? and role_type_id = ? and status = ? and patient_id = ?`, expires, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_TEMP, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update patient_case_care_provider_assignment set expires = ? where provider_id = ? and role_type_id = ? and status = ? and patient_case_id = ?`, expires, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_TEMP, patientCaseId)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// PermanentlyAssignDoctorToCaseAndRouteToQueue assigns a case to a doctor that already has access to the patient file information. The call fails
// if the doctor does not have access to the patient file.
func (d *DataService) PermanentlyAssignDoctorToCaseAndRouteToQueue(doctorId int64, patientCase *common.PatientCase, queueItem *DoctorQueueItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	err = func() error {
		// first check to ensure that doctor is currently assigned to patient file
		var currentDoctorForPatient int64
		if err := tx.QueryRow(`
			SELECT provider_id 
			FROM patient_care_provider_assignment 
			WHERE role_type_id = ? AND provider_id = ? AND patient_id = ? AND status = ?`,
			d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientCase.PatientId.Int64(),
			STATUS_ACTIVE).Scan(&currentDoctorForPatient); err == sql.ErrNoRows {
			return CaseClaimForbidden("Doctor cannot claim case becase doctor is not assigned to patient file")
		} else if err != nil {
			return err
		}

		// update patient case to indicate that it is now claimed
		_, err = tx.Exec(`
			UPDATE patient_case 
			SET status = ? 
			WHERE id = ?`, common.PCStatusClaimed, patientCase.Id.Int64())
		if err != nil {
			return err
		}

		// update the patient visit (if that is the item we are working with) to indicate that it was routed
		if queueItem.EventType == DQEventTypePatientVisit {
			pvStatus := common.PVStatusRouted
			if err := updatePatientVisit(tx, queueItem.ItemId, &PatientVisitUpdate{Status: &pvStatus}); err != nil {
				return err
			}
		}

		// only add the doctor to the patient's care team for this case if the doctor doesn't already exist
		var existingDoctorID int64
		err = tx.QueryRow(`
		SELECT provider_id 
		FROM patient_case_care_provider_assignment 
		WHERE patient_case_id = ? and role_type_id = ?`, patientCase.Id.Int64(), d.roleTypeMapping[DOCTOR_ROLE]).Scan(&existingDoctorID)
		if err != sql.ErrNoRows && err != nil {
			return err
		} else if existingDoctorID != 0 && existingDoctorID != doctorId {
			return errors.New("Existing doctor for this case is different than incoming doctor for this case")
		} else if err == sql.ErrNoRows {

			// assign doctor to patient case
			_, err = tx.Exec(`
			INSERT INTO patient_case_care_provider_assignment 
			(provider_id, role_type_id, patient_case_id, status) 
			VALUES (?,?,?,?)`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.Id.Int64(), STATUS_ACTIVE)
			if err != nil {
				return err
			}
		}

		// insert item into doctor queue
		if err := insertItemIntoDoctorQueue(tx, queueItem); err != nil {
			return err
		}
		return nil
	}()

	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// TransitionToPermanentAssignmentOfDoctorToCaseAndPatient transitions from a temporary claim to a permanent claim on the patient case and the patient file. The item
// is consequently deleted from the unclaimed case queue.
func (d *DataService) TransitionToPermanentAssignmentOfDoctorToCaseAndPatient(doctorId int64, patientCase *common.PatientCase) error {
	tx, err := d.db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}

	// first check to ensure that the doctor is currently temporarily assigned to patient case and file
	var currentDoctorOnFile int64
	if err := tx.QueryRow(`select provider_id from patient_care_provider_assignment where role_type_id = ? and provider_id = ? and patient_id = ? and status = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientCase.PatientId.Int64(), STATUS_TEMP).Scan(&currentDoctorOnFile); err == sql.ErrNoRows {
		tx.Rollback()
		return JBCQItemClaimForbidden("Expected doctor to be temporarily assigned to patient file but wasnt")
	} else if err != nil {
		tx.Rollback()
		return err
	}

	var currentDoctorOnCase int64
	if err := tx.QueryRow(`select provider_id from patient_case_care_provider_assignment where role_type_id = ? and provider_id = ? and patient_case_id = ? and status = ?`, d.roleTypeMapping[DOCTOR_ROLE], doctorId, patientCase.Id.Int64(), STATUS_TEMP).Scan(&currentDoctorOnCase); err == sql.ErrNoRows {
		tx.Rollback()
		return JBCQItemClaimForbidden("Expected doctor to be temporarily assigned to patient case but wasnt")
	} else if err != nil {
		tx.Rollback()
		return err
	}

	// delete item from unclaimed queue
	_, err = tx.Exec(`delete from unclaimed_case_queue where patient_case_id = ? and doctor_id = ? and locked = ?`, patientCase.Id.Int64(), doctorId, true)
	if err != nil {
		tx.Rollback()
		return err
	}

	// update patient case to indicate that its now claimed
	_, err = tx.Exec(`update patient_case set status = ? where id = ?`, common.PCStatusClaimed, patientCase.Id.Int64())
	if err != nil {
		tx.Rollback()
		return err
	}

	// permanently assign doctor to patient
	_, err = tx.Exec(`update patient_care_provider_assignment set status = ?, expires = NULL where provider_id = ? and role_type_id = ? and patient_id = ? and status = ?`, STATUS_ACTIVE, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.PatientId.Int64(), STATUS_TEMP)
	if err != nil {
		tx.Rollback()
		return err
	}

	// permanent assign doctor to case
	_, err = tx.Exec(`update patient_case_care_provider_assignment set status = ?, expires = NULL where provider_id = ? and role_type_id = ? and patient_case_id = ? and status = ?`, STATUS_ACTIVE, doctorId, d.roleTypeMapping[DOCTOR_ROLE], patientCase.Id.Int64(), STATUS_TEMP)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetClaimedItemsInQueue() ([]*DoctorQueueItem, error) {
	rows, err := d.db.Query(`select id, event_type, item_id, patient_case_id, enqueue_date, status, doctor_id, expires from unclaimed_case_queue where locked = ?`, true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claimedItemsQueue := make([]*DoctorQueueItem, 0)
	for rows.Next() {
		var queueItem DoctorQueueItem
		if err := rows.Scan(&queueItem.Id,
			&queueItem.EventType,
			&queueItem.ItemId,
			&queueItem.PatientCaseId,
			&queueItem.EnqueueDate,
			&queueItem.Status,
			&queueItem.DoctorId,
			&queueItem.Expires); err != nil {
			return nil, err
		}
		claimedItemsQueue = append(claimedItemsQueue, &queueItem)
	}
	return claimedItemsQueue, rows.Err()
}

func (d *DataService) GetTempClaimedCaseInQueue(patientCaseId, doctorId int64) (*DoctorQueueItem, error) {
	var queueItem DoctorQueueItem
	err := d.db.QueryRow(`select id, event_type, item_id, patient_case_id, enqueue_date, status, doctor_id, expires from unclaimed_case_queue where locked = ? and patient_case_id = ? and doctor_id = ?`, true, patientCaseId, doctorId).Scan(
		&queueItem.Id,
		&queueItem.EventType,
		&queueItem.ItemId,
		&queueItem.PatientCaseId,
		&queueItem.EnqueueDate,
		&queueItem.Status,
		&queueItem.DoctorId,
		&queueItem.Expires)
	if err == sql.ErrNoRows {
		return nil, NoRowsError
	} else if err != nil {
		return nil, err
	}
	return &queueItem, nil
}

func (d *DataService) GetAllItemsInUnclaimedQueue() ([]*DoctorQueueItem, error) {
	rows, err := d.db.Query(`select id, event_type, item_id, patient_case_id, enqueue_date, status from unclaimed_case_queue order by enqueue_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return getUnclaimedItemsFromRows(rows)
}

func (d *DataService) OldestUnclaimedItems(maxItems int) ([]*ItemAge, error) {
	rows, err := d.db.Query(`
		SELECT id, enqueue_date 
		FROM unclaimed_case_queue 
		WHERE locked = 0
		ORDER BY enqueue_date
		LIMIT ?`, maxItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var caseAges []*ItemAge
	for rows.Next() {
		var caseAge ItemAge
		var enqueueDate time.Time
		if err := rows.Scan(
			&caseAge.ID,
			&enqueueDate); err != nil {
			return nil, err
		}

		caseAge.Age = time.Since(enqueueDate)
		caseAges = append(caseAges, &caseAge)
	}

	return caseAges, rows.Err()
}

func getUnclaimedItemsFromRows(rows *sql.Rows) ([]*DoctorQueueItem, error) {
	var queueItems []*DoctorQueueItem
	for rows.Next() {
		var queueItem DoctorQueueItem
		var enqueueDate mysql.NullTime
		if err := rows.Scan(
			&queueItem.Id,
			&queueItem.EventType,
			&queueItem.ItemId,
			&queueItem.PatientCaseId,
			&enqueueDate,
			&queueItem.Status); err != nil {
			return nil, err
		}
		queueItem.EnqueueDate = enqueueDate.Time
		queueItems = append(queueItems, &queueItem)
	}

	return queueItems, rows.Err()
}

func (d *DataService) GetElligibleItemsInUnclaimedQueue(doctorId int64) ([]*DoctorQueueItem, error) {
	// first get the list of care providing state ids where the doctor is registered to serve
	rows, err := d.db.Query(`select care_providing_state_id from care_provider_state_elligibility where provider_id = ? and role_type_id = ?`, doctorId, d.roleTypeMapping[DOCTOR_ROLE])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var careProvidingStateIds []int64
	for rows.Next() {
		var careProvidingStateId int64
		if err := rows.Scan(&careProvidingStateId); err != nil {
			return nil, err
		}
		careProvidingStateIds = append(careProvidingStateIds, careProvidingStateId)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(careProvidingStateIds) == 0 {
		return nil, errors.New("Doctor is not elligible to provide care for any health condition in any state")
	}

	// then get the items in the unclaimed queue that are not currently locked by another doctor
	params := appendInt64sToInterfaceSlice(nil, careProvidingStateIds)
	params = append(params, []interface{}{false, true, doctorId}...)
	rows2, err := d.db.Query(fmt.Sprintf(`select id, event_type, item_id, patient_case_id, enqueue_date, status from unclaimed_case_queue where care_providing_state_id in (%s) and locked = ? or (locked = ? and doctor_id = ?) order by enqueue_date`, nReplacements(len(careProvidingStateIds))), params...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	return getUnclaimedItemsFromRows(rows2)
}

// RevokeDoctorAccessToCase removes the temporary access that the doctor has so that the item can be picked up by another doctor from the jbcq
func (d *DataService) RevokeDoctorAccessToCase(patientCaseId, patientId, doctorId int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	// unclaim the item in the case queue
	_, err = tx.Exec(`update unclaimed_case_queue set doctor_id = NULL, expires = NULL, locked = 0 where doctor_id = ? and patient_case_id = ? and locked = 1`, doctorId, patientCaseId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// mark the patient case as unclaimed
	_, err = tx.Exec(`update patient_case set status = ? where id = ?`, common.PCStatusUnclaimed, patientCaseId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// revoke doctor access to patient case
	_, err = tx.Exec(`delete from patient_care_provider_assignment where provider_id = ? and role_type_id = ? and status = ? and patient_id = ?`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_TEMP, patientId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// revoke doctor access to patient file
	_, err = tx.Exec(`delete from patient_case_care_provider_assignment where provider_id = ? and role_type_id = ? and status = ? and patient_case_id = ?`, doctorId, d.roleTypeMapping[DOCTOR_ROLE], STATUS_TEMP, patientCaseId)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CareProvidingStatesWithUnclaimedCases() ([]int64, error) {
	rows, err := d.db.Query(`
			SELECT DISTINCT care_providing_state_id 
			FROM unclaimed_case_queue
			WHERE locked = 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var careProvidingStateIds []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		careProvidingStateIds = append(careProvidingStateIds, id)
	}

	return careProvidingStateIds, rows.Err()
}

type ByLastNotified []*DoctorNotify

func (c ByLastNotified) Len() int      { return len(c) }
func (c ByLastNotified) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c ByLastNotified) Less(i, j int) bool {
	return c[i].LastNotified == nil ||
		(c[j].LastNotified != nil && c[i].LastNotified.Before(*c[j].LastNotified))
}

func (d *DataService) DoctorsToNotifyInCareProvidingState(careProvidingStateID int64, avoidDoctorsRegisteredInStates []int64, timeThreshold time.Time) ([]*DoctorNotify, error) {

	doctorsToExclude := make(map[int64]bool)
	// identify doctors to exclude based on the states we are avoiding
	if len(avoidDoctorsRegisteredInStates) > 0 {
		vals := appendInt64sToInterfaceSlice(nil, avoidDoctorsRegisteredInStates)
		vals = append(vals, d.roleTypeMapping[DOCTOR_ROLE])
		rows, err := d.db.Query(`
			SELECT provider_id
			FROM care_provider_state_elligibility
			WHERE care_providing_state_id in (`+nReplacements(len(avoidDoctorsRegisteredInStates))+`)
			AND role_type_id = ?`, vals...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			doctorsToExclude[id] = true
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	rows, err := d.db.Query(`
		SELECT provider_id, last_notified
		FROM care_provider_state_elligibility
		LEFT OUTER JOIN doctor_case_notification ON provider_id = doctor_id
		WHERE role_type_id = ? AND notify = 1 AND care_providing_state_id = ?
		AND (last_notified is NULL or last_notified < ?)`, d.roleTypeMapping[DOCTOR_ROLE], careProvidingStateID, timeThreshold)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doctorsToNotify []*DoctorNotify
	for rows.Next() {
		var doctorNotify DoctorNotify
		if err := rows.Scan(&doctorNotify.DoctorID, &doctorNotify.LastNotified); err != nil {
			return nil, err
		}
		if !doctorsToExclude[doctorNotify.DoctorID] {
			doctorsToNotify = append(doctorsToNotify, &doctorNotify)
		}
	}

	sort.Sort(ByLastNotified(doctorsToNotify))

	return doctorsToNotify, rows.Err()
}

func (d *DataService) RecordDoctorNotifiedOfUnclaimedCases(doctorID int64) error {
	_, err := d.db.Exec(`REPLACE INTO doctor_case_notification (doctor_id) values (?)`, doctorID)
	return err
}

func (d *DataService) RecordCareProvidingStateNotified(careProvidingStateID int64) error {
	_, err := d.db.Exec(`REPLACE INTO care_providing_state_notification (care_providing_state_id) values (?)`, careProvidingStateID)
	return err
}

func (d *DataService) LastNotifiedTimeForCareProvidingState(careProvidingStateID int64) (time.Time, error) {
	var lastNotifiedTime time.Time
	err := d.db.QueryRow(`
		SELECT last_notified 
		FROM care_providing_state_notification 
		WHERE care_providing_state_id = ?`, careProvidingStateID).Scan(&lastNotifiedTime)
	if err == sql.ErrNoRows {
		return lastNotifiedTime, NoRowsError
	}
	return lastNotifiedTime, err
}
