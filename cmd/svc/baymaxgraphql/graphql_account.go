package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
)

var meType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Me",
		Fields: graphql.Fields{
			"account":             &graphql.Field{Type: graphql.NewNonNull(accountInterfaceType)},
			"clientEncryptionKey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var accountTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "AccountType",
	Values: graphql.EnumValueConfigMap{
		string(models.AccountTypePatient): &graphql.EnumValueConfig{
			Value:       string(models.AccountTypePatient),
			Description: "A patient account",
		},
		string(models.AccountTypeProvider): &graphql.EnumValueConfig{
			Value:       string(models.AccountTypeProvider),
			Description: "A provider account",
		},
	},
})

var accountInterfaceType = graphql.NewInterface(
	graphql.InterfaceConfig{
		Name: "Account",
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
			"type":          &graphql.Field{Type: graphql.NewNonNull(accountTypeEnum)},
			"organizations": &graphql.Field{Type: graphql.NewList(graphql.NewNonNull(organizationType))},
		},
	},
)

func init() {
	// This is done here rather than at declaration time to avoid an unresolvable compile time decleration loop
	accountInterfaceType.ResolveType = func(value interface{}, info graphql.ResolveInfo) *graphql.Object {
		switch value.(type) {
		case *models.ProviderAccount:
			return providerAccountType
		case *models.PatientAccount:
			return patientAccountType
		}
		return nil
	}
}

func lookupAccount(ctx context.Context, ram raccess.ResourceAccessor, accountID string) (interface{}, error) {
	account, err := ram.Account(ctx, accountID)
	if err != nil {
		return nil, err
	}
	// Since we only use the ID we don't really need to do the lookup, but
	// it allows us to check if the account exists.
	return &models.ProviderAccount{
		ID: account.ID,
	}, nil
}

func accountOrganizations(p graphql.ResolveParams, a models.Account) ([]*models.Organization, error) {
	svc := serviceFromParams(p)
	ram := raccess.ResourceAccess(p)
	ctx := p.Context
	if a == nil {
		// Shouldn't be possible I don't think
		return nil, errors.InternalError(ctx, errors.New("nil account"))
	}

	entities, err := ram.Entities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_EXTERNAL_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_ExternalID{
			ExternalID: a.GetID(),
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             1,
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS},
		},
		Statuses:   []directory.EntityStatus{directory.EntityStatus_ACTIVE},
		RootTypes:  []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_PATIENT},
		ChildTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return nil, errors.InternalError(ctx, fmt.Errorf("unable to lookup entities for %s: %s", a.GetID(), err))
	}

	sh := devicectx.SpruceHeaders(p.Context)

	var orgs []*models.Organization
	for _, e := range entities {
		entity, err := transformEntityToResponse(svc.staticURLPrefix, e, sh, gqlctx.Account(ctx))
		if err != nil {
			return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform entity: %+v", err))
		}
		for _, em := range e.Memberships {
			if em.Type == directory.EntityType_ORGANIZATION {
				oc, err := transformContactsToResponse(em.Contacts)
				if err != nil {
					return nil, errors.InternalError(ctx, fmt.Errorf("failed to transform org contacts: %+v", err))
				}
				orgs = append(orgs, &models.Organization{
					ID:       em.ID,
					Name:     em.Info.DisplayName,
					Contacts: oc,
					Entity:   entity,
				})
			}
		}
	}
	return orgs, nil
}
