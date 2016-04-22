package twilio

import (
	"fmt"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/settings"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"golang.org/x/net/context"
)

func TestAfterHours_IncomingCall_DefaultGreeting(t *testing.T) {
	orgID := "12345"
	// providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: practicePhoneNumber,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
			},
		},
	}, nil))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CreateIncomingCall, &models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhoneNumber),
		AfterHours:     true,
		CallSID:        callSID,
	}))

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyIncomingCallOption,
				Subkey: practicePhoneNumber,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: excommsSettings.IncomingCallOptionAfterHoursCallTriage,
						},
					},
				},
			},
		},
	}, nil))

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyAfterHoursGreetingOption,
				Subkey: practicePhoneNumber,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: excommsSettings.AfterHoursGreetingOptionDefault,
						},
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
	}

	twiml, err := processIncomingCall(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/afterhours_patient_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="alice">You have reached Dewabi Corp. If this is an emergency, please hang up and dial 9 1 1.</Say><Say voice="alice">Otherwise, press 1 to leave an urgent message, 2 to leave a non-urgent message.</Say></Gather><Redirect>/twilio/call/afterhours_greeting</Redirect></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_IncomingCall_CustomGreeting(t *testing.T) {
	orgID := "12345"
	// providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: practicePhoneNumber,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
			},
		},
	}, nil))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CreateIncomingCall, &models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhoneNumber),
		CallSID:        callSID,
		AfterHours:     true,
	}))

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyIncomingCallOption,
				Subkey: practicePhoneNumber,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID: excommsSettings.IncomingCallOptionAfterHoursCallTriage,
						},
					},
				},
			},
		},
	}, nil))

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyAfterHoursGreetingOption,
				Subkey: practicePhoneNumber,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID:               excommsSettings.AfterHoursGreetingOptionCustom,
							FreeTextResponse: "https://custom.voicemail",
						},
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
	}

	twiml, err := processIncomingCall(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/afterhours_patient_entered_digits" method="POST" timeout="5" numDigits="1"><Play>https://custom.voicemail</Play></Gather><Redirect>/twilio/call/afterhours_greeting</Redirect></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_PatientEnteredDigits_Urgent(t *testing.T) {
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdal := dalmock.New(t)
	defer mdal.Finish()

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		Urgent: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
		Digits:  "1",
	}

	twiml, err := afterHoursPatientEnteredDigits(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Redirect>/twilio/call/afterhours_voicemail</Redirect></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_PatientEnteredDigits_NonUrgent(t *testing.T) {
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdal := dalmock.New(t)
	defer mdal.Finish()

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		Urgent: ptr.Bool(false),
	}).WithReturns(int64(1), nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
		Digits:  "2",
	}

	twiml, err := afterHoursPatientEnteredDigits(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Redirect>/twilio/call/afterhours_voicemail</Redirect></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_Voicemail(t *testing.T) {
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"
	orgID := "o1"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: practicePhoneNumber,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
			},
		},
	}, nil))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
	}

	twiml, err := afterHoursVoicemailTWIML(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">Please leave a message after the tone.</Say><Record action="/twilio/call/afterhours_process_voicemail" timeout="60" maxLength="3600" transcribeCallback="/twilio/call/no_op" playBeep="true"></Record></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_Voicemail_Transcription(t *testing.T) {
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"
	orgID := "o1"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: practicePhoneNumber,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
			},
		},
	}, nil))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeyTranscribeVoicemail,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", nil, storage.NewTestStore(nil))
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: callSID,
	}

	twiml, err := afterHoursVoicemailTWIML(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">Please leave a message after the tone. Speak slowly and clearly as your message will be transcribed.</Say><Record action="/twilio/call/no_op" timeout="60" maxLength="3600" transcribeCallback="/twilio/call/afterhours_process_voicemail" playBeep="true"></Record></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}
