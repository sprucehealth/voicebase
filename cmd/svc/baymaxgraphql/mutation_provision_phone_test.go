package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	excommssettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"

	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestProvisionPhone(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account:12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"

	g.ra.Expect(mock.NewExpectation(g.ra.EntityForAccountID, entityID, acc.ID).WithReturns(
		&directory.Entity{
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
				{ID: entityID, Type: directory.EntityType_ORGANIZATION},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: entityID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: "+12068773590",
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateContact, &directory.CreateContactRequest{
		EntityID: entityID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_PHONE,
			Value:       "+12068773590",
			Provisioned: true,
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
			ID:   entityID,
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
		NodeID: entityID,
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
		"organizationId": entityID,
		"areaCode":       areaCode,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
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
}`, string(b))
}

func TestProvisionPhone_Unavailable(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID: "account:12345",
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"

	g.ra.Expect(mock.NewExpectation(g.ra.EntityForAccountID, entityID, acc.ID).WithReturns(
		&directory.Entity{
			ID:   "aodhigh",
			Type: directory.EntityType_INTERNAL,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
			Memberships: []*directory.Entity{
				{ID: entityID, Type: directory.EntityType_ORGANIZATION},
			},
		}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: entityID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{}, grpcErrorf(codes.InvalidArgument, "")))

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
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionPhoneNumber": {
			"clientMutationId": "a1b2c3",
			"errorCode": "UNAVAILABLE",
			"organization": null,
			"phoneNumber": null,
			"success": false
		}
	}
}`, string(b))
}
