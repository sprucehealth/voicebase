package medrecord

import "github.com/sprucehealth/backend/common"

type queueMessage struct {
	MedicalRecordID int64            `json:"medical_record_id"`
	PatientID       common.PatientID `json:"patient_id"`
}
