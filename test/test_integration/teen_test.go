package test_integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/test"
)

func TestTeenFlow(t *testing.T) {
	testData := SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	// Assuming test setup created at least one pathway
	pathway, err := testData.DataAPI.Pathway(1, api.POWithDetails)
	test.OK(t, err)
	test.OK(t, testData.DataAPI.UpdatePathway(pathway.ID, &api.PathwayUpdate{
		Details: &common.PathwayDetails{
			AgeRestrictions: []*common.PathwayAgeRestriction{
				{
					MaxAgeOfRange: ptr.Int(12),
					VisitAllowed:  false,
					Alert: &common.PathwayAlert{
						Message: "Sorry!",
					},
				},
				{
					MaxAgeOfRange: ptr.Int(17),
					VisitAllowed:  true,
				},
				{
					MaxAgeOfRange: ptr.Int(70),
					VisitAllowed:  true,
				},
				{
					MaxAgeOfRange: nil,
					VisitAllowed:  false,
					Alert: &common.PathwayAlert{
						Message: "Not Sorry!",
					},
				},
			},
		},
	}))

	pc := PatientClient(testData, t, 0)
	suRes, err := pc.SignUp(&patient.SignupPatientRequestData{
		Email:     "patient@sprucehealth.com",
		Password:  "12345",
		FirstName: "first",
		LastName:  "last",
		DOB:       fmt.Sprintf("%d-12-12", time.Now().Year()-16),
		Gender:    "female",
		ZipCode:   "94105",
		Phone:     "415-555-1212",
		StateCode: "CA",
	})
	test.OK(t, err)
	test.Assert(t, suRes.Token != "", "Auth token not returned")
	patientID := suRes.Patient.ID.Int64()
	pc.AuthToken = suRes.Token

	cvRes, err := pc.CreatePatientVisit(pathway.Tag, 0, SetupTestHeaders())
	test.OK(t, err)
	test.Equals(t, common.PVStatusOpen, cvRes.Status)
	test.Equals(t, false, cvRes.ParentalConsentGranted)
	test.Equals(t, true, cvRes.ParentalConsentRequired)
	test.Assert(t, cvRes.ParentalConsentInfo != nil, "Parental consent info missing")

	test.OK(t, pc.SelectPharmacy(&pharmacy.PharmacyData{}))
	test.OK(t, pc.Update(&patient.UpdateRequest{
		Address: &common.Address{
			AddressLine1: "116 New Montgomery St",
			AddressLine2: "Suite 250",
			City:         "San Francisco",
			State:        "CA",
			ZipCode:      "94105",
		},
	}))

	cvRes, err = pc.Visit(cvRes.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusOpen, cvRes.Status)
	test.Equals(t, false, cvRes.ParentalConsentGranted)
	test.Equals(t, true, cvRes.ParentalConsentRequired)
	test.Assert(t, cvRes.ParentalConsentInfo != nil, "Parental consent info missing")

	// Shouldn't be able to submit the visit
	err = pc.SubmitPatientVisit(cvRes.PatientVisitID)
	test.Assert(t, err != nil, "Should not be able to submit visit requiring consent before consent is granted")
	test.Assert(t, strings.Contains(err.Error(), "consent"), "Expected consent failure, got %s", err)

	test.OK(t, pc.ParentalConsentStepReached(cvRes.PatientVisitID))

	// Still shouldn't be able to submit the visit
	err = pc.SubmitPatientVisit(cvRes.PatientVisitID)
	test.Assert(t, err != nil, "Should not be able to submit visit requiring consent before consent is granted")
	test.Assert(t, strings.Contains(err.Error(), "consent"), "Expected consent failure, got %s", err)

	// Make sure the status has been updated properly
	cvRes, err = pc.Visit(cvRes.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusPendingParentalConsent, cvRes.Status)
	test.Equals(t, false, cvRes.ParentalConsentGranted)
	test.Equals(t, true, cvRes.ParentalConsentRequired)
	test.Assert(t, cvRes.ParentalConsentInfo != nil, "Parental consent info missing")

	// Sign up parent
	suRes, err = pc.SignUp(&patient.SignupPatientRequestData{
		Email:     "parent@sprucehealth.com",
		Password:  "12345",
		FirstName: "first",
		LastName:  "last",
		DOB:       fmt.Sprintf("%d-12-12", time.Now().Year()-35),
		Gender:    "male",
		ZipCode:   "94105",
		Phone:     "415-555-1212",
		StateCode: "CA",
	})
	test.OK(t, err)
	parentPatientID := suRes.Patient.ID.Int64()

	test.OK(t, testData.DataAPI.LinkParentChild(parentPatientID, patientID, "sensei"))

	// Still shouldn't be able to submit the visit
	err = pc.SubmitPatientVisit(cvRes.PatientVisitID)
	test.Assert(t, err != nil, "Should not be able to submit visit requiring consent before consent is granted")
	test.Assert(t, strings.Contains(err.Error(), "consent"), "Expected consent failure, got %s", err)

	test.OK(t, testData.DataAPI.GrantParentChildConsent(parentPatientID, patientID))

	// Make sure visit info updates to reflect consent being granted
	cvRes, err = pc.Visit(cvRes.PatientVisitID)
	test.OK(t, err)
	test.Equals(t, common.PVStatusReceivedParentalConsent, cvRes.Status)
	test.Equals(t, true, cvRes.ParentalConsentGranted)
	test.Equals(t, true, cvRes.ParentalConsentRequired)
	test.Assert(t, cvRes.ParentalConsentInfo == nil, "Parental consent info shouldn't be included when no consent required")

	// Finally now should be able to submit the visit
	err = pc.SubmitPatientVisit(cvRes.PatientVisitID)
	test.Assert(t, err == nil, "Should be able to submit visit after consent is granted")
}
