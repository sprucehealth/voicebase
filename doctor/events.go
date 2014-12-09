package doctor

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorResolvedEvent struct {
	DoctorID  int64
	ItemID    int64
	EventType common.ERxSourceType
}

type RefillRequestResolvedEvent struct {
	DoctorID        int64
	RefillRequestID int64
	Status          string
}

type DoctorLoggedInEvent struct {
	Doctor *common.Doctor
}

// TODO: Remove this event once we decouple the types of notifications
// from the application-specific events
type NotifyDoctorOfUnclaimedCaseEvent struct {
	DoctorID int64
}
