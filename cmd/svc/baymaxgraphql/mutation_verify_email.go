package main

import (
	"context"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
)

// verifyEmail

const (
	verifyEmailErrorCodeInvalidEmail   = "INVALID_EMAIL"
	verifyEmailErrorCodeInviteRequired = "INVITE_REQUIRED"
)

var verifyEmailErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "VerifyEmailErrorCode",
	Description: "Result of verifyEmail mutation",
	Values: graphql.EnumValueConfigMap{
		verifyEmailErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       verifyEmailErrorCodeInvalidEmail,
			Description: "The provided email is invalid",
		},
		verifyEmailErrorCodeInviteRequired: &graphql.EnumValueConfig{
			Value:       verifyEmailErrorCodeInviteRequired,
			Description: "An invite is required to perform email verification with this device",
		},
	},
})

type verifyEmailOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	Token            string `json:"token"`
	Message          string `json:"message"`
}

type verifyEmailInput struct {
	ClientMutationID      string `gql:"clientMutationId"`
	Email                 string `gql:"email"`
	AccountInviteClientID string `gql:"accountInviteClientID"`
}

var verifyEmailInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "VerifyEmailInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId":      newClientMutationIDInputField(),
			"email":                 &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"accountInviteClientID": accountInviteClientInputType,
		},
	},
)

var verifyEmailForAccountCreationInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "VerifyEmailForAccountCreationInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
		},
	},
)

var verifyEmailOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "VerifyEmailPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: verifyEmailErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"token":            &graphql.Field{Type: graphql.String},
			"message":          &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*verifyEmailOutput)
			return ok
		},
	},
)

var verifyEmailMutation = &graphql.Field{
	Type: graphql.NewNonNull(verifyEmailOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(verifyEmailInputType)},
	},
	Resolve: makeVerifyEmailResolve(false),
}

var verifyEmailForAccountCreationMutation = &graphql.Field{
	Type: graphql.NewNonNull(verifyEmailOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(verifyEmailForAccountCreationInputType)},
	},
	Resolve: makeVerifyEmailResolve(true),
}

func makeVerifyEmailResolve(forAccountCreation bool) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		var in verifyEmailInput
		if err := gqldecode.Decode(input, &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		var email string

		// If here for account creation then we require an invite to be mapped to the device
		if forAccountCreation {
			inv, _, err := svc.inviteAndAttributionInfo(ctx, in.AccountInviteClientID)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			if inv == nil {
				return &verifyEmailOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        verifyEmailErrorCodeInviteRequired,
					ErrorMessage:     "An invite is required to perform email verification with this device.",
				}, nil
			}
			var invEmail string
			switch inv.Invite.(type) {
			case *invite.LookupInviteResponse_Patient:
				// Since we don't store PHI for patients in the invites, get the email to verify from the parked entity contacts
				invEmail, err = contactForParkedEntity(ctx, ram, inv.GetPatient().Patient.ParkedEntityID, directory.ContactType_EMAIL)
				if err != nil {
					return nil, errors.InternalError(ctx, fmt.Errorf("Encountered error whil getting parked email for verification: %s", err))
				}
			case *invite.LookupInviteResponse_Colleague:
				invEmail = inv.GetColleague().Colleague.Email
			default:
				golog.Errorf("Unknown invite type %s", inv.Type.String())
			}
			email = invEmail
		} else {
			email = in.Email
			if !validate.Email(email) {
				return &verifyEmailOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        verifyEmailErrorCodeInvalidEmail,
					ErrorMessage:     "The provided email is invalid.",
				}, nil
			}
		}

		token, err := createAndSendVerificationEmail(ctx, ram, svc.emailTemplateIDs.emailVerification, email, svc.transactionalEmailSender)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		return &verifyEmailOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Token:            token,
			Message:          "A verification code has been sent to the invited email.",
		}, nil
	}
}

// createAndSendVerificationEmail creates and sends a verification code email
func createAndSendVerificationEmail(ctx context.Context, ram raccess.ResourceAccessor, templateID, email, transactionalEmailSender string) (string, error) {
	resp, err := ram.CreateVerificationCode(ctx, auth.VerificationCodeType_EMAIL, email)
	if err != nil {
		return "", errors.Trace(err)
	}

	body := fmt.Sprintf("During sign up, please enter this code when prompted: %s\nIf you have any troubles, we're here to help - simply reply to this email!\n\nThanks,\nThe Team at Spruce", resp.VerificationCode.Code)
	golog.Debugf("Sending email verification %q to %s", body, email)
	if err := ram.SendMessage(ctx, &excomms.SendMessageRequest{
		DeprecatedChannel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Your Email Verification Code",
				FromName:         "Spruce Support",
				FromEmailAddress: transactionalEmailSender,
				Transactional:    true,
				Body:             body,
				ToEmailAddress:   email,
				TemplateID:       templateID,
				TemplateSubstitutions: []*excomms.EmailMessage_Substitution{
					{Key: "{verification_code}", Value: resp.VerificationCode.Code},
				},
			},
		},
	}); err != nil {
		golog.Errorf("Error while sending verification email to %s: %s", email, err)
	}
	return resp.VerificationCode.Token, nil
}
