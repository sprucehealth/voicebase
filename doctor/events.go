package doctor

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorResolvedEvent struct {
	Patient   *common.Patient
	Doctor    *common.Doctor
	ItemID    int64
	EventType common.ERxSourceType
}

type RefillRequestResolvedEvent struct {
	Patient         *common.Patient
	Doctor          *common.Doctor
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
