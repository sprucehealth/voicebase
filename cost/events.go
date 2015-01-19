package cost

type VisitChargedEvent struct {
	AccountID     int64
	PatientID     int64
	VisitID       int64
	IsFollowup    bool
	PatientCaseID int64
}
