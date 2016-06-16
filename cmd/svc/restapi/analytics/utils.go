package analytics

import (
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
)

var ErrInvalidEventName = errors.New("analytics: invalid event name")

var (
	invalidEventNameCharRE = regexp.MustCompile(`[^` + ValidEventNameChar + `]`)
	dashRunRE              = regexp.MustCompile(`\-+`)
)

// JSONString returns the JSON version of the given value as a string. On
// error an empty string is returned and the error is logged. This function
// is useful for simplifying the use of the ExtraJSON in events.
func JSONString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		golog.Errorf(err.Error())
		return ""
	}
	return string(b)
}

// BadAnalyticsEvent creates and returns an Event that can be used to record
// invalid analytics events. This allows tracking the reason and number
// of dropped events.
func BadAnalyticsEvent(source, eventType, name, reason string) Event {
	return &ServerEvent{
		Event:     "bad_analytics_event",
		Timestamp: Time(time.Now()),
		ExtraJSON: JSONString(struct {
			Source string `json:"source"`
			Type   string `json:"type"`
			Name   string `json:"name"`
			Reason string `json:"reason"`
		}{
			Source: source,
			Type:   eventType,
			Name:   name,
			Reason: reason,
		}),
	}
}

// MangleEventName attempts to clean up an event name and
// makes sure it's valid.
func MangleEventName(name string) (string, error) {
	if len(name) == 0 {
		return name, ErrInvalidEventName
	}
	name = invalidEventNameCharRE.ReplaceAllString(name, "-")
	name = dashRunRE.ReplaceAllString(name, "-")
	if name == "-" {
		return name, ErrInvalidEventName
	}
	return name, nil
}
