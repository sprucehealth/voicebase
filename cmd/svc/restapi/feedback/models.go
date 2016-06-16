package feedback

import (
	"time"

	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
)

// PatientFeedback represents a piece of feedback
// from the patient.
type PatientFeedback struct {
	// ID represents the unique identifier for the patient feedback.
	ID int64
	// PatientID represents the patient that this feedback was authored by.
	PatientID common.PatientID

	// Rating represents the patient provided rating.
	Rating *int

	// Comment represents the comment the patient provided.
	Comment *string

	// Created represents the time at which the feedback was created.
	Created time.Time

	// Dismissed indicates whether or not the patient dismissed
	// the attempt to collect a rating.
	Dismissed bool

	// Pending indicates whether or not the feedback is yet
	// to be given by patient.
	Pending bool
}
