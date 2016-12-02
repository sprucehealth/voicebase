package twilio

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/settings"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
)

func TestAfterHours_IncomingCall_SendAllCallsToVM_DefaultGreeting(t *testing.T) {
	orgID := "12345"
	providerPersonalPhone := "+14152222222"
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
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL},
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
	}))

	mdal.Expect(mock.NewExpectation(mdal.LookupBlockedNumbers, phone.Number(practicePhoneNumber)))

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingListTimeout,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyPauseBeforeCallConnect,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyExposeCaller,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
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
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
			{
				Type: settings.ConfigType_INTEGER,
				Value: &settings.Value_Integer{
					Integer: &settings.IntegerValue{
						Value: 30,
					},
				},
			},
			{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeyForwardingList,
					Subkey: practicePhoneNumber,
				},
				Type: settings.ConfigType_STRING_LIST,
				Value: &settings.Value_StringList{
					StringList: &settings.StringListValue{
						Values: []string{providerPersonalPhone},
					},
				},
			},
			{
				Type: settings.ConfigType_INTEGER,
				Value: &settings.Value_Integer{
					Integer: &settings.IntegerValue{
						Value: 0,
					},
				},
			},
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

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
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
							ID: excommsSettings.VoicemailOptionDefault,
						},
					},
				},
			},
		},
	}, nil))

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		Afterhours: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", signer)
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

func TestAfterHours_IncomingCallStatus(t *testing.T) {
	orgID := "12345"
	// providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesResponse{
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

	mdal.Expect(mock.NewExpectation(mdal.LookupIncomingCall, callSID).WithReturns(&models.IncomingCall{
		OrganizationID: orgID,
	}, nil))

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
					Subkey: practicePhoneNumber,
				},
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		Afterhours: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
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
							ID: excommsSettings.VoicemailOptionDefault,
						},
					},
				},
			},
		},
	}, nil))

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", signer)
	params := &rawmsg.TwilioParams{
		From:           patientPhone,
		To:             practicePhoneNumber,
		CallSID:        callSID,
		DialCallStatus: rawmsg.TwilioParams_NO_ANSWER,
		CallStatus:     rawmsg.TwilioParams_IN_PROGRESS,
	}

	twiml, err := processDialedCallStatus(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/afterhours_patient_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="alice">You have reached Dewabi Corp. If this is an emergency, please hang up and dial 9 1 1.</Say><Say voice="alice">Otherwise, press 1 to leave an urgent message, 2 to leave a non-urgent message.</Say></Gather><Redirect>/twilio/call/afterhours_greeting</Redirect></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestAfterHours_IncomingCall_SendAllCallsToVM_CustomGreeting(t *testing.T) {
	orgID := "12345"
	providerPersonalPhone := "+14152222222"
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
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_ORGANIZATION},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION, directory.EntityType_INTERNAL},
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
	}))

	mdal.Expect(mock.NewExpectation(mdal.LookupBlockedNumbers, phone.Number(practicePhoneNumber)))

	msettings := settingsmock.New(t)
	defer msettings.Finish()

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyAfterHoursVociemailEnabled,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingListTimeout,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyPauseBeforeCallConnect,
				Subkey: practicePhoneNumber,
			},
			{
				Key:    excommsSettings.ConfigKeyExposeCaller,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
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
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
			{
				Type: settings.ConfigType_INTEGER,
				Value: &settings.Value_Integer{
					Integer: &settings.IntegerValue{
						Value: 30,
					},
				},
			},
			{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeyForwardingList,
					Subkey: practicePhoneNumber,
				},
				Type: settings.ConfigType_STRING_LIST,
				Value: &settings.Value_StringList{
					StringList: &settings.StringListValue{
						Values: []string{providerPersonalPhone},
					},
				},
			},
			{
				Type: settings.ConfigType_INTEGER,
				Value: &settings.Value_Integer{
					Integer: &settings.IntegerValue{
						Value: 0,
					},
				},
			},
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

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		Afterhours: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID:               excommsSettings.VoicemailOptionCustom,
							FreeTextResponse: "123456789",
						},
					},
				},
			},
		},
	}, nil))

	mc := clock.NewManaged(time.Now())
	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	expectedURL, err := signer.SignedURL(fmt.Sprintf("/media/%s", "123456789"), url.Values{}, ptr.Time(mc.Now().Add(time.Hour)))
	test.OK(t, err)

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, mc, nil, "https://test.com", "", "", "", signer)
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
<Response><Gather action="/twilio/call/afterhours_patient_entered_digits" method="POST" timeout="5" numDigits="1"><Play>%s</Play></Gather><Redirect>/twilio/call/afterhours_greeting</Redirect></Response>`, html.EscapeString(expectedURL))

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

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", signer)
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

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", signer)
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
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
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
	mclock := clock.NewManaged(time.Now())

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		SentToVoicemail: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

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

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, mclock, nil, "https://test.com", "", "", "", signer)
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
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
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

	mdal.Expect(mock.NewExpectation(mdal.UpdateIncomingCall, callSID, &dal.IncomingCallUpdate{
		SentToVoicemail: ptr.Bool(true),
	}).WithReturns(int64(1), nil))

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

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, msettings, mdal, &mockSNS_Twilio{}, clock.New(), nil, "https://test.com", "", "", "", signer)
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

func TestAfterHours_Voicemail_Process(t *testing.T) {
	conc.Testing = true
	params := &rawmsg.TwilioParams{
		From:              "+12068773590",
		To:                "+17348465522",
		RecordingDuration: 10,
		RecordingURL:      "http://google.com",
	}

	mclock := clock.NewManaged(time.Now())

	ms := &mockSNS_Twilio{}
	md := dalmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.LookupIncomingCall, params.CallSID).WithReturns(&models.IncomingCall{
		OrganizationID: "o1",
		Source:         phone.Number(params.From),
		Destination:    phone.Number(params.To),
		AfterHours:     true,
	}, nil))

	md.Expect(mock.NewExpectation(md.StoreIncomingRawMessage, &rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	}))

	md.Expect(mock.NewExpectation(md.LookupIncomingCall, params.CallSID).WithReturns(&models.IncomingCall{
		OrganizationID: "o1",
		Source:         phone.Number(params.From),
		Destination:    phone.Number(params.To),
	}, nil))

	md.Expect(mock.NewExpectation(md.UpdateIncomingCall, params.CallSID, &dal.IncomingCallUpdate{
		LeftVoicemail:     ptr.Bool(true),
		LeftVoicemailTime: ptr.Time(mclock.Now()),
	}).WithReturns(int64(1), nil))

	mdir := dirmock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "o1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_EXTERNAL_IDS,
				directory.EntityInformation_MEMBERS,
			},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "o1",
				Type: directory.EntityType_ORGANIZATION,
				Members: []*directory.Entity{
					{
						ID:   "p1",
						Type: directory.EntityType_INTERNAL,
					},
				},
				ExternalIDs: []string{"account_1"},
			},
		},
	}, nil))

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, nil, md, ms, mclock, nil, "", "", "", "", signer)

	twiml, err := afterHoursProcessVoicemail(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	} else if twiml != "" {
		t.Fatalf("Expected %s got %s", "", twiml)
	}

	if len(ms.published) != 1 {
		t.Fatalf("Expected 1 but got %d", len(ms.published))
	}
}
