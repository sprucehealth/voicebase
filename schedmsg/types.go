package schedmsg

import (
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/messages"
)

type EmailMessage struct {
	email.Email
}

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

func (e *EmailMessage) TypeName() string {
	return common.SMEmailMessageType
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
		common.SMEmailMessageType:        reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&EmailMessage{})).Interface()),
		common.SMTreatmanPlanMessageType: reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&TreatmentPlanMessage{})).Interface()),
	}
)
