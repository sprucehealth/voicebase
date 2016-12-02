package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	proxynumber "github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber/mock"
	exsettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/twilio"
	twiliomock "github.com/sprucehealth/backend/libs/twilio/mock"
	"github.com/sprucehealth/backend/libs/urlutil"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/events"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/settings/settingsmock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func init() {
	conc.Testing = true
}

type mockIDGen struct{}

func newMockIDGen() idGenerator {
	return &mockIDGen{}
}

func (*mockIDGen) NewID() (uint64, error) {
	return uint64(1), nil
}

type mockAvailablePhoneNumberService_Excomms struct {
	twilio.AvailablephoneNumbersIFace
	*mock.Expector

	phoneNumbers []*twilio.AvailablePhoneNumber
}

func (m *mockAvailablePhoneNumberService_Excomms) ListLocal(params twilio.AvailablePhoneNumbersParams) ([]*twilio.AvailablePhoneNumber, *twilio.Response, error) {
	defer m.Record(params)
	return m.phoneNumbers, nil, nil
}

type mockIncomingPhoneNumberService_Excomms struct {
	twilio.IncomingPhoneNumberIFace
	*mock.Expector
	pn *twilio.IncomingPhoneNumber
}

func (m *mockIncomingPhoneNumberService_Excomms) PurchaseLocal(params twilio.PurchasePhoneNumberParams) (*twilio.IncomingPhoneNumber, *twilio.Response, error) {
	defer m.Record(params)
	return m.pn, &twilio.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(&bytes.Reader{}),
		},
	}, nil
}

type mockMessages_Excomms struct {
	twilio.MessageIFace
	*mock.Expector
	msg *twilio.Message
}

func (m *mockMessages_Excomms) SendSMS(from, to, body string) (*twilio.Message, *twilio.Response, error) {
	defer m.Record(from, to, body)
	return m.msg, nil, nil
}
func (m *mockMessages_Excomms) Send(from, to string, params twilio.MessageParams) (*twilio.Message, *twilio.Response, error) {
	defer m.Record(from, to, params)
	return m.msg, nil, nil
}

type mockEmail_Excomms struct {
	*mock.Expector
}

func (m *mockEmail_Excomms) SendMessage(em *models.EmailMessage) error {
	defer m.Record(em)
	return nil
}

type mockDirectory_Excomms struct {
	directory.DirectoryClient
	*mock.Expector
	res        *directory.LookupEntitiesResponse
	resErr     error
	contactRes map[string]*directory.LookupEntitiesByContactResponse
	contactErr map[string]error
}

func (m *mockDirectory_Excomms) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	defer func() {
		if len(opts) > 0 {
			m.Record(ctx, in, opts)
		} else {
			m.Record(ctx, in)
		}
	}()
	return m.res, m.resErr
}
func (m *mockDirectory_Excomms) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	defer func() {
		if len(opts) > 0 {
			m.Record(ctx, in, opts)
		} else {
			m.Record(ctx, in)
		}
	}()
	return m.contactRes[in.ContactValue], m.contactErr[in.ContactValue]
}

