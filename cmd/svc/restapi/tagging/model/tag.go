package model

import "time"

type Tag struct {
	ID     int64
	Text   string
	Common bool
}

type TagUpdate struct {
	ID     int64
	Common bool
}

type TagMembership struct {
	TagID       int64
	CaseID      *int64
	TriggerTime *time.Time
	Created     time.Time
	Hidden      bool
}

type TagMembershipUpdate struct {
	TagID       int64
	CaseID      *int64
	TriggerTime *time.Time
}

type TagSavedSearch struct {
	ID          int64
	Title       string
	Query       string
	CreatedTime time.Time
}
