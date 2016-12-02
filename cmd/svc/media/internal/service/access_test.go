package service

import (
	"testing"

	"context"

	"github.com/golang/mock/gomock"
	"github.com/sprucehealth/backend/cmd/svc/media/internal/dal"
	mock_dl "github.com/sprucehealth/backend/cmd/svc/media/internal/dal/test"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/care"
	mock_care "github.com/sprucehealth/backend/svc/care/mock"
	"github.com/sprucehealth/backend/svc/directory"
	mock_directory "github.com/sprucehealth/backend/svc/directory/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/backend/svc/threading/threadingmock"
)

type tservice struct {
	service *service
	ctrl    *gomock.Controller
	md      *mock_directory.Client
	mdl     *mock_dl.MockDAL
	mv      *mock_care.Client
	mt      *threadingmock.MockThreadsClient
	finish  []mock.Finisher
}

func (t *tservice) Finish() {
	t.ctrl.Finish()
	mock.FinishAll(t.finish...)
}

func newTService(t *testing.T) *tservice {
	ctrl := gomock.NewController(t)
	md := mock_directory.New(t)
	mt := threadingmock.NewMockThreadsClient(ctrl)
	mv := mock_care.New(t)
	mdl := mock_dl.New(t)
	return &tservice{
		ctrl: ctrl,
		md:   md,
		mt:   mt,
		mv:   mv,
		mdl:  mdl,
		service: &service{
			directory: md,
			threads:   mt,
			dal:       mdl,
			care:      mv,
		},
		finish: []mock.Finisher{md, mdl, mv},
	}
}

