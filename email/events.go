package email

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
)

type SendEvent struct {
	Recipients []*api.Recipient
	Type       string
}

func (se *SendEvent) Events() []analytics.Event {
	if len(se.Recipients) == 0 {
		return nil
	}
	now := time.Now()
	ev := make([]analytics.Event, len(se.Recipients))
	for i, r := range se.Recipients {
		ev[i] = &analytics.ServerEvent{
			Event:     "email-send",
			Timestamp: analytics.Time(now),
			AccountID: r.AccountID,
			ExtraJSON: analytics.JSONString(
				struct {
					Type string `json:"type"`
				}{
					Type: se.Type,
				},
			),
		}
	}
	return ev
}
