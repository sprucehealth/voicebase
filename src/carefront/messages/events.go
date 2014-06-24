package messages

import "carefront/common"

type PostEvent struct {
	Message *common.CaseMessage
	Person  *common.Person
	Case    *common.PatientCase
}

type ReadEvent struct {
	CaseID int64
	Person *common.Person
}
