package appmsg

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	exsettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	rsettings "github.com/sprucehealth/backend/cmd/svc/routing/internal/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/settings/settingsmock"
	"github.com/sprucehealth/backend/svc/threading"
	"google.golang.org/grpc"
)

type mockDirectoryService struct {
	*mock.Expector
	directory.DirectoryClient
	entityIDToEntityMapping map[string]*directory.Entity
}

func (s *mockDirectoryService) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	s.Record(ctx, in)
	entity := s.entityIDToEntityMapping[in.GetEntityID()]
	var entities []*directory.Entity
	if entity != nil {
		entities = append(entities, entity)
	}

	return &directory.LookupEntitiesResponse{
		Entities: entities,
	}, nil
}

type mockExCommsService struct {
	*mock.Expector
	excomms.ExCommsClient
	messageSendRequest *excomms.SendMessageRequest
}

func (e *mockExCommsService) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	e.Record(ctx, in)
	e.messageSendRequest = in
	return &excomms.SendMessageResponse{}, nil
}

func TestSendMessage_SMS_RevealSender(t *testing.T) {
	testSendMessageSMS(t, true, true)
}

func TestSendMessage_SMS_DontRevealSender(t *testing.T) {
	testSendMessageSMS(t, false, false)
}

func testSendMessageSMS(t *testing.T, revealSender, entityHasDefaultNumber bool) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	ctx := context.Background()

	orgEntity := &directory.Entity{
		Type: directory.EntityType_ORGANIZATION,
		ID:   "10",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+17348465522",
				Provisioned: true,
			},
		},
	}

	externalEntity := &directory.Entity{
		Type: directory.EntityType_EXTERNAL,
		ID:   "20",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12068773590",
			},
		},
	}
	providerEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Smith",
		},
		Type: directory.EntityType_INTERNAL,
		ID:   "30",
	}

	var expectedNumber string
	if entityHasDefaultNumber {
		mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
			NodeID: providerEntity.ID,
			Keys: []*settings.ConfigKey{
				{
					Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
				},
			},
		}).Return(
			&settings.GetValuesResponse{
				Values: []*settings.Value{
					{
						Type: settings.ConfigType_TEXT,
						Value: &settings.Value_Text{
							Text: &settings.TextValue{
								Value: "+14155550001",
							},
						},
					},
				},
			}, nil)
		expectedNumber = "+14155550001"
	} else {
		mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
			NodeID: providerEntity.ID,
			Keys: []*settings.ConfigKey{
				{
					Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
				},
			},
		}).Return(
			&settings.GetValuesResponse{
				Values: nil,
			}, nil)
		expectedNumber = "+17348465522"
	}

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}

	text := "Hello"
	if revealSender {
		text = "Dr. Smith: Hello"
	}
	me.Expect(mock.NewExpectation(me.SendMessage, ctx, &excomms.SendMessageRequest{
		UUID:              "11000",
		DeprecatedChannel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				FromPhoneNumber: expectedNumber,
				ToPhoneNumber:   "+12068773590",
				Text:            text,
				MediaIDs:        []string{"s3://image/attachment/url"},
			},
		},
	}))

	md := &mockDirectoryService{
		Expector: &mock.Expector{
			T: t,
		},
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
			providerEntity.ID: providerEntity,
		},
	}
	md.Expect(mock.NewExpectation(md.LookupEntities, ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	if revealSender {
		md.Expect(mock.NewExpectation(md.LookupEntities, ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: providerEntity.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		}))
	}

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: rsettings.ConfigKeyRevealSenderAcrossExcomms,
			},
		},
	}).Return(
		&settings.GetValuesResponse{
			Values: []*settings.Value{
				{
					Type: settings.ConfigType_BOOLEAN,
					Value: &settings.Value_Boolean{
						Boolean: &settings.BooleanValue{
							Value: revealSender,
						},
					},
				},
			},
		}, nil)

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: providerEntity.ID,
			Internal:      false,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.ENDPOINT_CHANNEL_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "+12068773590",
							Channel: threading.ENDPOINT_CHANNEL_SMS,
						},
					},
					Attachments: []*threading.Attachment{
						{
							URL: "image/attachment/url",
							Data: &threading.Attachment_Image{
								Image: &threading.ImageAttachment{
									MediaID: "s3://image/attachment/url",
								},
							},
						},
						{
							URL:  "generic/url",
							Data: &threading.Attachment_GenericURL{},
						},
					},
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest == nil {
		t.Fatal("Expected message to be sent but it wasnt")
	}

	me.Finish()
	md.Finish()
}

