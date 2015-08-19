package messages

import "github.com/sprucehealth/backend/common"

type PostEvent struct {
	Message *common.CaseMessage
	Person  *common.Person
	Case    *common.PatientCase
}

type CaseAssignEvent struct {
	Message *common.CaseMessage
	Person  *common.Person
	MA      *common.Doctor
	Doctor  *common.Doctor
	Case    *common.PatientCase
}
