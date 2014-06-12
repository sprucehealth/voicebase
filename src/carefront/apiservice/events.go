package apiservice

import "carefront/common"

type VisitReviewSubmittedEvent struct {
	PatientId       int64
	DoctorId        int64
	VisitId         int64
	TreatmentPlanId int64
	Status          string
	Patient         *common.Patient // Setting Patient is an optional optimization. If this is nil then PatientId can be used.
}

type RxTransmissionErrorResolvedEvent struct {
	DoctorId  int64
	ItemId    int64
	EventType common.StatusEventCheckType
}

type RefillRequestResolvedEvent struct {
	DoctorId        int64
	RefillRequestId int64
	Status          string
}

type DiagnosisModifiedEvent struct {
	DoctorId        int64
	PatientVisitId  int64
	TreatmentPlanId int64
}
