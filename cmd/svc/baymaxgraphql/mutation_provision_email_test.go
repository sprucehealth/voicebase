package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func TestProvisionEmail_Organization(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	organizationID := "e1"
	entityID := "e12"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	// Looking up the orgnaization entity
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   organizationID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
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
	})

	// Lookup whether the domain exists or not
	g.ra.Expect(mock.NewExpectation(g.ra.EntityDomain, organizationID, "").WithReturns(&directory.LookupEntityDomainResponse{}, grpc.Errorf(codes.NotFound, "")))

	// Create domain
	g.ra.Expect(mock.NewExpectation(g.ra.CreateEntityDomain, organizationID, "pup").WithReturns(&directory.CreateEntityDomainResponse{}, nil))

	// Provision email address
	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: organizationID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateContact, &directory.CreateContactRequest{
		EntityID: organizationID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
			Provisioned: true,
			Verified:    true,
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
			Memberships: []*directory.Entity{},
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "sup@pup.amdava.com",
					Provisioned: true,
				},
			},
		},
	}, nil))

	res := g.query(ctx, `
		mutation _ ($organizationID: ID!, $localPart: String!, $subdomain: String!) {
			provisionEmail(input: {
				clientMutationId: "a1b2c3",
				localPart: $localPart,
				subdomain: $subdomain,
				organizationID: $organizationID,
			}) {
				clientMutationId
				success
				organization {
					id
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"organizationID": organizationID,
		"localPart":      localPart,
		"subdomain":      subdomain,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionEmail": {
			"clientMutationId": "a1b2c3",
			"organization": {
				"contacts": [
					{
						"provisioned": true,
						"type": "EMAIL",
						"value": "sup@pup.amdava.com"
					}
				],
				"id": "e1"
			},
			"success": true
		}
	}
}`, string(b))
}

