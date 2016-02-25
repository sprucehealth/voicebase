package twilio

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	proxynumber "github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	directorymock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	settingsmock "github.com/sprucehealth/backend/svc/settings/mock"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type mockDirectoryService_Twilio struct {
	directory.DirectoryClient
	entities     map[string][]*directory.Entity
	entitiesList []*directory.Entity
}

func (m *mockDirectoryService_Twilio) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	return &directory.LookupEntitiesResponse{
		Entities: m.entities[in.GetEntityID()],
	}, nil
}

func (m *mockDirectoryService_Twilio) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	return &directory.LookupEntitiesByContactResponse{
		Entities: m.entitiesList,
	}, nil
}

type mockDAL_Twilio struct {
	*mock.Expector
	dal.DAL
	cr   *models.CallRequest
	ppnr *models.ProxyPhoneNumberReservation
}

func (m *mockDAL_Twilio) StoreIncomingRawMessage(msg *rawmsg.Incoming) (uint64, error) {
	m.Record(msg)
	return 0, nil
}

func (m *mockDAL_Twilio) LookupCallRequest(fromPhoneNumber string) (*models.CallRequest, error) {
	m.Record(fromPhoneNumber)
	return m.cr, nil
}
func (m *mockDAL_Twilio) CreateCallRequest(cr *models.CallRequest) error {
	m.Record(cr)
	return nil
}

type mockSNS_Twilio struct {
	snsiface.SNSAPI
	published []*sns.PublishInput
}

func (m *mockSNS_Twilio) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	m.published = append(m.published, input)
	return nil, nil
}

func TestOutgoing_Process(t *testing.T) {
	testOutgoing(t, false, "")
}

func TestOutgoing_Expired(t *testing.T) {
	testOutgoing(t, true, "")
}

func TestOutgoing_PatientName(t *testing.T) {
	testOutgoing(t, false, "Joe")
}

func testOutgoing(t *testing.T, testExpired bool, patientName string) {
	practicePhoneNumber := phone.Number("+12068773590")
	patientPhoneNumber := phone.Number("+11234567890")
	providerPersonalPhoneNumber := phone.Number("+17348465522")
	proxyPhoneNumber := phone.Number("+14152222222")
	organizationID := "1234"
	destinationEntityID := "6789"
	providerID := "0000"
	callSID := "8888"

	md := &mockDirectoryService_Twilio{
		entities: map[string][]*directory.Entity{
			"1234": {
				{
					ID:   organizationID,
					Type: directory.EntityType_ORGANIZATION,
					Contacts: []*directory.Contact{
						{
							Provisioned: true,
							ContactType: directory.ContactType_PHONE,
							Value:       practicePhoneNumber.String(),
						},
					},
				},
			},
			"6789": {
				{
					ID:   destinationEntityID,
					Type: directory.EntityType_EXTERNAL,
					Info: &directory.EntityInfo{
						DisplayName: patientName,
					},
					Contacts: []*directory.Contact{
						{
							Provisioned: false,
							ContactType: directory.ContactType_PHONE,
							Value:       patientPhoneNumber.String(),
						},
					},
				},
			},
		},
	}

	mdal := &mockDAL_Twilio{
		Expector: &mock.Expector{
			T: t,
		},
	}
	mclock := clock.NewManaged(time.Now())

	if !testExpired {
		mdal.Expect(mock.NewExpectation(mdal.CreateCallRequest, &models.CallRequest{
			Source:         phone.Number(providerPersonalPhoneNumber),
			Destination:    phone.Number(patientPhoneNumber),
			Proxy:          proxyPhoneNumber,
			OrganizationID: organizationID,
			CallSID:        callSID,
			Requested:      mclock.Now(),
			CallerEntityID: providerID,
			CalleeEntityID: destinationEntityID,
		}))
	}

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	if testExpired {
		mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ActiveReservation, providerPersonalPhoneNumber, proxyPhoneNumber).WithReturns(
			&models.ProxyPhoneNumberReservation{},
			dal.ErrProxyPhoneNumberReservationNotFound))
	} else {
		mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ActiveReservation, providerPersonalPhoneNumber, proxyPhoneNumber).WithReturns(&models.ProxyPhoneNumberReservation{
			ProxyPhoneNumber:       proxyPhoneNumber,
			OriginatingPhoneNumber: providerPersonalPhoneNumber,
			DestinationPhoneNumber: patientPhoneNumber,
			DestinationEntityID:    destinationEntityID,
			OwnerEntityID:          providerID,
			OrganizationID:         organizationID,
		}, nil))

		mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.CallStarted, providerPersonalPhoneNumber, proxyPhoneNumber))
	}

	es := NewEventHandler(md, nil, mdal, nil, mclock, mproxynumberManager, "https://test.com", "", "")

	params := &rawmsg.TwilioParams{
		From:    providerPersonalPhoneNumber.String(),
		To:      proxyPhoneNumber.String(),
		CallSID: callSID,
	}

	twiml, err := processOutgoingCall(context.Background(), params, es.(*eventsHandler))
	if testExpired {
		if err == nil {
			t.Fatalf("Expected the call to not go through and be expired.")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		var expected string
		if patientName != "" {
			expected = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">You will be connected to %s</Say><Dial callerId="+12068773590"><Number statusCallbackEvent="ringing answered completed" statusCallback="https://test.com/twilio/call/process_outgoing_call_status">+11234567890</Number></Dial></Response>`, patientName)
		} else {
			expected = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">You will be connected to 123 456 7890</Say><Dial callerId="+12068773590"><Number statusCallbackEvent="ringing answered completed" statusCallback="https://test.com/twilio/call/process_outgoing_call_status">+11234567890</Number></Dial></Response>`)
		}

		if twiml != expected {
			t.Fatalf("\nExpected %s\nGot %s", expected, twiml)
		}
	}
	mock.FinishAll(mdal)
}

