package patient_file

import "github.com/sprucehealth/backend/common"

type PatientVisitOpenedEvent struct {
	PatientVisit *common.PatientVisit
	PatientID    int64
	DoctorID     int64
	Role         string
}
