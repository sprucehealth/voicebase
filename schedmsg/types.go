package schedmsg

import (
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/messages"
)

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
