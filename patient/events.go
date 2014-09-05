package patient

import "github.com/sprucehealth/backend/common"

type CareTeamAssingmentEvent struct {
	PatientId   int64
	Assignments []*common.CareProviderAssignment
}

type AccountLoggedOutEvent struct {
	AccountId int64
}

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
