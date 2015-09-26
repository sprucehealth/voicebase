package analytics

// DebugLogger is an analytics logger that supports using a custom log function
type DebugLogger struct {
	Logf func(f string, a ...interface{})
}

// WriteEvents writes events to the provided Log if non-nil
func (l DebugLogger) WriteEvents(events []Event) {
	for _, e := range events {
		if l.Logf != nil {
			l.Logf("%s %+v", e.Category(), e)
		}
	}
}

// Start is a noop;
func (DebugLogger) Start() error {
	return nil
}

// Stop is a noop
func (DebugLogger) Stop() error {
	return nil
}