func TestProvisionEmail_Internal(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	organizationID := "o1"
	g.svc.emailDomain = "amdava.com"
	entityID := "e12"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_EXTERNAL_IDS,
			},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
	}).WithReturns(
		[]*directory.Entity{
			{
				ID:   entityID,
				Type: directory.EntityType_INTERNAL,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
				Memberships: []*directory.Entity{
					{
						ID:   organizationID,
						Type: directory.EntityType_ORGANIZATION,
					},
				},
				ExternalIDs: []string{
					acc.ID,
				},
			},
		}, nil))

	// Lookup whether the domain exists or not
	g.ra.Expect(mock.NewExpectation(g.ra.EntityDomain, organizationID, "").WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// Provision email address
	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: entityID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
	}, nil))

	g.ra.Expect(mock.NewExpectation(g.ra.CreateContact, &directory.CreateContactRequest{
		EntityID: entityID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
			Provisioned: true,
			Verified:    true,
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
			Memberships: []*directory.Entity{},
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "sup@pup.amdava.com",
					Provisioned: true,
				},
			},
		},
	}, nil))

	// Provisioning email address

	res := g.query(ctx, `
		mutation _ ($entityID: ID!, $localPart: String!, $subdomain: String!) {
			provisionEmail(input: {
				clientMutationId: "a1b2c3",
				localPart: $localPart,
				subdomain: $subdomain,
				entityID: $entityID,
			}) {
				clientMutationId
				success
				entity {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"entityID":  entityID,
		"localPart": localPart,
		"subdomain": subdomain,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionEmail": {
			"clientMutationId": "a1b2c3",
			"entity": {
				"contacts": [
					{
						"provisioned": true,
						"type": "EMAIL",
						"value": "sup@pup.amdava.com"
					}
				]
			},
			"success": true
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_DomainExists(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	organizationID := "o1"
	entityID := "e1"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   organizationID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
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
	})

	// Lookup whether the domain exists or not
	g.ra.Expect(mock.NewExpectation(g.ra.EntityDomain, organizationID, "").WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// Provision email address
	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: organizationID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
	}, nil))

	// provision the email address
	g.ra.Expect(mock.NewExpectation(g.ra.CreateContact, &directory.CreateContactRequest{
		EntityID: organizationID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
			Provisioned: true,
			Verified:    true,
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
			Memberships: []*directory.Entity{},
			Contacts: []*directory.Contact{
				{
					ContactType: directory.ContactType_EMAIL,
					Value:       "sup@pup.amdava.com",
					Provisioned: true,
				},
			},
		},
	}, nil))

	// Provisioning email address

	res := g.query(ctx, `
		mutation _ ($organizationID: ID!, $localPart: String!, $subdomain: String!) {
			provisionEmail(input: {
				clientMutationId: "a1b2c3",
				localPart: $localPart,
				subdomain: $subdomain,
				organizationID: $organizationID,
			}) {
				clientMutationId
				success
				organization {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"organizationID": organizationID,
		"localPart":      localPart,
		"subdomain":      subdomain,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionEmail": {
			"clientMutationId": "a1b2c3",
			"organization": {
				"contacts": [
					{
						"provisioned": true,
						"type": "EMAIL",
						"value": "sup@pup.amdava.com"
					}
				]
			},
			"success": true
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_DomainInUse(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	entityID := "e1"
	organizationID := "o1"
	localPart := "sup"
	subdomain := "pup"

	// Looking up the orgnaization entity
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   organizationID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   entityID,
			Type: directory.EntityType_INTERNAL,
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
	})

	// Lookup whether the domain exists or not
	g.ra.Expect(mock.NewExpectation(g.ra.EntityDomain, organizationID, "").WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "dup",
	}, nil))

	// Provisioning email address

	res := g.query(ctx, `
		mutation _ ($organizationID: ID!, $localPart: String!, $subdomain: String!) {
			provisionEmail(input: {
				clientMutationId: "a1b2c3",
				localPart: $localPart,
				subdomain: $subdomain,
				organizationID: $organizationID,
			}) {
				clientMutationId
				success
				errorCode
				entity {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"organizationID": organizationID,
		"localPart":      localPart,
		"subdomain":      subdomain,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionEmail": {
			"clientMutationId": "a1b2c3",
			"entity": null,
			"errorCode": "SUBDOMAIN_IN_USE",
			"success": false
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_EmailInUse(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &auth.Account{
		ID:   "account:12345",
		Type: auth.AccountType_PROVIDER,
	}
	ctx = gqlctx.WithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	// entityID := "e1"
	organizationID := "o1"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	// Looking up the orgnaization entity
	g.ra.Expect(mock.NewExpectation(g.ra.Entities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
		Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	}).WithReturns([]*directory.Entity{
		{
			ID:   organizationID,
			Type: directory.EntityType_ORGANIZATION,
			Info: &directory.EntityInfo{
				DisplayName: "Schmee",
			},
		},
	}, nil))

	expectEntityInOrgForAccountID(g.ra, acc.ID, []*directory.Entity{
		{
			ID:   organizationID,
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
	})
	// Lookup whether the domain exists or not
	g.ra.Expect(mock.NewExpectation(g.ra.EntityDomain, organizationID, "").WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// provision the email address
	g.ra.Expect(mock.NewExpectation(g.ra.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: organizationID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{}, grpc.Errorf(codes.AlreadyExists, "")))
	// Provisioning email address

	res := g.query(ctx, `
		mutation _ ($organizationID: ID!, $localPart: String!, $subdomain: String!) {
			provisionEmail(input: {
				clientMutationId: "a1b2c3",
				localPart: $localPart,
				subdomain: $subdomain,
				organizationID: $organizationID,
			}) {
				clientMutationId
				success
				errorCode
				organization {
					 contacts {
						type
						value
						provisioned
					}
				}
			}
		}`, map[string]interface{}{
		"localPart":      localPart,
		"subdomain":      subdomain,
		"organizationID": organizationID,
	})
	b, err := json.MarshalIndent(res, "", "\t")
	test.OK(t, err)
	test.Equals(t, `{
	"data": {
		"provisionEmail": {
			"clientMutationId": "a1b2c3",
			"errorCode": "LOCAL_PART_IN_USE",
			"organization": null,
			"success": false
		}
	}
}`, string(b))
}
