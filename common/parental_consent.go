package common

// ParentalConsent represents a parent/child relationship having to do with consent to treatment.
type ParentalConsent struct {
	ParentPatientID int64
	Consented       bool
	Relationship    string
}
