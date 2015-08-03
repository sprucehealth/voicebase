package campaigns

import (
	"fmt"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/config"
)

type mockDataAPIListeners struct {
	api.DataAPI
	patientParams        []int64
	patientErrs          []error
	patients             []*common.Patient
	patientCallCount     int
	patientParentIDParam int64
	patientParentIDErr   error
	patientParentID      int64
}

func (m *mockDataAPIListeners) Patient(id int64, basic bool) (*common.Patient, error) {
	defer func() { m.patientCallCount++ }()
	m.patientParams = append(m.patientParams, id)
	return m.patients[m.patientCallCount], m.patientErrs[m.patientCallCount]
}

func (m *mockDataAPIListeners) ParentalConsent(id int64) ([]*common.ParentalConsent, error) {
	m.patientParentIDParam = id
	return []*common.ParentalConsent{
		{
			ParentPatientID: m.patientParentID,
		},
	}, m.patientParentIDErr
}

const (
	emailWebDomain = "www.spruce.test"
)

func TestEmailCampaignWelcomeOnSignup(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.WelcomeEmailEnabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{
		patients: []*common.Patient{
			{
				DOB: encoding.Date{
					Year:  1920,
					Month: 1,
					Day:   1,
				},
			},
		},
		patientErrs: []error{
			nil,
		},
	}

	var accountID int64 = 12345
	var vars map[int64][]mandrill.Var
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient.SignupEvent{AccountID: accountID})
	emails := emailService.Reset()
	test.Equals(t, 1, len(emails))
	test.Equals(t, patientSignupEmailType, emails[0].Type)
	test.Equals(t, []int64{accountID}, emails[0].AccountIDs)
	test.Equals(t, vars, emails[0].Vars)
	test.Equals(t, &mandrill.Message{}, emails[0].Msg)
}

func TestEmailCampaignWelcomeUnder18OnSignup(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.WelcomeEmailEnabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{
		patients: []*common.Patient{
			{
				DOB: encoding.Date{
					Year:  2014,
					Month: 1,
					Day:   1,
				},
			},
		},
		patientErrs: []error{
			nil,
		},
	}

	var accountID int64 = 12345
	var vars map[int64][]mandrill.Var
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient.SignupEvent{AccountID: accountID})
	emails := emailService.Reset()
	test.Equals(t, 1, len(emails))
	test.Equals(t, patientUnder18SignupEmailType, emails[0].Type)
	test.Equals(t, []int64{accountID}, emails[0].AccountIDs)
	test.Equals(t, vars, emails[0].Vars)
	test.Equals(t, &mandrill.Message{}, emails[0].Msg)
}

func TestEmailCampaignWelcomeOnSignupCfgDisabled(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.WelcomeEmailDisabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient.SignupEvent{AccountID: 12345})
	test.Equals(t, 0, len(emailService.Reset()))
}

func TestEmailCampaignMinorTreatmentPlanIssued(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.MinorTreatmentPlanIssuedEmailEnabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	var parentPatientID int64 = 54321
	var parentAccountID int64 = 56789
	var patientID int64 = 12345
	patientFirstName := "Child"
	parentFirstName := "Parent"
	dataAPI := &mockDataAPIListeners{
		patients: []*common.Patient{
			&common.Patient{
				ID:                 encoding.NewObjectID(patientID),
				HasParentalConsent: true,
				DOB:                encoding.Date{Month: 1, Day: 1, Year: time.Now().Year() - 16},
				FirstName:          patientFirstName,
			},
			&common.Patient{
				AccountID: encoding.NewObjectID(parentAccountID),
				FirstName: parentFirstName,
			},
		},
		patientErrs:     []error{nil, nil},
		patientParentID: parentPatientID,
	}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&doctor_treatment_plan.TreatmentPlanActivatedEvent{PatientID: patientID})
	emails := emailService.Reset()
	test.Equals(t, 1, len(emails))
	test.Equals(t, minorTreatmentPlanIssuedEmailType, emails[0].Type)
	test.Equals(t, []int64{parentAccountID}, emails[0].AccountIDs)
	test.Equals(t, map[int64][]mandrill.Var{
		parentAccountID: []mandrill.Var{
			mandrill.Var{Name: varParentFirstNameName, Content: parentFirstName},
			mandrill.Var{Name: varPatientFirstNameName, Content: patientFirstName},
			mandrill.Var{Name: varParentFrequentlyAskedQuestionsURLName, Content: "https://" + emailWebDomain + faqURLPath},
			mandrill.Var{Name: varPatientMedrecordURLName, Content: "https://" + emailWebDomain + fmt.Sprintf(medRecordURLPathFormatString, patientID)},
		},
	}, emails[0].Vars)
	test.Equals(t, &mandrill.Message{}, emails[0].Msg)
	test.Equals(t, 2, len(dataAPI.patientParams))
	test.Equals(t, patientID, dataAPI.patientParams[0])
	test.Equals(t, parentPatientID, dataAPI.patientParams[1])
	test.Equals(t, patientID, dataAPI.patientParentIDParam)
}

func TestEmailCampaignMinorTreatmentPlanIssuedCfgDisabled(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.MinorTreatmentPlanIssuedEmailDisabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&doctor_treatment_plan.TreatmentPlanActivatedEvent{PatientID: 12345})
	test.Equals(t, 0, len(emailService.Reset()))
}

