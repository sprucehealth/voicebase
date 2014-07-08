package app_event

type AppEvent struct {
	Action     string
	Resource   string
	ResourceId int64
	AccountId  int64
	Role       string
}
