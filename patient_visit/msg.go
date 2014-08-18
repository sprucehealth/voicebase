package patient_visit

type visitMessage struct {
	PatientVisitID int64
	PatientID      int64
	PatientCaseID  int64
	ItemType       string
}
