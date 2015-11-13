package messages

import "github.com/sprucehealth/backend/common"

// PostEvent is an event that is dispatched after a case message is sent.
type PostEvent struct {
	// Message is the case message that was sent.
	Message *common.CaseMessage
	// Person is the person sending the case message.
	Person *common.Person
	// Case is the thread in which the case message was sent.
	Case *common.PatientCase
	// IsAutomated indicates whether the message sent was an automated or manual one.
	IsAutomated bool
}

// CaseAssignEvent is an event that is dispatched when a case assignment occurs.
type CaseAssignEvent struct {
	// Message indicates the message that was sent.
	Message *common.CaseMessage
	// Person indicates the person sending the mesasge.
	Person *common.Person
	// MA indicates the care coordinator on case. Optional
	MA *common.Doctor
	// Doctor indicates the doctor on case. Optional.
	Doctor *common.Doctor
	// Case indicates the case that owns the assignment.
	Case *common.PatientCase
	// IsAutomated indicates whether the case assignment was automated or explicit.
	IsAutomated bool
}