func TestIncoming_Organization(t *testing.T) {
	orgID := "12345"
	providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       practicePhoneNumber,
						Provisioned: true,
					},
				},
			},
		},
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CreateIncomingCall, &models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhoneNumber),
		CallSID:        callSID,
	}))

	msettings := settingsmock.New(t)
	defer msettings.Finish()
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
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
		},
	}, nil))

	es := NewEventHandler(md, msettings, mdal, nil, clock.New(), nil, "https://test.com", "", "")
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
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="%s"><Number url="/twilio/call/provider_call_connected">%s</Number></Dial></Response>`, practicePhoneNumber, providerPersonalPhone)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncoming_Organization_SingleProvider_DirectAllCallsToVoicemail(t *testing.T) {
	orgID := "12345"
	providerID := "p1"
	providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       practicePhoneNumber,
						Provisioned: true,
					},
				},
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
				Members: []*directory.Entity{
					{
						ID: providerID,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       providerPersonalPhone,
							},
						},
					},
				},
			},
		},
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CreateIncomingCall, &models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhoneNumber),
		CallSID:        callSID,
	}))

	msettings := settingsmock.New(t)
	defer msettings.Finish()
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
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
		},
	}, nil))
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeySendCallsToVoicemail,
			},
		},
		NodeID: providerID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key: excommsSettings.ConfigKeySendCallsToVoicemail,
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

	es := NewEventHandler(md, msettings, mdal, nil, clock.New(), nil, "https://test.com", "", "")
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
<Response><Say voice="alice">You have reached Dewabi Corp. Please leave a message after the tone.</Say><Record action="/twilio/call/process_voicemail" timeout="60" playBeep="true"></Record></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncoming_Organization_MultipleContacts(t *testing.T) {
	orgID := "12345"
	listedNumber1 := "+14152222222"
	listedNumber2 := "+14153333333"
	listedNumber3 := "+14154444444"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Provisioned: true,
						Value:       practicePhoneNumber,
					},
				},
			},
		},
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CreateIncomingCall, &models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhoneNumber),
		CallSID:        callSID,
	}))

	msettings := settingsmock.New(t)
	defer msettings.Finish()
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeyForwardingList,
					Subkey: practicePhoneNumber,
				},
				Type: settings.ConfigType_STRING_LIST,
				Value: &settings.Value_StringList{
					StringList: &settings.StringListValue{
						Values: []string{listedNumber1, listedNumber2, listedNumber3},
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(md, msettings, mdal, nil, clock.New(), nil, "https://test.com", "", "")
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
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="+14150000000"><Number url="/twilio/call/provider_call_connected">+14152222222</Number><Number url="/twilio/call/provider_call_connected">+14153333333</Number><Number url="/twilio/call/provider_call_connected">+14154444444</Number></Dial></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncoming_Organization_MultipleContacts_SendToVoicemail(t *testing.T) {
	orgID := "12345"
	listedNumber1 := "+14152222222"
	providerID1 := "p1"
	listedNumber2 := "+14153333333"
	listedNumber3 := "+14154444444"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       practicePhoneNumber,
						Provisioned: true,
					},
				},
				Members: []*directory.Entity{
					{
						ID: providerID1,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       listedNumber1,
							},
						},
					},
				},
			},
		},
	}

	msettings := settingsmock.New(t)
	defer msettings.Finish()
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key:    excommsSettings.ConfigKeyForwardingList,
				Subkey: practicePhoneNumber,
			},
		},
		NodeID: orgID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeyForwardingList,
					Subkey: practicePhoneNumber,
				},
				Type: settings.ConfigType_STRING_LIST,
				Value: &settings.Value_StringList{
					StringList: &settings.StringListValue{
						Values: []string{listedNumber1, listedNumber2, listedNumber3},
					},
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

	// configure the situation to have one of the numbers in the list belong to a provider
	// who has their send to voicemail setting on.
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeySendCallsToVoicemail,
			},
		},
		NodeID: providerID1,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key: excommsSettings.ConfigKeySendCallsToVoicemail,
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

	es := NewEventHandler(md, msettings, mdal, nil, clock.New(), nil, "https://test.com", "", "")
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "12345",
	}

	twiml, err := processIncomingCall(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="+14150000000"><Number url="/twilio/call/provider_call_connected">+14153333333</Number><Number url="/twilio/call/provider_call_connected">+14154444444</Number></Dial></Response>`)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncoming_Provider(t *testing.T) {
	orgID := "12345"
	providerID := "6789"
	providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"
	callSID := "12345"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   providerID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       practicePhoneNumber,
						Provisioned: true,
					},
					{
						ContactType: directory.ContactType_PHONE,
						Value:       providerPersonalPhone,
					},
				},
			},
		},
	}

	msettings := settingsmock.New(t)
	defer msettings.Finish()
	msettings.Expect(mock.NewExpectation(msettings.GetValues, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{
				Key: excommsSettings.ConfigKeySendCallsToVoicemail,
			},
		},
		NodeID: providerID,
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key: excommsSettings.ConfigKeySendCallsToVoicemail,
				},
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
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

	es := NewEventHandler(md, msettings, mdal, nil, clock.New(), nil, "https://test.com", "", "")
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
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="%s"><Number url="/twilio/call/provider_call_connected">%s</Number></Dial></Response>`, practicePhoneNumber, providerPersonalPhone)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestProviderCallConnected(t *testing.T) {
	patientPhone := "+12061111111"
	practicePhone := "+12062222222"
	providerPhone := "+12063333333"
	orgID := "o1"

	// the params are intended to simulate the dial leg of the call
	// where the call shows up as originating from the practice phone to
	// the number of the provider in the forwarding list
	params := &rawmsg.TwilioParams{
		From:          practicePhone,
		To:            providerPhone,
		ParentCallSID: "12345",
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.LookupIncomingCall, params.ParentCallSID).WithReturns(&models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhone),
		CallSID:        params.ParentCallSID,
	}, nil))

	mdirectory := directorymock.New(t)
	defer mdirectory.Finish()

	mdirectory.Expect(mock.NewExpectation(mdirectory.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: patientPhone,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				Type: directory.EntityType_EXTERNAL,
				Info: &directory.EntityInfo{
					FirstName:   "J",
					LastName:    "S",
					DisplayName: "JS",
				},
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdirectory, nil, mdal, nil, clock.New(), nil, "https://test.com", "", "")

	twiml, err := providerCallConnected(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err.Error())
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/provider_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="alice">You have an incoming call from JS</Say><Say voice="alice">Press 1 to answer.</Say></Gather><Hangup></Hangup></Response>`

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestProviderCallConnected_NoName(t *testing.T) {
	patientPhone := "+12061111111"
	practicePhone := "+12062222222"
	providerPhone := "+12063333333"
	orgID := "o1"

	// the params are intended to simulate the dial leg of the call
	// where the call shows up as originating from the practice phone to
	// the number of the provider in the forwarding list
	params := &rawmsg.TwilioParams{
		From:          practicePhone,
		To:            providerPhone,
		ParentCallSID: "12345",
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.LookupIncomingCall, params.ParentCallSID).WithReturns(&models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhone),
		CallSID:        params.ParentCallSID,
	}, nil))

	mdirectory := directorymock.New(t)
	defer mdirectory.Finish()

	mdirectory.Expect(mock.NewExpectation(mdirectory.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: patientPhone,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				Type: directory.EntityType_EXTERNAL,
				Info: &directory.EntityInfo{},
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdirectory, nil, mdal, nil, clock.New(), nil, "https://test.com", "", "")

	twiml, err := providerCallConnected(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err.Error())
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/provider_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="alice">You have an incoming call from 206 111 1111</Say><Say voice="alice">Press 1 to answer.</Say></Gather><Hangup></Hangup></Response>`

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestProviderEnteredDigits_Entered1(t *testing.T) {
	params := &rawmsg.TwilioParams{
		From:   "+14151111111",
		To:     "+14152222222",
		Digits: "1",
	}

	twiml, err := providerEnteredDigits(context.Background(), params, nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response></Response>`
	if expected != twiml {
		t.Fatalf(`\nExpected: %s\nGot:%s`, expected, twiml)
	}
}