func TestSearchAvailablePhoneNumbers(t *testing.T) {
	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
	}

	m := &mockAvailablePhoneNumberService_Excomms{
		phoneNumbers: []*twilio.AvailablePhoneNumber{
			{
				PhoneNumber: "+14152222222",
				Capabilities: map[string]bool{
					"voice": true,
					"mms":   true,
					"sms":   true,
				},
			},
			{
				PhoneNumber: "+14155555555",
				Capabilities: map[string]bool{
					"voice": true,
					"mms":   false,
					"sms":   false,
				},
			},
		},
		Expector: &mock.Expector{
			T: t,
		},
	}
	es.twilio.AvailablePhoneNumbers = m

	m.Expect(mock.NewExpectation(m.ListLocal, twilio.AvailablePhoneNumbersParams{
		AreaCode:                      "415",
		ExcludeAllAddressRequired:     true,
		ExcludeLocalAddressRequired:   true,
		ExcludeForeignAddressRequired: true,
		VoiceEnabled:                  true,
		SMSEnabled:                    false,
		MMSEnabled:                    false,
	}))

	res, err := es.SearchAvailablePhoneNumbers(context.Background(), &excomms.SearchAvailablePhoneNumbersRequest{
		AreaCode: "415",
		Capabilities: []excomms.PhoneNumberCapability{
			excomms.PhoneNumberCapability_VOICE_ENABLED,
		},
	})
	test.OK(t, err)
	test.Equals(t, len(m.phoneNumbers), len(res.PhoneNumbers))
	test.Equals(t, m.phoneNumbers[0].PhoneNumber, res.PhoneNumbers[0].PhoneNumber)
	test.Equals(t, m.phoneNumbers[1].PhoneNumber, res.PhoneNumbers[1].PhoneNumber)
	test.Equals(t, []excomms.PhoneNumberCapability{excomms.PhoneNumberCapability_VOICE_ENABLED, excomms.PhoneNumberCapability_SMS_ENABLED, excomms.PhoneNumberCapability_MMS_ENABLED}, res.PhoneNumbers[0].Capabilities)
	test.Equals(t, []excomms.PhoneNumberCapability{excomms.PhoneNumberCapability_VOICE_ENABLED}, res.PhoneNumbers[1].Capabilities)
	m.Finish()
}

func TestProvisionPhoneNumber_NotProvisioned_AreaCode(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	mi := &mockIncomingPhoneNumberService_Excomms{
		pn: &twilio.IncomingPhoneNumber{
			PhoneNumber: "+14152222222",
		},
		Expector: &mock.Expector{
			T: t,
		},
	}
	defer mi.Finish()

	snsC := mock.NewSNSAPI(t)
	es := &excommsService{
		twilio:     twilio.NewClient("", "", nil),
		dal:        md,
		sns:        snsC,
		eventTopic: "eventsTopic",
	}
	es.twilio.IncomingPhoneNumber = mi

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpointByUUID, "abc").WithReturns((*models.ProvisionedEndpoint)(nil), dal.ErrProvisionedEndpointNotFound))

	mi.Expect(mock.NewExpectation(mi.PurchaseLocal, twilio.PurchasePhoneNumberParams{
		AreaCode: "415",
	}))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "+14152222222",
		EndpointType:   models.EndpointTypePhone,
	}, "abc"))

	eventData, err := events.MarshalEnvelope(events.Service_EXCOMMS, &excomms.Event{
		Type: excomms.Event_PROVISIONED_ENDPOINT,
		Details: &excomms.Event_ProvisionedEndpoint{
			ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
				ForEntityID:  "test",
				EndpointType: excomms.EndpointType_PHONE,
				Endpoint:     "+14152222222",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: "415",
		},
		UUID: "abc",
	})
	test.OK(t, err)
	test.Equals(t, mi.pn.PhoneNumber, res.PhoneNumber)
}

func TestProvisionPhoneNumber_NotProvisioned_PhoneNumber(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	mi := &mockIncomingPhoneNumberService_Excomms{
		pn: &twilio.IncomingPhoneNumber{
			PhoneNumber: "+14152222222",
		},
		Expector: &mock.Expector{
			T: t,
		},
	}
	defer mi.Finish()

	snsC := mock.NewSNSAPI(t)
	es := &excommsService{
		twilio:     twilio.NewClient("", "", nil),
		dal:        md,
		sns:        snsC,
		eventTopic: "eventsTopic",
	}
	es.twilio.IncomingPhoneNumber = mi

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpointByUUID, "abc").WithReturns((*models.ProvisionedEndpoint)(nil), dal.ErrProvisionedEndpointNotFound))

	mi.Expect(mock.NewExpectation(mi.PurchaseLocal, twilio.PurchasePhoneNumberParams{
		PhoneNumber: "+14152222222",
	}))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "+14152222222",
		EndpointType:   models.EndpointTypePhone,
	}, "abc"))

	eventData, err := events.MarshalEnvelope(events.Service_EXCOMMS, &excomms.Event{
		Type: excomms.Event_PROVISIONED_ENDPOINT,
		Details: &excomms.Event_ProvisionedEndpoint{
			ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
				ForEntityID:  "test",
				EndpointType: excomms.EndpointType_PHONE,
				Endpoint:     "+14152222222",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_PhoneNumber{
			PhoneNumber: "+14152222222",
		},
		UUID: "abc",
	})
	test.OK(t, err)
	test.Equals(t, mi.pn.PhoneNumber, res.PhoneNumber)
}

