package main

import (
	"fmt"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type provisionPhoneNumberOutput struct {
	ClientMutationID string        `json:"clientMutationId,omitempty"`
	PhoneNumber      string        `json:"phoneNumber,omitempty"`
	Organization     *organization `json:"organization,omitempty"`
	Result           string        `json:"result"`
}

var provisionPhoneNumberInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ProvisionPhoneNumberInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"organizationID": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "OrganizationID of the organization for which we are provisioning a phone number",
			},
			"areaCode": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Area code in which to provision a particular phone number",
			},
		},
	},
)

const (
	provisionPhoneNumberResultSuccess     = "SUCCESS"
	provisionPhoneNumberResultUnavailable = "UNAVAILABLE"
)

var provisionPhoneNumberResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ProvisionPhoneNumberResult",
		Description: "Result of provisionPhoneNumber mutation",
		Values: graphql.EnumValueConfigMap{
			provisionPhoneNumberResultSuccess: &graphql.EnumValueConfig{
				Value:       provisionPhoneNumberResultSuccess,
				Description: "Success",
			},
			provisionPhoneNumberResultUnavailable: &graphql.EnumValueConfig{
				Value:       provisionPhoneNumberResultUnavailable,
				Description: "No phone numbers found for area code",
			},
		},
	},
)

var provisionPhoneNumberOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProvisionPhoneNumberPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"phoneNumber":      &graphql.Field{Type: graphql.String},
			"organization":     &graphql.Field{Type: organizationType},
			"result":           &graphql.Field{Type: graphql.NewNonNull(provisionPhoneNumberResultType)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*provisionPhoneNumberOutput)
			return ok
		},
	},
)

var provisionPhoneNumberMutation = &graphql.Field{
	Type: graphql.NewNonNull(provisionPhoneNumberOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(provisionPhoneNumberInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		acc := accountFromContext(ctx)
		if acc == nil {
			return nil, errNotAuthenticated
		}

		input := p.Args["input"].(map[string]interface{})
		organizationID, _ := input["organizationID"].(string)
		mutationID, _ := input["clientMutationId"].(string)
		areaCode, _ := input["areaCode"].(string)

		if organizationID == "" {
			return nil, fmt.Errorf("organizationID required")
		} else if areaCode == "" {
			return nil, fmt.Errorf("areaCode required")
		}

		entity, err := svc.entityForAccountID(ctx, organizationID, acc.ID)
		if err != nil {
			return nil, internalError(err)
		} else if entity == nil {
			return nil, fmt.Errorf("No entity found in organization %s", organizationID)
		}

		res, err := svc.exComms.ProvisionPhoneNumber(ctx, &excomms.ProvisionPhoneNumberRequest{
			ProvisionFor: organizationID,
			Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
				AreaCode: areaCode,
			},
		})
		if grpc.Code(err) == codes.InvalidArgument || grpc.Code(err) == codes.NotFound {
			return &provisionPhoneNumberOutput{
				ClientMutationID: mutationID,
				Result:           provisionPhoneNumberResultUnavailable,
			}, nil
		} else if err != nil {
			return nil, internalError(err)
		}

		// lets go ahead and create a contact for the entity for which the number was provisioned
		createContactRes, err := svc.directory.CreateContact(ctx, &directory.CreateContactRequest{
			EntityID: organizationID,
			Contact: &directory.Contact{
				ContactType: directory.ContactType_PHONE,
				Provisioned: true,
				Value:       res.PhoneNumber,
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

		orgRes, err := transformOrganizationToResponse(createContactRes.Entity, entity)
		if err != nil {
			return nil, internalError(err)
		}

		return &provisionPhoneNumberOutput{
			PhoneNumber:      res.PhoneNumber,
			Organization:     orgRes,
			Result:           provisionPhoneNumberResultSuccess,
			ClientMutationID: mutationID,
		}, nil
	},
}