func TestProviderEnteredDigits_EnteredOtherDigit(t *testing.T) {

	patientPhone := "+12061111111"
	practicePhone := "+12062222222"
	orgID := "o1"

	// the params are intended to simulate the dial leg of the call
	// where the call shows up as originating from the practice phone to
	// the number of the provider in the forwarding list
	params := &rawmsg.TwilioParams{
		From:          "+14151111111",
		To:            "+14152222222",
		Digits:        "2",
		ParentCallSID: "12345",
	}

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.LookupIncomingCall, params.ParentCallSID).WithReturns(&models.IncomingCall{
		OrganizationID: orgID,
		Source:         phone.Number(patientPhone),
		Destination:    phone.Number(practicePhone),
		CallSID:        params.ParentCallSID,
	}, nil))

	mdirectory := directorymock.New(t)
	defer mdirectory.Finish()

	mdirectory.Expect(mock.NewExpectation(mdirectory.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: patientPhone,
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				Type: directory.EntityType_EXTERNAL,
				Info: &directory.EntityInfo{
					FirstName:   "J",
					LastName:    "S",
					DisplayName: "JS",
				},
				Memberships: []*directory.Entity{
					{
						ID:   orgID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	es := NewEventHandler(mdirectory, nil, mdal, nil, clock.New(), nil, "https://test.com", "", "")
	twiml, err := providerEnteredDigits(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/provider_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="alice">You have an incoming call from JS</Say><Say voice="alice">Press 1 to answer.</Say></Gather><Hangup></Hangup></Response>`

	if expected != twiml {
		t.Fatalf(`\nExpected: %s\nGot:%s`, expected, twiml)
	}
}

func TestVoicemailTwiML(t *testing.T) {
	orgID := "12345"
	providerID := "p1"
	practicePhoneNumber := "+14152222222"
	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       practicePhoneNumber,
						Provisioned: true,
					},
				},
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
				Members: []*directory.Entity{
					{
						ID: providerID,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       "+14151111111",
							},
						},
					},
				},
			},
		},
	}

	params := &rawmsg.TwilioParams{
		From:   "+14151111111",
		To:     "+14152222222",
		Digits: "2",
	}

	es := NewEventHandler(md, nil, nil, nil, clock.New(), nil, "https://test.com", "", "")

	twiml, err := voicemailTWIML(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">You have reached Dewabi Corp. Please leave a message after the tone.</Say><Record action="/twilio/call/process_voicemail" timeout="60" playBeep="true"></Record></Response>`

	if expected != twiml {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncomingCallStatus_CallAnswered(t *testing.T) {
	testIncomingCallStatus(t, rawmsg.TwilioParams_ANSWERED)
}

func TestIncomingCallStatus_CallCompleted(t *testing.T) {
	testIncomingCallStatus(t, rawmsg.TwilioParams_COMPLETED)
}

func TestIncomingCallStatus_OtherCallStatus(t *testing.T) {
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_FAILED)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_NO_ANSWER)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_IN_PROGRESS)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_QUEUED)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_INITIATED)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_BUSY)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_CANCELED)
	testIncomingCallStatus_Other(t, rawmsg.TwilioParams_RINGING)
}

