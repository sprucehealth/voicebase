package apiservice

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_LayoutForPatientVisit struct {
	api.DataAPI
	layoutData   []byte
	doctor       *common.Doctor
	doctorMember *common.CareProviderAssignment
	patient      *common.Patient
	patientCase  *common.PatientCase
}

func (m *mockDataAPI_LayoutForPatientVisit) GetPatientLayout(layoutVersionID, languageID int64) (*api.LayoutVersion, error) {
	return &api.LayoutVersion{
		Layout: m.layoutData,
	}, nil
}
func (m *mockDataAPI_LayoutForPatientVisit) GetPatientCaseFromID(caseID int64) (*common.PatientCase, error) {
	return m.patientCase, nil
}
func (m *mockDataAPI_LayoutForPatientVisit) GetActiveCareTeamMemberForCase(role string, caseID int64) (*common.CareProviderAssignment, error) {
	return m.doctorMember, nil
}
func (m *mockDataAPI_LayoutForPatientVisit) Patient(patientID int64, basicInfoOnly bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_LayoutForPatientVisit) Doctor(doctorID int64, basicInfoOnly bool) (*common.Doctor, error) {
	return m.doctor, nil
}

// this test is to ensure that parsing a templated layout
// works as we'd expect for a case where no doctor has been picked.
func TestTemplatedLayout_NoDoctorPicked(t *testing.T) {

	layout := `{
		  "is_templated" : true,
		  "visit_overview_header": {
		    "title": "{{.CaseName}} Visit",
		    "subtitle": "With {{.Doctor.Description}}",
		    "icon_url": "{{.Doctor.SmallThumbnailURL}}"
		  },
		  "additional_message": {
		    "title": "Is there anything else you'd like to ask or share with {{.Doctor.ShortDisplayName}}?",
		    "placeholder": "It's optional but this is your chance to let the doctor know what’s on your mind."
		  },
		  "checkout": {
		  	"header_image_url" : "{{.Doctor.SmallThumbnailURL}}",
		    "header_text": "{{.Doctor.ShortDisplayName | titleDoctor}} will review your visit and create your treatment plan within 24 hours.",
		    "footer_text": "There are no surprise medical bills with Spruce. If you're unsatisfied with your visit, we'll refund the full cost."
		  },
		  "submission_confirmation": {
		    "title": "Visit Submitted",
		    "top_text": "Your {{.CaseName}} visit has been submitted.",
		    "bottom_text": "{{.Doctor.ShortDisplayName | titleDoctor }} will review your visit and respond within 24 hours.",
		    "button_title": "Continue"
		  }
	}`
	m := &mockDataAPI_LayoutForPatientVisit{
		layoutData: []byte(layout),
		patientCase: &common.PatientCase{
			Name: "Acne",
		},
		doctorMember: &common.CareProviderAssignment{
			ProviderID: 1,
		},
	}

	intakeLayout, err := GetPatientLayoutForPatientVisit(&common.PatientVisit{}, 1, m, "api.spruce.local")
	test.OK(t, err)
	test.Equals(t, "Acne Visit", intakeLayout.Header.Title)
	test.Equals(t, "With First Available Doctor", intakeLayout.Header.Subtitle)
	test.Equals(t, "", intakeLayout.Header.IconURL)
	test.Equals(t, "", intakeLayout.Checkout.HeaderImageURL)
	test.Equals(t, "Your doctor will review your visit and create your treatment plan within 24 hours.", intakeLayout.Checkout.Header)
	test.Equals(t, "Your Acne visit has been submitted.", intakeLayout.SubmissionConfirmation.Top)
	test.Equals(t, "Your doctor will review your visit and respond within 24 hours.", intakeLayout.SubmissionConfirmation.Bottom)
	test.Equals(t, "Is there anything else you'd like to ask or share with your doctor?", intakeLayout.AdditionalMessage.Title)
}

// this test is to ensure that parsing a templated layout
// works as we'd expect for a case where a doctor has been picked.
func TestTemplatedLayout_DoctorPicked(t *testing.T) {

	layout := `{
		  "is_templated" : true,
		  "visit_overview_header": {
		    "title": "{{.CaseName}} Visit",
		    "subtitle": "With {{.Doctor.Description}}",
		    "icon_url": "{{.Doctor.SmallThumbnailURL}}"
		  },
		  "additional_message": {
		    "title": "Is there anything else you’d like to ask or share with {{.Doctor.ShortDisplayName}}?",
		    "placeholder": "It’s optional but this is your chance to let the doctor know what’s on your mind."
		  },
		  "checkout": {
		  	"header_image_url" : "{{.Doctor.SmallThumbnailURL}}",
		    "header_text": "{{.Doctor.ShortDisplayName | titleDoctor}} will review your visit and create your treatment plan within 24 hours.",
		    "footer_text": "There are no surprise medical bills with Spruce. If you're unsatisfied with your visit, we'll refund the full cost."
		  },
		  "submission_confirmation": {
		    "title": "Visit Submitted",
		    "top_text": "Your {{.CaseName}} visit has been submitted.",
		    "bottom_text": "{{.Doctor.ShortDisplayName | titleDoctor}} will review your visit and respond within 24 hours.",
		    "button_title": "Continue"
		  }
	}`
	m := &mockDataAPI_LayoutForPatientVisit{
		layoutData: []byte(layout),
		patientCase: &common.PatientCase{
			Name: "Acne",
		},
		doctorMember: &common.CareProviderAssignment{
			ProviderID: 1,
		},
		doctor: &common.Doctor{
			DoctorID:         encoding.NewObjectID(2),
			ShortDisplayName: "Dr. X",
		},
	}

	intakeLayout, err := GetPatientLayoutForPatientVisit(&common.PatientVisit{}, 1, m, "api.spruce.local")
	test.OK(t, err)
	test.Equals(t, "Acne Visit", intakeLayout.Header.Title)
	test.Equals(t, "With Dr. X", intakeLayout.Header.Subtitle)
	test.Equals(t, app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2), intakeLayout.Header.IconURL)
	test.Equals(t, app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2), intakeLayout.Checkout.HeaderImageURL)
	test.Equals(t, "Dr. X will review your visit and create your treatment plan within 24 hours.", intakeLayout.Checkout.Header)
	test.Equals(t, "Your Acne visit has been submitted.", intakeLayout.SubmissionConfirmation.Top)
	test.Equals(t, "Dr. X will review your visit and respond within 24 hours.", intakeLayout.SubmissionConfirmation.Bottom)
}

