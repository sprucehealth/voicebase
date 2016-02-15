package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	proxynumber "github.com/sprucehealth/backend/cmd/svc/excomms/internal/proxynumber/mock"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

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
		},
	}, nil
}

type mockDAL_Excomms struct {
	dal.DAL
	ppn     *models.ProvisionedEndpoint
	proxies []*models.ProxyPhoneNumber
	ppnr    *models.ProxyPhoneNumberReservation
	sm      *models.SentMessage
	*mock.Expector
}

func (m *mockDAL_Excomms) LookupProvisionedEndpoint(provisionedFor string, endpointType models.EndpointType) (*models.ProvisionedEndpoint, error) {
	defer m.Record(provisionedFor, endpointType)
	if m.ppn == nil {
		return nil, dal.ErrProvisionedEndpointNotFound
	}
	return m.ppn, nil
}
func (m *mockDAL_Excomms) ProvisionEndpoint(model *models.ProvisionedEndpoint) error {
	defer m.Record(model)
	return nil
}

func (m *mockDAL_Excomms) Transact(trans func(dal.DAL) error) error {
	return trans(m)
}
func (m *mockDAL_Excomms) CreateSentMessage(sm *models.SentMessage) error {
	defer m.Record(sm)
	return nil
}
func (m *mockDAL_Excomms) LookupSentMessageByUUID(uuid, destination string) (*models.SentMessage, error) {
	defer m.Record(uuid, destination)
	if m.sm == nil {
		return nil, dal.ErrSentMessageNotFound
	}
	return m.sm, nil
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
	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
	mi := &mockIncomingPhoneNumberService_Excomms{
		pn: &twilio.IncomingPhoneNumber{
			PhoneNumber: "+14152222222",
		},
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.IncomingPhoneNumber = mi

	mi.Expect(mock.NewExpectation(mi.PurchaseLocal, twilio.PurchasePhoneNumberParams{
		AreaCode: "415",
	}))
	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypePhone))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "+14152222222",
		EndpointType:   models.EndpointTypePhone,
	}))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: "415",
		},
	})
	test.OK(t, err)
	test.Equals(t, mi.pn.PhoneNumber, res.PhoneNumber)

	mi.Finish()
	md.Finish()
}

func TestProvisionPhoneNumber_NotProvisioned_PhoneNumber(t *testing.T) {
	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
	mi := &mockIncomingPhoneNumberService_Excomms{
		pn: &twilio.IncomingPhoneNumber{
			PhoneNumber: "+14152222222",
		},
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.IncomingPhoneNumber = mi

	mi.Expect(mock.NewExpectation(mi.PurchaseLocal, twilio.PurchasePhoneNumberParams{
		PhoneNumber: "+14152222222",
	}))
	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypePhone))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "+14152222222",
		EndpointType:   models.EndpointTypePhone,
	}))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_PhoneNumber{
			PhoneNumber: "+14152222222",
		},
	})
	test.OK(t, err)
	test.Equals(t, mi.pn.PhoneNumber, res.PhoneNumber)

	mi.Finish()
	md.Finish()
}

func TestProvisionPhoneNumber_Idempotent(t *testing.T) {
	md := &mockDAL_Excomms{
		ppn: &models.ProvisionedEndpoint{
			Endpoint:     "+14156666666",
			EndpointType: models.EndpointTypePhone,
		},
		Expector: &mock.Expector{
			T: t,
		},
	}
	mi := &mockIncomingPhoneNumberService_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.IncomingPhoneNumber = mi

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypePhone))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: "415",
		},
	})
	test.OK(t, err)
	test.Equals(t, md.ppn.Endpoint, res.PhoneNumber)

	mi.Finish()
	md.Finish()
}

func TestProvisionPhoneNumber_AlreadyProvisioned(t *testing.T) {
	md := &mockDAL_Excomms{
		ppn: &models.ProvisionedEndpoint{
			Endpoint:     "+14152222222",
			EndpointType: models.EndpointTypePhone,
		},
		Expector: &mock.Expector{
			T: t,
		},
	}
	mi := &mockIncomingPhoneNumberService_Excomms{
		pn: &twilio.IncomingPhoneNumber{
			PhoneNumber: "+14152222222",
		},
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}
	es.twilio.IncomingPhoneNumber = mi

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypePhone))

	_, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_PhoneNumber{
			PhoneNumber: "+14153333333",
		},
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.AlreadyExists, grpc.Code(err))

	mi.Finish()
	md.Finish()
}

func TestSendMessage_SMS(t *testing.T) {
	conc.Testing = true
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}
	mm.Expect(mock.NewExpectation(mm.SendSMS, "+17348465522", "+14152222222", "hello"))

	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
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
			},
		},
		Destination: "+14152222222",
	}))

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

	mm.Finish()
	md.Finish()
}

func TestSendMessage_SMSIdempotent(t *testing.T) {
	conc.Testing = true
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}

	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		sm: &models.SentMessage{},
	}
	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "+14152222222"))

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

	mm.Finish()
	md.Finish()
}

