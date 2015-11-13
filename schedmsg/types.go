package schedmsg

import (
	"reflect"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/messages"
)

// CaseMessage represents a case message in its complete
// form that is being scheduled.
type CaseMessage struct {
	// PatientCaseID identifies the case thread to which the message belongs.
	PatientCaseID int64
	// ProviderID identifies the provider sending the message.
	ProviderID int64
	// SenderPersonID is the id of the sender in the message thread.
	SenderPersonID int64
	// SenderRole identifies the type of care provider (CC or DOCTOR).
	SenderRole string
	// Message identifies the body of the message being sent.
	Message string
	// Attachments represents the list of attachemnts in the message.
	Attachments []*messages.Attachment
	// IsAutomated indicates whether the message was automatically scheduled based on some
	// trigger event or manually written by the provider.
	IsAutomated bool
}

// TreatmentPlanMessage represents a scheduled message originating from a treatment plan.
type TreatmentPlanMessage struct {
	// TreatmentPlanID identifies the treatment plan where the scheduled message originates from.
	TreatmentPlanID int64
	// MessageID identifies the message to be sent to the patient at the time of trigger.
	MessageID int64
}

// TypeName represents the type of scheduled message.
func (c *CaseMessage) TypeName() string {
	return common.SMCaseMessageType
}

// TypeName represents the type of scheduled message.
func (c *TreatmentPlanMessage) TypeName() string {
	return common.SMTreatmanPlanMessageType
}

var (
	//ScheduledMsgTypes is a mapping of schedules message type name to the type itself. Useful for instantiating the right concrete
	// scheduled message type.
	ScheduledMsgTypes = map[string]reflect.Type{
		common.SMCaseMessageType:         reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&CaseMessage{})).Interface()),
		common.SMTreatmanPlanMessageType: reflect.TypeOf(reflect.Indirect(reflect.ValueOf(&TreatmentPlanMessage{})).Interface()),
	}
)
