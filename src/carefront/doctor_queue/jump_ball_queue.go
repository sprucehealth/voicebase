package doctor_queue

import (
	"carefront/api"
	"carefront/doctor_treatment_plan"
	"carefront/libs/dispatch"
	"carefront/patient_file"
	"carefront/patient_visit"
	"time"
)

var (
	ExpireDuration = 15 * time.Minute
)

func initJumpBallCaseQueueListeners(dataAPI api.DataAPI, jbcqManager *api.JBCQManager) {

	// Grant temporary access to the patient case for an unclaimed case to the doctor requesting access to the case
	dispatch.Default.Subscribe(func(ev *patient_file.PatientVisitOpenedEvent) error {
		return jbcqManager.ClaimCaseForVisitIfUnclaimed(ev.PatientVisit.PatientVisitId.Int64(), ev.DoctorId, ExpireDuration)
	})

	// Extend the doctor's claim on the patient case if the doctor modifies the diagnosis associated with the case
	dispatch.Default.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		return jbcqManager.ExtendClaimOnPatientVisitDiagnosis(ev.PatientVisitId, ev.DoctorId, ExpireDuration)
	})

	// Extend the doctor's claim on the patient case if the doctor modifies any aspect of the treatment plan
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentsAddedEvent) error {
		return jbcqManager.ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.RegimenPlanAddedEvent) error {
		return jbcqManager.ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.AdviceAddedEvent) error {
		return jbcqManager.ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})

	// If the doctor successfully submits a treatment plan for an unclaimed case, the case is then considered
	// claimed by the doctor and the doctor is assigned to the case and made part of the patient's care team
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		return jbcqManager.PermanentlyAssignDoctorToCaseAndPatient(ev.VisitId, ev.DoctorId)

	})

	// If the doctor marks a case unsuitable for spruce, it is also considered claimed by the doctor
	// with the doctor permanently being assigned to the case and patient
	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		return jbcqManager.PermanentlyAssignDoctorToCaseAndPatient(ev.PatientVisitId, ev.DoctorId)
	})
}
