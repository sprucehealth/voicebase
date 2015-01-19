package cost

import "github.com/sprucehealth/backend/sku"

type VisitMessage struct {
	PatientVisitID int64
	IsFollowup     bool
	PatientID      int64
	AccountID      int64
	ItemCostID     int64
	PatientCaseID  int64
	ItemType       sku.SKU
	CardID         int64
}