func testIncomingCallStatus_Other(t *testing.T, incomingStatus rawmsg.TwilioParams_CallStatus) {
	conc.Testing = true
	ms := &mockSNS_Twilio{}
	params := &rawmsg.TwilioParams{
		From:           "+12068773590",
		To:             "+17348465522",
		DialCallStatus: incomingStatus,
	}

	orgID := "12345"
	providerID := "p1"
	providerPersonalPhone := "+14152222222"
	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       "+17348465522",
						Provisioned: true,
					},
				},
				Info: &directory.EntityInfo{
					DisplayName: "Dewabi Corp",
				},
				Members: []*directory.Entity{
					{
						ID: providerID,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       providerPersonalPhone,
							},
						},
					},
				},
			},
		},
	}

	es := NewEventHandler(md, nil, nil, ms, clock.New(), nil, "https://test.com", "", "")

	twiml, err := processIncomingCallStatus(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">You have reached Dewabi Corp. Please leave a message after the tone.</Say><Record action="/twilio/call/process_voicemail" timeout="60" playBeep="true"></Record></Response>`
	if expected != twiml {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}

	// ensure that item was published
	if len(ms.published) != 0 {
		t.Fatalf("Expected %d got %d", 0, len(ms.published))
	}
}

func testIncomingCallStatus(t *testing.T, incomingStatus rawmsg.TwilioParams_CallStatus) {
	conc.Testing = true
	ms := &mockSNS_Twilio{}
	params := &rawmsg.TwilioParams{
		From:           "+12068773590",
		To:             "+17348465522",
		DialCallStatus: incomingStatus,
	}

	es := NewEventHandler(nil, nil, nil, ms, clock.New(), nil, "", "", "")

	twiml, err := processIncomingCallStatus(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	} else if twiml != "" {
		t.Fatalf("Expected %s but got %s", "", twiml)
	}

	// ensure that item was published
	if len(ms.published) != 1 {
		t.Fatalf("Expected %d got %d", 1, len(ms.published))
	}

	pem, err := parsePublishedExternalMessage(*ms.published[0].Message)
	if err != nil {
		t.Fatal(err)
	} else if pem.FromChannelID != "+12068773590" {
		t.Fatalf("Expected %s but got %s", "+12068773590", params.From)
	} else if pem.ToChannelID != "+17348465522" {
		t.Fatalf("Expected %s but got %s", "+17348465522", params.To)
	} else if pem.Direction != excomms.PublishedExternalMessage_INBOUND {
		t.Fatalf("Expected %s but got %s", excomms.PublishedExternalMessage_INBOUND, pem.Direction)
	} else if pem.Type != excomms.PublishedExternalMessage_INCOMING_CALL_EVENT {
		t.Fatalf("Expected %s but got %s", excomms.PublishedExternalMessage_INCOMING_CALL_EVENT, pem.Type)
	}
}

