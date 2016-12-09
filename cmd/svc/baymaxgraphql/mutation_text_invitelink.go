package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type textInviteLinkInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	Token            string `gql:"token,nonempty"`
	PhoneNumber      string `gql:"phoneNumber"`
}

const textInviteLinkErrorCodeInvalidPhoneNumber = "INVALID_PHONE_NUMBER"
const textInviteLinkErrorCodeInvalidToken = "INVALID_TOKEN"

var textInviteLinkErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "textInviteLinkErrorCode",
	Values: graphql.EnumValueConfigMap{
		textInviteLinkErrorCodeInvalidPhoneNumber: &graphql.EnumValueConfig{
			Value:       textInviteLinkErrorCodeInvalidPhoneNumber,
			Description: "The provided phone number is invalid",
		},
		textInviteLinkErrorCodeInvalidToken: &graphql.EnumValueConfig{
			Value:       textInviteLinkErrorCodeInvalidToken,
			Description: "The provided invite code is invalid",
		},
	},
})

var textInviteLinkInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "TextInviteLinkInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"phoneNumber":      &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

type textInviteLinkOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var textInviteLinkOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "TextInviteLinkPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: textInviteLinkErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*textInviteLinkOutput)
			return ok
		},
	},
)

var textInviteLinkMutation = &graphql.Field{
	Description: "textInviteLink looks up an invite by token, and texts the invite link to the phone number specified or the user that the link is for.",
	Type:        graphql.NewNonNull(textInviteLinkOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(textInviteLinkInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context
		ram := raccess.ResourceAccess(p)

		var in textInviteLinkInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		res, err := svc.invite.LookupInvite(ctx, &invite.LookupInviteRequest{
			InviteToken: in.Token,
		})
		if grpc.Code(err) == codes.NotFound {
			return &textInviteLinkOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        textInviteLinkErrorCodeInvalidToken,
				ErrorMessage:     "Sorry, the invite code is not valid. Please re-enter the code or contact your healthcare provider.",
			}, nil
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		switch res.Invite.(type) {
		case *invite.LookupInviteResponse_Patient:

			// send invite to phone number in invite
			// not to the phone number entered

			if _, err := svc.invite.InvitePatients(ctx, &invite.InvitePatientsRequest{
				OrganizationEntityID: res.GetPatient().OrganizationEntityID,
				InviterEntityID:      res.GetPatient().InviterEntityID,
				Patients:             []*invite.Patient{res.GetPatient().Patient},
			}); err != nil {
				return nil, errors.InternalError(ctx, err)
			}

		case *invite.LookupInviteResponse_Organization:
			pn, err := phone.Format(in.PhoneNumber, phone.E164)
			if err != nil {
				return &textInviteLinkOutput{
					ClientMutationID: in.ClientMutationID,
					ErrorCode:        textInviteLinkErrorCodeInvalidPhoneNumber,
					ErrorMessage:     "Enter a valid phone number",
					Success:          false,
				}, nil
			}

			orgLink := invite.OrganizationInviteURL(svc.inviteAPIDomain, res.GetOrganization().Token)
			orgID := res.GetOrganization().OrganizationEntityID

			org, err := raccess.UnauthorizedEntity(ctx, ram, &directory.LookupEntitiesRequest{
				Key: &directory.LookupEntitiesRequest_EntityID{
					EntityID: orgID,
				},
			})
			if err != nil {
				return nil, errors.InternalError(ctx, fmt.Errorf("Error while looking up org %q for device association: %s", orgID, err))
			}

			if err := ram.SendMessage(ctx, &excomms.SendMessageRequest{
				DeprecatedChannel: excomms.ChannelType_SMS,
				Message: &excomms.SendMessageRequest_SMS{
					SMS: &excomms.SMSMessage{
						Text:            fmt.Sprintf("Download the Spruce app now and connect with %s: %s [code: %s]", org.Info.DisplayName, orgLink, res.GetOrganization().Token),
						FromPhoneNumber: svc.serviceNumber.String(),
						ToPhoneNumber:   pn,
					},
				},
			}); err != nil {
				golog.Warningf("Unable to send patient invite url to %s for token %s", pn, in.Token)
				return &textInviteLinkOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          true,
				}, nil
			}
		default:
			return &textInviteLinkOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        textInviteLinkErrorCodeInvalidToken,
				ErrorMessage:     "Sorry, the invite code is not valid. Please re-enter the code or contact your healthcare provider.",
			}, nil
		}

		return &textInviteLinkOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
		}, nil
	},
}
