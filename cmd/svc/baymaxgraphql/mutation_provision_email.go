package main

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// provision email

type provisionEmailOutput struct {
	ClientMutationID string        `json:"clientMutationId,omitempty"`
	Success          bool          `json:"success"`
	ErrorCode        string        `json:"errorCode,omitempty"`
	ErrorMessage     string        `json:"errorMessage,omitempty"`
	Organization     *organization `json:"organization"`
	Entity           *entity       `json:"entity"`
}

var provisionEmailInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ProvisionEmailInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"entityID": &graphql.InputObjectFieldConfig{
				Type:        graphql.ID,
				Description: "ID of the organization for which the email is being provisioned."},
			"organizationID": &graphql.InputObjectFieldConfig{
				Type:        graphql.ID,
				Description: "ID of the organization for which the email is being provisioned."},
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
	provisionEmailErrorCodeSubdomainInUse = "SUBDOMAIN_IN_USE"
	provisionEmailErrorCodeLocalPartInUse = "LOCAL_PART_IN_USE"
)

var provisionEmailErrorCodeEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ProvisionEmailErrorCode",
		Description: "Result of provisionEmail mutation",
		Values: graphql.EnumValueConfigMap{
			provisionEmailErrorCodeSubdomainInUse: &graphql.EnumValueConfig{
				Value:       provisionEmailErrorCodeSubdomainInUse,
				Description: "Subdomain not found",
			},
			provisionEmailErrorCodeLocalPartInUse: &graphql.EnumValueConfig{
				Value:       provisionEmailErrorCodeLocalPartInUse,
				Description: "Local part of the address is in use",
			},
		},
	},
)

var provisionEmailOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ProvisionEmailPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientmutationIDOutputField(),
		"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errorCode":        &graphql.Field{Type: provisionEmailErrorCodeEnum},
		"errorMessage":     &graphql.Field{Type: graphql.String},
		"entity":           &graphql.Field{Type: entityType},
		"organization":     &graphql.Field{Type: organizationType},
	},
	IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
		_, ok := value.(*provisionEmailOutput)
		return ok
	},
})

var provisionEmailMutation = &graphql.Field{
	Type: graphql.NewNonNull(provisionEmailOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(provisionEmailInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated(ctx)
		}

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		localPart, _ := input["localPart"].(string)
		subdomain, _ := input["subdomain"].(string)
		entityID, _ := input["entityID"].(string)
		organizationID, _ := input["organizationID"].(string)
		emailAddress := localPart + "@" + subdomain + "." + svc.emailDomain

		var ent *directory.Entity
		var orgEntity *directory.Entity
		var err error
		if entityID != "" {
			ent, err = svc.entity(ctx, entityID,
				directory.EntityInformation_CONTACTS,
				directory.EntityInformation_MEMBERSHIPS,
				directory.EntityInformation_EXTERNAL_IDS)
			if err != nil {
				return nil, internalError(ctx, err)
			} else if ent.Type != directory.EntityType_INTERNAL {
				return nil, fmt.Errorf("email can only be provisioned for a provider")
			}
			for _, m := range ent.Memberships {
				if m.Type == directory.EntityType_ORGANIZATION {
					orgEntity = m
					break
				}
			}
			if orgEntity == nil {
				return nil, fmt.Errorf("entity does not belong to an org")
			}

			// ensure that accountID is one of the external IDs for the entity
			entityBelongsToAccount := false
			for _, eID := range ent.ExternalIDs {
				if eID == acc.ID {
					entityBelongsToAccount = true
					break
				}
			}
			if !entityBelongsToAccount {
				return nil, fmt.Errorf("entity %s does not belong to account", entityID)
			}

		} else if organizationID != "" {
			orgEntity, err = svc.entity(ctx, organizationID)
			if err != nil {
				return nil, internalError(ctx, err)
			}
			ent, err = svc.entityForAccountID(ctx, organizationID, acc.ID)
			if err != nil {
				return nil, internalError(ctx, err)
			} else if ent == nil {
				return nil, fmt.Errorf("current user does not belong to the organization %s", organizationID)
			}
		}

		if !validate.Email(emailAddress) {
			return nil, errors.New("invalid email address")
		}

		if organizationID != "" {
			_, domain, err := svc.entityDomain(ctx, organizationID, "")
			if err != nil {
				return nil, err
			} else if domain != "" && domain != subdomain {
				return &provisionEmailOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeSubdomainInUse,
					ErrorMessage:     "The entered subdomain is already in use. Please pick another.",
				}, nil
			}

			// lets go ahead and create domain for organization
			if domain == "" {
				_, err := svc.directory.CreateEntityDomain(ctx, &directory.CreateEntityDomainRequest{
					EntityID: organizationID,
					Domain:   subdomain,
				})
				if err != nil {
					return nil, internalError(ctx, err)
				}
			}
		} else {
			if ent.Type != directory.EntityType_INTERNAL {
				return nil, fmt.Errorf("Cannot provision email for external entity")
			}

			_, domain, err := svc.entityDomain(ctx, orgEntity.ID, "")
			if err != nil {
				return nil, err
			} else if domain == "" {
				return nil, errors.New("no domain picked for organization yet")
			} else if domain != subdomain {
				return &provisionEmailOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeSubdomainInUse,
					ErrorMessage:     "The entered subdomain is already in use. Please pick another.",
				}, nil
			}
		}

		provisionFor := entityID
		if organizationID != "" {
			provisionFor = organizationID
		}

		// lets go ahead and provision the email for the entity specified
		_, err = svc.exComms.ProvisionEmailAddress(ctx, &excomms.ProvisionEmailAddressRequest{
			EmailAddress: emailAddress,
			ProvisionFor: provisionFor,
		})
		if err != nil {
			if grpc.Code(err) == codes.AlreadyExists {
				return &provisionEmailOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeLocalPartInUse,
					ErrorMessage:     "The entered email is already in use. Please pick another.",
				}, nil
			}
			return nil, internalError(ctx, err)
		}

		// lets go ahead and create the provisioned email address as a contact for the entity
		createContactRes, err := svc.directory.CreateContact(ctx, &directory.CreateContactRequest{
			EntityID: provisionFor,
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
			return nil, internalError(ctx, err)
		}

		var e *entity
		var o *organization
		if organizationID != "" {
			o, err = transformOrganizationToResponse(createContactRes.Entity, ent)
			if err != nil {
				return nil, internalError(ctx, err)
			}
		} else {
			e, err = transformEntityToResponse(createContactRes.Entity)
			if err != nil {
				return nil, internalError(ctx, err)
			}
		}

		return &provisionEmailOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Entity:           e,
			Organization:     o,
		}, nil
	},
}