func TestSendMessage_Email_RevealSender(t *testing.T) {
	testSendingEmail(t, true)
}

func TestSendMessage_Email_DontRevealSender(t *testing.T) {
	testSendingEmail(t, false)
}

func testSendingEmail(t *testing.T, revealSender bool) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	ctx := context.Background()

	orgEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Practice Name",
		},
		Type: directory.EntityType_ORGANIZATION,
		ID:   "10",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+17348465522",
				Provisioned: true,
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "doctor@practice.baymax.com",
				Provisioned: true,
			},
		},
	}

	externalEntity := &directory.Entity{
		Type: directory.EntityType_EXTERNAL,
		ID:   "20",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12068773590",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "patient@test.com",
			},
		},
	}

	providerEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Smith",
		},
		Type: directory.EntityType_INTERNAL,
		ID:   "30",
	}

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}

	fromName := "Practice Name"
	if revealSender {
		fromName = "Dr. Smith"
	}
	me.Expect(mock.NewExpectation(me.SendMessage, ctx, &excomms.SendMessageRequest{
		UUID:              "11000",
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Message from Practice Name",
				Body:             "Hello",
				FromName:         fromName,
				FromEmailAddress: "doctor@practice.baymax.com",
				ToEmailAddress:   "patient@test.com",
				MediaIDs:         []string{"s3://image/attachment/url"},
			},
		},
	}))

	md := &mockDirectoryService{
		Expector: &mock.Expector{
			T: t,
		},
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
			providerEntity.ID: providerEntity,
		},
	}
	md.Expect(mock.NewExpectation(md.LookupEntities, ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	if revealSender {
		md.Expect(mock.NewExpectation(md.LookupEntities, ctx, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: providerEntity.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
			},
		}))
	}

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: rsettings.ConfigKeyRevealSenderAcrossExcomms,
			},
		},
	}).Return(
		&settings.GetValuesResponse{
			Values: []*settings.Value{
				{
					Type: settings.ConfigType_BOOLEAN,
					Value: &settings.Value_Boolean{
						Boolean: &settings.BooleanValue{
							Value: revealSender,
						},
					},
				},
			},
		}, nil)

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.ENDPOINT_CHANNEL_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "patient@test.com",
							Channel: threading.ENDPOINT_CHANNEL_EMAIL,
						},
					},
					Attachments: []*threading.Attachment{
						{
							URL: "image/attachment/url",
							Data: &threading.Attachment_Image{
								Image: &threading.ImageAttachment{
									MediaID: "s3://image/attachment/url",
								},
							},
						},
						{
							URL:  "generic/url",
							Data: &threading.Attachment_GenericURL{},
						},
					},
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest == nil {
		t.Fatal("Expected message to be sent but it wasnt")
	}

	md.Finish()
	me.Finish()
}

