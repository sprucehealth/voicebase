package auth

import "github.com/sprucehealth/backend/apiservice"

type AuthenticatedEvent struct {
	AccountID     int64
	SpruceHeaders *apiservice.SpruceHeaders
}
