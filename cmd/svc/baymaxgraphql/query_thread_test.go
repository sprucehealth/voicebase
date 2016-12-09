package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	baymaxgraphqlsettings "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/settings"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/backend/svc/settings"
	smock "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/svc/threading"
	"github.com/sprucehealth/graphql"
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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
		Type:            threading.THREAD_TYPE_SECURE_EXTERNAL,
		PrimaryEntityID: primaryEntityID,
		OrganizationID:  orgID,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_EntityID{
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

	res := g.query(ctx, `
 query _ {
   thread(id: "threadID") {
    allowVisitAttachments
      }
 }
`, nil)

	responseEquals(t, `{"data":{"thread":{"allowVisitAttachments":true}}}`, res)
}

type testAllowPaymentRequestAttachmentParams struct {
	p  graphql.ResolveParams
	rm *ramock.ResourceAccessor
	sm *smock.Client
}

func (t *testAllowPaymentRequestAttachmentParams) Finishers() []mock.Finisher {
	return []mock.Finisher{t.rm, t.sm}
}

func TestAllowPaymentRequestAttachment(t *testing.T) {
	orgID := "orgID"
	cases := map[string]struct {
		tp          *testAllowPaymentRequestAttachmentParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success-Allowed": {
			tp: func() *testAllowPaymentRequestAttachmentParams {
				rm := ramock.New(t)
				sm := smock.New(t)
				sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
					NodeID: orgID,
					Keys: []*settings.ConfigKey{
						{
							Key: baymaxgraphqlsettings.ConfigKeyPayments,
						},
					},
				}).WithReturns(&settings.GetValuesResponse{
					Values: []*settings.Value{
						{
							Value: &settings.Value_Boolean{
								Boolean: &settings.BooleanValue{Value: true},
							},
						},
					},
				}, nil))
				rm.Expect(mock.NewExpectation(rm.VendorAccounts, &payments.VendorAccountsRequest{
					EntityID: orgID,
				}).WithReturns(&payments.VendorAccountsResponse{VendorAccounts: []*payments.VendorAccount{{}}}, nil))
				return &testAllowPaymentRequestAttachmentParams{
					p: graphql.ResolveParams{
						Context: gqlctx.WithAccount(context.Background(), &auth.Account{ID: "ID", Type: auth.AccountType_PROVIDER}),
						Source: &models.Thread{
							Type:           models.ThreadTypeSecureExternal,
							OrganizationID: orgID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
								"service":        &service{settings: sm},
							},
						},
					},
					rm: rm,
					sm: sm,
				}
			}(),
			Expected:    true,
			ExpectedErr: nil,
		},
		"Success-NotAllowed-NoVendorAccounts": {
			tp: func() *testAllowPaymentRequestAttachmentParams {
				rm := ramock.New(t)
				sm := smock.New(t)
				sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
					NodeID: orgID,
					Keys: []*settings.ConfigKey{
						{
							Key: baymaxgraphqlsettings.ConfigKeyPayments,
						},
					},
				}).WithReturns(&settings.GetValuesResponse{
					Values: []*settings.Value{
						{
							Value: &settings.Value_Boolean{
								Boolean: &settings.BooleanValue{Value: true},
							},
						},
					},
				}, nil))
				rm.Expect(mock.NewExpectation(rm.VendorAccounts, &payments.VendorAccountsRequest{
					EntityID: orgID,
				}).WithReturns(&payments.VendorAccountsResponse{VendorAccounts: []*payments.VendorAccount{}}, nil))
				return &testAllowPaymentRequestAttachmentParams{
					p: graphql.ResolveParams{
						Context: gqlctx.WithAccount(context.Background(), &auth.Account{ID: "ID", Type: auth.AccountType_PROVIDER}),
						Source: &models.Thread{
							Type:           models.ThreadTypeSecureExternal,
							OrganizationID: orgID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
								"service":        &service{settings: sm},
							},
						},
					},
					rm: rm,
					sm: sm,
				}
			}(),
			Expected:    false,
			ExpectedErr: nil,
		},
		"Success-NotAllowed-SettingDisabled": {
			tp: func() *testAllowPaymentRequestAttachmentParams {
				rm := ramock.New(t)
				sm := smock.New(t)
				sm.Expect(mock.NewExpectation(sm.GetValues, &settings.GetValuesRequest{
					NodeID: orgID,
					Keys: []*settings.ConfigKey{
						{
							Key: baymaxgraphqlsettings.ConfigKeyPayments,
						},
					},
				}).WithReturns(&settings.GetValuesResponse{
					Values: []*settings.Value{
						{
							Value: &settings.Value_Boolean{
								Boolean: &settings.BooleanValue{Value: false},
							},
						},
					},
				}, nil))
				return &testAllowPaymentRequestAttachmentParams{
					p: graphql.ResolveParams{
						Context: gqlctx.WithAccount(context.Background(), &auth.Account{ID: "ID", Type: auth.AccountType_PROVIDER}),
						Source: &models.Thread{
							Type:           models.ThreadTypeSecureExternal,
							OrganizationID: orgID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
								"service":        &service{settings: sm},
							},
						},
					},
					rm: rm,
					sm: sm,
				}
			}(),
			Expected:    false,
			ExpectedErr: nil,
		},
		"Success-NotAllowed-Patient": {
			tp: func() *testAllowPaymentRequestAttachmentParams {
				rm := ramock.New(t)
				sm := smock.New(t)
				return &testAllowPaymentRequestAttachmentParams{
					p: graphql.ResolveParams{
						Context: gqlctx.WithAccount(context.Background(), &auth.Account{ID: "ID", Type: auth.AccountType_PATIENT}),
						Source: &models.Thread{
							Type:           models.ThreadTypeSecureExternal,
							OrganizationID: orgID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
								"service":        &service{settings: sm},
							},
						},
					},
					rm: rm,
					sm: sm,
				}
			}(),
			Expected:    false,
			ExpectedErr: nil,
		},
		"Success-NotAllowed-ExternalThread": {
			tp: func() *testAllowPaymentRequestAttachmentParams {
				rm := ramock.New(t)
				sm := smock.New(t)
				return &testAllowPaymentRequestAttachmentParams{
					p: graphql.ResolveParams{
						Context: gqlctx.WithAccount(context.Background(), &auth.Account{ID: "ID", Type: auth.AccountType_PROVIDER}),
						Source: &models.Thread{
							Type:           models.ThreadTypeExternal,
							OrganizationID: orgID,
						},
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: rm,
								"service":        &service{settings: sm},
							},
						},
					},
					rm: rm,
					sm: sm,
				}
			}(),
			Expected:    false,
			ExpectedErr: nil,
		},
	}

	for cn, c := range cases {
		out, err := resolveAllowPaymentRequestAttachments(c.tp.p)
		test.EqualsCase(t, cn, c.Expected, out)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		mock.FinishAll(c.tp.Finishers()...)
	}
}
