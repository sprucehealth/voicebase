package model

import "time"

// AttributionData represents the external information to associate with an account/device
type AttributionData struct {
	ID           int64
	AccountID    *int64
	DeviceID     *string
	Data         map[string]interface{}
	CreationDate time.Time
	LastModified time.Time
}
