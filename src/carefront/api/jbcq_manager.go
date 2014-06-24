package api

import (
	"carefront/common"
	"carefront/libs/golog"
	"sync"
	"time"
)

// jbcqManager is responsible for managing access to items in the global jump ball queue
// where any elligible doctor can access an item for a period of time
type jbcqManager struct {
	dataAPI DataAPI
	qMutex  sync.Mutex
}

var jManager *jbcqManager
var jbcqManagerOnce sync.Once

func GetJBCQManager(dataAPI DataAPI) *jbcqManager {
	if jManager == nil {
		jManager = &jbcqManager{
			dataAPI: dataAPI,
			qMutex:  sync.Mutex{},
		}
	}
	return jManager
}

// ClaimCaseForVisitIfUnclaimed checks to see if the patient case is currently unclaimed, and
// temporarily hands over exclusive access to the doctor if this is so.
func (j *jbcqManager) ClaimCaseForVisitIfUnclaimed(patientVisitId, doctorId int64, expireDuration time.Duration) error {
	j.qMutex.Lock()
	defer j.qMutex.Unlock()

	// check if the visit is unclaimed and if so, claim it by updating the item in the jump ball queue
	// and temporarily assigning the doctor to the patient
	patientCase, err := j.dataAPI.GetPatientCaseFromPatientVisitId(patientVisitId)
	if err != nil {
		return err
	}

	// go ahead and claim case if no doctors are assigned to it
	if patientCase.Status == common.PCStatusUnclaimed {
		if err := j.dataAPI.temporarilyClaimCaseAndAssignDoctorToCaseAndPatient(doctorId, patientCase.Id.Int64(),
			patientCase.PatientId.Int64(), patientVisitId, DQEventTypePatientVisit, expireDuration); err != nil {
			golog.Errorf("Unable to temporarily assign the patient visit to the doctor: %s", err)
			return err
		}
	}
	return nil
}

// ExtendClaimOnPatientVisitDiagnosis extends the doctor's claim on the case if the doctor adds/modifies the diagnosis
// for the patient visits
func (j *jbcqManager) ExtendClaimOnPatientVisitDiagnosis(patientVisitId, doctorId int64, expireDuration time.Duration) error {
	j.qMutex.Lock()
	defer j.qMutex.Unlock()

	patientCase, err := j.dataAPI.GetPatientCaseFromPatientVisitId(patientVisitId)
	if err != nil {
		golog.Errorf("Unable to get patiente case from patient visit: %s", err)
		return err
	}

	if patientCase.Status == common.PCStatusTempClaimed {
		if err := j.dataAPI.extendClaimForDoctor(doctorId, patientVisitId, DQEventTypePatientVisit, expireDuration); err != nil {
			golog.Errorf("Unable to extend the claim on the case for the doctor: %s", err)
			return err
		}
	}

	return nil
}

// ExtendClaimOnTreatmentPlanModification extends the doctor's claim on the case if the doctor modifies a treatment plan
// pertaining to a temporarily claimed case
func (j *jbcqManager) ExtendClaimOnTreatmentPlanModification(treatmentPlanId, doctorId int64, expireDuration time.Duration) error {
	j.qMutex.Lock()
	defer j.qMutex.Unlock()

	patientCase, err := j.dataAPI.GetPatientCaseFromTreatmentPlanId(treatmentPlanId)
	if err != nil {
		golog.Errorf("Unable to get patient case from treatment plan id: %s", err)
		return err
	}

	patientVisitId, err := j.dataAPI.GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId)
	if err != nil {
		golog.Errorf("Unable to get patient visit id from treatment plan id: %s", err)
		return err
	}

	if patientCase.Status == common.PCStatusTempClaimed {
		if err := j.dataAPI.extendClaimForDoctor(doctorId, patientVisitId, DQEventTypePatientVisit, expireDuration); err != nil {
			golog.Errorf("Unable to extend claim on the case for the doctor: %s", err)
			return err
		}
	}

	return nil
}

// PermanentlyAssignDoctorToCaseAndPatient deletes the item from the jump ball queue and grants permanent
// access to the doctor
func (j *jbcqManager) PermanentlyAssignDoctorToCaseAndPatient(patientVisitId, doctorId int64) error {
	j.qMutex.Lock()
	defer j.qMutex.Unlock()

	patientCase, err := j.dataAPI.GetPatientCaseFromPatientVisitId(patientVisitId)
	if err != nil {
		return err
	}

	if patientCase.Status == common.PCStatusTempClaimed {
		if err := j.dataAPI.permanentlyAssignDoctorToCaseAndPatient(doctorId, patientCase.Id.Int64(),
			patientCase.PatientId.Int64(), patientVisitId, DQEventTypePatientVisit); err != nil {
			golog.Errorf("Unable to permanently assign doctor to case and patient: %s", err)
			return err
		}
	}

	return nil
}
