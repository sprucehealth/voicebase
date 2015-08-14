package cost

import "github.com/sprucehealth/backend/common"

type VisitMessage struct {
	PatientVisitID int64
	IsFollowup     bool
	PatientID      common.PatientID
	AccountID      int64
	ItemCostID     int64
	PatientCaseID  int64
	SKUType        string
	CardID         int64
}
