package main

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
)

func TestAllowVisitAttachmentsQuery(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	threadID := "threadID"
	primaryEntityID := "primaryEntityID"
	orgID := "organizationID"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type:      directory.EntityType_PATIENT,
			AccountID: "account_12345",
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.LastLoginForAccount, &auth.GetLastLoginInfoRequest{
		AccountID: acc.ID,
	}).WithReturns(&auth.GetLastLoginInfoResponse{
		Platform: auth.Platform_IOS,
	}, nil))

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":true}}}`, res)
}

func TestAllowVisitAttachmentsQuery_Android(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	threadID := "threadID"
	primaryEntityID := "primaryEntityID"
	orgID := "organizationID"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type:      directory.EntityType_PATIENT,
			AccountID: "account_12345",
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.LastLoginForAccount, &auth.GetLastLoginInfoRequest{
		AccountID: acc.ID,
	}).WithReturns(&auth.GetLastLoginInfoResponse{
		Platform: auth.Platform_ANDROID,
	}, nil))

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":false}}}`, res)
}

func TestAllowVisitAttachmentsQuery_NoAccount(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	threadID := "threadID"
	primaryEntityID := "primaryEntityID"
	orgID := "organizationID"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":true}}}`, res)
}

func TestAllowVisitAttachmentsQuery_FeatureDisabled(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	threadID := "threadID"
	primaryEntityID := "primaryEntityID"
	orgID := "organizationID"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":false}}}`, res)
}

func TestAllowVisitAttachmentsQuery_Allowed(t *testing.T) {
	acc := &auth.Account{ID: "account_12345", Type: auth.AccountType_PROVIDER}
	ctx := context.Background()
	ctx = gqlctx.WithAccount(ctx, acc)
	threadID := "threadID"
	primaryEntityID := "primaryEntityID"
	orgID := "organizationID"

	g := newGQL(t)
	defer g.finish()

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			Type: directory.EntityType_INTERNAL,
			Memberships: []*directory.Entity{
				{
					ID: orgID,
				},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.Thread, threadID, "").WithReturns(&threading.Thread{
		Type:            threading.ThreadType_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type: directory.EntityType_PATIENT,
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		NodeID: orgID,
		Keys: []*settings.ConfigKey{
			{
				Key: baymaxgraphqlsettings.ConfigKeyVisitAttachments,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: primaryEntityID,
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_PATIENT},
	}).WithReturns([]*directory.Entity{
		{
			Type:      directory.EntityType_PATIENT,
			AccountID: "account_12345",
			Info: &directory.EntityInfo{
				DisplayName: "patient",
			},
		},
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.LastLoginForAccount, &auth.GetLastLoginInfoRequest{
		AccountID: acc.ID,
	}).WithReturns(&auth.GetLastLoginInfoResponse{
		Platform: auth.Platform_IOS,
	}, nil))

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":true}}}`, res)
}