func TestOutgoingCallStatus(t *testing.T) {
	conc.Testing = true
	params := &rawmsg.TwilioParams{
		From:          "+12068773590",
		To:            "+17348465522",
		ParentCallSID: "12345",
		CallStatus:    rawmsg.TwilioParams_ANSWERED,
	}

	ms := &mockSNS_Twilio{}
	md := &mockDAL_Twilio{
		cr: &models.CallRequest{
			Source:      "+12068773590",
			Destination: "+14152222222",
			Proxy:       phone.Number("+17348465522"),
		},
	}

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.CallEnded, phone.Number(params.From), phone.Number(params.To)))

	es := NewEventHandler(nil, nil, md, ms, clock.New(), mproxynumberManager, "", "", "")

	twiml, err := processOutgoingCallStatus(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	} else if twiml != "" {
		t.Fatalf("Expected %s got %s", "", twiml)
	}

	if len(ms.published) != 1 {
		t.Fatalf("Expected 1 but got %d", len(ms.published))
	}

	pem, err := parsePublishedExternalMessage(*ms.published[0].Message)
	if err != nil {
		t.Fatal(err)
	} else if pem.FromChannelID != md.cr.Source.String() {
		t.Fatalf("Expected %s but got %s", md.cr.Source, pem.FromChannelID)
	} else if pem.ToChannelID != md.cr.Destination.String() {
		t.Fatalf("Expected %s but got %s", md.cr.Destination.String(), pem.ToChannelID)
	} else if pem.Direction != excomms.PublishedExternalMessage_OUTBOUND {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_OUTBOUND, pem.Direction)
	} else if pem.Type != excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT, pem.Type)
	} else if pem.GetOutgoing().Type != excomms.OutgoingCallEventItem_ANSWERED {
		t.Fatalf("Expectd %s but got %s", excomms.OutgoingCallEventItem_ANSWERED, pem.GetOutgoing().Type)
	} else if pem.GetOutgoing().DurationInSeconds != params.CallDuration {
		t.Fatalf("Expectd %d but got %d", params.CallDuration, pem.GetOutgoing().DurationInSeconds)
	}
}

