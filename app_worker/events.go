package app_worker

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorEvent struct {
	DoctorID  int64
	ItemID    int64
	EventType common.ERxSourceType
}

type RefillRequestCreatedEvent struct {
	DoctorID        int64
	RefillRequestID int64
	Status          string
}
