package models

import "time"

type Visit struct {
	ID                 VisitID
	Name               string
	LayoutVersionID    string
	EntityID           string
	Submitted          bool
	SubmittedTimestamp *time.Time
	Created            time.Time
}
