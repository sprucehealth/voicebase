package main

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"

	"github.com/graphql-go/graphql"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/excomms"
)

// requestPasswordReset

type requestPasswordResetOutput struct {
	ClientMutationID string `json:"clientMutationId"`
}

var requestPasswordResetInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "RequestPasswordResetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"email": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Specify the email associated with the account that you would like to reset the password for",
			},
		},
	},
)

var requestPasswordResetOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "RequestPasswordResetPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*requestPasswordResetOutput)
			return ok
		},
	},
)

var requestPasswordResetField = &graphql.Field{
	Type: graphql.NewNonNull(requestPasswordResetOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: requestPasswordResetInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		email, _ := input["email"].(string)

		if !validate.Email(email) {
			return nil, errors.New("invalid email")
		}

		conc.Go(func() {
			if err := svc.createAndSendPasswordResetEmail(ctx, email); err != nil {
				golog.Errorf("Error while sending password reset email: %s", err)
			}
		})

		return &requestPasswordResetOutput{
			ClientMutationID: mutationID,
		}, nil
	},
}

// checkPasswordResetToken

const (
	checkPasswordResetTokenResultSuccess = "SUCCESS"
	checkPasswordResetTokenResultFailure = "BAD_TOKEN"
	checkPasswordResetTokenResultExpired = "TOKEN_EXPIRED"
)

type checkPasswordResetTokenOutput struct {
	ClientMutationID          string `json:"clientMutationId"`
	Result                    string `json:"result"`
	PhoneNumberLastFourDigits string `json:"phone_number_last_four_digits"`
}

var checkPasswordResetTokenResultType = graphql.NewEnum(
	graphql.EnumConfig{
		Name:        "CheckVerificationCodeResult",
		Description: "Result of checkPasswordResetToken mutation",
		Values: graphql.EnumValueConfigMap{
			checkPasswordResetTokenResultSuccess: &graphql.EnumValueConfig{
				Value:       checkPasswordResetTokenResultSuccess,
				Description: "Success",
			},
			checkPasswordResetTokenResultExpired: &graphql.EnumValueConfig{
				Value:       checkPasswordResetTokenResultExpired,
				Description: "Code expired",
			},
			checkPasswordResetTokenResultFailure: &graphql.EnumValueConfig{
				Value:       checkPasswordResetTokenResultFailure,
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
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var checkPasswordResetTokenOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "CheckPasswordResetTokenPayload",
		Fields: graphql.Fields{
			"clientMutationId":          newClientmutationIDOutputField(),
			"result":                    &graphql.Field{Type: graphql.NewNonNull(checkVerificationCodeResultType)},
			"phoneNumberLastFourDigits": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*checkPasswordResetTokenOutput)
			return ok
		},
	},
)

var checkPasswordResetTokenField = &graphql.Field{
	Type: graphql.NewNonNull(checkPasswordResetTokenOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: checkPasswordResetTokenInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token, _ := input["token"].(string)

		resp, err := svc.auth.CheckPasswordResetToken(ctx, &auth.CheckPasswordResetTokenRequest{
			Token: token,
		})
		if grpc.Code(err) == auth.TokenExpired {
			return nil, errors.New("The provided token has expired")
		} else if err != nil {
			return nil, internalError(err)
		}

		last4Phone := resp.AccountPhoneNumber
		if len(last4Phone) > 4 {
			last4Phone = last4Phone[len(last4Phone)-4:]
		}
		return &checkPasswordResetTokenOutput{
			ClientMutationID:          mutationID,
			PhoneNumberLastFourDigits: last4Phone,
		}, nil
	},
}

// createAndSendPasswordResetEmail creates a token for the password reset link and embeds it in a link and sends it to the account's provided email
func (s *service) createAndSendPasswordResetEmail(ctx context.Context, email string) error {
	resp, err := s.auth.CreatePasswordResetToken(ctx, &auth.CreatePasswordResetTokenRequest{
		Email: email,
	})
	if err != nil {
		return errors.Trace(err)
	}

	body := fmt.Sprintf("Your password reset link is: %s", passwordResetURL(resp.Token))
	golog.Debugf("Sending password reset email %q to %s", body, email)
	if _, err := s.exComms.SendMessage(context.TODO(), &excomms.SendMessageRequest{
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
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
			"linkToken": &graphql.InputObjectFieldConfig{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The token contained in the password reset link",
			},
		},
	},
)

var verifyPhoneNumberForPasswordResetField = &graphql.Field{
	Type: graphql.NewNonNull(verifyPhoneNumberOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: verifyPhoneNumberForPasswordResetInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		linkToken, _ := input["linkToken"].(string)

		resp, err := svc.auth.CheckPasswordResetToken(ctx, &auth.CheckPasswordResetTokenRequest{
			Token: linkToken,
		})
		if grpc.Code(err) == auth.TokenExpired {
			return nil, errors.New("The provided token has expired")
		} else if err != nil {
			return nil, internalError(err)
		}

		token, err := svc.createAndSendSMSVerificationCode(ctx, auth.VerificationCodeType_PASSWORD_RESET, resp.AccountID, resp.AccountPhoneNumber)
		if err != nil {
			return nil, internalError(err)
		}

		last4Phone := resp.AccountPhoneNumber
		if len(last4Phone) > 4 {
			last4Phone = last4Phone[len(last4Phone)-4:]
		}
		return &verifyPhoneNumberOutput{
			ClientMutationID: mutationID,
			Token:            token,
			Message:          fmt.Sprintf("A verification code has been sent to the phone number ending in %s", last4Phone),
		}, nil
	},
}

// passwordReset

type passwordResetOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Message          string `json:"message"`
}

var passwordResetInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "PasswordResetInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"uuid":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
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

var passwordResetOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PasswordResetPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"message":          &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*passwordResetOutput)
			return ok
		},
	},
)

var passwordResetField = &graphql.Field{
	Type: graphql.NewNonNull(passwordResetOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: passwordResetInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ctx := p.Context

		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token, _ := input["token"].(string)
		code, _ := input["code"].(string)
		newPassword, _ := input["newPassword"].(string)

		_, err := svc.auth.UpdatePassword(ctx, &auth.UpdatePasswordRequest{
			Token:       token,
			Code:        code,
			NewPassword: newPassword,
		})
		if err != nil {
			switch grpc.Code(err) {
			case codes.NotFound:
				return nil, errors.New("Token not found")
			case auth.VerificationCodeExpired:
				return nil, errors.New("Code has expired")
			case auth.BadVerificationCode:
				return nil, errors.New("Bad verification code")
			default:
				return nil, internalError(err)
			}
		}

		return &passwordResetOutput{
			ClientMutationID: mutationID,
			Message:          "Password updated",
		}, nil
	},
}

func passwordResetURL(passwordResetToken string) string {
	return fmt.Sprintf("https://baymax.com/account/passwordReset/%s", passwordResetToken)
}
