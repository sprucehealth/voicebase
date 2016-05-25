package models

import "time"

type Visit struct {
	ID                 VisitID
	Name               string
	LayoutVersionID    string
	EntityID           string
	OrganizationID     string
	Submitted          bool
	SubmittedTimestamp *time.Time
	Created            time.Time
	CreatorID          string
	Triaged            bool
	TriagedTimestamp   *time.Time
}
