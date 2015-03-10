package patient_case

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	timePeriod = 15 * time.Minute
)

// worker is responsible for transitioning patient cases
// that reach their timeout threshold of being in the state they are in.
// The possible state transitions are:
// - ACTIVE -> INACTIVE
// - PRE_SUBMISSION_TRIAGE -> PRE_SUBMISSION_TRIAGE_DELETED
// - INACTIVE -> INACTIVE
// - PRE_SUBMISSION_TRIAGE_DELETED -> PRE_SUBMISSION_TRIAGE_DELETED
// - DELETED -> DELETED
type worker struct {
	dataAPI  api.DataAPI
	lockAPI  api.LockAPI
	stopChan chan bool
}

func NewWorker(dataAPI api.DataAPI, lockAPI api.LockAPI) *worker {
	return &worker{
		dataAPI:  dataAPI,
		lockAPI:  lockAPI,
		stopChan: make(chan bool),
	}
}

func (w *worker) Stop() {
	close(w.stopChan)
}

func (w *worker) Start() {
	go func() {
		defer w.lockAPI.Release()
		for {
			if !w.lockAPI.Wait() {
				return
			}

			select {
			case <-w.stopChan:
				return
			default:
			}

			if err := w.Do(); err != nil {
				golog.Errorf(err.Error())
			}

			select {
			case <-w.stopChan:
				return
			case <-time.After(timePeriod):
			}
		}
	}()
}

func (w *worker) Do() error {
	cases, err := w.dataAPI.TimedOutCases()
	if err != nil {
		golog.Errorf("Unable to get cases that have timed out: %s", err.Error())
		return err
	}

	for _, pc := range cases {
		var nextStatus *common.CaseStatus
		var timeoutDate api.NullableTime

		switch pc.Status {
		case common.PCStatusActive:
			status := common.PCStatusInactive
			nextStatus = &status
			timeoutDate.Valid = true

		case common.PCStatusPreSubmissionTriage:
			status := common.PCStatusPreSubmissionTriageDeleted
			nextStatus = &status
			timeoutDate.Valid = true

		case common.PCStatusInactive, common.PCStatusPreSubmissionTriageDeleted, common.PCStatusDeleted:
			timeoutDate.Valid = true

		default:
			golog.Errorf("Undetermined transition for patientCaseID: %d, currentStatus: %s", pc.ID.Int64(), pc.Status.String())
			return nil
		}

		if err := w.dataAPI.UpdatePatientCase(pc.ID.Int64(), &api.PatientCaseUpdate{
			Status:      nextStatus,
			TimeoutDate: timeoutDate,
		}); err != nil {
			golog.Errorf("Unable to update patient case: %s", err.Error())
			return err
		}
	}

	return nil
}
