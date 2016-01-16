package main

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	thmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestNodeQuery(t *testing.T) {
	nodeField := queryType.Fields()["node"]

	dirC := dirmock.New(t)
	thC := thmock.New(t)
	defer dirC.Finish()
	defer thC.Finish()

	acc := &account{ID: "account:12345"}
	ctx := context.Background()
	ctx = ctxWithAccount(ctx, acc)
	p := graphql.ResolveParams{
		Context: ctx,
		Info: graphql.ResolveInfo{
			RootValue: map[string]interface{}{
				"service": &service{
					// auth      auth.AuthClient
					directory: dirC,
					threading: thC,
					// exComms   excomms.ExCommsClient
				},
			},
		},
	}

	// Organization

	id := "entity:123"
	p.Args = map[string]interface{}{
		"id": id,
	}
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: id,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_ORGANIZATION,
					ID:   id,
					Name: "Org",
				},
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: "account:" + acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   "entity:222",
					Name: "Mem",
					Memberships: []*directory.Entity{
						{ID: id},
					},
				},
			},
		},
		nil))
	res, err := nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &organization{
		ID:       id,
		Name:     "Org",
		Contacts: []*contactInfo{},
		Entity: &entity{
			ID:       "entity:222",
			Name:     "Mem",
			Contacts: []*contactInfo{},
		},
	}, res)
	mock.FinishAll(dirC)

	// Entity

	id = "entity:123"
	p.Args = map[string]interface{}{
		"id": id,
	}
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: id,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   id,
					Name: "Someone",
				},
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &entity{ID: id, Name: "Someone", Contacts: []*contactInfo{}}, res)
	mock.FinishAll(dirC)

	// Thread

	id = "t_123"
	p.Args = map[string]interface{}{
		"id": id,
	}
	thC.Expect(mock.NewExpectation(thC.Thread,
		&threading.ThreadRequest{ThreadID: id},
	).WithReturns(
		&threading.ThreadResponse{
			Thread: &threading.Thread{
				ID:              id,
				OrganizationID:  "entity:1",
				PrimaryEntityID: "entity:2",
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "entity:2",
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_EXTERNAL,
					ID:   id,
					Name: "Someone",
				},
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: "account:" + acc.ID,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   "entity:222",
					Name: "Someone",
					Memberships: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							ID:   "entity:1",
						},
					},
				},
			},
		},
		nil))
	thC.Expect(mock.NewExpectation(thC.Thread,
		&threading.ThreadRequest{
			ThreadID:       id,
			ViewerEntityID: "entity:222",
		},
	).WithReturns(
		&threading.ThreadResponse{
			Thread: &threading.Thread{
				ID:              id,
				OrganizationID:  "entity:1",
				PrimaryEntityID: "entity:2",
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "entity:2",
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		},
	).WithReturns(
		&directory.LookupEntitiesResponse{
			Entities: []*directory.Entity{
				{
					Type: directory.EntityType_INTERNAL,
					ID:   "entity:2",
					Name: "Someone",
					Memberships: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							ID:   "entity:1",
						},
					},
				},
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &thread{ID: id, OrganizationID: "entity:1", PrimaryEntityID: "entity:2", Title: "Someone"}, res)
	mock.FinishAll(thC, dirC)

	// Thread item

	id = "ti_123"
	p.Args = map[string]interface{}{
		"id": id,
	}
	thC.Expect(mock.NewExpectation(thC.ThreadItem,
		&threading.ThreadItemRequest{ItemID: id},
	).WithReturns(
		&threading.ThreadItemResponse{
			Item: &threading.ThreadItem{
				ID:            id,
				Timestamp:     1234,
				ActorEntityID: "entity:1",
				Internal:      true,
				Type:          threading.ThreadItem_MESSAGE,
				Item: &threading.ThreadItem_Message{
					Message: &threading.Message{
						Title:  "abc",
						Text:   "hello",
						Status: threading.Message_NORMAL,
						Source: &threading.Endpoint{
							ID:      "555-555-5555",
							Channel: threading.Endpoint_VOICE,
						},
						TextRefs: []*threading.Reference{
							{ID: "e2", Type: threading.Reference_ENTITY},
						},
					},
				},
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &threadItem{
		ID:            id,
		Timestamp:     1234,
		ActorEntityID: "entity:1",
		Internal:      true,
		Data: &message{
			ThreadItemID: id,
			Title:        "abc",
			Text:         "hello",
			Status:       messageStatusNormal,
			Source: &endpoint{
				Channel: endpointChannelVoice,
				ID:      "555-555-5555",
			},
			Refs: []*reference{
				{ID: "e2", Type: "entity"},
			},
		},
	}, res)
	mock.FinishAll(thC)

	// Saved query

	id = "sq_123"
	p.Args = map[string]interface{}{
		"id": id,
	}
	thC.Expect(mock.NewExpectation(thC.SavedQuery,
		&threading.SavedQueryRequest{SavedQueryID: id},
	).WithReturns(
		&threading.SavedQueryResponse{
			SavedQuery: &threading.SavedQuery{
				ID:             id,
				OrganizationID: "entity:1",
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &savedThreadQuery{ID: id, OrganizationID: "entity:1"}, res)
	mock.FinishAll(thC)
}
