package appevent

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
)

// AppEvent is an even sent from an app/client
type AppEvent struct {
	Action     string
	Resource   string
	ResourceID int64
	AccountID  int64
	Role       string
}

// Events implements analytics.Eventer to create a version of the event for analytics
func (e *AppEvent) Events() []analytics.Event {
	return []analytics.Event{
		&analytics.ServerEvent{
			Event:     "app_event",
			Timestamp: analytics.Time(time.Now()),
			AccountID: e.AccountID,
			Role:      e.Role,
			ExtraJSON: analytics.JSONString(struct {
				Action     string `json:"action"`
				Resource   string `json:"resource"`
				ResourceID int64  `json:"resource_id,string"`
			}{
				Action:     e.Action,
				Resource:   e.Resource,
				ResourceID: e.ResourceID,
			}),
		},
	}
}
