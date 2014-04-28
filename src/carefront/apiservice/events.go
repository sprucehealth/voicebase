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
	PatientId int64
	DoctorId  int64
	VisitId   int64
	Patient   *common.Patient // Setting Patient is an optional optimization. If this is nil then PatientId can be used.
}
