package main

import (
	"fmt"
	"time"

	"github.com/segmentio/analytics-go"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/apiaccess"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/errors"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/gqlctx"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/models"
	"github.com/sprucehealth/backend/cmd/svc/baymaxgraphql/internal/raccess"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/device/devicectx"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/gqldecode"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/graphql"
	"github.com/sprucehealth/graphql/gqlerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// Token duration

const (
	tokenDurationShort  = "SHORT"
	tokenDurationMedium = "MEDIUM"
	tokenDurationLong   = "LONG"
)

var tokenDurationEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "TokenDurationEnum",
	Description: "Represents the durations of an authentication token",
	Values: graphql.EnumValueConfigMap{
		tokenDurationShort: &graphql.EnumValueConfig{
			Value:       tokenDurationShort,
			Description: "Short token duration",
		},
		tokenDurationMedium: &graphql.EnumValueConfig{
			Value:       tokenDurationMedium,
			Description: "Medium token duration",
		},
		tokenDurationLong: &graphql.EnumValueConfig{
			Value:       tokenDurationLong,
			Description: "Long token duration",
		},
	},
})

// authenticate

const (
	authenticateErrorCodeTwoFactorRequired         = "TWO_FACTOR_REQUIRED"
	authenticateErrorCodeAccountNotFound           = "ACCOUNT_NOT_FOUND"
	authenticateErrorCodePasswordMismatch          = "PASSWORD_MISMATCH"
	authenticateErrorCodeInvalidCode               = "INVALID_CODE"
	authenticateErrorCodePatientPlatformNotAllowed = "PATIENT_PLATFORM_NOT_ALLOWED"
)

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
		authenticateErrorCodePatientPlatformNotAllowed: &graphql.EnumValueConfig{
			Value:       authenticateErrorCodePatientPlatformNotAllowed,
			Description: "Patient accounts are not allowed to authenticate on this platform",
		},
	},
})

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

var authenticateOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "AuthenticatePayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
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

type authenticateInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	Email            string `gql:"email,nonempty"`
	Password         string `gql:"password,nonempty"`
	Duration         string `gql:"duration"`
}

var authenticateInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"email":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"password":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"duration":         &graphql.InputObjectFieldConfig{Type: tokenDurationEnum},
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

		var in authenticateInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}
		if !validate.Email(in.Email) {
			return &authenticateOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        authenticateErrorCodeAccountNotFound,
				ErrorMessage:     "No account exists with the provided email.",
			}, nil
		}
		if in.Duration == "" {
			in.Duration = auth.TokenDuration_SHORT.String()
		}
		res, err := ram.AuthenticateLogin(ctx, in.Email, in.Password, auth.TokenDuration(auth.TokenDuration_value[in.Duration]))
		if err != nil {
			switch grpc.Code(err) {
			case auth.EmailNotFound:
				return &authenticateOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "No account exists with the provided email.",
				}, nil
			case auth.BadPassword:
				return &authenticateOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodePasswordMismatch,
					ErrorMessage:     "The password does not match. Please try typing it again.",
				}, nil
			case auth.AccountBlocked:
				return &authenticateOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "Your account has been blocked. Please contact help@sprucehealth.com.",
				}, nil
			case auth.AccountSuspended:
				return &authenticateOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeAccountNotFound,
					ErrorMessage:     "Your account has been suspended. Please contact help@sprucehealth.com.",
				}, nil
			default:
				return nil, errors.InternalError(ctx, err)
			}
		}
		headers := devicectx.SpruceHeaders(ctx)
		if res.Account.Type == auth.AccountType_PATIENT && (headers.Platform != device.Android && headers.Platform != device.IOS) {
			return &authenticateOutput{
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        authenticateErrorCodePatientPlatformNotAllowed,
				ErrorMessage:     "Patient accounts may only log in on Android and iOS.",
			}, nil
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
				ClientMutationID: in.ClientMutationID,
				Success:          false,
				ErrorCode:        authenticateErrorCodeTwoFactorRequired,
				ErrorMessage:     "A verification code has been sent to your primary phone number. You must enter the received code to complete authentication.",
				Token:            token,
				PhoneNumberLastDigits: phoneNumberLastDigits,
			}, nil
		}

		token := res.Token.Value
		expires := time.Unix(int64(res.Token.ExpirationEpoch), 0)

		// Track the authentication
		conc.Go(func() { trackAuthentication(p, res.Account.ID, devicectx.SpruceHeaders(ctx)) })

		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		gqlctx.InPlaceWithAccount(ctx, res.Account)
		result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
		result.Set("auth_token", token)
		result.Set("auth_expiration", expires)

		return &authenticateOutput{
			ClientMutationID:    in.ClientMutationID,
			Success:             true,
			Token:               token,
			Account:             transformAccountToResponse(res.Account),
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
		}, nil
	},
}

