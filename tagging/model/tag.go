package model

import "time"

type Tag struct {
	ID   int64
	Text string
}

type TagMembership struct {
	TagID       int64
	CaseID      *int64
	TriggerTime *time.Time
	Hidden      bool
}
