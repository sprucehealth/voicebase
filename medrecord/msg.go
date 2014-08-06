package medrecord

type queueMessage struct {
	MedicalRecordID int64 `json:"medical_record_id"`
	PatientID       int64 `json:"patient_id"`
}