func TestCanAccess(t *testing.T) {
	cases := map[string]struct {
		tservice  *tservice
		mediaID   string
		accountID string
		expected  error
		finishers []mock.Finisher
	}{
		"LegacyMedia-CanAccess": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					(*dal.Media)(nil), dal.ErrNotFound))
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerAccountID-AccountIDMatches": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeAccount,
						OwnerID:   "accountID",
					}, nil))
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerAccountID-AccountIDMismatch": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeAccount,
						OwnerID:   "differentAccountID",
					}, nil))
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerEntity-SameEntity": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeEntity,
						OwnerID:   "entityID",
					}, nil))

				// entitiesForAccountID
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerEntity-DifferentEntity": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeEntity,
						OwnerID:   "entityID",
					}, nil))

				// entitiesForAccountID
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerOrg-OrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeOrganization,
						OwnerID:   "orgID",
					}, nil))

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerOrg-NotOrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeOrganization,
						OwnerID:   "orgID",
					}, nil))

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerThread-OrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				gomock.InOrder(
					ts.mt.EXPECT().Thread(context.Background(), &threading.ThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadResponse{
						Thread: &threading.Thread{
							Type:           threading.THREAD_TYPE_EXTERNAL,
							OrganizationID: "orgID",
						},
					}, nil),
				)

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerThread-NotOrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				gomock.InOrder(
					ts.mt.EXPECT().Thread(context.Background(), &threading.ThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadResponse{
						Thread: &threading.Thread{
							Type:           threading.THREAD_TYPE_EXTERNAL,
							OrganizationID: "orgID",
						},
					}, nil),
				)

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerThread-TeamThreadThreadMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				gomock.InOrder(
					ts.mt.EXPECT().Thread(context.Background(), &threading.ThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadResponse{
						Thread: &threading.Thread{
							Type: threading.THREAD_TYPE_TEAM,
						},
					}, nil),
					ts.mt.EXPECT().ThreadMembers(context.Background(), &threading.ThreadMembersRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadMembersResponse{
						Members: []*threading.Member{{EntityID: "entityID"}},
					}, nil),
				)

				// ents for account
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerThread-TeamThreadNotThreadMember": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				gomock.InOrder(
					ts.mt.EXPECT().Thread(context.Background(), &threading.ThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadResponse{
						Thread: &threading.Thread{
							Type: threading.THREAD_TYPE_TEAM,
						},
					}, nil),
					ts.mt.EXPECT().ThreadMembers(context.Background(), &threading.ThreadMembersRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadMembersResponse{
						Members: []*threading.Member{{EntityID: "entityID"}},
					}, nil),
				)

				// ents for account
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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
				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  ErrAccessDenied,
		},
		"OwnerSupportThread-LinkedThreadAccess": {
			tservice: func() *tservice {
				ts := newTService(t)
				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeThread,
						OwnerID:   "threadID",
					}, nil))

				// thread
				gomock.InOrder(
					ts.mt.EXPECT().Thread(context.Background(), &threading.ThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.ThreadResponse{
						Thread: &threading.Thread{
							Type:           threading.THREAD_TYPE_SUPPORT,
							OrganizationID: "orgID2",
						},
					}, nil),
				)

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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

				// linked thread
				gomock.InOrder(
					ts.mt.EXPECT().LinkedThread(context.Background(), &threading.LinkedThreadRequest{
						ThreadID: "threadID",
					}).Return(&threading.LinkedThreadResponse{
						Thread: &threading.Thread{
							Type:           threading.THREAD_TYPE_SUPPORT,
							OrganizationID: "orgID",
						},
					}, nil),
				)

				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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

				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerVisit-OrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)

				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeVisit,
						OwnerID:   "visitID",
					}, nil))

				ts.mv.Expect(mock.NewExpectation(ts.mv.GetVisit, &care.GetVisitRequest{
					ID: "visitID",
				}).WithReturns(&care.GetVisitResponse{
					Visit: &care.Visit{
						OrganizationID: "orgID",
					},
				}, nil))

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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

				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerSavedMessage-Org": {
			tservice: func() *tservice {
				ts := newTService(t)

				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeSavedMessage,
						OwnerID:   "savedMessageID",
					}, nil))

				gomock.InOrder(
					ts.mt.EXPECT().SavedMessages(context.Background(), &threading.SavedMessagesRequest{
						By: &threading.SavedMessagesRequest_IDs{
							IDs: &threading.IDList{
								IDs: []string{"savedMessageID"},
							},
						},
					}).Return(&threading.SavedMessagesResponse{
						SavedMessages: []*threading.SavedMessage{
							{
								OwnerEntityID:  "orgID",
								OrganizationID: "orgID",
							},
						},
					}, nil),
				)

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
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

				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"OwnerSavedMessage-OrgMember": {
			tservice: func() *tservice {
				ts := newTService(t)

				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeSavedMessage,
						OwnerID:   "savedMessageID",
					}, nil))

				gomock.InOrder(
					ts.mt.EXPECT().SavedMessages(context.Background(), &threading.SavedMessagesRequest{
						By: &threading.SavedMessagesRequest_IDs{
							IDs: &threading.IDList{
								IDs: []string{"savedMessageID"},
							},
						},
					}).Return(&threading.SavedMessagesResponse{
						SavedMessages: []*threading.SavedMessage{
							{
								OwnerEntityID:  "entID",
								OrganizationID: "orgID",
							},
						},
					}, nil),
				)

				// ent memberships
				ts.md.Expect(mock.NewExpectation(ts.md.LookupEntities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ACCOUNT_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_AccountID{
						AccountID: "accountID",
					},
					Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
					RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
				}).WithReturns(
					&directory.LookupEntitiesResponse{

						Entities: []*directory.Entity{
							{ID: "entID", Memberships: []*directory.Entity{{ID: "orgID"}}},
						},
					}, nil))

				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
		"Success-PublicMedia": {
			tservice: func() *tservice {
				ts := newTService(t)

				ts.mdl.Expect(mock.NewExpectation(ts.mdl.Media, dal.MediaID("mediaID")).WithReturns(
					&dal.Media{
						OwnerType: dal.MediaOwnerTypeVisit,
						OwnerID:   "visitID",
						Public:    true,
					}, nil))

				return ts
			}(),
			mediaID:   "mediaID",
			accountID: "accountID",
			expected:  nil,
		},
	}

	for cn, c := range cases {
		mID, err := dal.ParseMediaID(c.mediaID)
		test.OK(t, err)
		test.EqualsCase(t, cn, c.expected, c.tservice.service.CanAccess(context.Background(), mID, c.accountID))
		c.tservice.Finish()
	}
}