func TestSendMessage_Multiple(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	ctx := context.Background()

	orgEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Practice Name",
		},
		Type: directory.EntityType_ORGANIZATION,
		ID:   "10",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+17348465522",
				Provisioned: true,
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "doctor@practice.baymax.com",
				Provisioned: true,
			},
		},
	}

	externalEntity := &directory.Entity{
		Type: directory.EntityType_EXTERNAL,
		ID:   "20",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12068773590",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "patient@test.com",
			},
		},
	}

	providerEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Smith",
		},
		Type: directory.EntityType_INTERNAL,
		ID:   "30",
	}

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: providerEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: exsettings.ConfigKeyDefaultProvisionedPhoneNumber,
			},
		},
	}).Return(
		&settings.GetValuesResponse{
			Values: nil,
		}, nil)

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}
	me.Expect(mock.NewExpectation(me.SendMessage, ctx, &excomms.SendMessageRequest{
		UUID:              "11000",
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Message from Practice Name",
				Body:             "Hello",
				FromName:         "Practice Name",
				FromEmailAddress: "doctor@practice.baymax.com",
				ToEmailAddress:   "patient@test.com",
			},
		},
	}))
	me.Expect(mock.NewExpectation(me.SendMessage, ctx, &excomms.SendMessageRequest{
		UUID:              "11000",
		DeprecatedChannel: excomms.ChannelType_SMS,
		Message: &excomms.SendMessageRequest_SMS{
			SMS: &excomms.SMSMessage{
				FromPhoneNumber: "+17348465522",
				ToPhoneNumber:   "+12068773590",
				Text:            "Hello",
			},
		},
	}))

	md := &mockDirectoryService{
		Expector: &mock.Expector{
			T: t,
		},
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
			providerEntity.ID: providerEntity,
		},
	}
	md.Expect(mock.NewExpectation(md.LookupEntities, ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
			},
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}))

	mockSettings.EXPECT().GetValues(ctx, &settings.GetValuesRequest{
		NodeID: orgEntity.ID,
		Keys: []*settings.ConfigKey{
			{
				Key: rsettings.ConfigKeyRevealSenderAcrossExcomms,
			},
		},
	}).Return(
		&settings.GetValuesResponse{
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
		}, nil)

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.ENDPOINT_CHANNEL_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "patient@test.com",
							Channel: threading.ENDPOINT_CHANNEL_EMAIL,
						},
						{
							ID:      "+12068773590",
							Channel: threading.ENDPOINT_CHANNEL_SMS,
						},
					},
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest == nil {
		t.Fatal("Expected message to be sent but it wasnt")
	}

	md.Finish()
	me.Finish()
}

func TestSendMessage_OnlyAppDestinations(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	orgEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Practice Name",
		},
		Type: directory.EntityType_ORGANIZATION,
		ID:   "10",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+17348465522",
				Provisioned: true,
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "doctor@practice.baymax.com",
				Provisioned: true,
			},
		},
	}

	externalEntity := &directory.Entity{
		Type: directory.EntityType_EXTERNAL,
		ID:   "20",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12068773590",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "patient@test.com",
			},
		},
	}

	providerEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Smith",
		},
		Type: directory.EntityType_INTERNAL,
		ID:   "30",
	}

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}

	md := &mockDirectoryService{
		Expector: &mock.Expector{
			T: t,
		},
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
			providerEntity.ID: providerEntity,
		},
	}

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.ENDPOINT_CHANNEL_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "APP",
							Channel: threading.ENDPOINT_CHANNEL_APP,
						},
					},
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest != nil {
		t.Fatal("Expected message to not be sent but it was")
	}

	md.Finish()
	me.Finish()
}

func TestSendMessage_NoDestinations(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	orgEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Practice Name",
		},
		Type: directory.EntityType_ORGANIZATION,
		ID:   "10",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+17348465522",
				Provisioned: true,
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "doctor@practice.baymax.com",
				Provisioned: true,
			},
		},
	}

	externalEntity := &directory.Entity{
		Type: directory.EntityType_EXTERNAL,
		ID:   "20",
		Contacts: []*directory.Contact{
			{
				ContactType: directory.ContactType_PHONE,
				Value:       "+12068773590",
			},
			{
				ContactType: directory.ContactType_EMAIL,
				Value:       "patient@test.com",
			},
		},
	}

	providerEntity := &directory.Entity{
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Smith",
		},
		Type: directory.EntityType_INTERNAL,
		ID:   "30",
	}

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}

	md := &mockDirectoryService{
		Expector: &mock.Expector{
			T: t,
		},
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
			providerEntity.ID: providerEntity,
		},
	}

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.ENDPOINT_CHANNEL_APP,
					},
					Destinations: []*threading.Endpoint{},
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest != nil {
		t.Fatal("Expected message to not be sent but it was")
	}

	md.Finish()
	me.Finish()
}

func TestSendMessage_Internal(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockSettings := settingsmock.NewMockSettingsClient(mockCtrl)

	me := &mockExCommsService{}
	md := &mockDirectoryService{}

	aw := NewWorker(nil, "", md, me, mockSettings)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  "99",
		ThreadID:        "100",
		PrimaryEntityID: "101",
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      true,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
				},
			},
		},
	}

	if err := aw.(*appMessageWorker).process(pti); err != nil {
		t.Fatal(err)
	}

	if me.messageSendRequest != nil {
		t.Fatal("No message should have been sent but one was")
	}

}
