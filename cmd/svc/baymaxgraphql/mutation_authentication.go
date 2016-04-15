package main

import (
	"time"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/graphql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// authenticate

const (
	authenticateErrorCodeTwoFactorRequired = "TWO_FACTOR_REQUIRED"
	authenticateErrorCodeAccountNotFound   = "ACCOUNT_NOT_FOUND"
	authenticateErrorCodePasswordMismatch  = "PASSWORD_MISMATCH"
	authenticateErrorCodeInvalidCode       = "INVALID_CODE"
)

type authenticateOutput struct {
	ClientMutationID      string         `json:"clientMutationId,omitempty"`
	Success               bool           `json:"success"`
	ErrorCode             string         `json:"errorCode,omitempty"`
	ErrorMessage          string         `json:"errorMessage,omitempty"`
	Token                 string         `json:"token,omitempty"`
	Account               models.Account `json:"account,omitempty"`
	PhoneNumberLastDigits string         `json:"phoneNumberLastDigits,omitempty"`
	ClientEncryptionKey   string         `json:"clientEncryptionKey,omitempty"`
}

var authenticateErrorCodeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "AuthenticateErrorCode",
	Description: "Result of authenticate mutation",
	Values: graphql.EnumValueConfigMap{
		authenticateErrorCodeTwoFactorRequired: &graphql.EnumValueConfig{
			Value:       authenticateErrorCodeTwoFactorRequired,
			Description: "2FA is required",
		},
		authenticateErrorCodeAccountNotFound: &graphql.EnumValueConfig{
			Value:       authenticateErrorCodeAccountNotFound,
			Description: "Account with email not found",
		},
		authenticateErrorCodePasswordMismatch: &graphql.EnumValueConfig{
			Value:       authenticateErrorCodePasswordMismatch,
			Description: "Password doesn't match",
		},
		authenticateErrorCodeInvalidCode: &graphql.EnumValueConfig{
			Value:       authenticateErrorCodeInvalidCode,
			Description: "Code doesn't match",
		},
	},
})

var authenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var authenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AuthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: authenticateErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
			"token":            &graphql.Field{Type: graphql.String},
			"account":          &graphql.Field{Type: accountInterfaceType},
			"phoneNumberLastDigits": &graphql.Field{
				Type:        graphql.String,
				Description: "Last couple digits of phone number used to send 2FA verification code. Only when errorCode=TWO_FACTOR_REQUIRED.",
			},
			"clientEncryptionKey": &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*authenticateOutput)
			return ok
		},
	},
)

var authenticateMutation = &graphql.Field{
	Type: graphql.NewNonNull(authenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		email := input["email"].(string)
		if !validate.Email(email) {
			return &authenticateOutput{
				ClientMutationID: mutationID,
				Success:          false,
				ErrorCode:        authenticateErrorCodeAccountNotFound,
				ErrorMessage:     "No account exists with the provided email.",
			}, nil
		}
		password := input["password"].(string)
		res, err := ram.AuthenticateLogin(ctx, email, password)
		if err != nil {
			switch grpc.Code(err) {
			case auth.EmailNotFound:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "No account exists with the provided email.",
				}, nil
			case auth.BadPassword:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodePasswordMismatch,
					ErrorMessage:     "The password does not match. Please try typing it again.",
				}, nil
			case auth.AccountBlocked:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "Your account has been blocked. Please contact help@sprucehealth.com.",
				}, nil
			case auth.AccountSuspended:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "Your account has been suspended. Please contact help@sprucehealth.com.",
				}, nil
			default:
				return nil, errors.InternalError(ctx, err)
			}
		}
		if res.TwoFactorRequired {
			twoFactorPhoneNumber, err := phone.ParseNumber(res.TwoFactorPhoneNumber)
			if err != nil {
				// Shouldn't fail
				return nil, errors.InternalError(ctx, err)
			}
			token, err := createAndSendSMSVerificationCode(ctx, ram, svc.serviceNumber, auth.VerificationCodeType_ACCOUNT_2FA, res.Account.ID, twoFactorPhoneNumber)
			if err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			phoneNumberLastDigits := res.TwoFactorPhoneNumber
			if len(phoneNumberLastDigits) > 2 {
				phoneNumberLastDigits = phoneNumberLastDigits[len(phoneNumberLastDigits)-2:]
			}
			return &authenticateOutput{
				ClientMutationID: mutationID,
				Success:          true,
				ErrorCode:        authenticateErrorCodeTwoFactorRequired,
				ErrorMessage:     "A verification code has been sent to your primary phone number. You must enter the received code to complete authentication.",
				Token:            token,
				PhoneNumberLastDigits: phoneNumberLastDigits,
			}, nil
		}

		token := res.Token.Value
		expires := time.Unix(int64(res.Token.ExpirationEpoch), 0)

		eh := gqlctx.SpruceHeaders(ctx)

		conc.Go(func() {
			svc.segmentio.Track(&analytics.Track{
				UserId: res.Account.ID,
				Event:  "signedin",
				Properties: map[string]interface{}{
					"platform": eh.Platform.String(),
				},
			})
			svc.segmentio.Identify(&analytics.Identify{
				UserId: res.Account.ID,
				Traits: map[string]interface{}{
					"platform": eh.Platform.String(),
				},
				Context: map[string]interface{}{
					"ip":        remoteAddrFromParams(p),
					"userAgent": userAgentFromParams(p),
				},
			})

		})

		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		gqlctx.InPlaceWithAccount(ctx, res.Account)
		result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
		result.Set("auth_token", token)
		result.Set("auth_expiration", expires)

		return &authenticateOutput{
			ClientMutationID:    mutationID,
			Success:             true,
			Token:               token,
			Account:             transformAccountToResponse(res.Account),
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
		}, nil
	},
}

