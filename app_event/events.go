package app_event

type AppEvent struct {
	Action     string
	Resource   string
	ResourceID int64
	AccountID  int64
	Role       string
}
