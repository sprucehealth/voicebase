package externalmsg

import (
	"testing"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type mockDirectoryService struct {
	directory.DirectoryClient
	entityIDToEntityMapping map[string]*directory.Entity
	contactToEntityMapping  map[string]*directory.Entity
	entityToCreate          *directory.Entity
}

func (s *mockDirectoryService) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	entity := s.entityIDToEntityMapping[in.GetEntityID()]
	var entities []*directory.Entity
	if entity != nil {
		entities = append(entities, entity)
	}

	return &directory.LookupEntitiesResponse{
		Entities: entities,
	}, nil
}
func (s *mockDirectoryService) CreateEntity(ctx context.Context, in *directory.CreateEntityRequest, opts ...grpc.CallOption) (*directory.CreateEntityResponse, error) {
	return &directory.CreateEntityResponse{
		Entity: s.entityToCreate,
	}, nil
}
func (s *mockDirectoryService) LookupEntitiesByContact(ctx context.Context, in *directory.LookupEntitiesByContactRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesByContactResponse, error) {
	entity := s.contactToEntityMapping[in.ContactValue]
	var entities []*directory.Entity
	if entity != nil {
		entities = append(entities, entity)
	}
	return &directory.LookupEntitiesByContactResponse{
		Entities: entities,
	}, nil
}

type mockThreadsService struct {
	threading.ThreadsClient

	threadsForMembers []*threading.Thread

	threadCreationRequested *threading.CreateThreadRequest
	postMessageRequested    *threading.PostMessageRequest
}

func (t *mockThreadsService) CreateThread(ctx context.Context, in *threading.CreateThreadRequest, opts ...grpc.CallOption) (*threading.CreateThreadResponse, error) {
	t.threadCreationRequested = in
	return &threading.CreateThreadResponse{}, nil
}
func (t *mockThreadsService) PostMessage(ctx context.Context, in *threading.PostMessageRequest, opts ...grpc.CallOption) (*threading.PostMessageResponse, error) {
	t.postMessageRequested = in
	return &threading.PostMessageResponse{}, nil
}
func (t *mockThreadsService) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest, opts ...grpc.CallOption) (*threading.ThreadsForMemberResponse, error) {
	return &threading.ThreadsForMemberResponse{
		Threads: t.threadsForMembers,
	}, nil
}

func TestIncomingSMS_NewUser(t *testing.T) {

	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "+17348465522",
			},
		},
	}
	providerEntity := &directory.Entity{
		ID:   "1",
		Type: directory.EntityType_INTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
	}
	externalEntityToBeCreated := &directory.Entity{
		ID:   "2",
		Type: directory.EntityType_EXTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Value: "+12068773590",
			},
		},
	}

	fromChannelID := "+12068773590"
	toChannelID := "+17348465522"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
		},
		contactToEntityMapping: map[string]*directory.Entity{
			toChannelID: organizationEntity,
		},
		entityToCreate: externalEntityToBeCreated,
	}
	mt := &mockThreadsService{}

	e := &externalMessageWorker{
		directory: md,
		threading: mt,
	}

	pem := &excomms.PublishedExternalMessage{
		FromChannelID: fromChannelID,
		ToChannelID:   toChannelID,
		Type:          excomms.PublishedExternalMessage_SMS,
		Item: &excomms.PublishedExternalMessage_SMSItem{
			SMSItem: &excomms.SMSItem{
				Text: "Hello",
				Attachments: []*excomms.MediaAttachment{
					{
						URL:         "http://google.com",
						ContentType: "image/jpeg",
					},
				},
			},
		},
	}

	if err := e.process(pem); err != nil {
		t.Fatal(err)
	}

	// at this point there should be a new thread created
	threadRequested := mt.threadCreationRequested
	if threadRequested == nil {
		t.Fatalf("Expected new thread to be created")
	}
	test.Equals(t, threadRequested.FromEntityID, externalEntityToBeCreated.ID)
	test.Equals(t, threadRequested.OrganizationID, organizationEntity.ID)
	test.Equals(t, threadRequested.Text, pem.GetSMSItem().Text)
	test.Equals(t, len(threadRequested.Attachments), len(pem.GetSMSItem().GetAttachments()))

	// ensure no call to post message to thread
	if mt.postMessageRequested != nil {
		t.Fatal("Expected no posting of message to thread given thread was just created")
	}
}

