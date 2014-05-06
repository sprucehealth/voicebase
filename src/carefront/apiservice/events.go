package apiservice

import "carefront/common"

type VisitStartedEvent struct {
	PatientId int64
	VisitId   int64
}

type VisitSubmittedEvent struct {
	PatientId int64
	DoctorId  int64
	VisitId   int64
}

type VisitReviewSubmittedEvent struct {
	PatientId       int64
	DoctorId        int64
	VisitId         int64
	TreatmentPlanId int64
	Status          string
	Patient         *common.Patient // Setting Patient is an optional optimization. If this is nil then PatientId can be used.
}

type TreatmentsAddedEvent struct {
	PatientVisitId int64
	DoctorId       int64
	Treatments     []*common.Treatment
}

type RegimenPlanAddedEvent struct {
	PatientVisitId int64
	DoctorId       int64
	RegimenPlan    *common.RegimenPlan
}

type AdviceAddedEvent struct {
	PatientVisitId int64
	DoctorId       int64
	Advice         *common.Advice
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