func TestProvisionPhoneNumber_Idempotent(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	mi := &mockIncomingPhoneNumberService_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
	defer mi.Finish()

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.IncomingPhoneNumber = mi

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpointByUUID, "abc").WithReturns(
		&models.ProvisionedEndpoint{
			ProvisionedFor: "test",
			EndpointType:   models.EndpointTypePhone,
			Endpoint:       "+16305551212",
		}, nil))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: "415",
		},
		UUID: "abc",
	})
	test.OK(t, err)
	test.Equals(t, "+16305551212", res.PhoneNumber)
}

func TestSendMessage_SMS(t *testing.T) {
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}
	clk := clock.NewManaged(time.Now())
	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())
	resizedURL1, err := signer.SignedURL("/media/mediaid1/thumbnail", url.Values{
		"width":  []string{"3264"},
		"height": []string{"3264"},
	}, ptr.Time(clk.Now().Add(time.Minute*15)))
	resizedURL2, err := signer.SignedURL("/media/mediaid2/thumbnail", url.Values{
		"width":  []string{"3264"},
		"height": []string{"3264"},
	}, ptr.Time(clk.Now().Add(time.Minute*15)))
	mHTTPClient := mock.NewHttpClient(t)
	mHTTPClient.Expect(mock.NewExpectation(mHTTPClient.Head, resizedURL1).WithReturns(&http.Response{
		Body: ioutil.NopCloser(nil),
	}, nil))
	mHTTPClient.Expect(mock.NewExpectation(mHTTPClient.Head, resizedURL2).WithReturns(&http.Response{
		Body: ioutil.NopCloser(nil),
	}, nil))
	mm.Expect(mock.NewExpectation(mm.Send, "+17348465522", "+14152222222", twilio.MessageParams{
		Body:           "hello",
		ApplicationSid: "1234",
		MediaUrl:       []string{resizedURL1, resizedURL2},
	}))

	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "+14152222222"))
	md.Expect(mock.NewExpectation(md.CreateSentMessage, &models.SentMessage{
		Type: models.SentMessage_SMS,
		UUID: "tag",
		Message: &models.SentMessage_SMSMsg{
			SMSMsg: &models.SMSMessage{
				ID:              "",
				FromPhoneNumber: "+17348465522",
				ToPhoneNumber:   "+14152222222",
				Text:            "hello",
				DateCreated:     uint64(time.Time{}.Unix()),
				DateSent:        uint64(time.Time{}.Unix()),
				MediaURLs:       []string{resizedURL1, resizedURL2},
			},
		},
		Destination: "+14152222222",
	}))

	es := &excommsService{
		twilio:               twilio.NewClient("", "", nil),
		dal:                  md,
		twilioApplicationSID: "1234",
		clock:                clk,
		signer:               signer,
		httpClient:           mHTTPClient,
	}
	es.twilio.Messages = mm

	_, err = es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		UUID:    "tag",
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				FromPhoneNumber: "+17348465522",
				ToPhoneNumber:   "+14152222222",
				Text:            "hello",
				MediaIDs:        []string{"s3://region/bucket/media/mediaid1", "s3://region/bucket/media/mediaid2"},
			},
		},
	})
	test.OK(t, err)

	mm.Finish()
	md.Finish()
}

func TestSendMessage_SMSIdempotent(t *testing.T) {
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}
	md := dalmock.New(t)
	defer mock.FinishAll(mm, md)

	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "+14152222222").WithReturns(&models.SentMessage{}, nil))

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.Messages = mm

	_, err := es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		UUID:    "tag",
		Channel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				FromPhoneNumber: "+17348465522",
				ToPhoneNumber:   "+14152222222",
				Text:            "hello",
			},
		},
	})
	test.OK(t, err)
}

