package notification

// DeviceRegistrationInfo represents the information required by the notification service to register a new device for push notifications
type DeviceRegistrationInfo struct {
	ExternalGroupID string `json:"external_group_id"`
	DeviceToken     string `json:"device_token"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	AppVersion      string `json:"app_version"`
	Device          string `json:"device"`
	DeviceModel     string `json:"device_model"`
	DeviceID        string `json:"device_id"`
}

// Notification represents the information to be transformed into a notification
type Notification struct {
	ShortMessage     string   `json:"short_message"`
	ThreadID         string   `json:"thread_id"`
	OrganizationID   string   `json:"organization_id"`
	EntitiesToNotify []string `json:"entities_to_notify"`
}
