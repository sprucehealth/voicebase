package analytics

import "time"

const timeFormat = "2006-01-02 15:04:05"

type Logger interface {
	WriteEvents(category string, events []interface{})
	Start() error
	Stop() error
}

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

type ClientEvent struct {
	ID           int64   `json:"id"`
	Event        string  `json:"event"`
	Time         Time    `json:"time"`
	SessionID    string  `json:"session_id"`
	DeviceID     string  `json:"device_id"`
	AccountID    int64   `json:"account_id,omitempty"`
	PatientID    int64   `json:"patient_id,omitempty"`
	VisitID      int64   `json:"visit_id,omitempty"`
	TimeSpent    int     `json:"time_spent,omitempty"`
	AppType      string  `json:"app_type,omitempty"`
	AppEnv       string  `json:"app_env,omitempty"`
	AppVersion   string  `json:"app_version,omitempty"`
	AppBuild     string  `json:"app_build,omitempty"`
	OS           string  `json:"os,omitempty"`
	OSVersion    string  `json:"os_version,omitempty"`
	DeviceType   string  `json:"device_type,omitempty"`
	DeviceModel  string  `json:"device_model,omitempty"`
	ScreenWidth  int     `json:"screen_width,omitempty"`
	ScreenHeight int     `json:"screen_height,omitempty"`
	DPI          int     `json:"dpi,omitempty"`
	Scale        float64 `json:"scale,omitempty"`
	ExtraJSON    []byte  `json:"extra_json,omitempty"`
}
