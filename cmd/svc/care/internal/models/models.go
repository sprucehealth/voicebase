package models

type Visit struct {
	ID                 VisitID
	Name               string
	LayoutVersionID    string
	EntityID           string
	Submitted          bool
	SubmittedTimestamp *uint64
	Created            int64
}
