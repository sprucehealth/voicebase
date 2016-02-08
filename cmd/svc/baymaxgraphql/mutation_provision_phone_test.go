package main

import (
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestProvisionPhone(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"

	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: entityID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{
		PhoneNumber: "+12068773590",
	}, nil))

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
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
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "aodigh",
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{
						Type: directory.EntityType_ORGANIZATION,
						ID:   entityID,
					},
				},
			},
		},
	}, nil))

	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateContact, &directory.CreateContactRequest{
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
	res := g.query(ctx, `
		mutation _ ($organizationId: ID!, $areaCode: String!) {
			provisionPhoneNumber(input: {
				clientMutationId: "a1b2c3",
				areaCode: $areaCode,
				organizationID: $organizationId,
			}) {
				clientMutationId
				result
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
			"phoneNumber": "+12068773590",
			"result": "SUCCESS"
		}
	}
}`, string(b))
}

func TestProvisionPhone_Unavailable(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	entityID := "12345"
	areaCode := "203"

	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionPhoneNumber, &excomms.ProvisionPhoneNumberRequest{
		ProvisionFor: entityID,
		Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
			AreaCode: areaCode,
		},
	}).WithReturns(&excomms.ProvisionPhoneNumberResponse{}, grpc.Errorf(codes.InvalidArgument, "")))

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
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
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
			{
				ID:   "aodigh",
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{
						Type: directory.EntityType_ORGANIZATION,
						ID:   entityID,
					},
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($organizationId: ID!, $areaCode: String!) {
			provisionPhoneNumber(input: {
				clientMutationId: "a1b2c3",
				areaCode: $areaCode,
				organizationID: $organizationId,
			}) {
				clientMutationId
				result
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
			"organization": null,
			"phoneNumber": "",
			"result": "UNAVAILABLE"
		}
	}
}`, string(b))
}
