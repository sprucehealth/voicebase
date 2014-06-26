package app_worker

import "carefront/common"

type RxTransmissionErrorEvent struct {
	DoctorId  int64
	ItemId    int64
	EventType common.StatusEventCheckType
}

type RefillRequestCreatedEvent struct {
	DoctorId        int64
	RefillRequestId int64
	Status          string
}
