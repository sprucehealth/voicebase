package main

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	excommssettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestProvisionPhone(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"
	organizationID := "org"

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "aodhigh",
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_PHONE,
					Value:       "+17348465522",
				},
			},
			Memberships: []*directory.Entity{
				{ID: organizationID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: organizationID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
		UUID: organizationID + ":primary",
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: "+12068773590",
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateContact, &directory.CreateContactRequest{
		EntityID: organizationID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12068773590",
			Provisioned: true,
			Verified:    true,
			Label:       "Primary",
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
			},
		},
	}).WithReturns(&directory.CreateContactResponse{
		Entity: &directory.Entity{
			ID:   organizationID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_PHONE,
					Provisioned: true,
					Value:       "+12068773590",
				},
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.SetValue, &settings.SetValueRequest{
		NodeID: organizationID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    excommssettings.ConfigKeyForwardingList,
				Subkey: "+12068773590",
			},
			Type: settings.ConfigType_STRING_LIST,
			Value: &settings.Value_StringList{
				StringList: &settings.StringListValue{
					Values: []string{
						"(734) 846-5522",
					},
				},
			},
		},
	}))

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		Key: &directory.LookupEntitiesRequest_AccountID{
			AccountID: acc.ID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   entityID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{
					ID:   organizationID,
					Type: directory.EntityType_ORGANIZATION,
				},
			},
		},
	}, nil))

	g.settingsC.Expect(mock.NewExpectation(g.settingsC.GetValues, &settings.GetValuesRequest{
		Keys:   []*settings.ConfigKey{{Key: "default_provisioned_phone_number"}},
		NodeID: entityID,
	}).WithReturns(&settings.GetValuesResponse{}, nil))

	res := g.query(ctx, `
		mutation _ ($organizationId: ID!, $areaCode: String!) {
			provisionPhoneNumber(input: {
				clientMutationId: "a1b2c3",
				areaCode: $areaCode,
				organizationID: $organizationId,
			}) {
				clientMutationId
				success
				phoneNumber
				organization {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"organizationId": organizationID,
		"areaCode":       areaCode,
	})
	responseEquals(t, `{
		"data": {
			"provisionPhoneNumber": {
				"clientMutationId": "a1b2c3",
				"organization": {
					"contacts": [
						{
							"provisioned": true,
							"type": "PHONE",
							"value": "+12068773590"
						}
					]
				},
				"phoneNumber": "(206) 877-3590",
				"success": true
			}
		}
	}`, res)
}

func TestProvisionPhone_Unavailable(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   "aodhigh",
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: entityID, Type: directory.EntityType_ORGANIZATION},
			},
		},
	})

	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: entityID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
		UUID: "12345:primary",
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{}, grpc.Errorf(codes.InvalidArgument, "")))

	res := g.query(ctx, `
		mutation _ ($organizationId: ID!, $areaCode: String!) {
			provisionPhoneNumber(input: {
				clientMutationId: "a1b2c3",
				areaCode: $areaCode,
				organizationID: $organizationId,
			}) {
				clientMutationId
				success
				errorCode
				phoneNumber
				organization {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
			}`, map[string]interface{}{
		"organizationId": entityID,
		"areaCode":       areaCode,
	})
	responseEquals(t, `{
		"data": {
			"provisionPhoneNumber": {
				"clientMutationId": "a1b2c3",
				"errorCode": "UNAVAILABLE",
				"organization": null,
				"phoneNumber": null,
				"success": false
			}
		}
	}`, res)
}
