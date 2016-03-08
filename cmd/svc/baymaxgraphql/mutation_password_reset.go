package main

import (
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/graphql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// requestPasswordReset

type requestPasswordResetOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var requestPasswordResetInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "RequestPasswordResetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"email": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Specify the email associated with the account that you would like to reset the password for",
			},
		},
	},
)

const (
	requestPasswordResetErrorCodeInvalidEmail = "INVALID_EMAIL"
)

var requestPasswordResetErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "RequestPasswordResetErrorCode",
	Values: graphql.EnumValueConfigMap{
		requestPasswordResetErrorCodeInvalidEmail: &graphql.EnumValueConfig{
			Value:       requestPasswordResetErrorCodeInvalidEmail,
			Description: "The provided email address is invalid",
		},
	},
})

var requestPasswordResetOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RequestPasswordResetPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: requestPasswordResetErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*requestPasswordResetOutput)
			return ok
		},
	},
)

var requestPasswordResetMutation = &graphql.Field{
	Type: graphql.NewNonNull(requestPasswordResetOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(requestPasswordResetInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		email, _ := input["email"].(string)

		if !validate.Email(email) {
			return &requestPasswordResetOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        requestPasswordResetErrorCodeInvalidEmail,
				ErrorMessage:     "The entered email address is invalid.",
			}, nil
		}

		conc.Go(func() {
			if err := createAndSendPasswordResetEmail(context.TODO(), ram, svc.webDomain, email); err != nil {
				golog.Errorf("Error while sending password reset email: %s", err)
			}
		})

		return &requestPasswordResetOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}

// checkPasswordResetToken

const (
	checkPasswordResetTokenErrorCodeFailure = "BAD_TOKEN"
	checkPasswordResetTokenErrorCodeExpired = "TOKEN_EXPIRED"
)

type checkPasswordResetTokenOutput struct {
	ClientMutationID          string `json:"clientMutationId,omitempty"`
	Success                   bool   `json:"success"`
	ErrorCode                 string `json:"errorCode,omitempty"`
	ErrorMessage              string `json:"errorMessage,omitempty"`
	PhoneNumberLastFourDigits string `json:"phoneNumberLastFourDigits"`
}

var checkPasswordResetTokenErrorCodeEnum = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CheckPasswordResetTokenErrorCode",
		Description: "Result of checkPasswordResetToken mutation",
		Values: graphql.EnumValueConfigMap{
			checkPasswordResetTokenErrorCodeFailure: &graphql.EnumValueConfig{
				Value:       checkPasswordResetTokenErrorCodeFailure,
				Description: "Code expired",
			},
			checkPasswordResetTokenErrorCodeExpired: &graphql.EnumValueConfig{
				Value:       checkPasswordResetTokenErrorCodeExpired,
				Description: "Code verifcation failed",
			},
		},
	},
)

var checkPasswordResetTokenInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "CheckPasswordResetTokenInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var checkPasswordResetTokenOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CheckPasswordResetTokenPayload",
		Fields: graphql.Fields{
			"clientMutationId":          newClientmutationIDOutputField(),
			"success":                   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":                 &graphql.Field{Type: checkPasswordResetTokenErrorCodeEnum},
			"errorMessage":              &graphql.Field{Type: graphql.String},
			"phoneNumberLastFourDigits": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*checkPasswordResetTokenOutput)
			return ok
		},
	},
)

var checkPasswordResetTokenMutation = &graphql.Field{
	Type: graphql.NewNonNull(checkPasswordResetTokenOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(checkPasswordResetTokenInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)

		resp, err := ram.CheckPasswordResetToken(ctx, token)
		if grpc.Code(err) == auth.TokenExpired {
			return &checkPasswordResetTokenOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        checkPasswordResetTokenErrorCodeExpired,
				ErrorMessage:     "Your reset link has expired. Please request a new password reset email.",
			}, nil
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		last4Phone := resp.AccountPhoneNumber
		if len(last4Phone) > 4 {
			last4Phone = last4Phone[len(last4Phone)-4:]
		}
		return &checkPasswordResetTokenOutput{
			ClientMutationID:          mutationID,
			Success:                   true,
			PhoneNumberLastFourDigits: last4Phone,
		}, nil
	},
}

