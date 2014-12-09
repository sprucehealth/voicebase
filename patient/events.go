package patient

import "github.com/sprucehealth/backend/common"

type CareTeamAssingmentEvent struct {
	PatientID   int64
	Assignments []*common.CareProviderAssignment
}

type AccountLoggedOutEvent struct {
	AccountID int64
}

type VisitStartedEvent struct {
	PatientID     int64
	VisitID       int64
	PatientCaseID int64
}

type VisitSubmittedEvent struct {
	PatientID     int64
	AccountID     int64
	VisitID       int64
	PatientCaseID int64
	Visit         *common.PatientVisit
	CardID        int64
}
