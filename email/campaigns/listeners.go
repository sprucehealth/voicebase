package campaigns

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/patient"
	"github.com/sprucehealth/backend/patient_visit"
)

var welcomeEmailEnabledDef = &cfg.ValueDef{
	Name:        "Email.Campaign.Welcome.Enabled",
	Description: "Enable or disable the welcome email.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

var minorTreatmentPlanIssuedEmailEnabledDef = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Treatment.Plan.Issued.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been issued a treatment plan.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

var minorTriagedEmailEnabledDef = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Triaged.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been triaged.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

var parentWelcomeEmailEnabledDef = &cfg.ValueDef{
	Name:        "Email.Campaign.Parent.Welcome.Enabled",
	Description: "Enable or disable the email welcoming parents after consenting.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

var parentFrequentlyAskedQuestionsURLPathDef = &cfg.ValueDef{
	Name:        "Parent.Frequently.Asked.Question.URL.Path",
	Description: "Change the path applied to the web domain for the location of the parent FAQ.",
	Type:        cfg.ValueTypeString,
	Default:     "/pc/faq",
}

const (
	patientSignupEmailType                   = "welcome"
	minorTreatmentPlanIssuedEmailType        = "minor-treatment-plan-issued"
	minorTriagedEmailType                    = "minor-triaged"
	parentWelcomeEmailType                   = "parent-welcome"
	varParentFirstNameName                   = "parent_first_name"
	varPatientFirstNameName                  = "patient_first_name"
	varParentFrequentlyAskedQuestionsURLName = "spruce_parent_faq_url"
)

// InitListeners bootstraps the listeners related to email campaigns triggered by events in the system
func InitListeners(dispatch *dispatch.Dispatcher, cfgStore cfg.Store, emailService email.Service, dataAPI api.DataAPI, webDomain string) {
	cfgStore.Register(welcomeEmailEnabledDef)
	cfgStore.Register(minorTreatmentPlanIssuedEmailEnabledDef)
	cfgStore.Register(minorTriagedEmailEnabledDef)
	cfgStore.Register(parentWelcomeEmailEnabledDef)
	cfgStore.Register(parentFrequentlyAskedQuestionsURLPathDef)
	dispatch.SubscribeAsync(func(ev *patient.SignupEvent) error {
		if cfgStore.Snapshot().Bool(welcomeEmailEnabledDef.Name) {
			if _, err := emailService.Send([]int64{ev.AccountID}, patientSignupEmailType, nil, &mandrill.Message{}, email.OnlyOnce|email.CanOptOut); err != nil {
				golog.Errorf("Failed to send welcome email to account %d: %s", ev.AccountID, err)
			}
		}
		return nil
	})
	dispatch.SubscribeAsync(func(ev *doctor_treatment_plan.TreatmentPlanActivatedEvent) error {
		if cfgStore.Snapshot().Bool(minorTreatmentPlanIssuedEmailEnabledDef.Name) {
			if err := sendToPatientParent(ev.PatientID, minorTreatmentPlanIssuedEmailType, webDomain, email.CanOptOut, emailService, dataAPI, cfgStore); err != nil {
				golog.Errorf("%s", err)
			}
		}
		return nil
	})
	dispatch.SubscribeAsync(func(ev *patient_visit.PatientVisitMarkedUnsuitableEvent) error {
		if cfgStore.Snapshot().Bool(minorTriagedEmailEnabledDef.Name) {
			if err := sendToPatientParent(ev.PatientID, minorTriagedEmailType, webDomain, email.CanOptOut, emailService, dataAPI, cfgStore); err != nil {
				golog.Errorf("%s", err)
			}
		}
		return nil
	})
	// Send the consenting parent a welcome email but only do it once
	dispatch.SubscribeAsync(func(ev *patient.ParentalConsentCompletedEvent) error {
		if cfgStore.Snapshot().Bool(parentWelcomeEmailEnabledDef.Name) {
			if err := sendToPatientParent(ev.ChildPatientID, parentWelcomeEmailType, webDomain, email.CanOptOut, emailService, dataAPI, cfgStore); err != nil {
				golog.Errorf("%s", err)
			}
		}
		return nil
	})
}

func sendToPatientParent(childPatientID int64, emailType, webDomain string, opt email.Option, emailService email.Service, dataAPI api.DataAPI, cfgStore cfg.Store) error {
	faqURL := "https://" + strings.Join([]string{webDomain, cfgStore.Snapshot().String(parentFrequentlyAskedQuestionsURLPathDef.Name)}, "/")
	patient, err := dataAPI.Patient(childPatientID, true)
	if err != nil {
		return errors.Trace(fmt.Errorf("Failed to send %s email to parent account of child patient id %d: %s", emailType, childPatientID, err))
	}
	if patient.IsUnder18() && patient.HasParentalConsent {
		consents, err := dataAPI.ParentalConsent(childPatientID)
		if err != nil {
			return errors.Trace(fmt.Errorf("Failed to send %s email to parent account of child account %d: %s", emailType, patient.AccountID.Int64(), err))
		}

		// notify all parents that have granted consent
		for _, consent := range consents {
			parent, err := dataAPI.Patient(consent.ParentPatientID, true)
			if err != nil {
				return errors.Trace(fmt.Errorf("Failed to send %s email to parent account of child account %d: %s", emailType, patient.AccountID.Int64(), err))
			}
			if _, err := emailService.Send(
				[]int64{parent.AccountID.Int64()},
				emailType,
				map[int64][]mandrill.Var{
					parent.AccountID.Int64(): []mandrill.Var{
						mandrill.Var{Name: varParentFirstNameName, Content: parent.FirstName},
						mandrill.Var{Name: varPatientFirstNameName, Content: patient.FirstName},
						mandrill.Var{Name: varParentFrequentlyAskedQuestionsURLName, Content: faqURL},
					},
				},
				&mandrill.Message{},
				opt); err != nil {
				return errors.Trace(fmt.Errorf("Failed to send %s issued email to account %d: %s", emailType, parent.AccountID.Int64(), err))
			}
		}
	}
	return nil
}
