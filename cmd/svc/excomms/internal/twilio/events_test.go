package twilio

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
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

func (m *mockDAL_Twilio) ActiveProxyPhoneNumberReservation(lookup *dal.ProxyPhoneNumberReservationLookup) (*models.ProxyPhoneNumberReservation, error) {
	m.Record(lookup)
	if m.ppnr == nil {
		return nil, dal.ErrProxyPhoneNumberReservationNotFound
	}
	return m.ppnr, nil
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
	practicePhoneNumber := "+12068773590"
	patientPhoneNumber := "+11234567890"
	providerPersonalPhoneNumber := "+17348465522"
	proxyPhoneNumber := phone.Number("+14152222222")
	organizationID := "1234"
	destinationEntityID := "6789"
	providerID := "0000"
	callSID := "8888"

	var ppnr *models.ProxyPhoneNumberReservation
	if !testExpired {
		ppnr = &models.ProxyPhoneNumberReservation{
			PhoneNumber:         proxyPhoneNumber,
			DestinationEntityID: destinationEntityID,
			OwnerEntityID:       providerID,
			OrganizationID:      organizationID,
		}
	}

	md := &mockDirectoryService_Twilio{
		entities: map[string][]*directory.Entity{
			"1234": []*directory.Entity{
				{
					ID:   organizationID,
					Type: directory.EntityType_ORGANIZATION,
					Contacts: []*directory.Contact{
						{
							Provisioned: true,
							ContactType: directory.ContactType_PHONE,
							Value:       practicePhoneNumber,
						},
					},
				},
			},
			"6789": []*directory.Entity{
				{
					ID:   destinationEntityID,
					Type: directory.EntityType_EXTERNAL,
					Name: patientName,
					Contacts: []*directory.Contact{
						{
							Provisioned: false,
							ContactType: directory.ContactType_PHONE,
							Value:       patientPhoneNumber,
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
		ppnr: ppnr,
	}
	mdal.Expect(mock.NewExpectation(mdal.ActiveProxyPhoneNumberReservation, &dal.ProxyPhoneNumberReservationLookup{
		ProxyPhoneNumber: ptr.String(proxyPhoneNumber.String()),
	}))

	mclock := clock.NewManaged(time.Now())

	if !testExpired {
		mdal.Expect(mock.NewExpectation(mdal.CreateCallRequest, &models.CallRequest{
			Source:         phone.Number(providerPersonalPhoneNumber),
			Destination:    phone.Number(patientPhoneNumber),
			Proxy:          proxyPhoneNumber,
			OrganizationID: organizationID,
			CallSID:        callSID,
			Requested:      mclock.Now(),
		}))
	}

	es := NewEventHandler(md, mdal, nil, mclock, "https://test.com", "", "")

	params := &rawmsg.TwilioParams{
		From:    providerPersonalPhoneNumber,
		To:      proxyPhoneNumber.String(),
		CallSID: callSID,
	}

	twiml, err := processOutgoingCall(params, es.(*eventsHandler))
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
	providerID := "6789"
	providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"

	md := &mockDirectoryService_Twilio{
		entitiesList: []*directory.Entity{
			{
				ID:   orgID,
				Type: directory.EntityType_ORGANIZATION,
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       providerPersonalPhone,
					},
				},
				Members: []*directory.Entity{
					{
						ID:   providerID,
						Type: directory.EntityType_INTERNAL,
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

	es := NewEventHandler(md, nil, nil, clock.New(), "https://test.com", "", "")
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="%s"><Number url="/twilio/call/provider_call_connected">%s</Number></Dial></Response>`, practicePhoneNumber, providerPersonalPhone)

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
					{
						ContactType: directory.ContactType_PHONE,
						Value:       listedNumber1,
					},
					{
						ContactType: directory.ContactType_PHONE,
						Value:       listedNumber2,
					},
					{
						ContactType: directory.ContactType_PHONE,
						Value:       listedNumber3,
					},
				},
			},
		},
	}

	es := NewEventHandler(md, nil, nil, clock.New(), "https://test.com", "", "")
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*eventsHandler))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/call/process_incoming_call_status" timeout="30" callerId="+14150000000"><Number url="/twilio/call/provider_call_connected">+14152222222</Number><Number url="/twilio/call/provider_call_connected">+14153333333</Number><Number url="/twilio/call/provider_call_connected">+14154444444</Number></Dial></Response>`)

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
						Value:       providerPersonalPhone,
					},
				},
			},
		},
	}

	es := NewEventHandler(md, nil, nil, clock.New(), "https://test.com", "", "")
	params := &rawmsg.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*eventsHandler))
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
	params := &rawmsg.TwilioParams{
		From: "+14151111111",
		To:   "+14152222222",
	}

	twiml, err := providerCallConnected(params, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/call/provider_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="woman">You have an incoming call. Press 1 to answer.</Say></Gather><Hangup></Hangup></Response>`

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

	twiml, err := providerEnteredDigits(params, nil)
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
	params := &rawmsg.TwilioParams{
		From:   "+14151111111",
		To:     "+14152222222",
		Digits: "2",
	}

	twiml, err := providerEnteredDigits(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Hangup></Hangup></Response>`
	if expected != twiml {
		t.Fatalf(`\nExpected: %s\nGot:%s`, expected, twiml)
	}
}

func TestVoicemailTwiML(t *testing.T) {
	twiml, err := voicemailTWIML(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Play><loop>0</loop><digits></digits>http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3</Play><Record action="/twilio/call/process_voicemail" timeout="60" playBeep="true"></Record></Response>`

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

	es := NewEventHandler(nil, nil, ms, clock.New(), "https://test.com", "", "")

	twiml, err := processIncomingCallStatus(params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Play><loop>0</loop><digits></digits>http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3</Play><Record action="/twilio/call/process_voicemail" timeout="60" playBeep="true"></Record></Response>`
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

	es := NewEventHandler(nil, nil, ms, clock.New(), "", "", "")

	twiml, err := processIncomingCallStatus(params, es.(*eventsHandler))
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
	} else if pem.Type != excomms.PublishedExternalMessage_CALL_EVENT {
		t.Fatalf("Expected %s but got %s", excomms.PublishedExternalMessage_CALL_EVENT, pem.Type)
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
		},
	}

	es := NewEventHandler(nil, md, ms, clock.New(), "", "", "")

	twiml, err := processOutgoingCallStatus(params, es.(*eventsHandler))
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
	} else if pem.Type != excomms.PublishedExternalMessage_CALL_EVENT {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_CALL_EVENT, pem.Type)
	} else if pem.GetCallEventItem().Type != excomms.CallEventItem_OUTGOING_ANSWERED {
		t.Fatalf("Expectd %s but got %s", excomms.CallEventItem_OUTGOING_ANSWERED, pem.GetCallEventItem().Type)
	} else if pem.GetCallEventItem().DurationInSeconds != params.CallDuration {
		t.Fatalf("Expectd %d but got %d", params.CallDuration, pem.GetCallEventItem().DurationInSeconds)
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

	es := NewEventHandler(nil, md, ms, clock.New(), "", "", "")

	twiml, err := processVoicemail(params, es.(*eventsHandler))
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
	var input struct {
		Default string `json:"default"`
	}
	if err := json.Unmarshal([]byte(message), &input); err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(input.Default)
	if err != nil {
		return nil, err
	}

	var pem excomms.PublishedExternalMessage
	if err := pem.Unmarshal(data); err != nil {
		return nil, err
	}

	return &pem, nil
}
