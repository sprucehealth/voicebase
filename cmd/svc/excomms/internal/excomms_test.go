package internal

import (
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/libs/twilio"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
	return m.pn, nil, nil
}

type mockDAL_Excomms struct {
	dal.DAL
	ppn *models.ProvisionedPhoneNumber
	*mock.Expector
}

func (m *mockDAL_Excomms) LookupProvisionedPhoneNumber(lookup *dal.ProvisionedNumberLookup) (*models.ProvisionedPhoneNumber, error) {
	defer m.Record(lookup)
	return m.ppn, nil
}
func (m *mockDAL_Excomms) ProvisionPhoneNumber(model *models.ProvisionedPhoneNumber) error {
	defer m.Record(model)
	return nil
}
func (m *mockDAL_Excomms) CreateCallRequest(cr *models.CallRequest) error {
	// defer m.Record(cr)
	return nil
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

type mockDirectory_Excomms struct {
	directory.DirectoryClient
	*mock.Expector
	res        *directory.LookupEntitiesResponse
	contactRes map[string]*directory.LookupEntitiesByContactResponse
}

func (m *mockDirectory_Excomms) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	defer func() {
		if len(opts) > 0 {
			m.Record(ctx, in, opts)
		} else {
			m.Record(ctx, in)
		}
	}()
	return m.res, nil
}
func (m *mockDirectory_Excomms) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	defer func() {
		if len(opts) > 0 {
			m.Record(ctx, in, opts)
		} else {
			m.Record(ctx, in)
		}
	}()
	return m.contactRes[in.ContactValue], nil
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
	md.Expect(mock.NewExpectation(md.LookupProvisionedPhoneNumber, &dal.ProvisionedNumberLookup{
		ProvisionedFor: ptr.String("test"),
	}))
	md.Expect(mock.NewExpectation(md.ProvisionPhoneNumber, &models.ProvisionedPhoneNumber{
		ProvisionedFor: "test",
		PhoneNumber:    "+14152222222",
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
	md.Expect(mock.NewExpectation(md.LookupProvisionedPhoneNumber, &dal.ProvisionedNumberLookup{
		ProvisionedFor: ptr.String("test"),
	}))
	md.Expect(mock.NewExpectation(md.ProvisionPhoneNumber, &models.ProvisionedPhoneNumber{
		ProvisionedFor: "test",
		PhoneNumber:    "+14152222222",
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
		ppn: &models.ProvisionedPhoneNumber{
			PhoneNumber: "+14156666666",
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

	md.Expect(mock.NewExpectation(md.LookupProvisionedPhoneNumber, &dal.ProvisionedNumberLookup{
		ProvisionedFor: ptr.String("test"),
	}))

	res, err := es.ProvisionPhoneNumber(context.Background(), &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: "test",
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: "415",
		},
	})
	test.OK(t, err)
	test.Equals(t, md.ppn.PhoneNumber, res.PhoneNumber)

	mi.Finish()
	md.Finish()
}

func TestProvisionPhoneNumber_AlreadyProvisioned(t *testing.T) {
	md := &mockDAL_Excomms{
		ppn: &models.ProvisionedPhoneNumber{
			PhoneNumber: "+14152222222",
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

	md.Expect(mock.NewExpectation(md.LookupProvisionedPhoneNumber, &dal.ProvisionedNumberLookup{
		ProvisionedFor: ptr.String("test"),
	}))

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

func TestSendMessage(t *testing.T) {
	mm := &mockMessages_Excomms{
		Expector: &mock.Expector{
			T: t,
		},
		msg: &twilio.Message{},
	}
	mm.Expect(mock.NewExpectation(mm.SendSMS, "+17348465522", "+14152222222", "hello"))

	es := &excommsService{
		twilio: twilio.NewClient("", "", nil),
	}
	es.twilio.Messages = mm

	_, err := es.SendMessage(context.Background(), &excomms.SendMessageRequest{
		FromChannelID: "+17348465522",
		ToChannelID:   "+14152222222",
		Text:          "hello",
		Channel:       excomms.ChannelType_SMS,
	})
	test.OK(t, err)

	mm.Finish()
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
		FromChannelID: "+17348465522",
		ToChannelID:   "+14152222222",
		Text:          "hello",
		Channel:       excomms.ChannelType_Voice,
	})
	test.Equals(t, true, err != nil)
	test.Equals(t, codes.Unimplemented, grpc.Code(err))

	mm.Finish()
}

func TestInitiatePhoneCall(t *testing.T) {
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
		res: &directory.LookupEntitiesResponse{
			Success: true,
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
		contactRes: map[string]*directory.LookupEntitiesByContactResponse{
			"+17348465522": &directory.LookupEntitiesByContactResponse{
				Success: true,
				Entities: []*directory.Entity{
					{
						Type: directory.EntityType_INTERNAL,
						Memberships: []*directory.Entity{
							{
								ID:   "1234",
								Type: directory.EntityType_ORGANIZATION,
							},
						},
					},
				},
			},
			"+14152222222": &directory.LookupEntitiesByContactResponse{
				Success: true,
				Entities: []*directory.Entity{
					{
						Type: directory.EntityType_EXTERNAL,
						Memberships: []*directory.Entity{
							{
								ID:   "1234",
								Type: directory.EntityType_ORGANIZATION,
							},
						},
					},
				},
			},
		},
	}

	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "1234",
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue: "+17348465522",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14152222222",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
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

	res, err := es.InitiatePhoneCall(context.Background(), &excomms.InitiatePhoneCallRequest{
		CallInitiationType: excomms.InitiatePhoneCallRequest_RETURN_PHONE_NUMBER,
		FromPhoneNumber:    "+17348465522",
		ToPhoneNumber:      "+14152222222",
		OrganizationID:     "1234",
	})
	test.OK(t, err)
	test.Equals(t, "+12064881903", res.PhoneNumber)
	md.Finish()
	mdal.Finish()
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
		res: &directory.LookupEntitiesResponse{
			Success: false,
			Failure: &directory.LookupEntitiesResponse_Failure{
				Reason: directory.LookupEntitiesResponse_Failure_NOT_FOUND,
			},
		},
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
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
		res: &directory.LookupEntitiesResponse{
			Success: true,
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
		contactRes: map[string]*directory.LookupEntitiesByContactResponse{
			"+17348465522": &directory.LookupEntitiesByContactResponse{
				Success: false,
				Failure: &directory.LookupEntitiesByContactResponse_Failure{
					Reason: directory.LookupEntitiesByContactResponse_Failure_NOT_FOUND,
				},
			},
		},
	}

	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "1234",
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue: "+17348465522",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
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

func TestInitiatePhoneCall_InvalidCallee(t *testing.T) {
	md := &mockDirectory_Excomms{
		Expector: &mock.Expector{T: t},
		res: &directory.LookupEntitiesResponse{
			Success: true,
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
		contactRes: map[string]*directory.LookupEntitiesByContactResponse{
			"+17348465522": &directory.LookupEntitiesByContactResponse{
				Success: true,
				Entities: []*directory.Entity{
					{
						Type: directory.EntityType_INTERNAL,
						Memberships: []*directory.Entity{
							{
								ID:   "1234",
								Type: directory.EntityType_ORGANIZATION,
							},
						},
					},
				},
			},
			"+14152222222": &directory.LookupEntitiesByContactResponse{
				Success: false,
				Failure: &directory.LookupEntitiesByContactResponse_Failure{
					Reason: directory.LookupEntitiesByContactResponse_Failure_NOT_FOUND,
				},
			},
		},
	}

	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "1234",
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue: "+17348465522",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntitiesByContact, context.Background(), &directory.LookupEntitiesByContactRequest{
		ContactValue: "+14152222222",
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
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
