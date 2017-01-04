package main

import (
	"fmt"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	excommssettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/analytics"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type provisionPhoneNumberInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	OrganizationID   string `gql:"organizationID"`
	AreaCode         string `gql:"areaCode"`
}

type provisionPhoneNumberOutput struct {
	ClientMutationID string               `json:"clientMutationId,omitempty"`
	Success          bool                 `json:"success"`
	ErrorCode        string               `json:"errorCode,omitempty"`
	ErrorMessage     string               `json:"errorMessage,omitempty"`
	PhoneNumber      string               `json:"phoneNumber,omitempty"`
	Organization     *models.Organization `json:"organization,omitempty"`
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
	provisionPhoneNumberErrorCodeUnavailable = "UNAVAILABLE"
)

var provisionPhoneNumberErrorCodeEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "ProvisionPhoneNumberErrorCode",
		Description: "Result of provisionPhoneNumber mutation",
		Values: graphql.EnumValueConfigMap{
			provisionPhoneNumberErrorCodeUnavailable: &graphql.EnumValueConfig{
				Value:       provisionPhoneNumberErrorCodeUnavailable,
				Description: "No phone numbers found for area code",
			},
		},
	},
)

var provisionPhoneNumberOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ProvisionPhoneNumberPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: provisionPhoneNumberErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"phoneNumber":      &graphql.Field{Type: graphql.String},
			"organization":     &graphql.Field{Type: organizationType},
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
	Resolve: apiaccess.Provider(func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		svc := serviceFromParams(p)
		acc := gqlctx.Account(ctx)
		sh := devicectx.SpruceHeaders(ctx)

		var in provisionPhoneNumberInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(p.Context, err)
		}

		if in.OrganizationID == "" {
			return nil, errors.Errorf("organizationID required")
		} else if in.AreaCode == "" {
			return nil, errors.Errorf("areaCode required")
		}

		entity, err := entityInOrgForAccountID(ctx, ram, in.OrganizationID, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		} else if entity == nil {
			return nil, errors.Errorf("No entity found in organization %s", in.OrganizationID)
		}

		res, err := ram.ProvisionPhoneNumber(ctx, &excomms.ProvisionPhoneNumberRequest{
			ProvisionFor: in.OrganizationID,
			Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
				AreaCode: in.AreaCode,
			},
			// Use the UUID to guarantee that the organization can only provision one number through this mutation.
			UUID: in.OrganizationID + ":" + "primary",
		})
		if grpc.Code(err) == codes.InvalidArgument || grpc.Code(err) == codes.NotFound {
			return &provisionPhoneNumberOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        provisionPhoneNumberErrorCodeUnavailable,
				ErrorMessage:     "No phone number is available for the chosen area code. Please choose another.",
			}, nil
		} else if grpc.Code(err) == excomms.ErrorCodePhoneNumberDeprovisioned {
			// lets try again with different UUID
			res, err = ram.ProvisionPhoneNumber(ctx, &excomms.ProvisionPhoneNumberRequest{
				ProvisionFor: in.OrganizationID,
				Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
					AreaCode: in.AreaCode,
				},
				// Use the UUID to guarantee that the organization can only provision one number through this mutation.
				UUID: in.OrganizationID + ":" + "primary" + ":" + time.Now().String(),
			})
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		// lets go ahead and create a contact for the entity for which the number was provisioned
		createContactRes, err := ram.CreateContact(ctx, &directory.CreateContactRequest{
			EntityID: in.OrganizationID,
			Contact: &directory.Contact{
				ContactType: directory.ContactType_PHONE,
				Provisioned: true,
				Value:       res.PhoneNumber,
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
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		analytics.SegmentTrack(&segment.Track{
			Event:  "provisioned-phone",
			UserId: acc.ID,
			Properties: map[string]interface{}{
				"phone_number": res.PhoneNumber,
			},
		})

		// identify the phone number associated with the provider
		// provisioning the number for the organization.
		var phoneNumber string
		for _, c := range entity.Contacts {
			if !c.Provisioned && c.ContactType == directory.ContactType_PHONE {
				phoneNumber, err = phone.Format(c.Value, phone.Pretty)
				if err != nil {
					return nil, errors.InternalError(ctx, err)
				}
				break
			}
		}

		// lets go ahead and add the mobile number of the user to the forwarding list
		// so that there is a number in the forwarding list by default.
		_, err = svc.settings.SetValue(ctx, &settings.SetValueRequest{
			NodeID: in.OrganizationID,
			Value: &settings.Value{
				Key: &settings.ConfigKey{
					Key:    excommssettings.ConfigKeyForwardingList,
					Subkey: res.PhoneNumber,
				},
				Type: settings.ConfigType_STRING_LIST,
				Value: &settings.Value_StringList{
					StringList: &settings.StringListValue{
						Values: []string{
							phoneNumber,
						},
					},
				},
			},
		})
		if err != nil {
			golog.ContextLogger(ctx).Errorf("Unable to create forwarding list for the provisioned phone number: %s", err)
		}

		orgRes, err := transformOrganizationToResponse(ctx, svc.staticURLPrefix, createContactRes.Entity, entity, sh, acc)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		pn, err := phone.Format(res.PhoneNumber, phone.Pretty)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &provisionPhoneNumberOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			PhoneNumber:      pn,
			Organization:     orgRes,
		}, nil
	}),
}
