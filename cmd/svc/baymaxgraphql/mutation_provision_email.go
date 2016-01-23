package main

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// provision email

type provisionEmailOutput struct {
	ClientMutationID string  `json:"clientMutationId"`
	Result           string  `json:"result"`
	Entity           *entity `json:"entity"`
}

var provisionEmailInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ProvisionEmailInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Specify the entityID of the provider or the organization here, depending on who the email is being provisioned for."},
			"localPart": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Email address to provision.",
			},
			"subdomain": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Subdomain to use for email address",
			},
		},
	},
)

const (
	provisionEmailResultSuccess        = "SUCCESS"
	provisionEmailResultSubdomainInUse = "SUBDOMAIN_IN_USE"
	provisionEmailResultLocalPartInUse = "LOCAL_PART_IN_USE"
)

var provisionEmailResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ProvisionEmailResult",
		Description: "Result of provisionEmail mutation",
		Values: graphql.EnumValueConfigMap{
			provisionEmailResultSuccess: &graphql.EnumValueConfig{
				Value:       provisionEmailResultSuccess,
				Description: "Success",
			},
			provisionEmailResultSubdomainInUse: &graphql.EnumValueConfig{
				Value:       provisionEmailResultSubdomainInUse,
				Description: "Subdomain not found",
			},
			provisionEmailResultLocalPartInUse: &graphql.EnumValueConfig{
				Value:       provisionEmailResultLocalPartInUse,
				Description: "Local part of the address is in use",
			},
		},
	},
)

var provisionEmailOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProvisionEmailPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"result":           &graphql.Field{Type: graphql.NewNonNull(provisionEmailResultType)},
			"entity": &graphql.Field{
				Type: entityType,
			},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*provisionEmailOutput)
			return ok
		},
	},
)

var provisionEmailField = &graphql.Field{
	Type: graphql.NewNonNull(provisionEmailOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: provisionEmailInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		localPart, _ := input["localPart"].(string)
		subdomain, _ := input["subdomain"].(string)
		entityID, _ := input["entityID"].(string)
		emailAddress := localPart + "@" + subdomain + "." + svc.emailDomain

		entity, err := svc.entity(ctx, entityID)
		if err != nil {
			return nil, internalError(err)
		}

		if !validate.Email(emailAddress) {
			return nil, errors.New("invalid email address")
		}

		switch entity.Type {
		case directory.EntityType_ORGANIZATION:

			_, domain, err := svc.entityDomain(ctx, entityID, "")
			if err != nil {
				return nil, err
			} else if domain != "" && domain != subdomain {
				return &provisionEmailOutput{
					Result:           provisionEmailResultSubdomainInUse,
					ClientMutationID: mutationID,
				}, nil
			}

			// lets go ahead and create domain for organization
			if domain == "" {
				_, err := svc.directory.CreateEntityDomain(ctx, &directory.CreateEntityDomainRequest{
					EntityID: entityID,
					Domain:   subdomain,
				})
				if err != nil {
					return nil, internalError(err)
				}
			}
		case directory.EntityType_INTERNAL:
			var orgID string
			for _, e := range entity.Memberships {
				if e.Type == directory.EntityType_ORGANIZATION {
					orgID = e.ID
					break
				}
			}
			if orgID == "" {
				return nil, errors.New("internal entity is not part of any organization")
			}

			_, domain, err := svc.entityDomain(ctx, orgID, "")
			if err != nil {
				return nil, err
			} else if domain == "" {
				return nil, errors.New("no domain picked for organization yet")
			} else if domain != subdomain {
				return &provisionEmailOutput{
					Result:           provisionEmailResultSubdomainInUse,
					ClientMutationID: mutationID,
				}, nil
			}

		case directory.EntityType_EXTERNAL:
			return nil, errors.New("cannot provision email for external entity")
		}

		// lets go ahead and provision the email for the entity specified
		_, err = svc.exComms.ProvisionEmailAddress(ctx, &excomms.ProvisionEmailAddressRequest{
			EmailAddress: emailAddress,
			ProvisionFor: entityID,
		})
		if grpc.Code(err) == codes.AlreadyExists {
			return &provisionEmailOutput{
				Result:           provisionEmailResultLocalPartInUse,
				ClientMutationID: mutationID,
			}, nil
		}

		// lets go ahead and create the provisioned email address as a contact for the entity
		createContactRes, err := svc.directory.CreateContact(ctx, &directory.CreateContactRequest{
			EntityID: entityID,
			Contact: &directory.Contact{
				ContactType: directory.ContactType_EMAIL,
				Provisioned: true,
				Value:       emailAddress,
			},
			RequestedInformation: &directory.RequestedInformation{
				Depth: 0,
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_MEMBERSHIPS,
					directory.EntityInformation_CONTACTS,
				},
			},
		})
		if err != nil {
			return nil, internalError(err)
		}

		e, err := transformEntityToResponse(createContactRes.Entity)
		if err != nil {
			return nil, internalError(err)
		}

		return &provisionEmailOutput{
			Result:           provisionEmailResultSuccess,
			ClientMutationID: mutationID,
			Entity:           e,
		}, nil
	},
}
