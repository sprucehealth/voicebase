package doctor_queue

import (
	"carefront/api"
	"carefront/common"
	"carefront/doctor_treatment_plan"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	"carefront/patient_file"
	"carefront/patient_visit"

	"github.com/samuel/go-metrics/metrics"
)

func initJumpBallCaseQueueListeners(dataAPI api.DataAPI, statsRegistry metrics.Registry) {

	tempClaimSucess := metrics.NewCounter()
	tempClaimFailure := metrics.NewCounter()
	permanentClaimSuccess := metrics.NewCounter()
	permanentClaimFailure := metrics.NewCounter()
	claimExtensionSucess := metrics.NewCounter()
	claimExtensionFailure := metrics.NewCounter()

	statsRegistry.Add("temp_claim/success", tempClaimSucess)
	statsRegistry.Add("temp_claim/failure", tempClaimFailure)
	statsRegistry.Add("perm_claim/success", permanentClaimSuccess)
	statsRegistry.Add("perm_claim/failure", permanentClaimFailure)
	statsRegistry.Add("claim_extension/success", claimExtensionSucess)
	statsRegistry.Add("claim_extension/failure", claimExtensionFailure)

	// Grant temporary access to the patient case for an unclaimed case to the doctor requesting access to the case
	dispatch.Default.Subscribe(func(ev *patient_file.PatientVisitOpenedEvent) error {
		// check if the visit is unclaimed and if so, claim it by updating the item in the jump ball queue
		// and temporarily assigning the doctor to the patient
		patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(ev.PatientVisit.PatientVisitId.Int64())
		if err != nil {
			return err
		}

		// go ahead and claim case if no doctors are assigned to it
		if patientCase.Status == common.PCStatusUnclaimed {
			if err := dataAPI.TemporarilyClaimCaseAndAssignDoctorToCaseAndPatient(ev.DoctorId, patientCase, ExpireDuration); err != nil {
				tempClaimFailure.Inc(1)
				golog.Errorf("Unable to temporarily assign the patient visit to the doctor: %s", err)
				return err
			}
			tempClaimSucess.Inc(1)
		}
		return nil
	})

	// Extend the doctor's claim on the patient case if the doctor modifies the diagnosis associated with the case
	dispatch.Default.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(ev.PatientVisitId)
		if err != nil {
			golog.Errorf("Unable to get patiente case from patient visit: %s", err)
			return err
		}

		if patientCase.Status == common.PCStatusTempClaimed {
			if err := dataAPI.ExtendClaimForDoctor(ev.DoctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), ExpireDuration); err != nil {
				golog.Errorf("Unable to extend the claim on the case for the doctor: %s", err)
				claimExtensionFailure.Inc(1)
				return err
			}
			claimExtensionSucess.Inc(1)
		}
		return nil
	})

	// Extend the doctor's claim on the patient case if the doctor modifies any aspect of the treatment plan
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentsAddedEvent) error {
		return extendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, dataAPI, claimExtensionSucess, claimExtensionFailure)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.RegimenPlanAddedEvent) error {
		return extendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, dataAPI, claimExtensionSucess, claimExtensionFailure)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.AdviceAddedEvent) error {
		return extendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, dataAPI, claimExtensionSucess, claimExtensionFailure)
	})

	// If the doctor successfully submits a treatment plan for an unclaimed case, the case is then considered
	// claimed by the doctor and the doctor is assigned to the case and made part of the patient's care team
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		return permanentlyAssignDoctorToCaseAndPatient(ev.VisitId, ev.DoctorId, dataAPI, permanentClaimSuccess, permanentClaimFailure)

	})

	// If the doctor marks a case unsuitable for spruce, it is also considered claimed by the doctor
	// with the doctor permanently being assigned to the case and patient
	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		return permanentlyAssignDoctorToCaseAndPatient(ev.PatientVisitId, ev.DoctorId, dataAPI, permanentClaimSuccess, permanentClaimFailure)
	})
}

func permanentlyAssignDoctorToCaseAndPatient(patientVisitId, doctorId int64, dataAPI api.DataAPI, permClaimSuccess, permClaimFailure metrics.Counter) error {
	patientCase, err := dataAPI.GetPatientCaseFromPatientVisitId(patientVisitId)
	if err != nil {
		return err
	}

	if patientCase.Status == common.PCStatusTempClaimed {
		if err := dataAPI.PermanentlyAssignDoctorToCaseAndPatient(doctorId, patientCase); err != nil {
			golog.Errorf("Unable to permanently assign doctor to case and patient: %s", err)
			permClaimFailure.Inc(1)
			return err
		}
		permClaimSuccess.Inc(1)
	}

	return nil
}

func extendClaimOnTreatmentPlanModification(treatmentPlanId, doctorId int64, dataAPI api.DataAPI, claimExtensionSucess, claimExtensionFailure metrics.Counter) error {
	patientCase, err := dataAPI.GetPatientCaseFromTreatmentPlanId(treatmentPlanId)
	if err != nil {
		golog.Errorf("Unable to get patient case from treatment plan id: %s", err)
		return err
	}

	if patientCase.Status == common.PCStatusTempClaimed {
		if err := dataAPI.ExtendClaimForDoctor(doctorId, patientCase.PatientId.Int64(), patientCase.Id.Int64(), ExpireDuration); err != nil {
			golog.Errorf("Unable to extend claim on the case for the doctor: %s", err)
			claimExtensionFailure.Inc(1)
			return err
		}
		claimExtensionSucess.Inc(1)
	}

	return nil
}