func TestProcessVoicemail(t *testing.T) {
	conc.Testing = true
	params := &rawmsg.TwilioParams{
		From:              "+12068773590",
		To:                "+17348465522",
		RecordingDuration: 10,
		RecordingURL:      "http://google.com",
	}

	ms := &mockSNS_Twilio{}

	md := &mockDAL_Twilio{
		Expector: &mock.Expector{
			T: t,
		},
	}
	md.Expect(mock.NewExpectation(md.StoreIncomingRawMessage, &rawmsg.Incoming{
		Type: rawmsg.Incoming_TWILIO_VOICEMAIL,
		Message: &rawmsg.Incoming_Twilio{
			Twilio: params,
		},
	}))

	es := NewEventHandler(nil, nil, md, ms, clock.New(), nil, "", "", "")

	twiml, err := processVoicemail(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	} else if twiml != "" {
		t.Fatalf("Expected %s got %s", "", twiml)
	}

	if len(ms.published) != 1 {
		t.Fatalf("Expected 1 but got %d", len(ms.published))
	}

	md.Finish()
}

func parsePublishedExternalMessage(message string) (*excomms.PublishedExternalMessage, error) {
	data, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, err
	}

	var pem excomms.PublishedExternalMessage
	if err := pem.Unmarshal(data); err != nil {
		return nil, err
	}

	return &pem, nil
}
