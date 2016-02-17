package main

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dirmock "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	thmock "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestNodeQuery(t *testing.T) {
	nodeField := queryType.Fields()["node"]

	dirC := dirmock.New(t)
	thC := thmock.New(t)
	defer dirC.Finish()
	defer thC.Finish()

	acc := &account{ID: "account_12345"}
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

	id := "entity_123"
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
					Info: &directory.EntityInfo{
						DisplayName: "Org",
					},
				},
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
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
					ID:   "entity_222",
					Info: &directory.EntityInfo{
						DisplayName: "Mem",
					},
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
			ID:          "entity_222",
			IsEditable:  true,
			DisplayName: "Mem",
			Contacts:    []*contactInfo{},
			IsInternal:  true,
		},
	}, res)
	mock.FinishAll(dirC)

	// Entity

	id = "entity_123"
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
					Type: directory.EntityType_EXTERNAL,
					ID:   id,
					Info: &directory.EntityInfo{
						DisplayName: "Someone",
					},
				},
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &entity{ID: id, IsEditable: true, IsInternal: false, DisplayName: "Someone", Contacts: []*contactInfo{}}, res)
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
				OrganizationID:  "entity_1",
				PrimaryEntityID: "entity_2",
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "entity_2",
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
					Info: &directory.EntityInfo{
						DisplayName: "Someone",
					},
				},
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
				ExternalID: acc.ID,
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
					ID:   "entity_222",
					Info: &directory.EntityInfo{
						DisplayName: "Someone",
					},
					Memberships: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							ID:   "entity_1",
						},
					},
				},
			},
		},
		nil))
	thC.Expect(mock.NewExpectation(thC.Thread,
		&threading.ThreadRequest{
			ThreadID:       id,
			ViewerEntityID: "entity_222",
		},
	).WithReturns(
		&threading.ThreadResponse{
			Thread: &threading.Thread{
				ID:              id,
				OrganizationID:  "entity_1",
				PrimaryEntityID: "entity_2",
			},
		},
		nil))
	dirC.Expect(mock.NewExpectation(dirC.LookupEntities,
		&directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: "entity_2",
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
					ID:   "entity_2",
					Info: &directory.EntityInfo{
						DisplayName: "Someone",
					},
					Memberships: []*directory.Entity{
						{
							Type: directory.EntityType_ORGANIZATION,
							ID:   "entity_1",
						},
					},
				},
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	res.(*thread).primaryEntity = nil
	test.Equals(t, &thread{
		ID: id,
		AllowInternalMessages: true,
		IsDeletable:           true,
		OrganizationID:        "entity_1",
		PrimaryEntityID:       "entity_2",
		Title:                 "Someone",
		LastPrimaryEntityEndpoints: []*endpoint{},
	}, res)
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
				ActorEntityID: "entity_1",
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
		ActorEntityID: "entity_1",
		Internal:      true,
		Data: &message{
			ThreadItemID:  id,
			SummaryMarkup: "abc",
			TextMarkup:    "hello",
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
				OrganizationID: "entity_1",
			},
		},
		nil))
	res, err = nodeField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &savedThreadQuery{ID: id, OrganizationID: "entity_1"}, res)
	mock.FinishAll(thC)
}

func TestSubdomainQuery_Unavailable(t *testing.T) {
	subdomainField := queryType.Fields()["subdomain"]

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

	p.Args = map[string]interface{}{
		"value": "mypractice",
	}
	dirC.Expect(mock.NewExpectation(dirC.LookupEntityDomain,
		&directory.LookupEntityDomainRequest{
			Domain: "mypractice",
		},
	).WithReturns(
		&directory.LookupEntityDomainResponse{
			EntityID: "dkgj",
			Domain:   "mypractice",
		},
		nil),
	)

	res, err := subdomainField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &subdomain{
		Available: false,
	}, res)
	mock.FinishAll(dirC)
}

func TestSubdomainQuery_Available(t *testing.T) {
	subdomainField := queryType.Fields()["subdomain"]

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

	// Available
	p.Args = map[string]interface{}{
		"value": "anotherpractice",
	}
	dirC.Expect(mock.NewExpectation(dirC.LookupEntityDomain,
		&directory.LookupEntityDomainRequest{
			Domain: "anotherpractice",
		},
	).WithReturns(
		&directory.LookupEntityDomainResponse{},
		grpc.Errorf(codes.NotFound, "entity_domain not found")),
	)
	res, err := subdomainField.Resolve(p)
	test.OK(t, err)
	test.Equals(t, &subdomain{
		Available: true,
	}, res)
	mock.FinishAll(dirC)
}