// createAndSendPasswordResetEmail creates a token for the password reset link and embeds it in a link and sends it to the account's provided email
func createAndSendPasswordResetEmail(ctx context.Context, ram raccess.ResourceAccessor, webDomain string, email string) error {
	resp, err := ram.CreatePasswordResetToken(ctx, email)
	if grpc.Code(err) == codes.NotFound {
		golog.Warningf("PasswordReset: Unable to find account for email %s", email)
		return nil
	} else if err != nil {
		return errors.Trace(err)
	}

	body := fmt.Sprintf("Your password reset link is: %s", passwordResetURL(webDomain, resp.Token))
	golog.Debugf("Sending password reset email %q to %s", body, email)
	if err := ram.SendMessage(ctx, &excomms.SendMessageRequest{
		Channel: excomms.ChannelType_EMAIL,
		Message: &excomms.SendMessageRequest_Email{
			Email: &excomms.EmailMessage{
				Subject:          "Password Reset",
				FromName:         "Spruce Support",
				FromEmailAddress: "support@sprucehealth.com",
				Body:             body,
				ToEmailAddress:   email,
			},
		},
	}); err != nil {
		golog.Errorf("Error while sending password reset email to %s: %s", email, err)
	}
	return nil
}

// verifyPhoneNumberForPasswordReset
var verifyPhoneNumberForPasswordResetInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "VerifyPhoneNumberForPasswordResetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             newUUIDInputField(),
			"linkToken": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The token contained in the password reset link",
			},
		},
	},
)

var verifyPhoneNumberForPasswordResetMutation = &graphql.Field{
	Type: graphql.NewNonNull(verifyPhoneNumberOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(verifyPhoneNumberForPasswordResetInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		linkToken, _ := input["linkToken"].(string)

		resp, err := ram.CheckPasswordResetToken(ctx, linkToken)
		if grpc.Code(err) == auth.TokenExpired {
			return nil, errors.UserError(ctx, errors.ErrTypeExpired, "Your phone verification token has expired. Please go back and enter your number again.")
		} else if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		accountPhoneNumber, err := phone.ParseNumber(resp.AccountPhoneNumber)
		if err != nil {
			// Shouldn't fail
			return nil, errors.InternalError(ctx, err)
		}
		token, err := createAndSendSMSVerificationCode(ctx, ram, svc.serviceNumber, auth.VerificationCodeType_PASSWORD_RESET, resp.AccountID, accountPhoneNumber)
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}

		last4Phone := resp.AccountPhoneNumber
		if len(last4Phone) > 4 {
			last4Phone = last4Phone[len(last4Phone)-4:]
		}
		return &verifyPhoneNumberOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to the phone number ending in %s", last4Phone),
		}, nil
	},
}

// passwordReset

type passwordResetOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	Message          string `json:"message"`
}

var passwordResetInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "PasswordResetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.ID},
			"token": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The token associated with the password reset phone verification request",
			},
			"code": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The code associated with the password reset phone verification request",
			},
			"newPassword": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The new password to map to the account",
			},
		},
	},
)

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var passwordResetErrorCodeEnum = graphql.String

var passwordResetOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PasswordResetPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: passwordResetErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"message":          &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*passwordResetOutput)
			return ok
		},
	},
)

var passwordResetMutation = &graphql.Field{
	Type: graphql.NewNonNull(passwordResetOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(passwordResetInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token, _ := input["token"].(string)
		code, _ := input["code"].(string)
		newPassword, _ := input["newPassword"].(string)

		if err := ram.UpdatePassword(ctx, token, code, newPassword); err != nil {
			switch grpc.Code(err) {
			case codes.NotFound:
				return nil, errors.New("Token not found")
			case auth.VerificationCodeExpired:
				return nil, errors.New("Code has expired")
			case auth.BadVerificationCode:
				return nil, errors.New("Bad verification code")
			default:
				return nil, errors.InternalError(ctx, err)
			}
		}

		return &passwordResetOutput{
			ClientMutationID: mutationID,
			Success:          true,
			Message:          "Password updated",
		}, nil
	},
}

func passwordResetURL(webDomain, passwordResetToken string) string {
	return fmt.Sprintf("https://%s/account/password-reset/%s", webDomain, passwordResetToken)
}