func TestSendMessage_Email(t *testing.T) {
	me := &mockEmail_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
	clk := clock.NewManaged(time.Now())
	sig, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	signer := urlutil.NewSigner("apiDomain", sig, clock.New())
	resizedURL1, err := signer.SignedURL("/media/mediaid1/thumbnail", url.Values{
		"width":  []string{"3264"},
		"height": []string{"3264"},
	}, ptr.Time(clk.Now().Add(time.Minute*15)))
	test.OK(t, err)
	resizedURL2, err := signer.SignedURL("/media/mediaid2/thumbnail", url.Values{
		"width":  []string{"3264"},
		"height": []string{"3264"},
	}, ptr.Time(clk.Now().Add(time.Minute*15)))
	test.OK(t, err)
	mHTTPClient := mock.NewHttpClient(t)
	mHTTPClient.Expect(mock.NewExpectation(mHTTPClient.Head, resizedURL1).WithReturns(&http.Response{
		Body: ioutil.NopCloser(nil),
	}, nil))
	mHTTPClient.Expect(mock.NewExpectation(mHTTPClient.Head, resizedURL2).WithReturns(&http.Response{
		Body: ioutil.NopCloser(nil),
	}, nil))
	em := &models.EmailMessage{
		ID:                    "1",
		Subject:               "Hi",
		Body:                  "Hello",
		FromName:              "Joe Schmoe",
		FromEmail:             "joe@schmoe.com",
		ToEmail:               "patient@example.com",
		MediaURLs:             []string{resizedURL1, resizedURL2},
		TemplateSubstitutions: []*models.EmailMessage_Substitution{},
	}
	me.Expect(mock.NewExpectation(me.SendMessage, em))

	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "patient@example.com"))
	md.Expect(mock.NewExpectation(md.CreateSentMessage, &models.SentMessage{
		ID:   1,
		Type: models.SentMessage_EMAIL,
		UUID: "tag",
		Message: &models.SentMessage_EmailMsg{
			EmailMsg: em,
		},
		Destination: "patient@example.com",
	}))

	es := &excommsService{
		twilio:            twilio.NewClient("", "", nil),
		dal:               md,
		emailClient:       me,
		idgen:             newMockIDGen(),
		clock:             clk,
		signer:            signer,
		httpClient:        mHTTPClient,
		spruceEmailDomain: "schmoe",
	}

	_, err = es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		UUID:    "tag",
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Hi",
				Body:             "Hello",
				FromName:         "Joe Schmoe",
				FromEmailAddress: "joe@schmoe.com",
				ToEmailAddress:   "patient@example.com",
				MediaIDs:         []string{"s3://region/bucket/media/mediaid1", "s3://region/bucket/media/mediaid2"},
			},
		},
	})
	test.OK(t, err)

	me.Finish()
	md.Finish()
}

func TestSendMessage_EmailIdempotent(t *testing.T) {
	me := &mockEmail_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

	md := dalmock.New(t)
	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "patient@example.com").WithReturns(&models.SentMessage{}, nil))

	es := &excommsService{
		twilio:      twilio.NewClient("", "", nil),
		dal:         md,
		emailClient: me,
		idgen:       newMockIDGen(),
	}

	_, err := es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		UUID:    "tag",
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Hi",
				Body:             "Hello",
				FromName:         "Joe Schmoe",
				FromEmailAddress: "joe@schmoe.com",
				ToEmailAddress:   "patient@example.com",
			},
		},
	})
	test.OK(t, err)

	me.Finish()
	md.Finish()
}

func TestSendMessage_VoiceNotSupported(t *testing.T) {
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
	}
	es.twilio.Messages = mm

	_, err := es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_VOICE,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				FromPhoneNumber: "+17348465522",
				ToPhoneNumber:   "+14152222222",
				Text:            "hello",
			},
		},
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.Unimplemented, grpc.Code(err))

	mm.Finish()
}

