package patient_file

import "github.com/sprucehealth/backend/common"

type PatientVisitOpenedEvent struct {
	PatientVisit *common.PatientVisit
	PatientId    int64
	DoctorId     int64
}
