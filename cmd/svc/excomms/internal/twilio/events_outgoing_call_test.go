package twilio

import (
	"fmt"
	"testing"
	"time"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	proxynumber "github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/rawmsg"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/directory"
	directorymock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/excomms"
)

func TestOutgoing_Process(t *testing.T) {
	testOutgoing(t, false, "")
}

func TestOutgoing_Expired(t *testing.T) {
	testOutgoing(t, true, "")
}

func TestOutgoing_BlockedCallerID(t *testing.T) {
	providerPersonalPhoneNumber := phone.Number(phone.NumberBlocked)
	proxyPhoneNumber := phone.Number("+14152222222")
	organizationID := "1234"
	providerID := "0000"
	callSID := "8888"
	conc.Testing = true

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.ActiveProxyPhoneNumberReservation, (*phone.Number)(nil), (*phone.Number)(nil), phone.Ptr(proxyPhoneNumber)).WithReturns(&models.ProxyPhoneNumberReservation{
		OrganizationID: organizationID,
		OwnerEntityID:  providerID,
	}, nil))

	mclock := clock.NewManaged(time.Now())

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(nil, nil, mdal, &mockSNS_Twilio{}, mclock, nil, "https://test.com", "", "", "", signer)

	params := &rawmsg.TwilioParams{
		From:    providerPersonalPhoneNumber.String(),
		To:      proxyPhoneNumber.String(),
		CallSID: callSID,
	}

	twiml, err := processOutgoingCall(context.Background(), params, es.(*eventsHandler))

	if err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">Outbound calls cannot be made from a phone with blocked caller ID. We use the phone number you are calling from to verify your identity and connect the call via your Spruce phone number. Please try again after unblocking your caller ID.</Say><Say voice="alice">Thank you!</Say></Response>`)
	if twiml != expected {
		t.Fatalf("\nExpected %s\nGot %s", expected, twiml)
	}
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

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(md, nil, mdal, &mockSNS_Twilio{}, mclock, mproxynumberManager, "https://test.com", "", "", "", signer)

	params := &rawmsg.TwilioParams{
		From:    providerPersonalPhoneNumber.String(),
		To:      proxyPhoneNumber.String(),
		CallSID: callSID,
	}

	twiml, err := processOutgoingCall(context.Background(), params, es.(*eventsHandler))
	if testExpired {
		if err != nil {
			t.Fatal(err)
		}

		expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response><Say voice="alice">Outbound calls to patients should be initiated from within the Spruce app. Please hang up and call the patient you are trying to reach by tapping the phone icon within their conversation thread. Thank you!</Say></Response>`)
		if twiml != expected {
			t.Fatalf("\nExpected %s\nGot %s", expected, twiml)
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
			Source:         "+12068773590",
			Destination:    "+14152222222",
			Proxy:          phone.Number("+17348465522"),
			CallerEntityID: "e1",
		},
	}

	mdir := directorymock.New(t)
	defer mdir.Finish()

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "e1",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_EXTERNAL_IDS,
			},
		},
		Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:          "e1",
				ExternalIDs: []string{"account_1"},
			},
		},
	}, nil))

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.CallEnded, phone.Number(params.From), phone.Number(params.To)))

	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())

	es := NewEventHandler(mdir, nil, md, ms, clock.New(), mproxynumberManager, "", "", "", "", signer)

	twiml, err := processOutgoingCallStatus(context.Background(), params, es.(*eventsHandler))
	if err != nil {
		t.Fatal(err)
	} else if twiml != "" {
		t.Fatalf("Expected %s got %s", "", twiml)
	}

	if len(ms.published) != 3 {
		t.Fatalf("Expected 3 but got %d", len(ms.published))
	}

	// Note that the first two items in the sqs queue should be delete resource requests
	pem, err := parsePublishedExternalMessage(*ms.published[2].Message)
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
