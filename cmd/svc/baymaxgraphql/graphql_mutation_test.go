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

func TestProvisionEmail_Organization(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	organizationID := "e1"
	entityID := "e12"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	// Looking up the orgnaization entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
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
		},
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
		},
	}, nil))

	// Lookup whether the domain exists or not
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntityDomain, &directory.LookupEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "",
	}).WithReturns(&directory.LookupEntityDomainResponse{}, grpc.Errorf(codes.NotFound, "")))

	// Create domain
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateEntityDomain, &directory.CreateEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "pup",
	}).WithReturns(&directory.CreateEntityDomainResponse{}, nil))

	// provision the email address
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateContact, &directory.CreateContactRequest{
		EntityID: organizationID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
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

	// Provision email address
	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: organizationID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
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
				result
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
			"result": "SUCCESS"
		}
	}
}`, string(b))
}

func TestProvisionEmail_Internal(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	organizationID := "o1"
	g.svc.emailDomain = "amdava.com"
	entityID := "e12"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	// Looking up the organization entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
			EntityInformation: []directory.EntityInformation{
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_EXTERNAL_IDS,
			},
		},
	}).WithReturns(&directory.LookupEntitiesResponse{
		Entities: []*directory.Entity{
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
		},
	}, nil))

	// Lookup whether the domain exists or not
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntityDomain, &directory.LookupEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "",
	}).WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// provision the email address
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateContact, &directory.CreateContactRequest{
		EntityID: entityID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
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

	// Provision email address
	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: entityID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
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
				result
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
			"result": "SUCCESS"
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_DomainExists(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	organizationID := "o1"
	entityID := "e1"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
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
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
			},
		},
	}, nil))

	// Looking up the orgnaization entity
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
		},
	}, nil))

	// Lookup whether the domain exists or not
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntityDomain, &directory.LookupEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "",
	}).WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// provision the email address
	g.dirC.Expect(mock.NewExpectation(g.dirC.CreateContact, &directory.CreateContactRequest{
		EntityID: organizationID,
		Contact: &directory.Contact{
			ContactType: directory.ContactType_EMAIL,
			Value:       "sup@pup.amdava.com",
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

	// Provision email address
	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
		ProvisionFor: organizationID,
		EmailAddress: emailToProvision,
	}).WithReturns(&excomms.ProvisionEmailAddressResponse{
		EmailAddress: emailToProvision,
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
				result
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
			"result": "SUCCESS"
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_DomainInUse(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	entityID := "e1"
	organizationID := "o1"
	localPart := "sup"
	subdomain := "pup"

	// Looking up the orgnaization entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
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
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
			},
		},
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
		},
	}, nil))

	// Lookup whether the domain exists or not
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntityDomain, &directory.LookupEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "",
	}).WithReturns(&directory.LookupEntityDomainResponse{
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
				result
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
			"result": "SUBDOMAIN_IN_USE"
		}
	}
}`, string(b))
}

func TestProvisionEmail_Organization_EmailInUse(t *testing.T) {
	g := newGQL(t)
	defer g.finish()

	ctx := context.Background()
	acc := &account{
		ID: "account:12345",
	}
	ctx = ctxWithAccount(ctx, acc)

	g.svc.emailDomain = "amdava.com"
	entityID := "e1"
	organizationID := "o1"
	localPart := "sup"
	subdomain := "pup"
	emailToProvision := "sup@pup.amdava.com"

	// Looking up the orgnaization entity
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntities, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: organizationID,
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
				ID:   organizationID,
				Type: directory.EntityType_ORGANIZATION,
				Info: &directory.EntityInfo{
					DisplayName: "Schmee",
				},
			},
		},
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
		},
	}, nil))

	// Lookup whether the domain exists or not
	g.dirC.Expect(mock.NewExpectation(g.dirC.LookupEntityDomain, &directory.LookupEntityDomainRequest{
		EntityID: organizationID,
		Domain:   "",
	}).WithReturns(&directory.LookupEntityDomainResponse{
		EntityID: organizationID,
		Domain:   "pup",
	}, nil))

	// provision the email address
	g.exC.Expect(mock.NewExpectation(g.exC.ProvisionEmailAddress, &excomms.ProvisionEmailAddressRequest{
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
				result
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
			"organization": null,
			"result": "LOCAL_PART_IN_USE"
		}
	}
}`, string(b))
}
