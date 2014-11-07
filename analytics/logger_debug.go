package analytics

import (
	"testing"

	"github.com/sprucehealth/backend/libs/golog"
)

type DebugLogger struct {
	T *testing.T
}

func (l DebugLogger) WriteEvents(events []Event) {
	for _, e := range events {
		if l.T != nil {
			l.T.Logf("%s %+v", e.Category(), e)
		} else {
			golog.Debugf("%s %+v", e.Category(), e)
		}
	}
}

func (DebugLogger) Start() error {
	return nil
}

func (DebugLogger) Stop() error {
	return nil
}
