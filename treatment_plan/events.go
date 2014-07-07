package treatment_plan

import "github.com/sprucehealth/backend/common"

type TreatmentPlanOpenedEvent struct {
	RoleType      string
	RoleId        int64
	TreatmentPlan *common.TreatmentPlan
}
