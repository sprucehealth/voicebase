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

// DeviceDeregistrationInfo represents the information required by the notification service to unregister an existing device
type DeviceDeregistrationInfo struct {
	DeviceID string `json:"device_id"`
}

// Type represents the type associated with the notification info
type Type string

const (
	// DeprecatedNewMessageOnThread is for backwards compatibility and should be removed once all producers have been updated
	DeprecatedNewMessageOnThread Type = ""
	// NewMessageOnInternalThread represents that the notification is for activity on an internal thread
	NewMessageOnInternalThread Type = "new_message_on_internal_thread"
	// NewMessageOnExternalThread represents that the notification is for activity on an external thread
	NewMessageOnExternalThread Type = "new_message_on_external_thread"
	// IncomingIPCall notifies if an incoming video or voip call
	IncomingIPCall Type = "incoming_ipcall"
	// BadgeUpdate notification is a silent/empty notification just to update the app badge count
	BadgeUpdate Type = "badge_update"
)

// Notification represents the information to be transformed into a notification
type Notification struct {
	ShortMessages    map[string]string `json:"short_message"`
	CollapseKey      string            `json:"collapse_key"`
	DedupeKey        string            `json:"dedupe_key"`
	OrganizationID   string            `json:"organization_id"`
	EntitiesToNotify []string          `json:"entities_to_notify"`
	Type             Type              `json:"type"`

	// For NewMessageOnInternalThread and NewMessageOnExternalThread
	UnreadCounts         map[string]int      `json:"unread_counts"`
	SavedQueryID         string              `json:"saved_query_id"`
	ThreadID             string              `json:"thread_id"`
	MessageID            string              `json:"message_id"`
	EntitiesAtReferenced map[string]struct{} `json:"entities_at_referenced"`

	// For IncomingIPCall
	CallID string `json:"call_id"`
}