func trackAuthentication(p graphql.ResolveParams, accountID string, eh *device.SpruceHeaders) {
	svc := serviceFromParams(p)
	svc.segmentio.Track(&analytics.Track{
		UserId: accountID,
		Event:  "signedin",
		Properties: map[string]interface{}{
			"platform": eh.Platform.String(),
		},
	})
	svc.segmentio.Identify(&analytics.Identify{
		UserId: accountID,
		Traits: map[string]interface{}{
			"platform": eh.Platform.String(),
		},
		Context: map[string]interface{}{
			"ip":        remoteAddrFromParams(p),
			"userAgent": userAgentFromParams(p),
		},
	})
}

// authenticateWithCode

type authenticateWithCodeInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	Token            string `gql:"token,nonempty"`
	Code             string `gql:"code,nonempty"`
	Duration         string `gql:"duration"`
}

var authenticateWithCodeInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "AuthenticateWithCodeInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"token":            &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"code":             &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
			"duration":         &graphql.InputObjectFieldConfig{Type: tokenDurationEnum},
		},
	},
)

var authenticateWithCodeMutation = &graphql.Field{
	Type: graphql.NewNonNull(authenticateOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(authenticateWithCodeInputType)},
	},
	Resolve: func(p graphql.ResolveParams) (interface{}, error) {
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		var in authenticateWithCodeInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}
		if in.Duration == "" {
			in.Duration = auth.TokenDuration_SHORT.String()
		}
		res, err := ram.AuthenticateLoginWithCode(ctx, in.Token, in.Code, auth.TokenDuration(auth.TokenDuration_value[in.Duration]))
		if err != nil {
			switch grpc.Code(err) {
			case auth.BadVerificationCode, codes.NotFound:
				return &authenticateOutput{
					ClientMutationID: in.ClientMutationID,
					Success:          false,
					ErrorCode:        authenticateErrorCodeInvalidCode,
					ErrorMessage:     "The verification code you provided is incorrect.",
				}, nil
			default:
				return nil, errors.InternalError(ctx, err)
			}
		}
		result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
		result.Set("auth_token", res.Token.Value)
		result.Set("auth_expiration", time.Unix(int64(res.Token.ExpirationEpoch), 0))

		// Track the authentication
		conc.Go(func() { trackAuthentication(p, res.Account.ID, devicectx.SpruceHeaders(ctx)) })

		// TODO: updating the context this is safe for now because the GraphQL pkg serializes mutations.
		// that likely won't change, but this still isn't a great way to update the context.
		gqlctx.InPlaceWithAccount(ctx, res.Account)
		return &authenticateOutput{
			ClientMutationID:    in.ClientMutationID,
			Success:             true,
			Token:               res.Token.Value,
			ClientEncryptionKey: res.Token.ClientEncryptionKey,
			Account:             transformAccountToResponse(res.Account),
		}, nil
	},
}

// modifyTokenDuration

var modifyTokenDurationErrorCodeEnum = graphql.String