// this test is to ensure that parsing of a templated layout for followup
// visits works as expected
func TestTemplatedLayout_FollowupVisit(t *testing.T) {
	layout := `{
	  "is_templated": true,
	  "visit_overview_header": {
	    "title": "Follow-up Visit",
	    "subtitle": "With {{.Doctor.ShortDisplayName}}",
	    "icon_url": "{{.Doctor.SmallThumbnailURL}}"
	  },
	  "additional_message": {
	    "title": "Is there anything else you’d like to ask or share with {{.Doctor.ShortDisplayName}}?",
	    "placeholder": "It’s optional but this is your chance to let the doctor know what’s on your mind."
	  },
	  "checkout": {
	    "header_text": "Submit your follow-up visit for {{.Doctor.ShortDisplayName}} to review",
	    "footer_text": "There are no surprise medical bills with Spruce. If you're unsatisfied with your visit, we'll refund the full cost."
	  },
	  "submission_confirmation": {
	    "title": "Visit Submitted",
	    "top_text": "Your follow-up visit has been submitted.",
	    "bottom_text": "{{.Doctor.ShortDisplayName}} will review your visit and respond within 24 hours.",
	    "button_title": "Continue"
	  },
	  "transitions": [
	    {
	      "message": "Welcome to your follow-up visit, {{.Patient.FirstName}}. We'll begin by asking you about your current treatment plan.",
	      "buttons": [
	        {
	          "button_text": "Begin",
	          "tap_url": "spruce:///action/view_next_visit_section",
	          "style": "filled"
	        }
	      ]
	    },
	    {
	      "message": "Next we'll take photos so {{.Doctor.ShortDisplayName}} can see your progress.",
	      "buttons": [
	        {
	          "button_text": "Begin",
	          "tap_url": "spruce:///action/view_next_visit_section",
	          "style": "filled"
	        }
	      ]
	    },
	    {
	      "message": "Next we'll make sure your medical history is up to date.",
	      "buttons": [
	        {
	          "button_text": "Continue",
	          "tap_url": "spruce:///action/view_next_visit_section",
	          "style": "filled"
	        }
	      ]
	    },
	    {
	      "message": "That's all the information {{.Doctor.ShortDisplayName}} needs for your follow-up.",
	      "buttons": [
	        {
	          "button_text": "Continue",
	          "tap_url": "spruce:///action/view_next_visit_section",
	          "style": "filled"
	        }
	      ]
	    }
	  ]
	}`

	m := &mockDataAPI_LayoutForPatientVisit{
		layoutData: []byte(layout),
		patientCase: &common.PatientCase{
			Name: "Acne",
		},
		doctorMember: &common.CareProviderAssignment{
			ProviderID: 1,
		},
		doctor: &common.Doctor{
			DoctorID:         encoding.NewObjectID(2),
			ShortDisplayName: "Dr. X",
		},
		patient: &common.Patient{
			FirstName: "Ben",
		},
	}

	intakeLayout, err := GetPatientLayoutForPatientVisit(&common.PatientVisit{}, 1, m, "api.spruce.local")
	test.OK(t, err)
	test.Equals(t, "Follow-up Visit", intakeLayout.Header.Title)
	test.Equals(t, "With Dr. X", intakeLayout.Header.Subtitle)
	test.Equals(t, app_url.ThumbnailURL("api.spruce.local", api.DOCTOR_ROLE, 2), intakeLayout.Header.IconURL)
	test.Equals(t, "Submit your follow-up visit for Dr. X to review", intakeLayout.Checkout.Header)
	test.Equals(t, "Your follow-up visit has been submitted.", intakeLayout.SubmissionConfirmation.Top)
	test.Equals(t, "Dr. X will review your visit and respond within 24 hours.", intakeLayout.SubmissionConfirmation.Bottom)
	test.Equals(t, "Welcome to your follow-up visit, Ben. We'll begin by asking you about your current treatment plan.", intakeLayout.Transitions[0].Message)

}
