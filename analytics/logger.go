package analytics

import "time"

const timeFormat = "2006-01-02 15:04:05.000"

type Logger interface {
	WriteEvents([]Event)
	Start() error
	Stop() error
}

type NullLogger struct{}

func (NullLogger) WriteEvents([]Event) {}
func (NullLogger) Start() error        { return nil }
func (NullLogger) Stop() error         { return nil }

type Time time.Time

func (t Time) MarshalText() ([]byte, error) {
	return []byte(time.Time(t).UTC().Format(timeFormat)), nil
}

func (t *Time) UnmarshalText(data []byte) error {
	tt, err := time.Parse(timeFormat, string(data))
	if err != nil {
		return err
	}
	*t = Time(tt)
	return nil
}

type Event interface {
	Category() string
	Time() time.Time
}
