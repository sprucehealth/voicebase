package app_worker

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorEvent struct {
	DoctorId  int64
	ItemId    int64
	EventType common.ERxSourceType
}

type RefillRequestCreatedEvent struct {
	DoctorId        int64
	RefillRequestId int64
	Status          string
}
