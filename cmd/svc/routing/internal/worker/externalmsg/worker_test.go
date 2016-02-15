package externalmsg

import (
	"testing"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type mockDirectoryService struct {
	directory.DirectoryClient
	entityIDToEntityMapping  map[string]*directory.Entity
	contactToEntitiesMapping map[string][]*directory.Entity
	entityToCreate           *directory.Entity
}

func (s *mockDirectoryService) LookupEntities(ctx context.Context, in *directory.LookupEntitiesRequest, opts ...grpc.CallOption) (*directory.LookupEntitiesResponse, error) {
	entity := s.entityIDToEntityMapping[in.GetEntityID()]
	var entities []*directory.Entity
	if entity != nil {
		entities = append(entities, entity)
	} else {
		return nil, grpc.Errorf(codes.NotFound, "")
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
	entities := s.contactToEntitiesMapping[in.ContactValue]
	if len(entities) == 0 {
		return nil, grpc.Errorf(codes.NotFound, "")
	}

	return &directory.LookupEntitiesByContactResponse{
		Entities: entities,
	}, nil
}

type mockThreadsService struct {
	threading.ThreadsClient

	threadsForMembers []*threading.Thread

	threadCreationRequested *threading.CreateThreadRequest
	postMessageRequests     []*threading.PostMessageRequest
}

func (t *mockThreadsService) CreateThread(ctx context.Context, in *threading.CreateThreadRequest, opts ...grpc.CallOption) (*threading.CreateThreadResponse, error) {
	t.threadCreationRequested = in
	return &threading.CreateThreadResponse{}, nil
}
func (t *mockThreadsService) PostMessage(ctx context.Context, in *threading.PostMessageRequest, opts ...grpc.CallOption) (*threading.PostMessageResponse, error) {
	t.postMessageRequests = append(t.postMessageRequests, in)
	return &threading.PostMessageResponse{}, nil
}
func (t *mockThreadsService) ThreadsForMember(ctx context.Context, in *threading.ThreadsForMemberRequest, opts ...grpc.CallOption) (*threading.ThreadsForMemberResponse, error) {
	return &threading.ThreadsForMemberResponse{
		Threads: t.threadsForMembers,
	}, nil
}

func TestIncomingSMS_NewUser_SMS(t *testing.T) {

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
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID: []*directory.Entity{organizationEntity},
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
	test.Equals(t, threadRequested.Title, "SMS")
	test.Equals(t, threadRequested.Text, pem.GetSMSItem().Text)
	test.Equals(t, len(threadRequested.Attachments), len(pem.GetSMSItem().GetAttachments()))

	// ensure no call to post message to thread
	if len(mt.postMessageRequests) != 0 {
		t.Fatal("Expected no posting of message to thread given thread was just created")
	}
}

func TestIncomingSMS_NewUser_Email(t *testing.T) {

	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "doctor@mypractice.baymax.com",
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
				Value:       "patient@example.com",
				ContactType: directory.ContactType_EMAIL,
			},
		},
	}

	fromChannelID := "patient@example.com"
	toChannelID := "doctor@mypractice.baymax.com"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
		},
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID: []*directory.Entity{organizationEntity},
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
		Type:          excomms.PublishedExternalMessage_EMAIL,
		Item: &excomms.PublishedExternalMessage_EmailItem{
			EmailItem: &excomms.EmailItem{
				Subject: "Hello",
				Body:    "body",
				Attachments: []*excomms.MediaAttachment{
					{
						URL:         "s3://test/1234",
						ContentType: "image/jpeg",
						Name:        "Testing",
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
	test.Equals(t, threadRequested.Title, "Email")
	test.Equals(t, threadRequested.Text, "Subject: Hello\n\n"+pem.GetEmailItem().Body)
	test.Equals(t, pem.GetEmailItem().Attachments[0].URL, threadRequested.Attachments[0].GetImage().URL)
	test.Equals(t, pem.GetEmailItem().Attachments[0].Name, threadRequested.Attachments[0].Title)
	test.Equals(t, pem.GetEmailItem().Attachments[0].ContentType, threadRequested.Attachments[0].GetImage().Mimetype)

	// ensure no call to post message to thread
	if len(mt.postMessageRequests) != 0 {
		t.Fatal("Expected no posting of message to thread given thread was just created")
	}
}

func TestIncomingSMS_ExistingUser_SMS(t *testing.T) {

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
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID:   []*directory.Entity{organizationEntity},
			fromChannelID: []*directory.Entity{externalEntity},
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
	if len(mt.postMessageRequests) != 1 {
		t.Fatal("Expected message to be posted to existing thread")
	}
	test.Equals(t, mt.postMessageRequests[0].FromEntityID, externalEntity.ID)
	test.Equals(t, mt.postMessageRequests[0].Title, "SMS")
	test.Equals(t, mt.postMessageRequests[0].Text, pem.GetSMSItem().Text)
	test.Equals(t, len(mt.postMessageRequests[0].Attachments), len(pem.GetSMSItem().GetAttachments()))
}

func TestIncomingSMS_ExistingUser_Email(t *testing.T) {

	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "doctor@mypractice.baymax.com",
				ContactType: directory.ContactType_EMAIL,
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
				Value:       "patient@example.com",
				ContactType: directory.ContactType_EMAIL,
			},
		},
	}

	fromChannelID := "patient@example.com"
	toChannelID := "doctor@mypractice.baymax.com"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
			externalEntity.ID:     externalEntity,
		},
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID:   []*directory.Entity{organizationEntity},
			fromChannelID: []*directory.Entity{externalEntity},
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
		Type:          excomms.PublishedExternalMessage_EMAIL,
		Item: &excomms.PublishedExternalMessage_EmailItem{
			EmailItem: &excomms.EmailItem{
				Subject: "Hello",
				Body:    "Body",
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
	if len(mt.postMessageRequests) != 1 {
		t.Fatal("Expected message to be posted to existing thread")
	}
	test.Equals(t, mt.postMessageRequests[0].FromEntityID, externalEntity.ID)
	test.Equals(t, mt.postMessageRequests[0].Title, "Email")
	test.Equals(t, mt.postMessageRequests[0].Text, "Subject: Hello\n\n"+pem.GetEmailItem().Body)
}

func TestIncomingSMS_MultipleExistingUsers_Email(t *testing.T) {

	// Setup
	organizationEntity := &directory.Entity{
		ID:   "10",
		Type: directory.EntityType_ORGANIZATION,
		Contacts: []*directory.Contact{
			{
				Provisioned: true,
				Value:       "doctor@mypractice.baymax.com",
				ContactType: directory.ContactType_EMAIL,
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
				Value:       "patient@example.com",
				ContactType: directory.ContactType_EMAIL,
			},
		},
	}
	externalEntity2 := &directory.Entity{
		ID:   "3",
		Type: directory.EntityType_EXTERNAL,
		Memberships: []*directory.Entity{
			organizationEntity,
		},
		Contacts: []*directory.Contact{
			{
				Value:       "patient@example.com",
				ContactType: directory.ContactType_EMAIL,
			},
		},
	}

	fromChannelID := "patient@example.com"
	toChannelID := "doctor@mypractice.baymax.com"

	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			organizationEntity.ID: organizationEntity,
			providerEntity.ID:     providerEntity,
			externalEntity.ID:     externalEntity,
		},
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID:   []*directory.Entity{organizationEntity},
			fromChannelID: []*directory.Entity{externalEntity, externalEntity2},
		},
	}
	mt := &mockThreadsService{
		threadsForMembers: []*threading.Thread{
			{
				ID:              "1000",
				OrganizationID:  "10",
				PrimaryEntityID: externalEntity.ID,
			},
			{
				ID:              "10000",
				OrganizationID:  "10",
				PrimaryEntityID: externalEntity2.ID,
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
		Type:          excomms.PublishedExternalMessage_EMAIL,
		Item: &excomms.PublishedExternalMessage_EmailItem{
			EmailItem: &excomms.EmailItem{
				Subject: "Hello",
				Body:    "Body",
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
	if len(mt.postMessageRequests) != 2 {
		t.Fatal("Expected 2 messages to be posted to existing thread")
	}

	test.Equals(t, "1000", mt.postMessageRequests[0].ThreadID)
	test.Equals(t, "10000", mt.postMessageRequests[1].ThreadID)
	test.Equals(t, externalEntity.ID, mt.postMessageRequests[0].FromEntityID)
	test.Equals(t, externalEntity2.ID, mt.postMessageRequests[1].FromEntityID)
	test.Equals(t, "Email", mt.postMessageRequests[0].Title)
	test.Equals(t, "Subject: Hello\n\n"+pem.GetEmailItem().Body, mt.postMessageRequests[0].Text)
	test.Equals(t, "Email", mt.postMessageRequests[1].Title)
	test.Equals(t, "Subject: Hello\n\n"+pem.GetEmailItem().Body, mt.postMessageRequests[1].Text)
}

func TestIncomingVoicemail_NewUser(t *testing.T) {
	// Setup
	organizationEntity := &directory.Entity{
		ID: "10",
		Info: &directory.EntityInfo{
			DisplayName: "Spruce Practice",
		},
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
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID: []*directory.Entity{organizationEntity},
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
		Type:          excomms.PublishedExternalMessage_INCOMING_CALL_EVENT,
		Item: &excomms.PublishedExternalMessage_Incoming{
			Incoming: &excomms.IncomingCallEventItem{
				Type:              excomms.IncomingCallEventItem_LEFT_VOICEMAIL,
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
	test.Equals(t, "", threadRequested.Text)
	test.Equals(t, "Voicemail", threadRequested.Title)
	test.Equals(t, pem.GetIncoming().DurationInSeconds, threadRequested.GetAttachments()[0].GetAudio().DurationInSeconds)
	test.Equals(t, pem.GetIncoming().URL, threadRequested.GetAttachments()[0].GetAudio().URL)

	// ensure no call to post message to thread
	if len(mt.postMessageRequests) != 0 {
		t.Fatal("Expected no posting of message to thread given thread was just created")
	}
}

func TestOutgoingCallEvent(t *testing.T) {
	// Setup
	organizationEntity := &directory.Entity{
		ID: "10",
		Info: &directory.EntityInfo{
			DisplayName: "Spruce Practice",
		},
		Type: directory.EntityType_ORGANIZATION,
	}
	providerEntity := &directory.Entity{
		ID: "1",
		Info: &directory.EntityInfo{
			DisplayName: "Dr. Craig",
		},
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
		contactToEntitiesMapping: map[string][]*directory.Entity{
			toChannelID:   []*directory.Entity{externalEntity},
			fromChannelID: []*directory.Entity{providerEntity},
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
		Type:          excomms.PublishedExternalMessage_OUTGOING_CALL_EVENT,
		Direction:     excomms.PublishedExternalMessage_OUTBOUND,
		Item: &excomms.PublishedExternalMessage_Outgoing{
			Outgoing: &excomms.OutgoingCallEventItem{
				Type:           excomms.OutgoingCallEventItem_PLACED,
				CallerEntityID: providerEntity.ID,
				CalleeEntityID: externalEntity.ID,
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
	if len(mt.postMessageRequests) != 1 {
		t.Fatal("Expected message to be posted to existing thread")
	}
	test.Equals(t, providerEntity.ID, mt.postMessageRequests[0].FromEntityID)
	test.Equals(t, "", mt.postMessageRequests[0].Text)
	test.Equals(t, "Outbound call", mt.postMessageRequests[0].Title)

}