type modifyTokenDurationOutput struct {
	ClientMutationID string `json:"clientMutationId,omitempty"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	Token            string `json:"token,omitempty"`
}

var modifyTokenDurationOutputType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "ModifyTokenDurationPayload",
		Fields: graphql.Fields{
			"clientMutationId": newClientMutationIDOutputField(),
			"token":            &graphql.Field{Type: graphql.String},
		},
		IsTypeOf: func(value interface{}, info graphql.ResolveInfo) bool {
			_, ok := value.(*modifyTokenDurationOutput)
			return ok
		},
	},
)

type modifyTokenDurationInput struct {
	ClientMutationID string `gql:"clientMutationId"`
	Duration         string `gql:"duration,nonempty"`
}

var modifyTokenDurationInputType = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ModifyTokenDurationInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"clientMutationId": newClientMutationIDInputField(),
			"duration":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(tokenDurationEnum)},
		},
	},
)

var modifyTokenDurationMutation = &graphql.Field{
	Type: graphql.NewNonNull(modifyTokenDurationOutputType),
	Args: graphql.FieldConfigArgument{
		"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(modifyTokenDurationInputType)},
	},
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		var in modifyTokenDurationInput
		if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
			switch err := err.(type) {
			case gqldecode.ErrValidationFailed:
				return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
			}
			return nil, errors.InternalError(ctx, err)
		}

		token, err := ram.UpdateAuthToken(ctx, &auth.UpdateAuthTokenRequest{
			Token:    gqlctx.AuthToken(ctx),
			Duration: auth.TokenDuration(auth.TokenDuration_value[in.Duration]),
		})
		if err != nil {
			return nil, errors.InternalError(ctx, err)
		}
		result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
		result.Set("auth_token", token.Value)
		result.Set("auth_expiration", time.Unix(int64(token.ExpirationEpoch), 0))

		conc.Go(func() {
			svc.segmentio.Track(&analytics.Track{
				UserId: gqlctx.Account(ctx).ID,
				Event:  "modifytokenduration",
				Properties: map[string]interface{}{
					"platform": devicectx.SpruceHeaders(ctx).Platform.String(),
					"duration": in.Duration,
				},
			})
		})

		return &modifyTokenDurationOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
			Token:            token.Value,
		}, nil
	}),
}

/// unauthenticate

type unauthenticateOutput struct {
	ClientMutationID string `json:"clientMutationId"`
	Success          bool   `json:"success"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type unauthenticateInput struct {
	ClientMutationID string `gql:"clientMutationId"`
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
			"clientMutationId": newClientMutationIDOutputField(),
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
	Resolve: apiaccess.Authenticated(func(p graphql.ResolveParams) (interface{}, error) {
		svc := serviceFromParams(p)
		ram := raccess.ResourceAccess(p)
		ctx := p.Context

		var in unauthenticateInput
		if p.Args["input"] != nil {
			if err := gqldecode.Decode(p.Args["input"].(map[string]interface{}), &in); err != nil {
				switch err := err.(type) {
				case gqldecode.ErrValidationFailed:
					return nil, gqlerrors.FormatError(fmt.Errorf("%s is invalid: %s", err.Field, err.Reason))
				}
				return nil, errors.InternalError(ctx, err)
			}
		}

		token := gqlctx.AuthToken(ctx)
		if token != "" {
			if err := ram.Unauthenticate(ctx, token); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			result := p.Info.RootValue.(map[string]interface{})["result"].(*conc.Map)
			result.Set("unauthenticated", true)
		}

		msg := "Unauthenticate called."
		headers := devicectx.SpruceHeaders(ctx)
		if headers != nil && headers.DeviceID != "" {
			if err := svc.notification.DeregisterDeviceForPush(headers.DeviceID); err != nil {
				return nil, errors.InternalError(ctx, err)
			}
			msg += " Device ID: " + headers.DeviceID
		}

		conc.Go(func() {
			svc.segmentio.Track(&analytics.Track{
				UserId: gqlctx.Account(ctx).ID,
				Event:  "signedout",
				Properties: map[string]interface{}{
					"platform": devicectx.SpruceHeaders(ctx).Platform.String(),
				},
			})
		})

		golog.Infof(msg)

		return &unauthenticateOutput{
			ClientMutationID: in.ClientMutationID,
			Success:          true,
		}, nil
	}),
}
