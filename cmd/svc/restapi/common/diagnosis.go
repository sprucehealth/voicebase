package common

import (
	"time"

	"github.com/sprucehealth/backend/encoding"
)

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
	CodeID          string
	LayoutVersionID *int64
}

type DiagnosisDetailsIntake struct {
	ID      int64
	CodeID  string
	Layout  Typed
	Version *encoding.Version
	Created time.Time
	Active  bool
}
