package appmsg

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

type mockExCommsService struct {
	excomms.ExCommsClient
	messageSendRequest *excomms.SendMessageRequest
}

func (e *mockExCommsService) SendMessage(ctx context.Context, in *excomms.SendMessageRequest, opts ...grpc.CallOption) (*excomms.SendMessageResponse, error) {
	e.messageSendRequest = in
	return &excomms.SendMessageResponse{}, nil
}

func TestSendMessage(t *testing.T) {

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

	me := &mockExCommsService{}
	md := &mockDirectoryService{
		entityIDToEntityMapping: map[string]*directory.Entity{
			orgEntity.ID:      orgEntity,
			externalEntity.ID: externalEntity,
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
	test.Equals(t, pti.GetItem().GetMessage().Text, me.messageSendRequest.Text)
	test.Equals(t, orgEntity.Contacts[0].Value, me.messageSendRequest.FromChannelID)
	test.Equals(t, externalEntity.Contacts[0].Value, me.messageSendRequest.ToChannelID)
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