func TestInitiatePhoneCall_OriginatingNumberSpecified(t *testing.T) {
	mclock := clock.NewManaged(time.Now())
	callerEntityID := "e1"
	destinationEntityID := "d1"
	organizationID := "1234"

	originatingNumber, err := phone.ParseNumber("+17348465522")
	test.OK(t, err)

	destinationPhoneNumber, err := phone.ParseNumber("+14152222222")
	test.OK(t, err)

	proxyPhoneNumber, err := phone.ParseNumber("+12061111111")
	test.OK(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	md := dirmock.New(t)
	defer md.Finish()
	mdal := dalmock.New(t)
	defer mdal.Finish()
	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   callerEntityID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:       organizationID,
						Type:     directory.EntityType_ORGANIZATION,
						Contacts: nil,
					},
				},
			},
		},
	}, nil))

	// callee lookup
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: destinationPhoneNumber.String(),
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   destinationEntityID,
				Type: directory.EntityType_EXTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID, "deviceID"))

	ctx := context.Background()

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: callerEntityID,
		Keys: []*settings.ConfigKey{
			{
				Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
			},
		},
	}).Return(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Text{
					Text: &settings.TextValue{Value: "+14155550001"},
				},
			},
		},
	}, nil)

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, phone.Number("+14155550001"), destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		settings:           mockSettings,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(ctx, &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    originatingNumber.String(),
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
		DeviceID:           "deviceID",
	})
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber.String(), res.ProxyPhoneNumber)
	test.Equals(t, originatingNumber.String(), res.OriginatingPhoneNumber)
}

func TestInitiatePhoneCall_OriginatingNumberNotSpecified_ButExists(t *testing.T) {
	mclock := clock.NewManaged(time.Now())
	callerEntityID := "e1"
	destinationEntityID := "d1"
	organizationID := "1234"

	originatingNumber, err := phone.ParseNumber("+17348465522")
	test.OK(t, err)

	destinationPhoneNumber, err := phone.ParseNumber("+14152222222")
	test.OK(t, err)

	proxyPhoneNumber, err := phone.ParseNumber("+12061111111")
	test.OK(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	md := dirmock.New(t)
	defer md.Finish()
	mdal := dalmock.New(t)
	defer mdal.Finish()
	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   callerEntityID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
						Contacts: []*directory.Contact{
							{
								ContactType: directory.ContactType_PHONE,
								Value:       "+14155550002",
								Provisioned: true,
							},
						},
					},
				},
			},
		},
	}, nil))

	// callee lookup
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: destinationPhoneNumber.String(),
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   destinationEntityID,
				Type: directory.EntityType_EXTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	mdal.Expect(mock.NewExpectation(mdal.CurrentOriginatingNumber, callerEntityID, "deviceID").WithReturns(originatingNumber, nil))

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID, "deviceID"))

	ctx := context.Background()

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: callerEntityID,
		Keys: []*settings.ConfigKey{
			{
				Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
			},
		},
	}).Return(&settings.GetValuesResponse{Values: nil}, nil)

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, phone.Number("+14155550002"), destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		settings:           mockSettings,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(ctx, &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
		DeviceID:           "deviceID",
	})
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber.String(), res.ProxyPhoneNumber)
	test.Equals(t, originatingNumber.String(), res.OriginatingPhoneNumber)
}

func TestInitiatePhoneCall_OriginatingNumberNotSpecified_DoesNotExist(t *testing.T) {
	mclock := clock.NewManaged(time.Now())
	callerEntityID := "e1"
	destinationEntityID := "d1"
	organizationID := "1234"

	originatingNumber, err := phone.ParseNumber("+17348465522")
	test.OK(t, err)

	destinationPhoneNumber, err := phone.ParseNumber("+14152222222")
	test.OK(t, err)

	proxyPhoneNumber, err := phone.ParseNumber("+12061111111")
	test.OK(t, err)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	md := dirmock.New(t)
	defer md.Finish()
	mdal := dalmock.New(t)
	defer mdal.Finish()
	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   callerEntityID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       originatingNumber.String(),
					},
				},
			},
		},
	}, nil))

	// callee lookup
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: destinationPhoneNumber.String(),
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{
		Entities: []*directory.Entity{
			{
				ID:   destinationEntityID,
				Type: directory.EntityType_EXTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	mdal.Expect(mock.NewExpectation(mdal.CurrentOriginatingNumber, callerEntityID, "deviceID").WithReturns(phone.Number(""), dal.ErrOriginatingNumberNotFound))

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID, "deviceID"))

	ctx := context.Background()

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: callerEntityID,
		Keys: []*settings.ConfigKey{
			{
				Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
			},
		},
	}).Return(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Value: &settings.Value_Text{
					Text: &settings.TextValue{Value: "+14155550003"},
				},
			},
		},
	}, nil)

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, phone.Number("+14155550003"), destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		settings:           mockSettings,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(ctx, &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
		DeviceID:           "deviceID",
	})
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber.String(), res.ProxyPhoneNumber)
	test.Equals(t, originatingNumber.String(), res.OriginatingPhoneNumber)
}

