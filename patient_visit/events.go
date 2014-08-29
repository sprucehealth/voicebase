package patient_visit

import "github.com/sprucehealth/backend/common"

type VisitStartedEvent struct {
	PatientId     int64
	VisitId       int64
	PatientCaseId int64
}

type VisitSubmittedEvent struct {
	PatientId     int64
	VisitId       int64
	PatientCaseId int64
	Visit         *common.PatientVisit
}

type VisitChargedEvent struct {
	PatientID     int64
	VisitID       int64
	PatientCaseID int64
}

type DiagnosisModifiedEvent struct {
	DoctorId        int64
	PatientVisitId  int64
	TreatmentPlanId int64
	Diagnosis       string
	PatientCaseId   int64
}

type PatientVisitMarkedUnsuitableEvent struct {
	PatientVisitId int64
	CaseID         int64
	DoctorId       int64
	InternalReason string
}