func TestEmailCampaignMinorTriaged(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.MinorTriagedEmailEnabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	var parentPatientID int64 = 54321
	var parentAccountID int64 = 56789
	var patientID int64 = 12345
	patientFirstName := "Child"
	parentFirstName := "Parent"
	dataAPI := &mockDataAPIListeners{
		patients: []*common.Patient{
			&common.Patient{
				ID:                 encoding.NewObjectID(patientID),
				HasParentalConsent: true,
				DOB:                encoding.Date{Month: 1, Day: 1, Year: time.Now().Year() - 16},
				FirstName:          patientFirstName,
			},
			&common.Patient{
				AccountID: encoding.NewObjectID(parentAccountID),
				FirstName: parentFirstName,
			},
		},
		patientErrs:     []error{nil, nil},
		patientParentID: parentPatientID,
	}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient_visit.PatientVisitMarkedUnsuitableEvent{PatientID: patientID})
	emails := emailService.Reset()
	test.Equals(t, 1, len(emails))
	test.Equals(t, minorTriagedEmailType, emails[0].Type)
	test.Equals(t, []int64{parentAccountID}, emails[0].AccountIDs)
	test.Equals(t, map[int64][]mandrill.Var{
		parentAccountID: []mandrill.Var{
			mandrill.Var{Name: varParentFirstNameName, Content: parentFirstName},
			mandrill.Var{Name: varPatientFirstNameName, Content: patientFirstName},
			mandrill.Var{Name: varParentFrequentlyAskedQuestionsURLName, Content: "https://" + emailWebDomain + faqURLPath},
			mandrill.Var{Name: varPatientMedrecordURLName, Content: "https://" + emailWebDomain + fmt.Sprintf(medRecordURLPathFormatString, patientID)},
		},
	}, emails[0].Vars)
	test.Equals(t, &mandrill.Message{}, emails[0].Msg)
	test.Equals(t, 2, len(dataAPI.patientParams))
	test.Equals(t, patientID, dataAPI.patientParams[0])
	test.Equals(t, parentPatientID, dataAPI.patientParams[1])
	test.Equals(t, patientID, dataAPI.patientParentIDParam)
}

func TestEmailCampaignMinorTriagedCfgDisabled(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.MinorTriagedEmailDisabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient_visit.PatientVisitMarkedUnsuitableEvent{PatientID: 12345})
	test.Equals(t, 0, len(emailService.Reset()))
}

func TestEmailCampaignParentWelcome(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.ParentWelcomeEmailEnabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	var parentPatientID int64 = 12345
	var parentAccountID int64 = 56789
	var patientID int64 = 12345
	patientFirstName := "Child"
	parentFirstName := "Parent"
	dataAPI := &mockDataAPIListeners{
		patients: []*common.Patient{
			&common.Patient{
				ID:                 encoding.NewObjectID(patientID),
				HasParentalConsent: true,
				DOB:                encoding.Date{Month: 1, Day: 1, Year: time.Now().Year() - 16},
				FirstName:          patientFirstName,
			},
			&common.Patient{
				AccountID: encoding.NewObjectID(parentAccountID),
				FirstName: parentFirstName,
			},
		},
		patientErrs:     []error{nil, nil},
		patientParentID: parentPatientID,
	}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient.ParentalConsentCompletedEvent{ChildPatientID: patientID})
	emails := emailService.Reset()
	test.Equals(t, 1, len(emails))
	test.Equals(t, parentWelcomeEmailType, emails[0].Type)
	test.Equals(t, []int64{parentAccountID}, emails[0].AccountIDs)
	test.Equals(t, map[int64][]mandrill.Var{
		parentAccountID: []mandrill.Var{
			mandrill.Var{Name: varParentFirstNameName, Content: parentFirstName},
			mandrill.Var{Name: varPatientFirstNameName, Content: patientFirstName},
			mandrill.Var{Name: varParentFrequentlyAskedQuestionsURLName, Content: "https://" + emailWebDomain + faqURLPath},
			mandrill.Var{Name: varPatientMedrecordURLName, Content: "https://" + emailWebDomain + fmt.Sprintf(medRecordURLPathFormatString, patientID)},
		},
	}, emails[0].Vars)
	test.Equals(t, &mandrill.Message{}, emails[0].Msg)
	test.Equals(t, 2, len(dataAPI.patientParams))
	test.Equals(t, patientID, dataAPI.patientParams[0])
	test.Equals(t, parentPatientID, dataAPI.patientParams[1])
	test.Equals(t, patientID, dataAPI.patientParentIDParam)
}

func TestEmailCampaignParentWelcomeCfgDisabled(t *testing.T) {
	dispatch.Testing = true
	dispatcher := dispatch.New()
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.ParentWelcomeEmailDisabled})
	test.OK(t, err)
	emailService := &email.TestService{}
	dataAPI := &mockDataAPIListeners{}
	InitListeners(dispatcher, cfgStore, emailService, dataAPI, emailWebDomain)
	dispatcher.PublishAsync(&patient.ParentalConsentCompletedEvent{ParentPatientID: 12345})
	test.Equals(t, 0, len(emailService.Reset()))
}
