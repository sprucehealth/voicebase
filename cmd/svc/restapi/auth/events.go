package auth

import "github.com/sprucehealth/backend/device"

type AuthenticatedEvent struct {
	AccountID     int64
	SpruceHeaders *device.SpruceHeaders
}
