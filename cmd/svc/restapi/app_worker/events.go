package app_worker

import "github.com/sprucehealth/backend/cmd/svc/restapi/common"

type RxTransmissionErrorEvent struct {
	Patient      *common.Patient
	ProviderID   int64
	ProviderRole string
	ItemID       int64
	EventType    common.ERxSourceType
}

type RefillRequestCreatedEvent struct {
	Patient         *common.Patient
	DoctorID        int64
	RefillRequestID int64
	Status          string
}
