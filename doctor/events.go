package doctor

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorResolvedEvent struct {
	DoctorId  int64
	ItemId    int64
	EventType common.ERxSourceType
}

type RefillRequestResolvedEvent struct {
	DoctorId        int64
	RefillRequestId int64
	Status          string
}

// TODO: Remove this event once we decouple the types of notifications
// from the application-specific events
type NotifyDoctorOfUnclaimedCaseEvent struct {
	DoctorID int64
}
