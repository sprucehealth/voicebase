package schedmsg

import (
	"reflect"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/messages"
)

const (
	// message types supported
	smVisitChargedEventType   = "visit_charged"
	smTreatmentPlanViewedType = "treatment_plan_viewed"
)

type schedSQSMessage struct {
	ScheduledMessageID int64
	ScheduledTime      time.Time
}

type emailMessage struct {
	email.Email
}

func (e emailMessage) TypeName() string {
	return common.SMEmailMessageType
}

type caseMessage struct {
	PatientCaseID  int64
	ProviderID     int64
	SenderPersonID int64
	SenderRole     string
	Message        string
	Attachments    []*messages.Attachment
}

func (c caseMessage) TypeName() string {
	return common.SMCaseMessageType
}

var (
	scheduledMsgTypes = map[string]reflect.Type{
		common.SMCaseMessageType:  reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&caseMessage{})).Interface()),
		common.SMEmailMessageType: reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&emailMessage{})).Interface()),
	}
)
