package cache

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	dmock "github.com/sprucehealth/backend/svc/directory/mock"
)

type tCachedClient struct {
	cachedClient *CachedClient
	finishers    []mock.Finisher
}

func TestCachedClientLookupEntities(t *testing.T) {
	cases := map[string]struct {
		tCachedClient *tCachedClient
		ctx           context.Context
		requests      []*directory.LookupEntitiesRequest
		expected      []*directory.LookupEntitiesResponse
	}{
		"UnintializedContextCache": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{}).WithReturns(&directory.LookupEntitiesResponse{}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx:      context.Background(),
			requests: []*directory.LookupEntitiesRequest{&directory.LookupEntitiesRequest{}},
			expected: []*directory.LookupEntitiesResponse{&directory.LookupEntitiesResponse{}},
		},
		"SingleCall": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{}).WithReturns(&directory.LookupEntitiesResponse{}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx:      InitEntityCache(context.Background()),
			requests: []*directory.LookupEntitiesRequest{&directory.LookupEntitiesRequest{}},
			expected: []*directory.LookupEntitiesResponse{&directory.LookupEntitiesResponse{}},
		},
		"SimpleCall-CacheHit": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity1",
					},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx: InitEntityCache(context.Background()),
			requests: []*directory.LookupEntitiesRequest{
				&directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity1",
					},
				},
				&directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity1",
					},
				}},
			expected: []*directory.LookupEntitiesResponse{
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}},
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}},
			},
		},
		"SimpleCall-CacheMiss": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity1",
					},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}}, nil))
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity2",
					},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity2"}}}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx: InitEntityCache(context.Background()),
			requests: []*directory.LookupEntitiesRequest{
				&directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity1",
					},
				},
				&directory.LookupEntitiesRequest{
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: "entity2",
					},
				}},
			expected: []*directory.LookupEntitiesResponse{
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}},
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity2"}}},
			},
		},
		"SortedData-CacheHit": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity1", "entity2"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE, directory.EntityStatus_DELETED},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}, &directory.Entity{ID: "entity2"}}}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx: InitEntityCache(context.Background()),
			requests: []*directory.LookupEntitiesRequest{
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity1", "entity2"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_MEMBERS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE, directory.EntityStatus_DELETED},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
				},
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity2", "entity1"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_DELETED, directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_INTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_INTERNAL},
				}},
			expected: []*directory.LookupEntitiesResponse{
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}, &directory.Entity{ID: "entity2"}}},
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}, &directory.Entity{ID: "entity2"}}},
			},
		},
		"FilterData-CacheMiss": {
			tCachedClient: func() *tCachedClient {
				dClient := dmock.New(t)
				cClient := &CachedClient{dc: dClient}
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity1", "entity2"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}}, nil))
				dClient.Expect(mock.NewExpectation(dClient.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity1", "entity2"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE, directory.EntityStatus_DELETED},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_EXTERNAL},
				}).WithReturns(&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}, &directory.Entity{ID: "entity2"}}}, nil))
				return &tCachedClient{
					cachedClient: cClient,
					finishers:    []mock.Finisher{dClient},
				}
			}(),
			ctx: InitEntityCache(context.Background()),
			requests: []*directory.LookupEntitiesRequest{
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity1", "entity2"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_INTERNAL},
				},
				&directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_BATCH_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_BatchEntityID{
						BatchEntityID: &directory.IDList{IDs: []string{"entity2", "entity1"}},
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERS, directory.EntityInformation_CONTACTS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_DELETED, directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_INTERNAL},
					ChildTypes: []directory.EntityType{directory.EntityType_EXTERNAL, directory.EntityType_INTERNAL},
				}},
			expected: []*directory.LookupEntitiesResponse{
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}}},
				&directory.LookupEntitiesResponse{Entities: []*directory.Entity{&directory.Entity{ID: "entity1"}, &directory.Entity{ID: "entity2"}}},
			},
		},
	}

	for cn, c := range cases {
		responses := make([]*directory.LookupEntitiesResponse, len(c.requests))
		for i, r := range c.requests {
			resp, err := c.tCachedClient.cachedClient.LookupEntities(c.ctx, r)
			test.OKCase(t, cn, err)
			responses[i] = resp
		}
		test.EqualsCase(t, cn, c.expected, responses)
		mock.FinishAll(c.tCachedClient.finishers...)
	}
}

func TestBustCache(t *testing.T) {
	ctx := InitEntityCache(context.Background())
	cc := &CachedClient{}
	cc.cache(ctx, "Hello", []*directory.Entity{&directory.Entity{ID: "entity1"}})
	cc.bustCache(ctx)
	test.AssertNil(t, cc.checkCache(ctx, "Hello"))
}