func TestInitiatePhoneCall_ConnectCallers(t *testing.T) {
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
	}

	mdal := dalmock.New(t)

	es := &excommsService{
		dal:       mdal,
		directory: md,
	}

	_, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_CONNECT_PARTIES,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		OrganizationID:     "1234",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.Unimplemented, grpc.Code(err))

	md.Finish()
	mdal.Finish()
}

func TestInitiatePhoneCall_CallerNotInOrg(t *testing.T) {
	callerEntityID := "e1"
	md := dirmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   callerEntityID,
				Type: directory.EntityType_INTERNAL,
				Memberships: []*directory.Entity{
					{
						ID:   "nooooope",
						Type: directory.EntityType_ORGANIZATION,
					},
				},
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Value:       "+17348465522",
					},
				},
			},
		},
	}, nil))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	es := &excommsService{
		dal:       mdal,
		directory: md,
	}

	_, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		CallerEntityID:     callerEntityID,
		OrganizationID:     "1234",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestInitiatePhoneCall_InvalidCaller(t *testing.T) {
	callerEntityID := "e1"
	md := dirmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{}, grpcErrorf(codes.NotFound, "")))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	es := &excommsService{
		dal:       mdal,
		directory: md,
	}

	_, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		CallerEntityID:     callerEntityID,
		OrganizationID:     "1234",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestInitiatePhoneCall_InvalidCallee(t *testing.T) {
	callerEntityID := "e1"
	organizationID := "1234"
	md := dirmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: callerEntityID,
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
			},
		},
	}, nil))

	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14152222222",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{}, grpcErrorf(codes.NotFound, "")))

	mdal := dalmock.New(t)
	defer mdal.Finish()

	es := &excommsService{
		dal:       mdal,
		directory: md,
	}

	_, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		CallerEntityID:     callerEntityID,
		OrganizationID:     "1234",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestProvisionEmailAddress(t *testing.T) {
	md := dalmock.New(t)

	snsC := mock.NewSNSAPI(t)
	es := &excommsService{
		twilio:     twilio.NewClient("", "", nil),
		dal:        md,
		sns:        snsC,
		eventTopic: "eventsTopic",
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail).WithReturns((*models.ProvisionedEndpoint)(nil), dal.ErrProvisionedEndpointNotFound))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "test@subdomain.domain.com",
		EndpointType:   models.EndpointTypeEmail,
	}, ""))

	eventData, err := events.MarshalEnvelope(events.Service_EXCOMMS, &excomms.Event{
		Type: excomms.Event_PROVISIONED_ENDPOINT,
		Details: &excomms.Event_ProvisionedEndpoint{
			ProvisionedEndpoint: &excomms.ProvisionedEndpoint{
				ForEntityID:  "test",
				EndpointType: excomms.EndpointType_EMAIL,
				Endpoint:     "test@subdomain.domain.com",
			},
		},
	})
	test.OK(t, err)
	snsC.Expect(mock.NewExpectation(snsC.Publish, &sns.PublishInput{
		Message:  ptr.String(base64.StdEncoding.EncodeToString(eventData)),
		TopicArn: ptr.String("eventsTopic"),
	}).WithReturns(&sns.PublishOutput{}, nil))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.OK(t, err)
	test.Equals(t, "test@subdomain.domain.com", res.EmailAddress)
}

func TestProvisionEmailAddress_Idempotent(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail).WithReturns(
		&models.ProvisionedEndpoint{
			ProvisionedFor: "test",
			Endpoint:       "test@subdomain.domain.com",
			EndpointType:   models.EndpointTypeEmail,
		}, nil))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.OK(t, err)
	test.Equals(t, "test@subdomain.domain.com", res.EmailAddress)
}

