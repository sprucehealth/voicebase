package appmsg

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
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

func TestSendMessage_SMS(t *testing.T) {

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

	me := &mockExCommsService{
		Expector: &mock.Expector{
			T: t,
		},
	}
	me.Expect(mock.NewExpectation(me.SendMessage, context.Background(), &excomms.SendMessageRequest{
		UUID:    "11000",
		Channel: excomms.ChannelType_SMS,
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
		},
	}
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: externalEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "+12068773590",
							Channel: threading.Endpoint_SMS,
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

func TestSendMessage_Email(t *testing.T) {

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
	me.Expect(mock.NewExpectation(me.SendMessage, context.Background(), &excomms.SendMessageRequest{
		UUID:    "11000",
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Message from Dr. Smith, Practice Name",
				Body:             "Hello",
				FromName:         "Dr. Smith",
				FromEmailAddress: "doctor@practice.baymax.com",
				ToEmailAddress:   "patient@test.com",
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
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: externalEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: providerEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}))

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "patient@test.com",
							Channel: threading.Endpoint_EMAIL,
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
	me.Expect(mock.NewExpectation(me.SendMessage, context.Background(), &excomms.SendMessageRequest{
		UUID:    "11000",
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Message from Dr. Smith, Practice Name",
				Body:             "Hello",
				FromName:         "Dr. Smith",
				FromEmailAddress: "doctor@practice.baymax.com",
				ToEmailAddress:   "patient@test.com",
			},
		},
	}))
	me.Expect(mock.NewExpectation(me.SendMessage, context.Background(), &excomms.SendMessageRequest{
		UUID:    "11000",
		Channel: excomms.ChannelType_SMS,
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
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: externalEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 1,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}))
	md.Expect(mock.NewExpectation(md.LookupEntities, context.Background(), &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: providerEntity.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	}))

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "patient@test.com",
							Channel: threading.Endpoint_EMAIL,
						},
						{
							ID:      "+12068773590",
							Channel: threading.Endpoint_SMS,
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

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
					},
					Destinations: []*threading.Endpoint{
						{
							ID:      "APP",
							Channel: threading.Endpoint_APP,
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

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  orgEntity.ID,
		ThreadID:        "100",
		PrimaryEntityID: externalEntity.ID,
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      false,
			Type:          threading.ThreadItem_MESSAGE,
			Item: &threading.ThreadItem_Message{
				Message: &threading.Message{
					Text: "Hello",
					Source: &threading.Endpoint{
						Channel: threading.Endpoint_APP,
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
	me := &mockExCommsService{}
	md := &mockDirectoryService{}

	aw := NewWorker(nil, "", md, me)

	pti := &threading.PublishedThreadItem{
		OrganizationID:  "99",
		ThreadID:        "100",
		PrimaryEntityID: "101",
		Item: &threading.ThreadItem{
			ID:            "11000",
			ActorEntityID: "30",
			Internal:      true,
			Type:          threading.ThreadItem_MESSAGE,
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
