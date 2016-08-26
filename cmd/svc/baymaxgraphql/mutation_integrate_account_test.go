package main

import (
	"fmt"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	ramock "github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess/mock"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/patientsync"
	"github.com/sprucehealth/backend/svc/payments"
	"github.com/sprucehealth/graphql"
)

type testConnectVendorAccountParams struct {
	p         graphql.ResolveParams
	finishers []mock.Finisher
}

func TestConnectVendorStripeAccount(t *testing.T) {
	mutationID := "mutationID"
	entityID := "entityID"
	code := "code"
	cases := map[string]struct {
		In          integrateAccountInput
		TestParams  *testConnectVendorAccountParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Error-NotFound-NoEntity": {
			In: integrateAccountInput{
				ClientMutationID: mutationID,
				EntityID:         entityID,
				Code:             code,
			},
			TestParams: func() *testConnectVendorAccountParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: entityID,
					},
				}).WithReturns(([]*directory.Entity)(nil), grpc.Errorf(codes.NotFound, "Not Found")))
				return &testConnectVendorAccountParams{
					p: graphql.ResolveParams{
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &integrateAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        integrateAccountErrorCodeEntityNotFound,
				ErrorMessage:     fmt.Sprintf("Entity %s Not Found", entityID),
			},
		},
		"Error-NotOrg": {
			In: integrateAccountInput{
				ClientMutationID: mutationID,
				EntityID:         entityID,
				Code:             code,
			},
			TestParams: func() *testConnectVendorAccountParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: entityID,
					},
				}).WithReturns([]*directory.Entity{
					{
						Type: directory.EntityType_INTERNAL,
					},
				}, nil))
				return &testConnectVendorAccountParams{
					p: graphql.ResolveParams{
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &integrateAccountOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        integrateAccountErrorCodeEntityNotSupported,
				ErrorMessage:     fmt.Sprintf("Entity %s is not supported for this integration. Expect an Organization", entityID),
			},
		},
		"Success": {
			In: integrateAccountInput{
				ClientMutationID: mutationID,
				EntityID:         entityID,
				Code:             code,
			},
			TestParams: func() *testConnectVendorAccountParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.Entities, &directory.LookupEntitiesRequest{
					LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
					LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
						EntityID: entityID,
					},
				}).WithReturns([]*directory.Entity{
					{
						Type: directory.EntityType_ORGANIZATION,
					},
				}, nil))
				mra.Expect(mock.NewExpectation(mra.ConnectVendorAccount, &payments.ConnectVendorAccountRequest{
					EntityID: entityID,
					Type:     payments.VENDOR_ACCOUNT_TYPE_STRIPE,
					ConnectVendorAccountOneof: &payments.ConnectVendorAccountRequest_StripeRequest{
						StripeRequest: &payments.StripeAccountConnectRequest{
							Code: code,
						},
					},
				}))
				return &testConnectVendorAccountParams{
					p: graphql.ResolveParams{
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &integrateAccountOutput{
				ClientMutationID: mutationID,
				Success:          true,
			},
		},
	}
	for cn, c := range cases {
		out, err := connectVendorStripeAccount(c.TestParams.p, c.In)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}

func TestConfigureSync(t *testing.T) {
	mutationID := "mutationID"
	entityID := "entityID"
	code := "code"
	cases := map[string]struct {
		In          integrateAccountInput
		TestParams  *testConnectVendorAccountParams
		Expected    interface{}
		ExpectedErr error
	}{
		"Success": {
			In: integrateAccountInput{
				ClientMutationID: mutationID,
				EntityID:         entityID,
				Code:             code,
			},
			TestParams: func() *testConnectVendorAccountParams {
				mra := ramock.New(t)
				mra.Expect(mock.NewExpectation(mra.ConfigurePatientSync, &patientsync.ConfigureSyncRequest{
					OrganizationEntityID: entityID,
					Token:                code,
					Source:               patientsync.SOURCE_HINT,
				}))
				return &testConnectVendorAccountParams{
					p: graphql.ResolveParams{
						Info: graphql.ResolveInfo{
							RootValue: map[string]interface{}{
								raccess.ParamKey: mra,
							},
						},
					},
					finishers: []mock.Finisher{mra},
				}
			}(),
			ExpectedErr: nil,
			Expected: &integrateAccountOutput{
				ClientMutationID: mutationID,
				Success:          true,
			},
		},
	}
	for cn, c := range cases {
		out, err := configureSync(c.TestParams.p, c.In)
		test.EqualsCase(t, cn, c.ExpectedErr, err)
		test.EqualsCase(t, cn, c.Expected, out)
		mock.FinishAll(c.TestParams.finishers...)
	}
}
