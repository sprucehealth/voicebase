package common

import "time"

type Diagnosis struct {
	ID          int64
	Code        string
	Description string
	Billable    bool
}

type VisitDiagnosisSet struct {
	ID               int64
	VisitID          int64
	DoctorID         int64
	Active           bool
	Created          time.Time
	Notes            string
	Unsuitable       bool
	UnsuitableReason string
	Items            []*VisitDiagnosisItem
}

type VisitDiagnosisItem struct {
	ID              int64
	CodeID          int64
	LayoutVersionID *int64
}

type DiagnosisDetailsIntake struct {
	ID      int64
	CodeID  int64
	Code    string
	Layout  Typed
	Version *Version
	Created time.Time
	Active  bool
}
