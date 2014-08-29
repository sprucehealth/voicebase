package analytics

import (
	"github.com/sprucehealth/backend/libs/golog"
)

type DebugLogger struct{}

func (DebugLogger) WriteEvents(events []Event) {
	for _, e := range events {
		golog.Debugf("%s %+v", e.Category(), e)
	}
}

func (DebugLogger) Start() error {
	return nil
}

func (DebugLogger) Stop() error {
	return nil
}
