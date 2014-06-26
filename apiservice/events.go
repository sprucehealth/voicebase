package apiservice

import "github.com/sprucehealth/backend/common"

type RxTransmissionErrorResolvedEvent struct {
	DoctorId  int64
	ItemId    int64
	EventType common.StatusEventCheckType
}

type RefillRequestResolvedEvent struct {
	DoctorId        int64
	RefillRequestId int64
	Status          string
}
