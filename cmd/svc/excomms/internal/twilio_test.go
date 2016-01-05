package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/sprucehealth/backend/libs/conc"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

type mockDirectoryService_Twilio struct {
	directory.DirectoryClient
	entities []*directory.Entity
}

func (m *mockDirectoryService_Twilio) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	return &directory.LookupEntitiesResponse{
		Entities: m.entities,
	}, nil
}
func (m *mockDirectoryService_Twilio) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	return &directory.LookupEntitiesByContactResponse{
		Entities: m.entities,
	}, nil
}

type mockDAL_Twilio struct {
	dal.DAL
	cr      *models.CallRequest
	callSID string
}

func (m *mockDAL_Twilio) ValidCallRequest(sourcePhontNumber string) (*models.CallRequest, error) {
	return m.cr, nil
}

func (m *mockDAL_Twilio) UpdateCallRequest(fromPhoneNumber, callSID string) (int64, error) {
	m.callSID = callSID
	return 1, nil
}

func (m *mockDAL_Twilio) LookupCallRequest(fromPhoneNumber string) (*models.CallRequest, error) {
	return m.cr, nil
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
	testOutgoing(t, false)
}

func TestOutgoing_Expired(t *testing.T) {
	testOutgoing(t, true)
}

func testOutgoing(t *testing.T, testExpired bool) {
	practicePhoneNumber := "+12068773590"
	patientPhoneNumber := "+11234567890"
	providerPersonalPhoneNumber := "+17348465522"
	proxyPhoneNumber := "+14152222222"
	organizationID := "1234"
	callSID := "8888"
	requested := time.Now().Add(-time.Hour)
	var expires time.Time
	if testExpired {
		expires = time.Now().Add(-1 * time.Hour / 2)
	} else {
		expires = time.Now().Add(2 * time.Hour)
	}

	md := &mockDirectoryService_Twilio{
		entities: []*directory.Entity{
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
	}

	mdal := &mockDAL_Twilio{
		cr: &models.CallRequest{
			Source:         providerPersonalPhoneNumber,
			Destination:    patientPhoneNumber,
			Proxy:          proxyPhoneNumber,
			OrganizationID: organizationID,
			Requested:      requested,
			Expires:        expires,
		},
	}
	es := NewService("", "", "", mdal, "https://test.com", md, nil, "")

	params := &excomms.TwilioParams{
		From:    providerPersonalPhoneNumber,
		To:      proxyPhoneNumber,
		CallSID: callSID,
	}

	twiml, err := processOutgoingCall(params, es.(*excommsService))
	if testExpired {
		if err == nil {
			t.Fatalf("Expected the call to not go through and be expired.")
		}
	} else {
		if err != nil {
			t.Fatal(err)
		}
		expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial callerId="%s"><Number statusCallbackEvent="ringing answered completed" statusCallback="https://test.com/twilio/process_outgoing_call_status">%s</Number></Dial></Response>`, practicePhoneNumber, patientPhoneNumber)
		if twiml != expected {
			t.Fatalf("\nExpected %s\nGot %s", expected, twiml)
		}
		if callSID != mdal.callSID {
			t.Fatalf("Expected call request to be updated with %s, but got %s", callSID, mdal.callSID)
		}
	}
}

func TestIncoming_Organization(t *testing.T) {
	orgID := "12345"
	providerID := "6789"
	providerPersonalPhone := "+14152222222"
	patientPhone := "+14151111111"
	practicePhoneNumber := "+14150000000"

	md := &mockDirectoryService_Twilio{
		entities: []*directory.Entity{
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

	es := NewService("", "", "", nil, "https://test.com", md, nil, "")
	params := &excomms.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*excommsService))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/process_incoming_call_status" timeout="30" callerId="%s"><Number url="/twilio/provider_call_connected">%s</Number></Dial></Response>`, practicePhoneNumber, providerPersonalPhone)

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
		entities: []*directory.Entity{
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

	es := NewService("", "", "", nil, "https://test.com", md, nil, "")
	params := &excomms.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*excommsService))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/process_incoming_call_status" timeout="30" callerId="+14150000000"><Number url="/twilio/provider_call_connected">+14152222222</Number><Number url="/twilio/provider_call_connected">+14153333333</Number><Number url="/twilio/provider_call_connected">+14154444444</Number></Dial></Response>`)

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
		entities: []*directory.Entity{
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

	es := NewService("", "", "", nil, "https://test.com", md, nil, "")
	params := &excomms.TwilioParams{
		From:    patientPhone,
		To:      practicePhoneNumber,
		CallSID: "",
	}

	twiml, err := processIncomingCall(params, es.(*excommsService))
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Dial action="/twilio/process_incoming_call_status" timeout="30" callerId="%s"><Number url="/twilio/provider_call_connected">%s</Number></Dial></Response>`, practicePhoneNumber, providerPersonalPhone)

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestProviderCallConnected(t *testing.T) {
	params := &excomms.TwilioParams{
		From: "+14151111111",
		To:   "+14152222222",
	}

	twiml, err := providerCallConnected(params, nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Gather action="/twilio/provider_entered_digits" method="POST" timeout="5" numDigits="1"><Say voice="woman">You have an incoming call. Press 1 to answer.</Say></Gather><Hangup></Hangup></Response>`

	if twiml != expected {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestProviderEnteredDigits_Entered1(t *testing.T) {
	params := &excomms.TwilioParams{
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
	params := &excomms.TwilioParams{
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
<Response><Play><loop>0</loop><digits></digits>http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3</Play><Record action="/twilio/process_voicemail" timeout="60" playBeep="true"></Record></Response>`

	if expected != twiml {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}
}

func TestIncomingCallStatus_CallAnswered(t *testing.T) {
	testIncomingCallStatus(t, excomms.TwilioParams_ANSWERED)
}

func TestIncomingCallStatus_CallCompleted(t *testing.T) {
	testIncomingCallStatus(t, excomms.TwilioParams_COMPLETED)
}

func TestIncomingCallStatus_OtherCallStatus(t *testing.T) {
	testIncomingCallStatus_Other(t, excomms.TwilioParams_FAILED)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_NO_ANSWER)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_IN_PROGRESS)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_QUEUED)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_INITIATED)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_BUSY)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_CANCELED)
	testIncomingCallStatus_Other(t, excomms.TwilioParams_RINGING)
}

func testIncomingCallStatus_Other(t *testing.T, incomingStatus excomms.TwilioParams_CallStatus) {
	conc.Testing = true
	ms := &mockSNS_Twilio{}
	params := &excomms.TwilioParams{
		From:           "+12068773590",
		To:             "+17348465522",
		DialCallStatus: incomingStatus,
	}
	es := NewService("", "", "", nil, "https://test.com", nil, ms, "")

	twiml, err := processIncomingCallStatus(params, es.(*excommsService))
	if err != nil {
		t.Fatal(err)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response><Play><loop>0</loop><digits></digits>http://dev-twilio.s3.amazonaws.com/kunal_clinic_voicemail.mp3</Play><Record action="/twilio/process_voicemail" timeout="60" playBeep="true"></Record></Response>`
	if expected != twiml {
		t.Fatalf("\nExpected: %s\nGot: %s", expected, twiml)
	}

	// ensure that item was published
	if len(ms.published) != 0 {
		t.Fatalf("Expected %d got %d", 0, len(ms.published))
	}
}

func testIncomingCallStatus(t *testing.T, incomingStatus excomms.TwilioParams_CallStatus) {
	conc.Testing = true
	ms := &mockSNS_Twilio{}
	params := &excomms.TwilioParams{
		From:           "+12068773590",
		To:             "+17348465522",
		DialCallStatus: incomingStatus,
	}
	es := NewService("", "", "", nil, "https://test.com", nil, ms, "")

	twiml, err := processIncomingCallStatus(params, es.(*excommsService))
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
	params := &excomms.TwilioParams{
		From:          "+12068773590",
		To:            "+17348465522",
		ParentCallSID: "12345",
		CallStatus:    excomms.TwilioParams_ANSWERED,
	}

	ms := &mockSNS_Twilio{}
	md := &mockDAL_Twilio{
		cr: &models.CallRequest{
			Source:      "+12068773590",
			Destination: "+14152222222",
		},
	}

	es := NewService("", "", "", md, "", nil, ms, "")

	twiml, err := processOutgoingCallStatus(params, es.(*excommsService))
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
	} else if pem.FromChannelID != md.cr.Source {
		t.Fatalf("Expected %s but got %s", md.cr.Source, pem.FromChannelID)
	} else if pem.ToChannelID != md.cr.Destination {
		t.Fatalf("Expected %s but got %s", md.cr.Destination, pem.ToChannelID)
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
	params := &excomms.TwilioParams{
		From:              "+12068773590",
		To:                "+17348465522",
		RecordingDuration: 10,
		RecordingURL:      "http://google.com",
	}

	ms := &mockSNS_Twilio{}

	es := NewService("", "", "", nil, "", nil, ms, "")

	twiml, err := processVoicemail(params, es.(*excommsService))
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
	} else if pem.FromChannelID != params.From {
		t.Fatalf("Expected %s but got %s", params.From, pem.FromChannelID)
	} else if pem.ToChannelID != params.To {
		t.Fatalf("Expected %s but got %s", params.To, pem.ToChannelID)
	} else if pem.Direction != excomms.PublishedExternalMessage_INBOUND {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_INBOUND, pem.Direction)
	} else if pem.Type != excomms.PublishedExternalMessage_CALL_EVENT {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_CALL_EVENT, pem.Type)
	} else if pem.GetCallEventItem().Type != excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL {
		t.Fatalf("Expectd %s but got %s", excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL, pem.GetCallEventItem().Type)
	} else if pem.GetCallEventItem().DurationInSeconds != params.RecordingDuration {
		t.Fatalf("Expectd %d but got %d", params.RecordingDuration, pem.GetCallEventItem().DurationInSeconds)
	} else if pem.GetCallEventItem().URL != (params.RecordingURL + ".mp3") {
		t.Fatalf("Expectd %s but got %s", params.RecordingURL, pem.GetCallEventItem().URL)
	}
}

func TestProcessIncomingSMS(t *testing.T) {
	conc.Testing = true
	params := &excomms.TwilioParams{
		From:     "+12068773590",
		To:       "+17348465522",
		Body:     "sms",
		NumMedia: 2,
		MediaItems: []*excomms.TwilioParams_TwilioMediaItem{
			{
				MediaURL:    "http://1.com",
				ContentType: "test",
			},
			{
				MediaURL:    "http://2.com",
				ContentType: "test",
			},
		},
	}

	ms := &mockSNS_Twilio{}
	es := NewService("", "", "", nil, "", nil, ms, "")

	twiml, err := processIncomingSMS(params, es.(*excommsService))
	if err != nil {
		t.Fatal(err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Response></Response>`
	if twiml != expected {
		t.Fatalf("\nExpected %s\nGot %s", expected, twiml)
	}

	if len(ms.published) != 1 {
		t.Fatalf("Expected %d got %d", 1, len(ms.published))
	}

	pem, err := parsePublishedExternalMessage(*ms.published[0].Message)
	if err != nil {
		t.Fatal(err)
	} else if pem.FromChannelID != params.From {
		t.Fatalf("Expected %s but got %s", params.From, pem.FromChannelID)
	} else if pem.ToChannelID != params.To {
		t.Fatalf("Expected %s but got %s", params.To, pem.ToChannelID)
	} else if pem.Direction != excomms.PublishedExternalMessage_INBOUND {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_INBOUND, pem.Direction)
	} else if pem.Type != excomms.PublishedExternalMessage_SMS {
		t.Fatalf("Expectd %s but got %s", excomms.PublishedExternalMessage_SMS, pem.Type)
	} else if pem.GetSMSItem().Text != params.Body {
		t.Fatalf("Expected %s but got %s", pem.GetSMSItem().Text, params.Body)
	} else if len(pem.GetSMSItem().Attachments) != len(params.MediaItems) {
		t.Fatalf("Expected %d but got %d", len(pem.GetSMSItem().Attachments), len(params.MediaItems))
	}
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