func TestProvisionEmailAddress_AlreadyProvisionedWithDifferentAddress(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail).WithReturns(
		&models.ProvisionedEndpoint{
			ProvisionedFor: "test",
			Endpoint:       "test12345@subdomain.domain.com",
			EndpointType:   models.EndpointTypeEmail,
		}, nil))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.AlreadyExists, grpc.Code(err))
	test.Equals(t, true, res == nil)
}

func TestDeprovisionPhoneNumber(t *testing.T) {
	phoneNumber := "+17348465522"
	mc := clock.NewManaged(time.Now())

	md := dalmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.UpdateProvisionedEndpoint, phoneNumber, models.EndpointTypePhone, &dal.ProvisionedEndpointUpdate{
		Deprovisioned:          ptr.Bool(true),
		DeprovisionedTimestamp: ptr.Time(mc.Now()),
		DeprovisionedReason:    ptr.String("sup"),
	}))

	ipn := twiliomock.NewIncomingPhoneNumber(t)
	defer ipn.Finish()

	ipn.Expect(mock.NewExpectation(ipn.List, twilio.ListPurchasedPhoneNumberParams{
		PhoneNumber: phoneNumber,
	}).WithReturns(&twilio.ListPurchasedPhoneNumbersResponse{
		IncomingPhoneNumbers: []*twilio.IncomingPhoneNumber{
			{
				SID: "1",
			},
		},
	}, &twilio.Response{}, nil))

	ipn.Expect(mock.NewExpectation(ipn.Delete, "1"))

	es := &excommsService{
		twilio:     twilio.NewClient("", "", nil),
		dal:        md,
		eventTopic: "eventsTopic",
		clock:      mc,
	}
	es.twilio.IncomingPhoneNumber = ipn

	_, err := es.DeprovisionPhoneNumber(context.Background(), &excomms.DeprovisionPhoneNumberRequest{
		PhoneNumber: phoneNumber,
		Reason:      "sup",
	})
	test.OK(t, err)
}

func TestBlockNumber(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	blockedNumber := phone.Number("+12222222222")
	provisionedPhoneNumber := phone.Number("+13333333333")
	orgID := "orgID1"

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, orgID, models.EndpointTypePhone).WithReturns(
		&models.ProvisionedEndpoint{
			ProvisionedFor: orgID,
			Endpoint:       provisionedPhoneNumber.String(),
		}, nil))
	md.Expect(mock.NewExpectation(md.InsertBlockedNumber, blockedNumber, provisionedPhoneNumber))
	md.Expect(mock.NewExpectation(md.LookupBlockedNumbers, provisionedPhoneNumber))
	es := &excommsService{
		dal: md,
	}
	_, err := es.BlockNumber(context.Background(), &excomms.BlockNumberRequest{
		Number:                 blockedNumber.String(),
		ProvisionedPhoneNumber: provisionedPhoneNumber.String(),
		OrgID: orgID,
	})
	test.OK(t, err)
}

func TestBlockNumber_WrongNumber(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	blockedNumber := phone.Number("+12222222222")
	provisionedPhoneNumber := phone.Number("+13333333333")
	orgID := "orgID1"

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, orgID, models.EndpointTypePhone).WithReturns(
		&models.ProvisionedEndpoint{
			ProvisionedFor: "org2",
			Endpoint:       blockedNumber.String(),
		}, nil))

	es := &excommsService{
		dal: md,
	}
	_, err := es.BlockNumber(context.Background(), &excomms.BlockNumberRequest{
		Number:                 blockedNumber.String(),
		ProvisionedPhoneNumber: provisionedPhoneNumber.String(),
		OrgID: orgID,
	})
	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
}

func TestUnblockNumber(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	blockedNumber := phone.Number("+12222222222")
	provisionedPhoneNumber := phone.Number("+13333333333")

	md.Expect(mock.NewExpectation(md.DeleteBlockedNumber, blockedNumber, provisionedPhoneNumber))
	md.Expect(mock.NewExpectation(md.LookupBlockedNumbers, provisionedPhoneNumber))

	es := &excommsService{
		dal: md,
	}
	_, err := es.UnblockNumber(context.Background(), &excomms.UnblockNumberRequest{
		Number:                 blockedNumber.String(),
		ProvisionedPhoneNumber: provisionedPhoneNumber.String(),
	})
	test.OK(t, err)

}