func TestIncomingSMS_ExistingUser(t *testing.T) {

	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "+17348465522",
			},
		},
	}
	providerEntity := &directory.Entity{
		ID:   "1",
		Type: directory.EntityType_INTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
	}
	externalEntity := &directory.Entity{
		ID:   "2",
		Type: directory.EntityType_EXTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Value: "+12068773590",
			},
		},
	}

	fromChannelID := "+12068773590"
	toChannelID := "+17348465522"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
			externalEntity.ID:     externalEntity,
		},
		contactToEntityMapping: map[string]*directory.Entity{
			toChannelID:   organizationEntity,
			fromChannelID: externalEntity,
		},
	}
	mt := &mockThreadsService{
		threadsForMembers: []*threading.Thread{
			{
				ID:              "1000",
				OrganizationID:  "10",
				PrimaryEntityID: externalEntity.ID,
			},
		},
	}

	e := &externalMessageWorker{
		directory: md,
		threading: mt,
	}

	pem := &excomms.PublishedExternalMessage{
		FromChannelID: fromChannelID,
		ToChannelID:   toChannelID,
		Type:          excomms.PublishedExternalMessage_SMS,
		Item: &excomms.PublishedExternalMessage_SMSItem{
			SMSItem: &excomms.SMSItem{
				Text: "Hello",
				Attachments: []*excomms.MediaAttachment{
					{
						URL:         "http://google.com",
						ContentType: "image/jpeg",
					},
				},
			},
		},
	}

	if err := e.process(pem); err != nil {
		t.Fatal(err)
	}

	// at this point there should be a new thread created
	threadRequested := mt.threadCreationRequested
	if threadRequested != nil {
		t.Fatalf("Expected no new thread to be created")
	}
	// ensure no call to post message to thread
	if mt.postMessageRequested == nil {
		t.Fatal("Expected message to be posted to existing thread")
	}
	test.Equals(t, mt.postMessageRequested.FromEntityID, externalEntity.ID)
	test.Equals(t, mt.postMessageRequested.Text, pem.GetSMSItem().Text)
	test.Equals(t, len(mt.postMessageRequested.Attachments), len(pem.GetSMSItem().GetAttachments()))
}

func TestIncomingVoicemail_NewUser(t *testing.T) {
	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Name: "Spruce Practice",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "+17348465522",
			},
		},
	}
	providerEntity := &directory.Entity{
		ID:   "1",
		Type: directory.EntityType_INTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
	}
	externalEntityToBeCreated := &directory.Entity{
		ID:   "2",
		Type: directory.EntityType_EXTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Value: "+12068773590",
			},
		},
	}

	fromChannelID := "+12068773590"
	toChannelID := "+17348465522"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
		},
		contactToEntityMapping: map[string]*directory.Entity{
			toChannelID: organizationEntity,
		},
		entityToCreate: externalEntityToBeCreated,
	}
	mt := &mockThreadsService{}

	e := &externalMessageWorker{
		directory: md,
		threading: mt,
	}

	pem := &excomms.PublishedExternalMessage{
		FromChannelID: fromChannelID,
		ToChannelID:   toChannelID,
		Type:          excomms.PublishedExternalMessage_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type:              excomms.CallEventItem_INCOMING_LEFT_VOICEMAIL,
				DurationInSeconds: 100,
				URL:               "http://voicemail.com",
			},
		},
	}

	if err := e.process(pem); err != nil {
		t.Fatal(err)
	}

	// at this point there should be a new thread created
	threadRequested := mt.threadCreationRequested
	if threadRequested == nil {
		t.Fatalf("Expected new thread to be created")
	}
	test.Equals(t, externalEntityToBeCreated.ID, threadRequested.FromEntityID)
	test.Equals(t, organizationEntity.ID, threadRequested.OrganizationID)
	test.Equals(t, "+12068773590 called Spruce Practice, left voicemail.", threadRequested.Text)
	test.Equals(t, pem.GetCallEventItem().DurationInSeconds, threadRequested.GetAttachments()[0].GetAudio().DurationInSeconds)
	test.Equals(t, pem.GetCallEventItem().URL, threadRequested.GetAttachments()[0].GetAudio().URL)

	// ensure no call to post message to thread
	if mt.postMessageRequested != nil {
		t.Fatal("Expected no posting of message to thread given thread was just created")
	}
}

func TestOutgoingCallEvent(t *testing.T) {
	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Name: "Spruce Practice",
		Type: directory.EntityType_ORGANIZATION,
	}
	providerEntity := &directory.Entity{
		ID:   "1",
		Name: "Dr. Craig",
		Type: directory.EntityType_INTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "+12068773590",
			},
		},
	}
	externalEntity := &directory.Entity{
		ID:   "2",
		Type: directory.EntityType_EXTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Value: "17348465522",
			},
		},
	}

	fromChannelID := "+12068773590"
	toChannelID := "+17348465522"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
			externalEntity.ID:     externalEntity,
		},
		contactToEntityMapping: map[string]*directory.Entity{
			toChannelID:   externalEntity,
			fromChannelID: providerEntity,
		},
	}
	mt := &mockThreadsService{
		threadsForMembers: []*threading.Thread{
			{
				ID:              "1000",
				OrganizationID:  "10",
				PrimaryEntityID: externalEntity.ID,
			},
		},
	}
	e := &externalMessageWorker{
		directory: md,
		threading: mt,
	}

	pem := &excomms.PublishedExternalMessage{
		FromChannelID: fromChannelID,
		ToChannelID:   toChannelID,
		Type:          excomms.PublishedExternalMessage_CALL_EVENT,
		Direction:     excomms.PublishedExternalMessage_OUTBOUND,
		Item: &excomms.PublishedExternalMessage_CallEventItem{
			CallEventItem: &excomms.CallEventItem{
				Type: excomms.CallEventItem_OUTGOING_PLACED,
			},
		},
	}

	if err := e.process(pem); err != nil {
		t.Fatal(err)
	}

	// at this point there should be a new thread created
	threadRequested := mt.threadCreationRequested
	if threadRequested != nil {
		t.Fatalf("No new thread should be created")
	}

	// ensure no call to post message to thread
	if mt.postMessageRequested == nil {
		t.Fatal("Expected message to be posted to existing thread")
	}
	test.Equals(t, providerEntity.ID, mt.postMessageRequested.FromEntityID)
	test.Equals(t, "Dr. Craig called +17348465522.", mt.postMessageRequested.Text)

}