func TestSendMessage_Email(t *testing.T) {
	conc.Testing = true
	me := &mockEmail_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
	em := &models.EmailMessage{
		ID:        "1",
		Subject:   "Hi",
		Body:      "Hello",
		FromName:  "Joe Schmoe",
		FromEmail: "joe@schmoe.com",
		ToEmail:   "patient@example.com",
	}
	me.Expect(mock.NewExpectation(me.SendMessage, em))

	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
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

func TestSendMessage_EmailIdempotent(t *testing.T) {
	conc.Testing = true
	me := &mockEmail_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		sm: &models.SentMessage{},
	}
	md.Expect(mock.NewExpectation(md.LookupSentMessageByUUID, "tag", "patient@example.com"))

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
	conc.Testing = true
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

	conc.Testing = true

	md := dirmock.New(t)
	defer md.Finish()

	// organization lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
			},
		},
	}, nil))

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
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
			},
		},
	}, nil))

	// calee lookup
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: destinationPhoneNumber.String(),
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
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

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID))

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    originatingNumber.String(),
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
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

	conc.Testing = true

	md := dirmock.New(t)
	defer md.Finish()

	// organization lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
			},
		},
	}, nil))

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
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
			},
		},
	}, nil))

	// callee lookup
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, &directory.LookupEntitiesByContactRequest{
		ContactValue: destinationPhoneNumber.String(),
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
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

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CurrentOriginatingNumber, callerEntityID).WithReturns(originatingNumber, nil))

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID))

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
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

	conc.Testing = true

	md := dirmock.New(t)
	defer md.Finish()

	// organization lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
			},
		},
	}, nil))

	// caller lookup
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
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
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
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

	mdal := dalmock.New(t)
	defer mdal.Finish()

	mdal.Expect(mock.NewExpectation(mdal.CurrentOriginatingNumber, callerEntityID).WithReturns(phone.Number(""), dal.ErrOriginatingNumberNotFound))

	mdal.Expect(mock.NewExpectation(mdal.SetCurrentOriginatingNumber, originatingNumber, callerEntityID))

	mproxynumberManager := proxynumber.NewMockManager(t)
	defer mproxynumberManager.Finish()

	mproxynumberManager.Expect(mock.NewExpectation(mproxynumberManager.ReserveNumber, originatingNumber, destinationPhoneNumber, destinationEntityID, callerEntityID, organizationID).WithReturns(proxyPhoneNumber, nil))

	es := &excommsService{
		dal:                mdal,
		directory:          md,
		clock:              mclock,
		proxyNumberManager: mproxynumberManager,
	}

	res, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		ToPhoneNumber:      destinationPhoneNumber.String(),
		OrganizationID:     "1234",
		CallerEntityID:     callerEntityID,
	})
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber.String(), res.ProxyPhoneNumber)
	test.Equals(t, originatingNumber.String(), res.OriginatingPhoneNumber)
}

func TestInitiatePhoneCall_ConnectCallers(t *testing.T) {
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
	}

	mdal := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

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

func TestInitiatePhoneCall_OrgNotFound(t *testing.T) {
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
		resErr:   grpc.Errorf(codes.NotFound, "Not Found"),
	}

	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "1234",
		},
	}))

	mdal := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		dal:       mdal,
		directory: md,
	}

	_, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		OrganizationID:     "1234",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.NotFound, grpc.Code(err))
	md.Finish()
	mdal.Finish()
}

func TestInitiatePhoneCall_InvalidCaller(t *testing.T) {
	callerEntityID := "e1"
	organizationID := "1234"
	md := dirmock.New(t)
	defer md.Finish()

	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: organizationID,
			},
		},
	}, nil))
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{}, grpc.Errorf(codes.NotFound, "")))

	mdal := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
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
			EntityID: organizationID,
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: organizationID,
			},
		},
	}, nil))
	md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: callerEntityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERSHIPS},
		},
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
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}).WithReturns(&directory.LookupEntitiesByContactResponse{}, grpc.Errorf(codes.NotFound, "")))

	mdal := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}
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
	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail))
	md.Expect(mock.NewExpectation(md.ProvisionEndpoint, &models.ProvisionedEndpoint{
		ProvisionedFor: "test",
		Endpoint:       "test@subdomain.domain.com",
		EndpointType:   models.EndpointTypeEmail,
	}))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.OK(t, err)
	test.Equals(t, "test@subdomain.domain.com", res.EmailAddress)
}

func TestProvisionEmailAddress_Idempotent(t *testing.T) {
	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		ppn: &models.ProvisionedEndpoint{
			ProvisionedFor: "test",
			Endpoint:       "test@subdomain.domain.com",
			EndpointType:   models.EndpointTypeEmail,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.OK(t, err)
	test.Equals(t, "test@subdomain.domain.com", res.EmailAddress)
}

func TestProvisionEmailAddress_AlreadyProvisionedWithDifferentAddress(t *testing.T) {
	md := &mockDAL_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		ppn: &models.ProvisionedEndpoint{
			ProvisionedFor: "test",
			Endpoint:       "test12345@subdomain.domain.com",
			EndpointType:   models.EndpointTypeEmail,
		},
	}

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
		dal:    md,
	}

	md.Expect(mock.NewExpectation(md.LookupProvisionedEndpoint, "test", models.EndpointTypeEmail))

	res, err := es.ProvisionEmailAddress(context.Background(), &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: "test",
		EmailAddress: "test@subdomain.domain.com",
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.AlreadyExists, grpc.Code(err))
	test.Equals(t, true, res == nil)
}
