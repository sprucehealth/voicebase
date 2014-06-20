package patient_file

import "carefront/common"

type PatientVisitOpenedEvent struct {
	PatientVisit *common.PatientVisit
	PatientId    int64
	DoctorId     int64
}
