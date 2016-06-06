package service

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	mock_dl "github.com/sprucehealth/backend/cmd/svc/media/internal/dal/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mock_directory "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	mock_threads "github.com/sprucehealth/backend/svc/threading/mock"
	"github.com/sprucehealth/backend/test"
)

type tservice struct {
	service *service
	finish  []mock.Finisher
}

func TestCanAccess(t *testing.T) {
	cases := map[string]struct {
		tservice  *tservice
		mediaID   string
		accountID string
		expected  error
		finishers []mock.Finisher
	}{
		"OwnerAccountID-AccountIDMatches": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeAccount,
						OwnerID:   "accountID",
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerAccountID-AccountIDMismatch": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeAccount,
						OwnerID:   "differentAccountID",
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerEntity-SameEntity": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeEntity,
						OwnerID:   "entityID",
					}, nil))

				// entitiesForAccountID
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{ID: "entityID"},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerEntity-DifferentEntity": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeEntity,
						OwnerID:   "entityID",
					}, nil))

				// entitiesForAccountID
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{ID: "differentEntityID"},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerOrg-OrgMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeOrganization,
						OwnerID:   "orgID",
					}, nil))

				// ent memberships
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
					ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{Memberships: []*directory.Entity{{ID: "orgID"}}},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerOrg-NotOrgMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeOrganization,
						OwnerID:   "orgID",
					}, nil))

				// ent memberships
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
					ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{Memberships: []*directory.Entity{{ID: "differentOrgID"}}},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerThread-OrgMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				mt.Expect(mock.NewExpectation(mt.Thread, &threading.ThreadRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadResponse{
					Thread: &threading.Thread{
						Type:           threading.ThreadType_EXTERNAL,
						OrganizationID: "orgID",
					},
				}, nil))

				// ent memberships
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
					ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{Memberships: []*directory.Entity{{ID: "orgID"}}},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerThread-NotOrgMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				mt.Expect(mock.NewExpectation(mt.Thread, &threading.ThreadRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadResponse{
					Thread: &threading.Thread{
						Type:           threading.ThreadType_EXTERNAL,
						OrganizationID: "orgID",
					},
				}, nil))

				// ent memberships
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					RequestedInformation: &directory.RequestedInformation{
						EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS},
					},
					Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
					ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{Memberships: []*directory.Entity{{ID: "differentOrgID"}}},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerThread-TeamThreadThreadMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				mt.Expect(mock.NewExpectation(mt.Thread, &threading.ThreadRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadResponse{
					Thread: &threading.Thread{
						Type: threading.ThreadType_TEAM,
					},
				}, nil))

				// thread members
				mt.Expect(mock.NewExpectation(mt.ThreadMembers, &threading.ThreadMembersRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadMembersResponse{
					Members: []*threading.Member{{EntityID: "entityID"}},
				}, nil))

				// ents for account
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{ID: "entityID"},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerThread-TeamThreadNotThreadMember": {
			tservice: func() *tservice {
				md := mock_directory.New(t)
				mt := mock_threads.New(t)
				mdl := mock_dl.New(t)
				mdl.Expect(mock.NewExpectation(mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				mt.Expect(mock.NewExpectation(mt.Thread, &threading.ThreadRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadResponse{
					Thread: &threading.Thread{
						Type: threading.ThreadType_TEAM,
					},
				}, nil))

				// thread members
				mt.Expect(mock.NewExpectation(mt.ThreadMembers, &threading.ThreadMembersRequest{
					ThreadID: "threadID",
				}).WithReturns(&threading.ThreadMembersResponse{
					Members: []*threading.Member{{EntityID: "entityID"}},
				}, nil))

				// ents for account
				md.Expect(mock.NewExpectation(md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
				}).WithReturns(
					&directory.LookupEntitiesResponse{
						Entities: []*directory.Entity{
							{ID: "differentEntityID"},
						},
					}, nil))
				return &tservice{
					service: &service{
						directory: md,
						threads:   mt,
						dal:       mdl,
					},
					finish: []mock.Finisher{md, mt, mdl},
				}
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
	}

	for cn, c := range cases {
		mID, err := dal.ParseMediaID(c.mediaID)
		test.OK(t, err)
		test.EqualsCase(t, cn, c.expected, c.tservice.service.CanAccess(context.Background(), mID, c.accountID))
		mock.FinishAll(c.tservice.finish...)
	}
}