// authenticateWithCode

var authenticateWithCodeInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateWithCodeInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		},
	},
)

var authenticateWithCodeMutation = &graphql.Field{
	Type: graphql.NewNonNull(authenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateWithCodeInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})
		mutationID, _ := input["clientMutationId"].(string)
		token := input["token"].(string)
		code := input["code"].(string)
		res, err := ram.AuthenticateLoginWithCode(ctx, token, code)
		if err != nil {
			switch grpc.Code(err) {
			case auth.BadVerificationCode, codes.NotFound:
				return &authenticateOutput{
					ClientMutationID: mutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeInvalidCode,
					ErrorMessage:     "The verification code you provided is incorrect.",
				}, nil
			default:
				return nil, errors.InternalError(ctx, err)
			}
		}
		result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
		result.Set("auth_token", res.Token.Value)
		result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

		eh := gqlctx.SpruceHeaders(ctx)

		conc.Go(func() {
			svc.segmentio.Track(&analytics.Track{
				UserId: res.Account.ID,
				Event:  "signedin",
				Properties: map[string]interface{}{
					"platform": eh.Platform.String(),
				},
			})

			svc.segmentio.Identify(&analytics.Identify{
				UserId: res.Account.ID,
				Traits: map[string]interface{}{
					"platform": eh.Platform.String(),
				},
				Context: map[string]interface{}{
					"ip":        remoteAddrFromParams(p),
					"userAgent": userAgentFromParams(p),
				},
			})
		})

		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		gqlctx.InPlaceWithAccount(ctx, res.Account)
		return &authenticateOutput{
			ClientMutationID:    mutationID,
			Success:             true,
			Token:               res.Token.Value,
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
			Account:             transformAccountToResponse(res.Account),
		}, nil
	},
}

/// unauthenticate

type unauthenticateOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

var unauthenticateInputType = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "UnauthenticateInput",
	Fields: graphql.InputObjectConfigFieldMap{
		"clientMutationId": newClientMutationIDInputField(),
	},
})

// JANK: can't have an empty enum and we want this field to always exist so make it a string until it's needed
var unauthenticateErrorCodeEnum = graphql.String

var unauthenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "UnauthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientmutationIDOutputField(),
			"success":          &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"errorCode":        &graphql.Field{Type: unauthenticateErrorCodeEnum},
			"errorMessage":     &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*unauthenticateOutput)
			return ok
		},
	},
)

var unauthenticateMutation = &graphql.Field{
	Type: graphql.NewNonNull(unauthenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: unauthenticateInputType},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context
		input, _ := p.Args["input"].(map[string]interface{})
		var mutationID string
		if input != nil {
			mutationID, _ = input["clientMutationId"].(string)
		}

		token := gqlctx.AuthToken(ctx)
		if token != "" {
			if err := ram.Unauthenticate(ctx, token); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			result := p.Info.RootValue.(map[string]interface{})["result"].(conc.Map)
			result.Set("unauthenticated", true)
		}

		msg := "Unauthenticate called."
		headers := gqlctx.SpruceHeaders(ctx)
		if headers != nil && headers.DeviceID != "" {
			if err := svc.notification.DeregisterDeviceForPush(headers.DeviceID); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			msg += " Device ID: " + headers.DeviceID
		}

		golog.Infof(msg)

		return &unauthenticateOutput{
			ClientMutationID: mutationID,
			Success:          true,
		}, nil
	},
}
