package worker

import (
	"testing"

	"context"

	dalmock "github.com/sprucehealth/backend/cmd/svc/operational/internal/dal/mock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	authmock "github.com/sprucehealth/backend/svc/auth/mock"
	"github.com/sprucehealth/backend/svc/directory"
	directorymock "github.com/sprucehealth/backend/svc/directory/mock"
	excommsmock "github.com/sprucehealth/backend/svc/excomms/mock"
	"github.com/sprucehealth/backend/svc/operational"
	threadingmock "github.com/sprucehealth/backend/svc/threading/mock"
)

func TestBlockAccountWorker(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	mt := threadingmock.New(t)
	defer mt.Finish()

	mdir := directorymock.New(t)
	defer mdir.Finish()

	me := excommsmock.New(t)
	defer me.Finish()

	ma := authmock.New(t)
	defer ma.Finish()

	accountID := "accountID"
	spruceOrgID := "spruceOrgID"

	ma.Expect(mock.NewExpectation(ma.BlockAccount, &auth.BlockAccountRequest{
		AccountID: accountID,
	}).WithReturns(&auth.BlockAccountResponse{
		Account: &auth.Account{
			ID:        accountID,
			FirstName: "Block",
			LastName:  "Example",
		},
	}, nil))

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "e1",
				Memberships: []*directory.Entity{
					{
						Type: directory.EntityType_ORGANIZATION,
						ID:   "o1",
					},
				},
			},
		},
	}, nil))

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "o1",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "o1",
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Provisioned: false,
					},
					{
						ContactType: directory.ContactType_PHONE,
						Provisioned: true,
						Value:       "+17348465522",
					},
				},
				Members: []*directory.Entity{
					{
						ID: "e1",
					},
				},
			},
		},
	}, nil))

	md.Expect(mock.NewExpectation(md.MarkAccountAsBlocked, accountID))

	w := NewBlockAccountWorker(ma, mdir, me, mt, nil, md, "sqs_url", spruceOrgID)
	test.OK(t, w.processEvent(context.Background(), &operational.BlockAccountRequest{
		AccountID: accountID,
	}))
}

func TestBlockAccountWorker_NoProvisionedPhoneNumber(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	mt := threadingmock.New(t)
	defer mt.Finish()

	mdir := directorymock.New(t)
	defer mdir.Finish()

	me := excommsmock.New(t)
	defer me.Finish()

	ma := authmock.New(t)
	defer ma.Finish()

	accountID := "accountID"
	spruceOrgID := "spruceOrgID"

	ma.Expect(mock.NewExpectation(ma.BlockAccount, &auth.BlockAccountRequest{
		AccountID: accountID,
	}).WithReturns(&auth.BlockAccountResponse{
		Account: &auth.Account{
			ID:        accountID,
			FirstName: "Block",
			LastName:  "Example",
		},
	}, nil))

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: accountID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "e1",
				Memberships: []*directory.Entity{
					{
						Type: directory.EntityType_ORGANIZATION,
						ID:   "o1",
					},
				},
			},
		},
	}, nil))

	mdir.Expect(mock.NewExpectation(mdir.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: "o1",
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID: "o1",
				Contacts: []*directory.Contact{
					{
						ContactType: directory.ContactType_PHONE,
						Provisioned: false,
					},
				},
				Members: []*directory.Entity{
					{
						ID: "e1",
					},
				},
			},
		},
	}, nil))

	md.Expect(mock.NewExpectation(md.MarkAccountAsBlocked, accountID))

	w := NewBlockAccountWorker(ma, mdir, me, mt, nil, md, "sqs_url", spruceOrgID)
	test.OK(t, w.processEvent(context.Background(), &operational.BlockAccountRequest{
		AccountID: accountID,
	}))
}
