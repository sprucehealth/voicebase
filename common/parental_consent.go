package common

// ParentalConsent represents a parent/child relationship having to do with consent to treatment.
type ParentalConsent struct {
	ParentPatientID PatientID
	Consented       bool
	Relationship    string
}
