package main

import (
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type provisionEmailInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	EntityID         string `gql:"entityID"`
	OrganizationID   string `gql:"organizationID"`
	LocalPart        string `gql:"localPart"`
	Subdomain        string `gql:"subdomain"`
}

type provisionEmailOutput struct {
	ClientMutationID string               `json:"clientMutationId,omitempty"`
	Success          bool                 `json:"success"`
	ErrorCode        string               `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	Organization     *models.Organization `json:"organization"`
	Entity           *models.Entity       `json:"entity"`
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
	provisionEmailErrorCodeSubdomainInUse  = "SUBDOMAIN_IN_USE"
	provisionEmailErrorCodeLocalPartInUse  = "LOCAL_PART_IN_USE"
	provisionEmailErrorInvalidEmailAddress = "INVALID_EMAIL"
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
			provisionEmailErrorInvalidEmailAddress: &graphql.EnumValueConfig{
				Value:       provisionEmailErrorInvalidEmailAddress,
				Description: "Invalid email address",
			},
		},
	},
)

var provisionEmailOutputType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ProvisionEmailPayload",
	Fields: graphql.Fields{
		"clientMutationId": newClientMutationIDOutputField(),
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
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		acc := gqlctx.Account(ctx)
		sh := devicectx.SpruceHeaders(ctx)

		var in provisionEmailInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		emailAddress := in.LocalPart + "@" + in.Subdomain + "." + svc.emailDomain

		var ent *directory.Entity
		var orgEntity *directory.Entity
		var err error
		if in.EntityID != "" {
			ent, err = raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				Key: &directory.LookupEntitiesRequest_EntityID{
					EntityID: in.EntityID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth:             0,
					EntityInformation: []directory.EntityInformation{directory.EntityInformation_MEMBERSHIPS, directory.EntityInformation_CONTACTS, directory.EntityInformation_EXTERNAL_IDS},
				},
				Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				RootTypes: []directory.EntityType{directory.EntityType_INTERNAL, directory.EntityType_ORGANIZATION},
			})
			if err != nil {
				return nil, err
			} else if ent.Type != directory.EntityType_INTERNAL {
				return nil, errors.Errorf("email can only be provisioned for a provider")
			}
			for _, m := range ent.Memberships {
				if m.Type == directory.EntityType_ORGANIZATION {
					orgEntity = m
					break
				}
			}
			if orgEntity == nil {
				return nil, errors.Errorf("entity does not belong to an org")
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
				return nil, errors.Errorf("entity %s does not belong to account", in.EntityID)
			}

		} else if in.OrganizationID != "" {
			orgEntity, err = raccess.Entity(ctx, ram, &directory.LookupEntitiesRequest{
				Key: &directory.LookupEntitiesRequest_EntityID{
					EntityID: in.OrganizationID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth:             0,
					EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
				},
				Statuses:  []directory.EntityStatus{directory.EntityStatus_ACTIVE},
				RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}

			ent, err = entityInOrgForAccountID(ctx, ram, in.OrganizationID, acc)
			if err != nil {
				return nil, err
			}
		}

		if !validate.Email(emailAddress) {
			return &provisionEmailOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        provisionEmailErrorInvalidEmailAddress,
				ErrorMessage:     "Please enter a valid email address",
			}, nil
		}

		if in.OrganizationID != "" {
			res, err := ram.EntityDomain(ctx, in.OrganizationID, "")
			if grpc.Code(err) == codes.NotFound {
				if err := ram.CreateEntityDomain(ctx, in.OrganizationID, in.Subdomain); err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			} else if res.Domain != "" && res.Domain != in.Subdomain {
				return &provisionEmailOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeSubdomainInUse,
					ErrorMessage:     "The entered subdomain is already in use. Please pick another.",
				}, nil
			}
		} else {
			if ent.Type != directory.EntityType_INTERNAL {
				return nil, errors.Errorf("Cannot provision email for external entity")
			}

			res, err := ram.EntityDomain(ctx, orgEntity.ID, "")
			if grpc.Code(err) == codes.NotFound || (err == nil && res.Domain == "") {
				return nil, errors.New("no domain picked for organization yet")
			} else if err != nil {
				return nil, err
			}

			if res.Domain != in.Subdomain {
				return &provisionEmailOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeSubdomainInUse,
					ErrorMessage:     "The entered subdomain is already in use. Please pick another.",
				}, nil
			}
		}

		provisionFor := in.EntityID
		if in.OrganizationID != "" {
			provisionFor = in.OrganizationID
		}

		// lets go ahead and provision the email for the entity specified
		_, err = ram.ProvisionEmailAddress(ctx, &excomms.ProvisionEmailAddressRequest{
			EmailAddress: emailAddress,
			ProvisionFor: provisionFor,
		})
		if err != nil {
			if grpc.Code(err) == codes.AlreadyExists {
				return &provisionEmailOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        provisionEmailErrorCodeLocalPartInUse,
					ErrorMessage:     "The entered email is already in use. Please pick another.",
				}, nil
			}
			return nil, errors.InternalError(ctx, err)
		}

		// lets go ahead and create the provisioned email address as a contact for the entity
		createContactRes, err := ram.CreateContact(ctx, &directory.CreateContactRequest{
			EntityID: provisionFor,
			Contact: &directory.Contact{
				ContactType: directory.ContactType_EMAIL,
				Provisioned: true,
				Value:       emailAddress,
				Verified:    true,
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
			return nil, errors.InternalError(ctx, err)
		}

		analytics.SegmentTrack(&segment.Track{
			Event:  "provisioned-email",
			UserId: acc.ID,
			Properties: map[string]interface{}{
				"email": emailAddress,
			},
		})

		var e *models.Entity
		var o *models.Organization
		if in.OrganizationID != "" {
			o, err = transformOrganizationToResponse(ctx, svc.staticURLPrefix, createContactRes.Entity, ent, sh, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		} else {
			sh := devicectx.SpruceHeaders(ctx)
			e, err = transformEntityToResponse(ctx, svc.staticURLPrefix, createContactRes.Entity, sh, acc)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		}

		return &provisionEmailOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Entity:           e,
			Organization:     o,
		}, nil
	}),
}
