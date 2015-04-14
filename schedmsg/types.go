package schedmsg

import (
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/messages"
)

type CaseMessage struct {
	PatientCaseID  int64
	ProviderID     int64
	SenderPersonID int64
	SenderRole     string
	Message        string
	Attachments    []*messages.Attachment
}

type TreatmentPlanMessage struct {
	TreatmentPlanID int64
	MessageID       int64
}

func (c *CaseMessage) TypeName() string {
	return common.SMCaseMessageType
}

func (c *TreatmentPlanMessage) TypeName() string {
	return common.SMTreatmanPlanMessageType
}

var (
	ScheduledMsgTypes = map[string]reflect.Type{
		common.SMCaseMessageType:         reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&CaseMessage{})).Interface()),
		common.SMTreatmanPlanMessageType: reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&TreatmentPlanMessage{})).Interface()),
	}
)
