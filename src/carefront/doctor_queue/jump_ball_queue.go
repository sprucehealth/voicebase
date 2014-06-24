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

func initJumpBallCaseQueueListeners(dataAPI api.DataAPI) {

	// Grant temporary access to the patient case for an unclaimed case to the doctor requesting access to the case
	dispatch.Default.Subscribe(func(ev *patient_file.PatientVisitOpenedEvent) error {
		return api.GetJBCQManager(dataAPI).ClaimCaseForVisitIfUnclaimed(ev.PatientVisit.PatientVisitId.Int64(), ev.DoctorId, ExpireDuration)
	})

	// Extend the doctor's claim on the patient case if the doctor modifies the diagnosis associated with the case
	dispatch.Default.Subscribe(func(ev *patient_visit.DiagnosisModifiedEvent) error {
		return api.GetJBCQManager(dataAPI).ExtendClaimOnPatientVisitDiagnosis(ev.PatientVisitId, ev.DoctorId, ExpireDuration)
	})

	// Extend the doctor's claim on the patient case if the doctor modifies any aspect of the treatment plan
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentsAddedEvent) error {
		return api.GetJBCQManager(dataAPI).ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.RegimenPlanAddedEvent) error {
		return api.GetJBCQManager(dataAPI).ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.AdviceAddedEvent) error {
		return api.GetJBCQManager(dataAPI).ExtendClaimOnTreatmentPlanModification(ev.TreatmentPlanId, ev.DoctorId, ExpireDuration)
	})

	// If the doctor successfully submits a treatment plan for an unclaimed case, the case is then considered
	// claimed by the doctor and the doctor is assigned to the case and made part of the patient's care team
	dispatch.Default.Subscribe(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		return api.GetJBCQManager(dataAPI).PermanentlyAssignDoctorToCaseAndPatient(ev.VisitId, ev.DoctorId)

	})

	// If the doctor marks a case unsuitable for spruce, it is also considered claimed by the doctor
	// with the doctor permanently being assigned to the case and patient
	dispatch.Default.Subscribe(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		return api.GetJBCQManager(dataAPI).PermanentlyAssignDoctorToCaseAndPatient(ev.PatientVisitId, ev.DoctorId)
	})
}
