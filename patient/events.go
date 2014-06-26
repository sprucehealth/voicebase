package patient

import "carefront/common"

type CareTeamAssingmentEvent struct {
	PatientId   int64
	Assignments []*common.CareProviderAssignment
}

type AccountLoggedOutEvent struct {
	AccountId int64
}
