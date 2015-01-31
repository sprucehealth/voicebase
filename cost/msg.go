package cost

type VisitMessage struct {
	PatientVisitID int64
	IsFollowup     bool
	PatientID      int64
	AccountID      int64
	ItemCostID     int64
	PatientCaseID  int64
	SKUType        string
	CardID         int64
}
